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

	// Parse brief
	if idx, ok := headerMap["brief"]; ok && idx < len(record) {
		product.Brief = record[idx]
	}

	// Parse images (pipe-separated URLs/filenames)
	if idx, ok := headerMap["images"]; ok && idx < len(record) && record[idx] != "" {
		imageStrs := strings.Split(record[idx], "|")
		for _, imgStr := range imageStrs {
			imgStr = strings.TrimSpace(imgStr)
			if imgStr == "" {
				continue
			}

			// Extract extension from filename
			ext := ""
			if dotIdx := strings.LastIndex(imgStr, "."); dotIdx != -1 {
				ext = imgStr[dotIdx+1:]
			}

			// Generate UUID for name
			name := security.RandomString()

			product.Images = append(product.Images, models.File{
				ID:       security.RandomString(),
				Name:     name,
				Ext:      ext,
				OrigName: imgStr,
			})
		}
	}

	// Parse attributes (pipe-separated strings)
	if idx, ok := headerMap["attributes"]; ok && idx < len(record) && record[idx] != "" {
		attrStrs := strings.Split(record[idx], "|")
		for _, attr := range attrStrs {
			trimmed := strings.TrimSpace(attr)
			if trimmed != "" {
				product.Attributes = append(product.Attributes, trimmed)
			}
		}
	}

	// Parse variants in B3 format
	hasVariants, options, variants, err := c.parseVariantsB3(record, headerMap)
	if err != nil {
		return product, fmt.Errorf("variant parsing error: %w", err)
	}
	product.HasVariants = hasVariants
	product.Options = options
	product.Variants = variants

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

// parseVariantsB3 parses variants in B3 format: [Option:Val1;Val2]=>VariantData|VariantData
func (c *CSVImporter) parseVariantsB3(record []string, headerMap map[string]int) (bool, []models.ProductOption, []models.ProductVariant, error) {
	idx, ok := headerMap["variants"]
	if !ok || idx >= len(record) || record[idx] == "" {
		return false, nil, nil, nil
	}

	variantStr := strings.TrimSpace(record[idx])
	if variantStr == "" {
		return false, nil, nil, nil
	}

	// Find => separator
	sepIdx := strings.Index(variantStr, "=>")
	if sepIdx == -1 {
		return false, nil, nil, fmt.Errorf("invalid variant format: missing '=>' separator")
	}

	optionsStr := variantStr[:sepIdx]
	variantsStr := variantStr[sepIdx+2:]

	// Parse option definitions [OptionName:Value1;Value2;Value3]
	options := []models.ProductOption{}
	position := 0

	// Find all [...] sections
	for {
		start := strings.Index(optionsStr, "[")
		if start == -1 {
			break
		}
		end := strings.Index(optionsStr[start:], "]")
		if end == -1 {
			return false, nil, nil, fmt.Errorf("invalid option definition: unmatched brackets")
		}
		end += start

		optionDef := optionsStr[start+1 : end]
		colonIdx := strings.Index(optionDef, ":")
		if colonIdx == -1 {
			return false, nil, nil, fmt.Errorf("invalid option definition: missing ':' separator")
		}

		optionName := strings.TrimSpace(optionDef[:colonIdx])
		valuesStr := optionDef[colonIdx+1:]
		valueStrs := strings.Split(valuesStr, ";")

		if len(valueStrs) == 0 {
			return false, nil, nil, fmt.Errorf("option must have at least one value")
		}

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

		if len(values) == 0 {
			return false, nil, nil, fmt.Errorf("option must have at least one value")
		}

		options = append(options, models.ProductOption{
			Name:     optionName,
			Values:   values,
			Position: position,
		})
		position++

		if position > 3 {
			return false, nil, nil, fmt.Errorf("maximum 3 options allowed")
		}

		optionsStr = optionsStr[end+1:]
	}

	if len(options) == 0 {
		return false, nil, nil, fmt.Errorf("no valid options found")
	}

	// Parse variant data: OptionValue1,OptionValue2,PriceSurcharge,Quantity,SKU
	variants := []models.ProductVariant{}
	variantStrs := strings.Split(variantsStr, "|")

	for _, vStr := range variantStrs {
		vStr = strings.TrimSpace(vStr)
		if vStr == "" {
			continue
		}

		parts := strings.Split(vStr, ",")
		expectedParts := len(options) + 3 // option values + price + quantity + sku

		if len(parts) != expectedParts {
			return false, nil, nil, fmt.Errorf("variant data doesn't match option structure: expected %d parts, got %d", expectedParts, len(parts))
		}

		// Build option values map
		optionValues := make(map[string]string)
		for i, option := range options {
			optionValues[option.Name] = strings.TrimSpace(parts[i])
		}

		// Parse price surcharge
		priceIdx := len(options)
		price, err := strconv.Atoi(strings.TrimSpace(parts[priceIdx]))
		if err != nil {
			return false, nil, nil, fmt.Errorf("invalid price surcharge: %s", parts[priceIdx])
		}

		// Parse quantity
		qtyIdx := len(options) + 1
		qty, err := strconv.Atoi(strings.TrimSpace(parts[qtyIdx]))
		if err != nil {
			return false, nil, nil, fmt.Errorf("invalid quantity: %s", parts[qtyIdx])
		}

		// Parse SKU
		skuIdx := len(options) + 2
		sku := strings.TrimSpace(parts[skuIdx])

		variants = append(variants, models.ProductVariant{
			OptionValues:   optionValues,
			PriceSurcharge: price,
			Quantity:       qty,
			SKU:            sku,
			Active:         true,
		})
	}

	return true, options, variants, nil
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
