package queries

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/shurco/mycart/internal/models"
)

// loadDetailVariantData loads the options, option values and variants of a
// product that has variants.
func (q *ProductQueries) loadDetailVariantData(ctx context.Context, product *models.Product) error {
	if err := q.loadDetailOptions(ctx, product); err != nil {
		return err
	}
	if err := q.loadDetailOptionValues(ctx, product); err != nil {
		return err
	}
	return q.loadDetailVariants(ctx, product)
}

// loadDetailOptions loads the options of the product, in position order.
func (q *ProductQueries) loadDetailOptions(ctx context.Context, product *models.Product) error {
	// Load all options first
	optionsQuery := `
			SELECT id, name, position
			FROM product_option
			WHERE product_id = ?
			ORDER BY position`

	optionRows, err := q.DB.QueryContext(ctx, optionsQuery, product.ID)
	if err != nil {
		return err
	}

	for optionRows.Next() {
		option := models.ProductOption{ProductID: product.ID}
		if err := optionRows.Scan(&option.ID, &option.Name, &option.Position); err != nil {
			optionRows.Close()
			return err
		}
		product.Options = append(product.Options, option)
	}
	if err := optionRows.Err(); err != nil {
		optionRows.Close()
		return err
	}
	optionRows.Close()

	return nil
}

// loadDetailOptionValues attaches the values to the options loaded before it.
func (q *ProductQueries) loadDetailOptionValues(ctx context.Context, product *models.Product) error {
	if len(product.Options) == 0 {
		return nil
	}

	valuesQuery := `
				SELECT id, option_id, value, position
				FROM product_option_value
				WHERE option_id IN (SELECT id FROM product_option WHERE product_id = ?)
				ORDER BY option_id, position`

	valuesRows, err := q.DB.QueryContext(ctx, valuesQuery, product.ID)
	if err != nil {
		return err
	}

	for valuesRows.Next() {
		value := models.ProductOptionValue{}
		if err := valuesRows.Scan(&value.ID, &value.OptionID, &value.Value, &value.Position); err != nil {
			valuesRows.Close()
			return err
		}

		// Find the option and append the value
		for i := range product.Options {
			if product.Options[i].ID == value.OptionID {
				product.Options[i].Values = append(product.Options[i].Values, value)
				break
			}
		}
	}
	if err := valuesRows.Err(); err != nil {
		valuesRows.Close()
		return err
	}
	valuesRows.Close()

	return nil
}

// loadDetailVariants loads the live variants of the product.
func (q *ProductQueries) loadDetailVariants(ctx context.Context, product *models.Product) error {
	// Load variants
	variantsQuery := `
			SELECT id, sku, price_surcharge, quantity, option_values, active
			FROM product_variant
			WHERE product_id = ? AND deleted = FALSE`

	variantRows, err := q.DB.QueryContext(ctx, variantsQuery, product.ID)
	if err != nil {
		return err
	}

	for variantRows.Next() {
		variant := models.ProductVariant{ProductID: product.ID}
		var optionValues string
		var sku sql.NullString

		if err := variantRows.Scan(&variant.ID, &sku, &variant.PriceSurcharge, &variant.Quantity, &optionValues, &variant.Active); err != nil {
			variantRows.Close()
			return err
		}

		if sku.Valid {
			variant.SKU = sku.String
		}

		if err := json.Unmarshal([]byte(optionValues), &variant.OptionValues); err != nil {
			variantRows.Close()
			return err
		}

		product.Variants = append(product.Variants, variant)
	}
	if err := variantRows.Err(); err != nil {
		variantRows.Close()
		return err
	}
	variantRows.Close()

	return nil
}
