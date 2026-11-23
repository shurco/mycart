package queries

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/errors"
)

// Product retrieves a product by its ID, with the option to fetch private or public data.
func (q *ProductQueries) Product(ctx context.Context, private bool, id string) (*models.Product, error) {
	product := &models.Product{}

	var raw productDetailRaw
	err := q.DB.QueryRowContext(ctx, q.productDetailQuery(private), id).Scan(raw.scanArgs(product, private)...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.ErrProductNotFound
		}
		return nil, err
	}

	if err := raw.decode(product, private); err != nil {
		return nil, err
	}

	// Load options and variants if product has variants
	if product.HasVariants {
		if err := q.loadDetailVariantData(ctx, product); err != nil {
			return nil, err
		}
	}

	return product, nil
}

// productDetailQuery builds the single-product SELECT: by id for the admin, by
// slug for the storefront.
func (q *ProductQueries) productDetailQuery(private bool) string {
	d := q.DB.Dialect()

	// The images are collected by a scalar subquery rather than a LEFT JOIN +
	// GROUP BY. The join produced an all-null placeholder row for a product
	// without images, which then had to be filtered out by comparing the
	// aggregated JSON against a literal — and PostgreSQL and SQLite format that
	// same JSON differently. A subquery simply yields NULL instead.
	imageObj := d.JSONObject("'id', pi.id, 'name', pi.name, 'ext', pi.ext, " +
		"'position', pi.position, 'is_representative', " + d.JSONBool("pi.is_representative"))
	// Wrap in subquery to ORDER BY before aggregation
	imagesJSON := d.JSONAgg(imageObj)

	query := fmt.Sprintf(`
			SELECT
				product.id,
				product.name,
				product.brief,
				product."desc",
				product.slug,
				product.amount,
				product.quantity,
				product.sku,
				product.has_variants,
				product.active,
				product.metadata,
				product.attribute,
				product.digital,
				product.seo,
				(SELECT %s FROM (SELECT * FROM product_image WHERE product_id = product.id ORDER BY position, id) pi) as images,
				%s,
				%s
	`, imagesJSON, d.Epoch("product.created"), d.Epoch("product.updated"))

	// Добавляем вычисление digital_filled для приватных запросов
	if private {
		query += `, (EXISTS(SELECT 1 FROM digital_data WHERE digital_data.product_id = product.id AND digital_data.cart_id IS NULL) OR
				EXISTS(SELECT 1 FROM digital_file WHERE digital_file.product_id = product.id)) AS digital_filled
			FROM product
			WHERE product.id = ?`
	} else {
		query += `
			FROM product
			WHERE product.slug = ? AND product.deleted = FALSE AND product.active = TRUE`
	}

	return query
}

// productDetailRaw holds the nullable columns of the single-product SELECT
// until they are decoded into a models.Product.
type productDetailRaw struct {
	images, metadata, attributes, digitalType, seo, sku sql.NullString
	updated, quantity                                   sql.NullInt64
	digitalFilled, hasVariants                          sql.NullBool
}

// scanArgs lists the destinations of the columns productDetailQuery selects.
func (r *productDetailRaw) scanArgs(product *models.Product, private bool) []any {
	scanArgs := []any{
		&product.ID,
		&product.Name,
		&product.Brief,
		&product.Description,
		&product.Slug,
		&product.Amount,
		&r.quantity,
		&r.sku,
		&r.hasVariants,
		&product.Active,
		&r.metadata,
		&r.attributes,
		&r.digitalType,
		&r.seo,
		&r.images,
		&product.Created,
		&r.updated,
	}

	// Добавляем digital_filled в scanArgs для приватных запросов
	if private {
		scanArgs = append(scanArgs, &r.digitalFilled)
	}

	return scanArgs
}

// decode copies the scanned columns into product.
func (r *productDetailRaw) decode(product *models.Product, private bool) error {
	if r.updated.Valid {
		product.Updated = r.updated.Int64
	}

	if r.images.Valid {
		if err := decodeFiles(r.images.String, &product.Images); err != nil {
			return err
		}
	}

	if r.attributes.Valid {
		if err := json.Unmarshal([]byte(r.attributes.String), &product.Attributes); err != nil {
			return err
		}
	}

	if r.metadata.Valid {
		if err := json.Unmarshal([]byte(r.metadata.String), &product.Metadata); err != nil {
			return err
		}
	}

	r.decodeDigital(product, private)

	if r.seo.Valid {
		if err := json.Unmarshal([]byte(r.seo.String), &product.Seo); err != nil {
			return err
		}
	}

	if r.quantity.Valid {
		product.Quantity = int(r.quantity.Int64)
	}

	if r.sku.Valid {
		product.SKU = r.sku.String
	}

	if r.hasVariants.Valid {
		product.HasVariants = r.hasVariants.Bool
	}

	return nil
}

// decodeDigital copies the delivery type, and for private reads whether the
// product has anything to deliver, into product.
func (r *productDetailRaw) decodeDigital(product *models.Product, private bool) {
	product.Digital.Type = r.digitalType.String

	// Устанавливаем digital.filled для приватных запросов
	if private && r.digitalType.Valid {
		if r.digitalFilled.Valid {
			product.Digital.Filled = r.digitalFilled.Bool
		} else {
			product.Digital.Filled = false
		}
	}
}
