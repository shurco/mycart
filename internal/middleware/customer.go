package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/golang-jwt/jwt/v5"

	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/jwtutil"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// CookieCustomerToken is the cookie a storefront session is carried in.
//
// It is deliberately not the admin's "token": both surfaces are served from the
// same origin, so a shared name would let signing in on one silently sign the
// other out.
const CookieCustomerToken = "customer_token"

// Locals keys under which CustomerJWTProtected publishes the authenticated
// session. Handlers read them through CustomerID, CustomerSessionID and
// CustomerEmail.
const (
	LocalsCustomerID        = "customer_id"
	LocalsCustomerSessionID = "customer_session_id"
	// LocalsCustomerEmail is the address of the authenticated customer. It is
	// published because everything the cabinet owns is keyed on the address
	// rather than on the account id: a purchase made before the account existed
	// is still the buyer's, and the carts table has no account column to point
	// at. Read it through CustomerEmail.
	LocalsCustomerEmail = "customer_email"
)

// customerTokenExtractor looks for a storefront token in the Authorization
// header first and in the cabinet cookie second, mirroring the admin surface.
var customerTokenExtractor = extractors.Chain(
	extractors.FromAuthHeader("Bearer"),
	extractors.FromCookie(CookieCustomerToken),
)

// AccountEnabled gates the storefront account API on the account_enabled
// setting.
//
// It answers 404 rather than 403: an installation that does not use the cabinet
// should present no sign-in form and no hint that one exists. The routes stay
// registered either way, so switching the setting takes effect on the next
// request instead of on the next restart.
func AccountEnabled() fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		enabled, err := queries.DB().AccountEnabled(ctx)
		if err != nil {
			logging.New().ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
		if !enabled {
			return webutil.StatusNotFound(c)
		}

		return c.Next()
	}
}

// CustomerID returns the id of the authenticated customer, or "" when the
// request did not come through CustomerJWTProtected.
func CustomerID(c fiber.Ctx) string {
	id, _ := c.Locals(LocalsCustomerID).(string)
	return id
}

// CustomerSessionID returns the session row id of the authenticated customer,
// or "" when the request did not come through CustomerJWTProtected.
func CustomerSessionID(c fiber.Ctx) string {
	id, _ := c.Locals(LocalsCustomerSessionID).(string)
	return id
}

// CustomerEmail returns the address of the authenticated customer, or "" when
// the request did not come through CustomerJWTProtected.
func CustomerEmail(c fiber.Ctx) string {
	email, _ := c.Locals(LocalsCustomerEmail).(string)
	return email
}

// CustomerJWTProtected guards the storefront cabinet endpoints.
//
// It is the admin JWTProtected's counterpart, and it differs from it in the
// three ways that keep the two surfaces apart: a signing key of its own
// (account_jwt_secret, never the admin jwt_secret), a cookie of its own, and a
// role check on the session row. All three have to agree before a request gets
// through, and every way of failing produces the same answer.
//
// Written by hand rather than assembled from the JWT middleware the admin uses,
// because this one has to publish who the session belongs to: the contrib
// middleware keeps the parsed token in an unexported context key, and reading
// it back means going through a lookup that prefers the request context over
// Fiber's locals — which the token is not in.
func CustomerJWTProtected() fiber.Handler {
	return func(c fiber.Ctx) error {
		raw, err := customerTokenExtractor.Extract(c)
		if err != nil {
			return customerUnauthorized(c)
		}

		ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
		defer cancel()

		secret, err := queries.DB().AccountSecret(ctx)
		if err != nil {
			logging.New().ErrorStack(err)
			return customerUnauthorized(c)
		}

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
			// Reject unexpected signing methods to prevent "alg confusion"
			// attacks (e.g. alg=none or asymmetric algorithms impersonating
			// HMAC). The admin surface enforces the same rule.
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			if err := jwtutil.ValidateHeader(t); err != nil {
				return nil, err
			}
			return []byte(secret), nil
		}, jwtutil.ParserOptions()...)
		if err != nil || !token.Valid {
			return customerUnauthorized(c)
		}

		sessionID, err := jwtutil.SessionID(claims)
		if err != nil {
			return customerUnauthorized(c)
		}

		// The session row is what makes a signature revocable: a signed token
		// is only accepted while its row exists, so signing out — or the row
		// expiring — ends the session before the token's own TTL would.
		value, err := queries.DB().GetSession(ctx, sessionID)
		if err != nil {
			return customerUnauthorized(c)
		}

		role, customerID := queries.ParseSessionValue(value)
		if role != queries.SessionRoleCustomer || customerID == "" {
			return customerUnauthorized(c)
		}

		// The account behind the session must still exist and be unblocked.
		//
		// Blocking a customer through the admin API revokes their sessions, so
		// this looks redundant — but revocation only covers the paths that go
		// through that API. Reading the flag here is what makes it an
		// authorisation input rather than a label: a row flipped directly in
		// the database ends the cabinet access it was meant to end, and a
		// session minted in the moment before a block does not survive it.
		customer, err := queries.DB().CustomerByID(ctx, customerID)
		if err != nil {
			if !errors.Is(err, errors.ErrCustomerNotFound) {
				logging.New().ErrorStack(err)
			}
			return customerUnauthorized(c)
		}
		if !customer.Active {
			return customerUnauthorized(c)
		}

		c.Locals(LocalsCustomerID, customerID)
		c.Locals(LocalsCustomerSessionID, sessionID)
		c.Locals(LocalsCustomerEmail, customer.Email)

		return c.Next()
	}
}

// customerUnauthorized is the single response every rejection produces, so that
// an expired session, a forged token and a session belonging to another surface
// are indistinguishable from the outside.
func customerUnauthorized(c fiber.Ctx) error {
	return webutil.Response(c, fiber.StatusUnauthorized, "unauthorized", "invalid or expired token")
}
