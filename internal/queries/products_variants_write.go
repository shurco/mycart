package queries

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/internal/models"
)

// AddProductWithVariants adds a product with its options and variants in a transaction
func (q *ProductQueries) AddProductWithVariants(ctx context.Context, product *models.Product) (*models.Product, error) {
	tx, err := q.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Insert product
	if err = q.insertProductRow(ctx, tx, product); err != nil {
		return nil, err
	}

	// 2. Insert product images
	if err = q.insertProductImages(ctx, tx, product.ID, product.Images); err != nil {
		return nil, err
	}

	// 3. Insert options and option values
	if err = q.insertProductOptions(ctx, tx, product.ID, product.Options); err != nil {
		return nil, err
	}

	// 4. Insert variants with relationships and images
	if err = q.insertProductVariants(ctx, tx, product); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return product, nil
}

// insertProductRow inserts the product row itself within a transaction
func (q *ProductQueries) insertProductRow(ctx context.Context, tx *database.Tx, product *models.Product) error {
	query := `
		INSERT INTO product (
			id, name, brief, "desc", slug, amount, quantity, sku,
			has_variants, metadata, attribute, seo, digital, active
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	metadata, attributes, seo, err := q.marshalProductJSON(product)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, query,
		product.ID,
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
		product.Digital.Type,
		product.Active,
	)
	if err != nil {
		return fmt.Errorf("insert product: %w", err)
	}

	return nil
}

// insertProductImages inserts all product images within a transaction
func (q *ProductQueries) insertProductImages(ctx context.Context, tx *database.Tx, productID string, images []models.File) error {
	for i, img := range images {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO product_image (id, product_id, name, ext, orig_name, position)
			VALUES (?, ?, ?, ?, ?, ?)
		`, img.ID, productID, img.Name, img.Ext, img.OrigName, i)
		if err != nil {
			return fmt.Errorf("insert image %d: %w", i, err)
		}
	}
	return nil
}

// insertProductOptions inserts options and their values within a transaction
func (q *ProductQueries) insertProductOptions(ctx context.Context, tx *database.Tx, productID string, options []models.ProductOption) error {
	for _, option := range options {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO product_option (id, product_id, name, position)
			VALUES (?, ?, ?, ?)
		`, option.ID, productID, option.Name, option.Position)
		if err != nil {
			return fmt.Errorf("insert option %s: %w", option.Name, err)
		}

		for _, value := range option.Values {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO product_option_value (id, option_id, value, position)
				VALUES (?, ?, ?, ?)
			`, value.ID, option.ID, value.Value, value.Position)
			if err != nil {
				return fmt.Errorf("insert option value %s: %w", value.Value, err)
			}
		}
	}
	return nil
}

// insertProductVariants inserts variants with relationships and images within a transaction
func (q *ProductQueries) insertProductVariants(ctx context.Context, tx *database.Tx, product *models.Product) error {
	for _, variant := range product.Variants {
		// Marshal option values to JSON
		optionValuesJSON, err := json.Marshal(variant.OptionValues)
		if err != nil {
			return fmt.Errorf("marshal option values: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO product_variant (
				id, product_id, sku, price_surcharge, quantity, option_values, active
			) VALUES (?, ?, ?, ?, ?, ?, ?)
		`, variant.ID, product.ID, skuOrNull(variant.SKU), variant.PriceSurcharge, variant.Quantity, string(optionValuesJSON), variant.Active)
		if err != nil {
			return fmt.Errorf("insert variant %s: %w", variant.SKU, err)
		}

		if err = q.insertVariantOptions(ctx, tx, product.ID, variant); err != nil {
			return err
		}

		if err = q.insertVariantImages(ctx, tx, variant); err != nil {
			return err
		}
	}
	return nil
}

// insertVariantOptions links a variant to the option values it is made of
func (q *ProductQueries) insertVariantOptions(ctx context.Context, tx *database.Tx, productID string, variant models.ProductVariant) error {
	for optionName, optionValue := range variant.OptionValues {
		// Find the option value ID
		var optionValueID string
		err := tx.QueryRowContext(ctx, `
				SELECT pov.id
				FROM product_option_value pov
				JOIN product_option po ON pov.option_id = po.id
				WHERE po.product_id = ? AND po.name = ? AND pov.value = ?
			`, productID, optionName, optionValue).Scan(&optionValueID)
		if err != nil {
			return fmt.Errorf("find option value ID for %s=%s: %w", optionName, optionValue, err)
		}

		_, err = tx.ExecContext(ctx, `
				INSERT INTO product_variant_option (variant_id, option_value_id)
				VALUES (?, ?)
			`, variant.ID, optionValueID)
		if err != nil {
			return fmt.Errorf("insert variant-option relationship: %w", err)
		}
	}
	return nil
}

// insertVariantImages inserts the images of a variant
func (q *ProductQueries) insertVariantImages(ctx context.Context, tx *database.Tx, variant models.ProductVariant) error {
	for i, img := range variant.Images {
		_, err := tx.ExecContext(ctx, `
				INSERT INTO product_variant_image (id, variant_id, name, ext, orig_name, position)
				VALUES (?, ?, ?, ?, ?, ?)
			`, img.ID, variant.ID, img.Name, img.Ext, img.OrigName, i)
		if err != nil {
			return fmt.Errorf("insert variant image %d: %w", i, err)
		}
	}
	return nil
}
