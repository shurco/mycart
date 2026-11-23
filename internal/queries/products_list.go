package queries

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/errors"
)

// ListProducts retrieves a list of products from the database.
// If cartID is provided, it will also include digital products that were purchased in that cart.
func (q *ProductQueries) ListProducts(ctx context.Context, private bool, limit, offset int, cartID string, idList ...models.CartProduct) (*models.Products, error) {
	currency, err := NewBase(q.DB).GetSettingByKey(ctx, "currency")
	if err != nil {
		return nil, err
	}

	products := &models.Products{
		Currency: currency["currency"].Value.(string),
	}

	query := q.productListSelect(private)
	queryPublic, queryAddon, params, countParams := productListFilter(private, cartID, idList)

	query += queryPublic
	query += queryAddon

	// Deterministic ordering is required before LIMIT/OFFSET so paginated
	// results stay stable across requests.
	query += " ORDER BY product.created DESC, product.id"

	// Add pagination
	query, params = paginate(query, params, limit, offset)

	rows, err := q.DB.QueryContext(ctx, query, params...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		product, err := scanProductListRow(rows, private)
		if err != nil {
			return nil, err
		}
		products.Products = append(products.Products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	products.Total, err = q.countListedProducts(ctx, queryPublic, queryAddon, countParams)
	if err != nil {
		return nil, err
	}

	return products, nil
}

// productListSelect builds the SELECT of the product listing, up to but not
// including the WHERE clause.
func (q *ProductQueries) productListSelect(private bool) string {
	// Build variants subquery based on private mode
	// For private mode (e.g., validation), include all variants
	d := q.DB.Dialect()

	// For public mode (e.g., storefront), only show active variants
	variantsFilter := ""
	if !private {
		variantsFilter = " AND active = TRUE"
	}

	// The aggregates are built from dialect fragments rather than written out
	// literally: SQLite and PostgreSQL disagree on both the function names and
	// on how a text column becomes a nested JSON value.
	imageObj := d.JSONObject(
		"'id', pi.id, 'name', pi.name, 'ext', pi.ext, " +
			"'position', pi.position, 'is_representative', " + d.JSONBool("pi.is_representative"))
	// Wrap in subquery to ORDER BY before aggregation
	imagesJSON := d.JSONAgg(imageObj)
	variantsJSON := d.JSONAgg(d.JSONObject(
		"'id', product_variant.id, 'sku', product_variant.sku, 'quantity', product_variant.quantity, " +
			"'price_surcharge', product_variant.price_surcharge, " +
			"'option_values', " + d.JSONValue("product_variant.option_values") + ", " +
			"'active', " + d.JSONBool("product_variant.active")))

	// No DISTINCT: the only table in FROM is product, and the two EXISTS below
	// are folded into the same row, so no row can be duplicated. DISTINCT would
	// also break the ORDER BY on PostgreSQL, which requires every ORDER BY
	// expression to appear in the select list.
	return fmt.Sprintf(`
			SELECT
			  product.id,
				product.name,
				product.brief,
				product.slug,
				product.amount,
				product.quantity,
				product.has_variants,
				product.active,
				product.digital,
				(EXISTS(SELECT 1 FROM digital_data WHERE digital_data.product_id = product.id AND digital_data.cart_id IS NULL) OR
				EXISTS(SELECT 1 FROM digital_file WHERE digital_file.product_id = product.id)) AS digital_filled,
				(SELECT %s FROM (SELECT * FROM product_image WHERE product_id = product.id ORDER BY position, id) pi) as image,
				(SELECT %s FROM product_variant WHERE product_id = product.id%s) as variants,
				%s
			FROM product
		`, imagesJSON, variantsJSON, variantsFilter, d.Epoch("created"))
}

// productListFilter builds the WHERE clause of the product listing and the
// parameters of its page query and of its count query.
func productListFilter(private bool, cartID string, idList []models.CartProduct) (queryPublic, queryAddon string, params, countParams []any) {
	if !private {
		queryPublic, params = publicProductFilter(cartID)
		countParams = append(countParams, params...)
	} else {
		// For private mode, add basic WHERE clause to filter deleted products
		queryPublic = " WHERE product.deleted = FALSE "
	}

	if len(idList) > 0 {
		for _, item := range idList {
			params = append(params, item.ProductID)
			countParams = append(countParams, item.ProductID)
		}
		queryAddon = fmt.Sprintf("AND product.id IN (%s)", inPlaceholders(len(idList)))
	}

	return queryPublic, queryAddon, params, countParams
}

// scanProductListRow reads one row of the product listing.
func scanProductListRow(rows *sql.Rows, private bool) (models.Product, error) {
	var image, variants, digitalType sql.NullString
	var digitalFilled, hasVariants sql.NullBool
	product := models.Product{}
	err := rows.Scan(
		&product.ID,
		&product.Name,
		&product.Brief,
		&product.Slug,
		&product.Amount,
		&product.Quantity,
		&hasVariants,
		&product.Active,
		&digitalType,
		&digitalFilled,
		&image,
		&variants,
		&product.Created,
	)
	if err != nil {
		return models.Product{}, err
	}

	if image.Valid {
		if err := decodeFiles(image.String, &product.Images); err != nil {
			return models.Product{}, err
		}
	}

	if hasVariants.Valid {
		product.HasVariants = hasVariants.Bool
	}

	if variants.Valid {
		if err := decodeVariants(variants.String, &product.Variants); err != nil {
			return models.Product{}, err
		}
	}

	product.Digital.Type = digitalType.String
	if private && digitalType.Valid {
		if digitalFilled.Valid {
			product.Digital.Filled = digitalFilled.Bool
		} else {
			product.Digital.Filled = false
		}
	}

	return product, nil
}

// countListedProducts counts the products the listing matches, without pagination.
func (q *ProductQueries) countListedProducts(ctx context.Context, queryPublic, queryAddon string, params []any) (int, error) {
	// Count total records (without pagination params)
	countQuery := `SELECT COUNT(DISTINCT product.id) FROM product`
	countQuery += queryPublic
	var total int
	err := q.DB.QueryRowContext(ctx, countQuery+queryAddon, params...).Scan(&total)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	return total, nil
}

// publicProductFilter builds the WHERE clause for storefront product queries.
// When cartID is set, purchased items remain visible even if the product was deactivated.
func publicProductFilter(cartID string) (string, []any) {
	if cartID != "" {
		return `
			WHERE product.deleted = FALSE AND (
				product.active = TRUE
				OR EXISTS (
					SELECT 1 FROM digital_data
					WHERE digital_data.product_id = product.id
					AND digital_data.cart_id = ?
				)
			)
		`, []any{cartID}
	}
	return ` WHERE product.deleted = FALSE AND product.active = TRUE `, nil
}
