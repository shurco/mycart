package queries

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/errors"
)

// CartQueries is a struct that holds a dialect-aware database handle.
// Calls on DB are automatically rebound for the configured dialect.
type CartQueries struct {
	DB *database.Conn
}

// PaymentList retrieves the status of different payment methods from the database.
func (q *CartQueries) PaymentList(ctx context.Context) (map[string]bool, error) {
	payments := map[string]bool{}
	keys := []any{
		"stripe_active", "paypal_active", "spectrocoin_active", "coinbase_active", "portone_active",
	}

	query := fmt.Sprintf("SELECT key, value FROM setting WHERE key IN (%s)", inPlaceholders(len(keys)))
	rows, err := q.DB.QueryContext(ctx, query, keys...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var key, value string
		err := rows.Scan(&key, &value)
		if err != nil {
			return nil, err
		}

		vBool, err := strconv.ParseBool(value)
		if err != nil {
			return nil, err
		}
		name := strings.ReplaceAll(key, "_active", "")
		payments[name] = vBool
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Dummy provider is always active (no database check needed)
	payments["dummy"] = true

	return payments, nil
}

// Cart retrieves a cart from the database using the provided cartId, or
// errors.ErrCartNotFound.
func (q *CartQueries) Cart(ctx context.Context, cartId string) (*models.Cart, error) {
	query := q.cartByIDQuery()

	var email, paymentID, cartJSON sql.NullString
	var created, updated sql.NullInt64
	cart := &models.Cart{}

	err := q.DB.QueryRowContext(ctx, query, cartId).
		Scan(
			&cart.ID,
			&email,
			&cartJSON,
			&cart.AmountTotal,
			&cart.Currency,
			&paymentID,
			&cart.PaymentStatus,
			&cart.PaymentSystem,
			&created,
			&updated,
		)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil, errors.ErrCartNotFound
		}
		return nil, err
	}

	cart.Email = email.String
	cart.PaymentID = paymentID.String
	if created.Valid {
		cart.Created = created.Int64
	}

	if updated.Valid {
		cart.Updated = updated.Int64
	}

	if err := decodeCartLines(cartJSON, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

// AddCart inserts a new cart into the database.
func (q *CartQueries) AddCart(ctx context.Context, cart *models.Cart) error {
	byteCart, err := json.Marshal(cart.Cart)
	if err != nil {
		return err
	}

	query := `INSERT INTO cart (id, email, cart, amount_total, currency, payment_id, payment_status, payment_system) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = q.DB.ExecContext(ctx, query, cart.ID, cart.Email, string(byteCart), cart.AmountTotal, cart.Currency, cart.PaymentID, cart.PaymentStatus, cart.PaymentSystem)
	return err
}

// UpdateCart updates the cart details in the database.
func (q *CartQueries) UpdateCart(ctx context.Context, cart *models.Cart) error {
	var (
		args []any
		sql  strings.Builder
	)

	sql.WriteString("UPDATE cart SET ")

	if cart.PaymentID != "" {
		sql.WriteString("payment_id = ?, ")
		args = append(args, cart.PaymentID)
	}

	if cart.PaymentStatus != "" {
		sql.WriteString("payment_status = ?, ")
		args = append(args, cart.PaymentStatus)
	}

	sql.WriteString("updated = CURRENT_TIMESTAMP WHERE id = ?")
	args = append(args, cart.ID)

	_, err := q.DB.ExecContext(ctx, sql.String(), args...)
	return err
}

// cartByIDQuery builds the single-cart SELECT, filtered by id.
func (q *CartQueries) cartByIDQuery() string {
	return fmt.Sprintf(`
	SELECT 
    id, 
    email, 
    cart,
    amount_total,
    currency,
    payment_id,
    payment_status,
    payment_system,
    %s,
    %s
	FROM cart
	WHERE id = ?
	`, q.DB.Dialect().Epoch("created"), q.DB.Dialect().Epoch("updated"))
}

// decodeCartLines unmarshals the cart JSON column into the lines of cart.
func decodeCartLines(cartJSON sql.NullString, cart *models.Cart) error {
	if cartJSON.Valid && cartJSON.String != "" {
		return json.Unmarshal([]byte(cartJSON.String), &cart.Cart)
	}
	return nil
}
