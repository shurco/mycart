package queries

import (
	"context"
	"database/sql"
	stderrors "errors"
	"strings"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/litepay"
)

// cartListSelect is the projection every cart listing shares, formatted for the
// configured dialect and stopping at the end of the column list, so each caller
// appends the FROM clause that suits it. A caller that needs the cart JSON —
// only the cabinet does — passes it as an extra column.
//
// Carts and CartsByEmail differ only in their WHERE clause, and a listing whose
// columns drift from another's is a bug that shows up as a zero amount on one
// screen and not the other — so the columns are built in one place.
func (q *CartQueries) cartListSelect(extra ...string) string {
	columns := append([]string{
		"id",
		"email",
		"amount_total",
		"currency",
		"payment_id",
		"payment_status",
		"payment_system",
		q.DB.Dialect().Epoch("created"),
		q.DB.Dialect().Epoch("updated"),
	}, extra...)

	return "\n\tSELECT\n\t\t" + strings.Join(columns, ",\n\t\t") + "\n\t"
}

// scanCartList reads a cart listing built by cartListSelect. withLines says
// whether the projection carried the cart JSON as its last column.
func scanCartList(rows *sql.Rows, withLines bool) ([]*models.Cart, error) {
	carts := []*models.Cart{}

	for rows.Next() {
		var email, paymentID, cartJSON sql.NullString
		var updated sql.NullInt64
		cart := &models.Cart{}

		dest := []any{
			&cart.ID,
			&email,
			&cart.AmountTotal,
			&cart.Currency,
			&paymentID,
			&cart.PaymentStatus,
			&cart.PaymentSystem,
			&cart.Created,
			&updated,
		}
		if withLines {
			dest = append(dest, &cartJSON)
		}

		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}

		cart.Email = email.String
		cart.PaymentID = paymentID.String
		if updated.Valid {
			cart.Updated = updated.Int64
		}

		if err := decodeCartLines(cartJSON, cart); err != nil {
			return nil, err
		}

		carts = append(carts, cart)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return carts, nil
}

// Carts retrieves a list of carts from the database.
func (q *CartQueries) Carts(ctx context.Context, limit, offset int) ([]*models.Cart, int, error) {
	query := q.cartListSelect() + "FROM cart\n\tORDER BY created DESC\n"

	// Add pagination
	query, params := paginate(query, nil, limit, offset)

	rows, err := q.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	carts, err := scanCartList(rows, false)
	if err != nil {
		return nil, 0, err
	}

	// Count total records
	var total int
	err = q.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM cart`).Scan(&total)
	if err != nil && !stderrors.Is(err, sql.ErrNoRows) {
		return nil, 0, err
	}

	return carts, total, nil
}

// CartsByEmail returns every cart created for an address, newest first, in any
// payment state.
//
// The address is matched after trimming and lower-casing, which is how the
// cabinet stores a login: checkout writes the address exactly as the buyer
// typed it, so "Anna@Example.com " and "anna@example.com" are one customer.
// Trusting the raw column instead would let a buyer's own purchases disappear
// from the cabinet they belong to. The comparison is on an expression, so no
// index on cart (email) can serve it — see the customer migration for why the
// index is nonetheless a plain one.
func (q *CartQueries) CartsByEmail(ctx context.Context, email string) ([]*models.Cart, error) {
	query := q.cartListSelect() +
		"FROM cart WHERE LOWER(TRIM(email)) = ? ORDER BY created DESC"

	rows, err := q.DB.QueryContext(ctx, query, NormalizeEmail(email))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanCartList(rows, false)
}

// PaidCartsByEmail returns the paid carts of an address, newest first, with the
// lines of each one.
//
// It is a third listing rather than a filter over CartsByEmail because the two
// callers want different things: the admin screen lists orders and shows
// nothing of what is in them, while the cabinet resolves every line against the
// catalogue. Leaving the cart JSON out of the shared projection is what keeps
// an admin page from carrying the contents of every abandoned cart along with
// it.
func (q *CartQueries) PaidCartsByEmail(ctx context.Context, email string) ([]*models.Cart, error) {
	query := q.cartListSelect("cart") +
		"FROM cart WHERE LOWER(TRIM(email)) = ? AND payment_status = ? ORDER BY created DESC"

	rows, err := q.DB.QueryContext(ctx, query, NormalizeEmail(email), litepay.PAID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanCartList(rows, true)
}
