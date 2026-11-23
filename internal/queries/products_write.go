package queries

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/security"
)

// AddProduct inserts a new product into the database and returns the product with the created timestamp.
func (q *ProductQueries) AddProduct(ctx context.Context, product *models.Product) (*models.Product, error) {
	product.ID = security.RandomString()

	metadata, attributes, seo, err := q.marshalProductJSON(product)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
			INSERT INTO product (
					id, name, amount, slug, metadata, attribute, brief, "desc", digital,
					quantity, sku, seo, active
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			RETURNING %s
	`, q.DB.Dialect().Epoch("created"))
	stmt, err := q.DB.PrepareContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stmt.Close() }()

	err = stmt.QueryRowContext(ctx,
		product.ID, product.Name, product.Amount, product.Slug,
		metadata, attributes, product.Brief, product.Description, product.Digital.Type,
		product.Quantity, skuOrNull(product.SKU), seo, product.Active,
	).Scan(&product.Created)
	if err != nil {
		return nil, err
	}

	return product, nil
}

// skuOrNull keeps the unique index on sku usable: it applies to every value
// that is not NULL, so rows without a SKU have to store NULL rather than the
// empty string — otherwise the second one would collide with the first.
func skuOrNull(sku string) any {
	if sku == "" {
		return nil
	}
	return sku
}

// UpdateProduct updates an existing product in the database with new values.
func (q *ProductQueries) UpdateProduct(ctx context.Context, product *models.Product) error {
	// Marshal JSON fields
	metadata, attributes, seo, err := q.marshalProductJSON(product)
	if err != nil {
		return err
	}

	// Start transaction for atomic updates
	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Update main product fields
	if err = q.updateProductMainFields(ctx, tx, product, metadata, attributes, seo); err != nil {
		return err
	}

	// Sync variant data
	if err = q.syncProductVariants(ctx, tx, product); err != nil {
		return err
	}

	return tx.Commit()
}

// marshalProductJSON marshals product metadata, attributes, and SEO to JSON
func (q *ProductQueries) marshalProductJSON(product *models.Product) (metadata, attributes, seo []byte, err error) {
	metadata, err = json.Marshal(product.Metadata)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal metadata: %w", err)
	}

	attributes, err = json.Marshal(product.Attributes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal attributes: %w", err)
	}

	seo, err = json.Marshal(product.Seo)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("marshal seo: %w", err)
	}

	return metadata, attributes, seo, nil
}

// updateProductMainFields executes the UPDATE statement for main product fields
func (q *ProductQueries) updateProductMainFields(ctx context.Context, tx *database.Tx, product *models.Product, metadata, attributes, seo []byte) error {
	stmt, err := tx.PrepareContext(ctx, `
		UPDATE product SET
			name = ?,
			brief = ?,
			"desc" = ?,
			slug = ?,
			amount = ?,
			quantity = ?,
			sku = ?,
			has_variants = ?,
			metadata = ?,
			attribute = ?,
			seo = ?,
			updated = CURRENT_TIMESTAMP
		WHERE id = ?
	`)
	if err != nil {
		return fmt.Errorf("prepare update statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	_, err = stmt.ExecContext(ctx,
		product.Name,
		product.Brief,
		product.Description,
		product.Slug,
		product.Amount,
		product.Quantity,
		skuOrNull(product.SKU),
		product.HasVariants,
		metadata,
		attributes,
		seo,
		product.ID,
	)
	if err != nil {
		return fmt.Errorf("execute update: %w", err)
	}

	return nil
}

// syncProductVariants handles all variant-related CRUD operations
func (q *ProductQueries) syncProductVariants(ctx context.Context, tx *database.Tx, product *models.Product) error {
	if !product.HasVariants {
		// If has_variants is false, clean up any existing variant data
		_, err := tx.ExecContext(ctx, `DELETE FROM product_option WHERE product_id = ?`, product.ID)
		if err != nil {
			return fmt.Errorf("cleanup options: %w", err)
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM product_variant WHERE product_id = ?`, product.ID)
		if err != nil {
			return fmt.Errorf("cleanup variants: %w", err)
		}
		return nil
	}

	// Delete existing options (cascades to option values)
	_, err := tx.ExecContext(ctx, `DELETE FROM product_option WHERE product_id = ?`, product.ID)
	if err != nil {
		return fmt.Errorf("delete options: %w", err)
	}

	// Delete existing variants
	_, err = tx.ExecContext(ctx, `DELETE FROM product_variant WHERE product_id = ?`, product.ID)
	if err != nil {
		return fmt.Errorf("delete variants: %w", err)
	}

	if err := q.replaceOptions(ctx, tx, product); err != nil {
		return err
	}

	return q.replaceVariants(ctx, tx, product)
}

// replaceOptions inserts the options of product and their values.
func (q *ProductQueries) replaceOptions(ctx context.Context, tx *database.Tx, product *models.Product) error {
	for _, option := range product.Options {
		optionID := security.RandomString()

		_, err := tx.ExecContext(ctx,
			`INSERT INTO product_option (id, product_id, name, position) VALUES (?, ?, ?, ?)`,
			optionID, product.ID, option.Name, option.Position,
		)
		if err != nil {
			return fmt.Errorf("insert option: %w", err)
		}

		// Insert option values
		for _, value := range option.Values {
			valueID := security.RandomString()
			_, err = tx.ExecContext(ctx,
				`INSERT INTO product_option_value (id, option_id, value, position) VALUES (?, ?, ?, ?)`,
				valueID, optionID, value.Value, value.Position,
			)
			if err != nil {
				return fmt.Errorf("insert option value: %w", err)
			}
		}
	}

	return nil
}

// replaceVariants inserts the variants of product.
func (q *ProductQueries) replaceVariants(ctx context.Context, tx *database.Tx, product *models.Product) error {
	for _, variant := range product.Variants {
		variantID := security.RandomString()

		// Marshal option_values map to JSON
		optionValuesJSON, err := json.Marshal(variant.OptionValues)
		if err != nil {
			return fmt.Errorf("marshal option values: %w", err)
		}

		_, err = tx.ExecContext(ctx,
			`INSERT INTO product_variant (id, product_id, sku, price_surcharge, quantity, option_values, active) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			variantID, product.ID, skuOrNull(variant.SKU), variant.PriceSurcharge, variant.Quantity, string(optionValuesJSON), variant.Active,
		)
		if err != nil {
			return fmt.Errorf("insert variant: %w", err)
		}
	}

	return nil
}
