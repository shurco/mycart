package queries

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/shurco/mycart/internal/models"
)

// GetProductWithVariants retrieves a product with all its options and variants
func (q *ProductQueries) GetProductWithVariants(ctx context.Context, productID string) (*models.Product, error) {
	product, err := q.loadProductBase(ctx, productID)
	if err != nil {
		return nil, err
	}

	if err = q.loadProductImages(ctx, product); err != nil {
		return nil, err
	}

	// Skip options/variants if product doesn't have variants
	if !product.HasVariants {
		return product, nil
	}

	// Continue in next part...
	return q.loadProductOptions(ctx, product)
}

// loadProductBase reads the product row and decodes its JSON columns
func (q *ProductQueries) loadProductBase(ctx context.Context, productID string) (*models.Product, error) {
	product := &models.Product{}

	// 1. Get product base data
	query := `
		SELECT id, name, brief, "desc", slug, amount, quantity, sku,
		       has_variants, metadata, attribute, digital, active
		FROM product
		WHERE id = ? AND deleted = FALSE
	`

	var metadata, attributes string
	var digitalType sql.NullString
	var sku sql.NullString

	err := q.DB.QueryRowContext(ctx, query, productID).Scan(
		&product.ID,
		&product.Name,
		&product.Brief,
		&product.Description,
		&product.Slug,
		&product.Amount,
		&product.Quantity,
		&sku,
		&product.HasVariants,
		&metadata,
		&attributes,
		&digitalType,
		&product.Active,
	)
	if err != nil {
		return nil, fmt.Errorf("query product: %w", err)
	}
	if sku.Valid {
		product.SKU = sku.String
	}

	// Parse JSON fields
	if err = json.Unmarshal([]byte(metadata), &product.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}
	if err = json.Unmarshal([]byte(attributes), &product.Attributes); err != nil {
		return nil, fmt.Errorf("unmarshal attributes: %w", err)
	}
	if digitalType.Valid {
		product.Digital.Type = digitalType.String
	}

	return product, nil
}

// loadProductImages reads the images of the product in position order
func (q *ProductQueries) loadProductImages(ctx context.Context, product *models.Product) error {
	imageRows, err := q.DB.QueryContext(ctx, `
		SELECT id, name, ext, orig_name
		FROM product_image
		WHERE product_id = ?
		ORDER BY position
	`, product.ID)
	if err != nil {
		return fmt.Errorf("query product images: %w", err)
	}
	defer imageRows.Close()

	for imageRows.Next() {
		var img models.File
		if err := imageRows.Scan(&img.ID, &img.Name, &img.Ext, &img.OrigName); err != nil {
			return fmt.Errorf("scan product image: %w", err)
		}
		product.Images = append(product.Images, img)
	}
	if err := imageRows.Err(); err != nil {
		return fmt.Errorf("iterate product images: %w", err)
	}

	return nil
}

// loadProductOptions loads options, option values, and variants for a product
func (q *ProductQueries) loadProductOptions(ctx context.Context, product *models.Product) (*models.Product, error) {
	// Get options
	optionRows, err := q.DB.QueryContext(ctx, `
		SELECT id, name, position
		FROM product_option
		WHERE product_id = ?
		ORDER BY position
	`, product.ID)
	if err != nil {
		return nil, fmt.Errorf("query options: %w", err)
	}
	defer optionRows.Close()

	for optionRows.Next() {
		option := models.ProductOption{ProductID: product.ID}
		if err := optionRows.Scan(&option.ID, &option.Name, &option.Position); err != nil {
			return nil, fmt.Errorf("scan option: %w", err)
		}
		product.Options = append(product.Options, option)
	}
	if err := optionRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate options: %w", err)
	}

	// Load option values
	if err = q.loadOptionValues(ctx, &product.Options); err != nil {
		return nil, err
	}

	// Load variants with their data
	variants, err := q.loadProductVariants(ctx, product.ID)
	if err != nil {
		return nil, err
	}
	product.Variants = variants

	return product, nil
}

// loadOptionValues loads values for all product options
func (q *ProductQueries) loadOptionValues(ctx context.Context, options *[]models.ProductOption) error {
	for i := range *options {
		valueRows, err := q.DB.QueryContext(ctx, `
			SELECT id, value, position
			FROM product_option_value
			WHERE option_id = ?
			ORDER BY position
		`, (*options)[i].ID)
		if err != nil {
			return fmt.Errorf("query option values: %w", err)
		}

		for valueRows.Next() {
			value := models.ProductOptionValue{OptionID: (*options)[i].ID}
			if err := valueRows.Scan(&value.ID, &value.Value, &value.Position); err != nil {
				valueRows.Close()
				return fmt.Errorf("scan option value: %w", err)
			}
			(*options)[i].Values = append((*options)[i].Values, value)
		}
		if err := valueRows.Err(); err != nil {
			valueRows.Close()
			return fmt.Errorf("iterate option values: %w", err)
		}
		valueRows.Close()
	}
	return nil
}

// loadProductVariants loads all variants with their option values and images
func (q *ProductQueries) loadProductVariants(ctx context.Context, productID string) ([]models.ProductVariant, error) {
	variantRows, err := q.DB.QueryContext(ctx, `
		SELECT id, sku, price_surcharge, quantity, active
		FROM product_variant
		WHERE product_id = ?
	`, productID)
	if err != nil {
		return nil, fmt.Errorf("query variants: %w", err)
	}
	defer variantRows.Close()

	var variants []models.ProductVariant

	for variantRows.Next() {
		variant := models.ProductVariant{
			ProductID:    productID,
			OptionValues: make(map[string]string),
		}

		var sku sql.NullString
		if err := variantRows.Scan(&variant.ID, &sku, &variant.PriceSurcharge, &variant.Quantity, &variant.Active); err != nil {
			return nil, fmt.Errorf("scan variant: %w", err)
		}
		if sku.Valid {
			variant.SKU = sku.String
		}

		if err := q.loadVariantOptionValues(ctx, &variant); err != nil {
			return nil, err
		}

		if err := q.loadVariantImages(ctx, &variant); err != nil {
			return nil, err
		}

		variants = append(variants, variant)
	}
	if err := variantRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate variants: %w", err)
	}

	return variants, nil
}

// loadVariantOptionValues reads the option name and value pairs a variant is made of
func (q *ProductQueries) loadVariantOptionValues(ctx context.Context, variant *models.ProductVariant) error {
	optValueRows, err := q.DB.QueryContext(ctx, `
			SELECT po.name, pov.value
			FROM product_variant_option pvo
			JOIN product_option_value pov ON pvo.option_value_id = pov.id
			JOIN product_option po ON pov.option_id = po.id
			WHERE pvo.variant_id = ?
		`, variant.ID)
	if err != nil {
		return fmt.Errorf("query variant option values: %w", err)
	}

	for optValueRows.Next() {
		var optionName, optionValue string
		if err := optValueRows.Scan(&optionName, &optionValue); err != nil {
			optValueRows.Close()
			return fmt.Errorf("scan variant option value: %w", err)
		}
		variant.OptionValues[optionName] = optionValue
	}
	if err := optValueRows.Err(); err != nil {
		optValueRows.Close()
		return fmt.Errorf("iterate variant option values: %w", err)
	}
	optValueRows.Close()

	return nil
}

// loadVariantImages reads the images of a variant in position order
func (q *ProductQueries) loadVariantImages(ctx context.Context, variant *models.ProductVariant) error {
	imgRows, err := q.DB.QueryContext(ctx, `
			SELECT id, name, ext, orig_name
			FROM product_variant_image
			WHERE variant_id = ?
			ORDER BY position
		`, variant.ID)
	if err != nil {
		return fmt.Errorf("query variant images: %w", err)
	}

	for imgRows.Next() {
		var img models.File
		if err := imgRows.Scan(&img.ID, &img.Name, &img.Ext, &img.OrigName); err != nil {
			imgRows.Close()
			return fmt.Errorf("scan variant image: %w", err)
		}
		variant.Images = append(variant.Images, img)
	}
	if err := imgRows.Err(); err != nil {
		imgRows.Close()
		return fmt.Errorf("iterate variant images: %w", err)
	}
	imgRows.Close()

	return nil
}
