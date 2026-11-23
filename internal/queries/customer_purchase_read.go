package queries

import (
	"context"
	"database/sql"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/errors"
)

// purchaseProducts reads the fields of the products a list of orders shows.
//
// Deliberately neither of the two lookups the rest of the shop uses. The public
// one hides anything the operator has not published, and a guide taken off sale
// is still owed to the people who paid for it. The admin one loads variants,
// options, metadata and images, none of which an order list shows and all of
// which the buyer is the wrong caller to hand them to.
//
// A product the shop has deleted since is absent from the answer, which is how
// a line whose product is gone drops out of the order.
func (q *CustomerQueries) purchaseProducts(ctx context.Context, productIDs []string) (map[string]*models.Product, error) {
	products := map[string]*models.Product{}
	if len(productIDs) == 0 {
		return products, nil
	}

	args := make([]any, len(productIDs))
	for i, id := range productIDs {
		args[i] = id
	}

	rows, err := q.DB.QueryContext(ctx,
		`SELECT id, name, slug, digital FROM product WHERE id IN (`+inPlaceholders(len(productIDs))+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		product := &models.Product{}
		var digital sql.NullString
		if err := rows.Scan(&product.ID, &product.Name, &product.Slug, &digital); err != nil {
			return nil, err
		}

		product.Digital.Type = digital.String
		products[product.ID] = product
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

// purchaseFiles lists the downloadable files of the given products, keyed by
// the product they belong to.
func (q *CustomerQueries) purchaseFiles(ctx context.Context, productIDs []string) (map[string][]models.CustomerPurchaseFile, error) {
	files := map[string][]models.CustomerPurchaseFile{}
	if len(productIDs) == 0 {
		return files, nil
	}

	args := make([]any, len(productIDs))
	for i, id := range productIDs {
		args[i] = id
	}

	// Ordered by the name the buyer will see rather than by the stored one: the
	// two differ only for a file stored without an upload name, and sorting on
	// the empty string would put exactly that file first. Rows of one product
	// keep that order inside the group they land in.
	rows, err := q.DB.QueryContext(ctx,
		`SELECT product_id, id, name, ext, orig_name FROM digital_file
			WHERE product_id IN (`+inPlaceholders(len(productIDs))+`)
			ORDER BY COALESCE(NULLIF(orig_name, ''), name || '.' || ext), id`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var productID, id, name, ext, origName string
		if err := rows.Scan(&productID, &id, &name, &ext, &origName); err != nil {
			return nil, err
		}

		// A row stored without a name would reach the buyer as a blank label on
		// a link; the stored name and extension are always there and make a
		// name of sorts. Same fallback the download endpoint puts into the
		// Content-Disposition header.
		if origName == "" {
			origName = name + "." + ext
		}

		files[productID] = append(files[productID], models.CustomerPurchaseFile{ID: id, OrigName: origName})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return files, nil
}

// purchaseCodes returns the licence keys the given orders claimed, keyed by the
// order and the product it claimed them for.
//
// Scoped to the orders as well as the product, and that scoping is the whole
// entitlement: a product holds one row per key, most of them unclaimed and
// waiting for a buyer, so a lookup by product alone would print the shop's
// entire remaining stock into the first cabinet that asked. The orders handed
// in are the buyer's own, which is what keeps one cabinet out of another's.
func (q *CustomerQueries) purchaseCodes(ctx context.Context, cartIDs []string) (map[string][]string, error) {
	codes := map[string][]string{}
	if len(cartIDs) == 0 {
		return codes, nil
	}

	args := make([]any, len(cartIDs))
	for i, id := range cartIDs {
		args[i] = id
	}

	rows, err := q.DB.QueryContext(ctx,
		`SELECT cart_id, product_id, content FROM digital_data
			WHERE cart_id IN (`+inPlaceholders(len(cartIDs))+`)
			ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var cartID, productID, content string
		if err := rows.Scan(&cartID, &productID, &content); err != nil {
			return nil, err
		}

		key := codeKey(cartID, productID)
		codes[key] = append(codes[key], content)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return codes, nil
}

// digitalFile reads one digital file and the product it belongs to.
func (q *CustomerQueries) digitalFile(ctx context.Context, fileID string) (*models.File, string, error) {
	file := &models.File{}
	var productID string

	err := q.DB.QueryRowContext(ctx,
		`SELECT id, name, ext, orig_name, product_id FROM digital_file WHERE id = ?`, fileID).
		Scan(&file.ID, &file.Name, &file.Ext, &file.OrigName, &productID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", errors.ErrProductNotFound
		}
		return nil, "", err
	}

	if file.OrigName == "" {
		file.OrigName = file.Name + "." + file.Ext
	}

	return file, productID, nil
}
