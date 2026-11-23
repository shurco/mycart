package queries

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/litepay"
)

// CartLetterPayment is ...
func (q *CartQueries) CartLetterPayment(ctx context.Context, email, amountPayment, paymentURL string) (*models.MessageMail, error) {
	mailLetter, err := NewBase(q.DB).GetSettingByKey(ctx, "site_name", "mail_letter_payment")
	if err != nil {
		return nil, err
	}
	letterTemplate := models.Letter{}
	if err := json.Unmarshal([]byte(mailLetter["mail_letter_payment"].Value.(string)), &letterTemplate); err != nil {
		return nil, err
	}

	mail := &models.MessageMail{
		To:     email,
		Letter: letterTemplate,
		Data: map[string]string{
			"Payment_URL":    paymentURL,
			"Site_Name":      mailLetter["site_name"].Value.(string),
			"Amount_Payment": amountPayment,
		},
	}

	return mail, nil
}

// CartLetterPurchase is ...
func (q *CartQueries) CartLetterPurchase(ctx context.Context, cartID string) (*models.MessageMail, error) {
	mail := &models.MessageMail{}

	email, products, err := q.paidCartLines(ctx, cartID)
	if err != nil {
		return nil, err
	}
	mail.To = email

	keys, files, err := q.claimPurchasedGoods(ctx, cartID, products)
	if err != nil {
		return nil, err
	}

	purchases := purchasesText(keys, files)

	// Fetch the 'mail_letter_purchase' setting value.
	mailLetter, err := NewBase(q.DB).GetSettingByKey(ctx, "email", "mail_letter_purchase")
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(mailLetter["mail_letter_purchase"].Value.(string)), &mail.Letter); err != nil {
		return nil, err
	}

	mail.Data = map[string]string{
		"Purchases":   purchases,
		"Admin_Email": mailLetter["email"].Value.(string),
	}
	mail.Files = files

	return mail, nil
}

// paidCartLines returns the buyer's address and the lines of a paid cart, or
// errors.ErrPageNotFound when there is no such paid cart.
func (q *CartQueries) paidCartLines(ctx context.Context, cartID string) (string, []models.CartProduct, error) {
	// Fetch the email, cart information, and 'email' setting in one query.
	var email, cartJSON string
	err := q.DB.QueryRowContext(ctx, `
        SELECT email, cart
        FROM cart
        WHERE payment_status = ? AND id = ?
    `, litepay.PAID, cartID).Scan(&email, &cartJSON)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return "", nil, errors.ErrPageNotFound
		}
		return "", nil, err
	}

	// Unmarshal the products from the cart JSON.
	products := []models.CartProduct{}
	if err := json.Unmarshal([]byte(cartJSON), &products); err != nil {
		return "", nil, err
	}

	return email, products, nil
}

// claimPurchasedGoods hands the buyer of cartID the files and keys of every
// digital product in products, within one transaction.
func (q *CartQueries) claimPurchasedGoods(ctx context.Context, cartID string, products []models.CartProduct) ([]models.Data, []models.File, error) {
	// Begin a transaction.
	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	keys := []models.Data{}
	files := []models.File{}
	for _, cart := range products {
		var digitalType string
		err := tx.QueryRowContext(ctx, `SELECT digital FROM product WHERE id = ?`, cart.ProductID).Scan(&digitalType)
		if err != nil {
			if stderrors.Is(err, sql.ErrNoRows) {
				return nil, nil, errors.ErrPageNotFound
			}
			return nil, nil, err
		}

		switch digitalType {
		case models.DigitalFile:
			productFiles, err := scanDigitalFiles(ctx, tx, cart.ProductID)
			if err != nil {
				return nil, nil, err
			}
			files = append(files, productFiles...)
		case models.DigitalData:
			key, err := claimDigitalData(ctx, tx, cartID, cart.ProductID)
			if err != nil {
				return nil, nil, err
			}
			keys = append(keys, key)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	return keys, files, nil
}

// purchasesText lists the claimed keys and files, numbered, for the purchase letter.
func purchasesText(keys []models.Data, files []models.File) string {
	// Construct the purchases information.
	var purchases strings.Builder
	count := 1
	if len(keys) > 0 {
		purchases.WriteString("Keys:\n")
		for _, key := range keys {
			purchases.WriteString(fmt.Sprintf("%v: %s\n", count, key.Content))
			count++
		}
	}
	if len(files) > 0 {
		purchases.WriteString("Files:\n")
		for _, file := range files {
			purchases.WriteString(fmt.Sprintf("%v: %s\n", count, file.OrigName))
			count++
		}
	}

	return purchases.String()
}

// claimDigitalData hands one unassigned digital_data row to cartID and returns
// it. Already owning a row makes the call idempotent, so a retried purchase
// mail keeps its keys.
//
// The row is claimed by an UPDATE guarded on `cart_id IS NULL`, and the claim
// only counts when it reports having changed a row. Reading a candidate and
// then updating it unconditionally — as this used to do — lets two concurrent
// purchases pick the same row and hands the same key to both buyers. SQLite hid
// the race behind `_txlock=immediate`; PostgreSQL does not.
func claimDigitalData(ctx context.Context, tx *database.Tx, cartID, productID string) (models.Data, error) {
	// Bounded so that a pathological amount of contention fails loudly instead
	// of spinning; each lost round is another buyer taking a key, so the loop
	// makes progress and this is a safety net, not the normal exit.
	const maxAttempts = 100

	for range maxAttempts {
		key := models.Data{}
		err := tx.QueryRowContext(ctx,
			`SELECT id, content FROM digital_data WHERE cart_id = ? AND product_id = ?`,
			cartID, productID).Scan(&key.ID, &key.Content)
		if err == nil {
			return key, nil
		}
		if !stderrors.Is(err, sql.ErrNoRows) {
			return models.Data{}, err
		}

		var candidate models.Data
		err = tx.QueryRowContext(ctx,
			`SELECT id, content FROM digital_data WHERE cart_id IS NULL AND product_id = ? ORDER BY id LIMIT 1`,
			productID).Scan(&candidate.ID, &candidate.Content)
		if err != nil {
			if stderrors.Is(err, sql.ErrNoRows) {
				return models.Data{}, errors.ErrPageNotFound
			}
			return models.Data{}, err
		}

		result, err := tx.ExecContext(ctx,
			`UPDATE digital_data SET cart_id = ? WHERE id = ? AND cart_id IS NULL`,
			cartID, candidate.ID)
		if err != nil {
			return models.Data{}, err
		}

		if claimed, err := result.RowsAffected(); err == nil && claimed == 1 {
			return candidate, nil
		}
	}

	return models.Data{}, fmt.Errorf("claim digital data for product %s: too much contention", productID)
}

// scanDigitalFiles loads every digital_file row for a product within the given transaction.
// Extracted from CartLetterPurchase to avoid a defer-in-loop leak when a cart contains
// many "file" products (each iteration used to accumulate an open sql.Rows until the
// enclosing function returned).
func scanDigitalFiles(ctx context.Context, tx *database.Tx, productID string) ([]models.File, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, name, ext, orig_name FROM digital_file WHERE product_id = ?`, productID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var files []models.File
	for rows.Next() {
		f := models.File{}
		if err := rows.Scan(&f.ID, &f.Name, &f.Ext, &f.OrigName); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}
