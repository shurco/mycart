package csvimport

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/security"
)

// CSVImporter handles CSV import operations
type CSVImporter struct {
	db *sql.DB
}

// NewCSVImporter creates a new CSV importer
func NewCSVImporter(db *sql.DB) *CSVImporter {
	return &CSVImporter{db: db}
}

// Required CSV columns
var requiredColumns = []string{"name", "slug", "amount", "digital"}

// ValidateAndPreview parses CSV and returns preview without importing
func (c *CSVImporter) ValidateAndPreview(file io.Reader) (*ImportResult, []models.Product, error) {
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("read header: %w", err)
	}

	// Validate required columns present
	headerMap := make(map[string]int)
	for i, col := range header {
		headerMap[col] = i
	}

	for _, req := range requiredColumns {
		if _, exists := headerMap[req]; !exists {
			return nil, nil, fmt.Errorf("missing required column: %s", req)
		}
	}

	result := &ImportResult{}
	products := []models.Product{}

	lineNum := 2 // Start from 2 (1 is header)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, Error{
				Line:    lineNum,
				Message: fmt.Sprintf("CSV parse error: %v", err),
			})
			lineNum++
			continue
		}

		product, err := c.parseProduct(record, headerMap)
		if err != nil {
			result.Errors = append(result.Errors, Error{
				Line:    lineNum,
				Message: err.Error(),
			})
			result.Skipped++
		} else {
			// Check if product exists
			exists, err := c.productExists(context.Background(), product.Slug)
			if err != nil {
				result.Errors = append(result.Errors, Error{
					Line:    lineNum,
					Message: fmt.Sprintf("database error: %v", err),
				})
			} else if exists {
				result.ToUpdate++
			} else {
				result.ToAdd++
			}
			products = append(products, product)
		}

		lineNum++
	}

	result.TotalRows = lineNum - 2
	return result, products, nil
}

// parseProduct parses a single CSV row into a Product model
func (c *CSVImporter) parseProduct(record []string, headerMap map[string]int) (models.Product, error) {
	product := models.Product{}

	// Parse required fields
	product.Name = record[headerMap["name"]]
	if product.Name == "" {
		return product, fmt.Errorf("name is required")
	}

	product.Slug = record[headerMap["slug"]]
	if product.Slug == "" {
		return product, fmt.Errorf("slug is required")
	}

	amountStr := record[headerMap["amount"]]
	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		return product, fmt.Errorf("invalid amount: %s", amountStr)
	}
	product.Amount = amount

	digitalType := record[headerMap["digital"]]
	if digitalType == "" {
		return product, fmt.Errorf("digital type is required")
	}
	product.Digital.Type = digitalType

	// Parse optional fields
	if idx, ok := headerMap["description"]; ok && idx < len(record) {
		product.Description = record[idx]
	}

	if idx, ok := headerMap["quantity"]; ok && idx < len(record) && record[idx] != "" {
		qty, err := strconv.Atoi(record[idx])
		if err != nil {
			return product, fmt.Errorf("invalid quantity: %s", record[idx])
		}
		product.Quantity = qty
	}

	if idx, ok := headerMap["sku"]; ok && idx < len(record) {
		product.SKU = record[idx]
	}

	if idx, ok := headerMap["active"]; ok && idx < len(record) {
		product.Active = record[idx] == "true" || record[idx] == "1"
	} else {
		product.Active = true
	}

	// Parse variant options
	product.HasVariants, product.Options, product.Variants = c.parseVariants(record, headerMap)

	// Generate IDs
	product.ID = security.RandomString()
	for i := range product.Options {
		product.Options[i].ID = security.RandomString()
		product.Options[i].ProductID = product.ID
		for j := range product.Options[i].Values {
			product.Options[i].Values[j].ID = security.RandomString()
			product.Options[i].Values[j].OptionID = product.Options[i].ID
		}
	}
	for i := range product.Variants {
		product.Variants[i].ID = security.RandomString()
		product.Variants[i].ProductID = product.ID
	}

	return product, nil
}

// parseVariants extracts variant data from CSV row
func (c *CSVImporter) parseVariants(record []string, headerMap map[string]int) (bool, []models.ProductOption, []models.ProductVariant) {
	options := []models.ProductOption{}
	variants := []models.ProductVariant{}

	// Check for option columns (option1_name, option2_name, option3_name)
	for i := 1; i <= 3; i++ {
		nameKey := fmt.Sprintf("option%d_name", i)
		valuesKey := fmt.Sprintf("option%d_values", i)

		nameIdx, hasName := headerMap[nameKey]
		valuesIdx, hasValues := headerMap[valuesKey]

		if !hasName || !hasValues || nameIdx >= len(record) || valuesIdx >= len(record) {
			break
		}

		optionName := record[nameIdx]
		optionValuesStr := record[valuesIdx]

		if optionName == "" || optionValuesStr == "" {
			break
		}

		// Parse option values (semicolon-separated)
		valueStrs := strings.Split(optionValuesStr, ";")
		values := []models.ProductOptionValue{}
		for pos, val := range valueStrs {
			val = strings.TrimSpace(val)
			if val != "" {
				values = append(values, models.ProductOptionValue{
					Value:    val,
					Position: pos,
				})
			}
		}

		if len(values) > 0 {
			options = append(options, models.ProductOption{
				Name:     optionName,
				Values:   values,
				Position: i - 1,
			})
		}
	}

	// If we have options, generate variants
	if len(options) > 0 {
		variants = c.generateVariants(options, record, headerMap)
		return true, options, variants
	}

	return false, options, variants
}

// generateVariants creates variant combinations from options
func (c *CSVImporter) generateVariants(options []models.ProductOption, record []string, headerMap map[string]int) []models.ProductVariant {
	// Generate cartesian product of option values
	combinations := c.cartesianProduct(options)

	// Parse variant-specific data (prices, quantities, SKUs)
	var prices []int
	var quantities []int
	var skus []string

	if idx, ok := headerMap["variant_prices"]; ok && idx < len(record) && record[idx] != "" {
		for _, p := range strings.Split(record[idx], ";") {
			price, _ := strconv.Atoi(strings.TrimSpace(p))
			prices = append(prices, price)
		}
	}

	if idx, ok := headerMap["variant_quantities"]; ok && idx < len(record) && record[idx] != "" {
		for _, q := range strings.Split(record[idx], ";") {
			qty, _ := strconv.Atoi(strings.TrimSpace(q))
			quantities = append(quantities, qty)
		}
	}

	if idx, ok := headerMap["variant_skus"]; ok && idx < len(record) && record[idx] != "" {
		for _, s := range strings.Split(record[idx], ";") {
			skus = append(skus, strings.TrimSpace(s))
		}
	}

	// Create variants
	variants := []models.ProductVariant{}
	for i, combo := range combinations {
		variant := models.ProductVariant{
			OptionValues: combo,
			Active:       true,
		}

		if i < len(prices) {
			variant.PriceSurcharge = prices[i]
		}

		if i < len(quantities) {
			variant.Quantity = quantities[i]
		} else {
			variant.Quantity = 0
		}

		if i < len(skus) && skus[i] != "" {
			variant.SKU = skus[i]
		}

		variants = append(variants, variant)
	}

	return variants
}

// cartesianProduct generates all combinations of option values
func (c *CSVImporter) cartesianProduct(options []models.ProductOption) []map[string]string {
	if len(options) == 0 {
		return []map[string]string{}
	}

	result := []map[string]string{{}}

	for _, option := range options {
		newResult := []map[string]string{}
		for _, existing := range result {
			for _, value := range option.Values {
				combo := make(map[string]string)
				for k, v := range existing {
					combo[k] = v
				}
				combo[option.Name] = value.Value
				newResult = append(newResult, combo)
			}
		}
		result = newResult
	}

	return result
}

// productExists checks if a product with the given slug exists
func (c *CSVImporter) productExists(ctx context.Context, slug string) (bool, error) {
	var count int
	err := c.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM product WHERE slug = ?", slug).Scan(&count)
	return count > 0, err
}

// Import executes the import operation
func (c *CSVImporter) Import(ctx context.Context, products []models.Product) (*ImportResult, error) {
	result := &ImportResult{}

	for _, product := range products {
		exists, err := c.productExists(ctx, product.Slug)
		if err != nil {
			result.Errors = append(result.Errors, Error{
				Message: fmt.Sprintf("Failed to check %s: %v", product.Slug, err),
			})
			continue
		}

		if exists {
			// Update logic would go here - for now, skip
			result.Skipped++
		} else {
			// Insert product (simplified - would use AddProductWithVariants in production)
			query := `INSERT INTO product (id, name, slug, desc, amount, quantity, digital, active, deleted)
			          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
			_, err = c.db.ExecContext(ctx, query,
				product.ID, product.Name, product.Slug, product.Description,
				product.Amount, product.Quantity, product.Digital.Type, product.Active, false)

			if err != nil {
				result.Errors = append(result.Errors, Error{
					Message: fmt.Sprintf("Failed to insert %s: %v", product.Slug, err),
				})
			} else {
				result.Imported++
			}
		}
	}

	result.TotalRows = len(products)
	return result, nil
}
