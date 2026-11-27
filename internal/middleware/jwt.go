package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	jwtMiddleware "github.com/gofiber/contrib/v3/jwt"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/golang-jwt/jwt/v5"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/jwtutil"
	"github.com/shurco/mycart/pkg/webutil"
)

func JWTProtected() fiber.Handler {
	config := jwtMiddleware.Config{
		KeyFunc:       customKeyFunc(),
		ErrorHandler:  jwtError,
		Extractor:     extractors.Chain(extractors.FromAuthHeader("Bearer"), extractors.FromCookie("token")),
		ParserOptions: jwtutil.ParserOptions(),
	}

	return jwtMiddleware.New(config)
}

func jwtError(c fiber.Ctx, err error) error {
	path := strings.Split(c.Path(), "/")[1]
	if path == "api" {
		if err != nil {
			if strings.Contains(err.Error(), "Missing") || strings.Contains(err.Error(), "malformed") {
				return webutil.Response(c, http.StatusBadRequest, "bad request", "missing or malformed token")
			}
			return webutil.Response(c, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
		}
	}

	return c.Redirect().To("/_/signin")
}

func customKeyFunc() jwt.Keyfunc {
	return func(t *jwt.Token) (any, error) {
		// Reject unexpected signing methods to prevent "alg confusion" attacks
		// (e.g. alg=none or asymmetric algorithms impersonating HMAC).
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		if err := jwtutil.ValidateHeader(t); err != nil {
			return nil, err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		db := queries.DB()
		settingJWT, err := queries.GetSettingByGroup[models.JWT](ctx, db)
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return nil, fmt.Errorf("database took too long to respond")
			}
			return nil, fmt.Errorf("database error: %w", err)
		}
		if settingJWT.Secret == "" {
			return nil, fmt.Errorf("JWT secret is empty or not configured")
		}

		// Revocation check: a signed token is only accepted while its session
		// row still exists (sign-out deletes the row), so leaked tokens can be
		// invalidated before their TTL expires.
		claims, ok := t.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("invalid token claims")
		}
		sessionID, err := jwtutil.SessionID(claims)
		if err != nil {
			return nil, err
		}
		value, err := db.GetSession(ctx, sessionID)
		if err != nil {
			return nil, fmt.Errorf("session not found or expired")
		}

		// Role check. The session table is shared with the storefront cabinet,
		// and both surfaces' tokens carry nothing but a session id, so the row
		// alone would otherwise be enough to be admitted here. The cabinet signs
		// with a key of its own, which already stops its tokens reaching this
		// function — this is the check that keeps holding when a future surface
		// does share the key, and it is what makes the "admin" value the admin
		// sign-in has always written mean something.
		if role, _ := queries.ParseSessionValue(value); role != queries.SessionRoleAdmin {
			return nil, fmt.Errorf("session is not an admin session")
		}

		return []byte(settingJWT.Secret), nil
	}
}
