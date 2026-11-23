package queries

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/security"
)

// DefaultAccountExpireHours is how long a cabinet session lasts when the
// operator leaves account_jwt_expire_hours at zero. Thirty days: long enough
// that a buyer is not asked to sign in again on every visit, short enough that
// an abandoned session on a shared device does not live forever.
const DefaultAccountExpireHours = 720

// accountSecretKey is the setting holding the storefront signing key.
const accountSecretKey = "account_jwt_secret"

// CustomerQueries is a struct that holds a dialect-aware database handle for
// the storefront customer accounts.
type CustomerQueries struct {
	DB *database.Conn
}

// NormalizeEmail trims and lower-cases an address so the same person cannot end
// up with two accounts, and so the lookup at sign-in matches what sign-up
// stored regardless of how the address was typed either time.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// customerExists reports whether an address already has an account. It is the
// uniqueness check of CreateCustomer, split out so the same question can be
// asked again after a losing race on the insert.
func (q *CustomerQueries) customerExists(ctx context.Context, email string) (bool, error) {
	var taken int
	if err := q.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM customer WHERE email = ?`, email).Scan(&taken); err != nil {
		return false, err
	}
	return taken > 0, nil
}

// CreateCustomer stores a new account with the password already hashed.
//
// The uniqueness check is a separate SELECT rather than reliance on the UNIQUE
// constraint, because the two engines report a violation differently and the
// caller needs to tell a taken address from a broken database. The constraint
// still backs it up: two simultaneous sign-ups for one address race past the
// SELECT, and the loser is told the address is taken rather than that the shop
// is broken.
func (q *CustomerQueries) CreateCustomer(ctx context.Context, email, password, name string) (*models.Customer, error) {
	email = NormalizeEmail(email)

	taken, err := q.customerExists(ctx, email)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, errors.ErrCustomerEmailTaken
	}

	hash, err := security.HashPassword(password)
	if err != nil {
		return nil, err
	}

	customer := &models.Customer{
		Core:     models.Core{ID: security.RandomString()},
		Email:    email,
		Password: hash,
		Name:     strings.TrimSpace(name),
		Active:   true,
	}

	query := fmt.Sprintf(
		`INSERT INTO customer (id, email, password, name, active) VALUES (?, ?, ?, ?, ?) RETURNING %s`,
		q.DB.Dialect().Epoch("created"))

	if err := q.DB.QueryRowContext(ctx, query,
		customer.ID, customer.Email, customer.Password, customer.Name, customer.Active,
	).Scan(&customer.Created); err != nil {
		// The loser of a race between two sign-ups fails here, on the UNIQUE
		// constraint. The engines spell that violation differently and the
		// query layer deliberately knows nothing about either, so the address
		// is looked up again instead: if it exists now, the caller is told what
		// a retry can act on.
		if taken, lookupErr := q.customerExists(ctx, email); lookupErr == nil && taken {
			return nil, errors.ErrCustomerEmailTaken
		}
		return nil, err
	}

	return customer, nil
}

// CustomerByEmail returns the account with the given address, or
// errors.ErrCustomerNotFound.
func (q *CustomerQueries) CustomerByEmail(ctx context.Context, email string) (*models.Customer, error) {
	return q.customer(ctx, "WHERE email = ?", NormalizeEmail(email))
}

// CustomerByID returns the account with the given id, or
// errors.ErrCustomerNotFound.
func (q *CustomerQueries) CustomerByID(ctx context.Context, id string) (*models.Customer, error) {
	return q.customer(ctx, "WHERE id = ?", id)
}

// customer runs a single-row lookup and turns "no rows" into the package's
// sentinel, so handlers branch with errors.Is instead of the driver's error.
//
// The timestamps come back as unix seconds like everywhere else, and are read
// through nullable holders: this is also the lookup the cabinet middleware
// makes on every request, so it stays a single indexed row.
func (q *CustomerQueries) customer(ctx context.Context, where string, arg any) (*models.Customer, error) {
	customer := &models.Customer{}
	var created, updated sql.NullInt64

	query := fmt.Sprintf(
		`SELECT id, email, password, name, active, %s, %s FROM customer %s`,
		q.DB.Dialect().Epoch("created"), q.DB.Dialect().Epoch("updated"), where)

	err := q.DB.QueryRowContext(ctx, query, arg).Scan(
		&customer.ID, &customer.Email, &customer.Password, &customer.Name, &customer.Active,
		&created, &updated)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.ErrCustomerNotFound
		}
		return nil, err
	}

	customer.Created = created.Int64
	customer.Updated = updated.Int64

	return customer, nil
}

// SetCustomerActive blocks or unblocks an account.
func (q *CustomerQueries) SetCustomerActive(ctx context.Context, id string, active bool) error {
	_, err := q.DB.ExecContext(ctx,
		`UPDATE customer SET active = ?, updated = CURRENT_TIMESTAMP WHERE id = ?`, active, id)
	return err
}

// SetCustomerPassword replaces the stored password hash.
//
// The hash is stored already hashed: this function is deliberately unable to
// take a plaintext password, so no caller can leave one lying in a column.
func (q *CustomerQueries) SetCustomerPassword(ctx context.Context, id, passwordHash string) error {
	_, err := q.DB.ExecContext(ctx,
		`UPDATE customer SET password = ?, updated = CURRENT_TIMESTAMP WHERE id = ?`, passwordHash, id)
	return err
}

// DeleteCustomer removes an account.
//
// The carts are left in place. They are the record of a payment that happened,
// and an operator deleting an account (a buyer asking to be forgotten, a
// duplicate created by a typo) must not thereby erase the shop's takings.
func (q *CustomerQueries) DeleteCustomer(ctx context.Context, id string) error {
	_, err := q.DB.ExecContext(ctx, `DELETE FROM customer WHERE id = ?`, id)
	return err
}

// RevokeCustomerSessions drops every cabinet session belonging to a customer.
//
// The session table is the revocation list for the storefront: a token is
// honoured only while its row exists, and the row's value carries the customer
// id, so deleting by value ends the session on every device the buyer signed in
// from rather than on the one the operator happens to know about.
func (q *CustomerQueries) RevokeCustomerSessions(ctx context.Context, customerID string) error {
	_, err := q.DB.ExecContext(ctx,
		`DELETE FROM session WHERE value = ?`, SessionValue(SessionRoleCustomer, customerID))
	return err
}
