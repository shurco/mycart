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

	if err := c.parseRequiredFields(&product, record, headerMap); err != nil {
		return product, err
	}

	c.parseOptionalFields(&product, record, headerMap)
	c.parseImages(&product, record, headerMap)
	c.parseAttributes(&product, record, headerMap)

	hasVariants, options, variants, err := c.parseVariantsB3(record, headerMap)
	if err != nil {
		return product, fmt.Errorf("variant parsing error: %w", err)
	}
	product.HasVariants = hasVariants
	product.Options = options
	product.Variants = variants

	c.generateProductIDs(&product)

	return product, nil
}

// parseRequiredFields parses required CSV fields into product
func (c *CSVImporter) parseRequiredFields(product *models.Product, record []string, headerMap map[string]int) error {
	product.Name = record[headerMap["name"]]
	if product.Name == "" {
		return fmt.Errorf("name is required")
	}

	product.Slug = record[headerMap["slug"]]
	if product.Slug == "" {
		return fmt.Errorf("slug is required")
	}

	amountStr := record[headerMap["amount"]]
	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		return fmt.Errorf("invalid amount: %s", amountStr)
	}
	product.Amount = amount

	digitalType := record[headerMap["digital"]]
	if digitalType == "" {
		return fmt.Errorf("digital type is required")
	}
	product.Digital.Type = digitalType

	return nil
}

// parseOptionalFields parses optional CSV fields into product
func (c *CSVImporter) parseOptionalFields(product *models.Product, record []string, headerMap map[string]int) {
	if idx, ok := headerMap["description"]; ok && idx < len(record) {
		product.Description = record[idx]
	}

	if idx, ok := headerMap["quantity"]; ok && idx < len(record) && record[idx] != "" {
		if qty, err := strconv.Atoi(record[idx]); err == nil {
			product.Quantity = qty
		}
	}

	if idx, ok := headerMap["sku"]; ok && idx < len(record) {
		product.SKU = record[idx]
	}

	if idx, ok := headerMap["brief"]; ok && idx < len(record) {
		product.Brief = record[idx]
	}

	if idx, ok := headerMap["active"]; ok && idx < len(record) {
		product.Active = record[idx] == "true" || record[idx] == "1"
	} else {
		product.Active = true
	}
}

// parseImages parses pipe-separated image URLs from CSV
func (c *CSVImporter) parseImages(product *models.Product, record []string, headerMap map[string]int) {
	idx, ok := headerMap["images"]
	if !ok || idx >= len(record) || record[idx] == "" {
		return
	}

	imageStrs := strings.Split(record[idx], "|")
	for _, imgStr := range imageStrs {
		imgStr = strings.TrimSpace(imgStr)
		if imgStr == "" {
			continue
		}

		ext := ""
		if dotIdx := strings.LastIndex(imgStr, "."); dotIdx != -1 {
			ext = imgStr[dotIdx+1:]
		}

		product.Images = append(product.Images, models.File{
			ID:       security.RandomString(),
			Name:     security.RandomString(),
			Ext:      ext,
			OrigName: imgStr,
		})
	}
}

// parseAttributes parses pipe-separated attributes from CSV
func (c *CSVImporter) parseAttributes(product *models.Product, record []string, headerMap map[string]int) {
	idx, ok := headerMap["attributes"]
	if !ok || idx >= len(record) || record[idx] == "" {
		return
	}

	attrStrs := strings.Split(record[idx], "|")
	for _, attr := range attrStrs {
		if trimmed := strings.TrimSpace(attr); trimmed != "" {
			product.Attributes = append(product.Attributes, trimmed)
		}
	}
}

// generateProductIDs generates UUIDs for product and all related entities
func (c *CSVImporter) generateProductIDs(product *models.Product) {
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
}

// parseVariantsB3 parses variants in B3 format: [Option:Val1;Val2]=>VariantData|VariantData
func (c *CSVImporter) parseVariantsB3(record []string, headerMap map[string]int) (bool, []models.ProductOption, []models.ProductVariant, error) {
	variantStr, ok := c.extractVariantString(record, headerMap)
	if !ok {
		return false, nil, nil, nil
	}

	optionsStr, variantsStr, err := c.splitVariantSections(variantStr)
	if err != nil {
		return false, nil, nil, err
	}

	options, err := c.parseOptionDefinitions(optionsStr)
	if err != nil {
		return false, nil, nil, err
	}

	variants, err := c.parseVariantDataRows(variantsStr, options)
	if err != nil {
		return false, nil, nil, err
	}

	return true, options, variants, nil
}

// extractVariantString extracts and validates variant string from CSV record
func (c *CSVImporter) extractVariantString(record []string, headerMap map[string]int) (string, bool) {
	idx, ok := headerMap["variants"]
	if !ok || idx >= len(record) {
		return "", false
	}

	variantStr := strings.TrimSpace(record[idx])
	return variantStr, variantStr != ""
}

// splitVariantSections splits variant string into options and variants parts
func (c *CSVImporter) splitVariantSections(variantStr string) (string, string, error) {
	sepIdx := strings.Index(variantStr, "=>")
	if sepIdx == -1 {
		return "", "", fmt.Errorf("invalid variant format: missing '=>' separator")
	}

	return variantStr[:sepIdx], variantStr[sepIdx+2:], nil
}

// parseOptionDefinitions parses option definitions from [Name:Val1;Val2] format
func (c *CSVImporter) parseOptionDefinitions(optionsStr string) ([]models.ProductOption, error) {
	options := []models.ProductOption{}
	position := 0

	for optionsStr != "" {
		start := strings.Index(optionsStr, "[")
		if start == -1 {
			break
		}

		end := strings.Index(optionsStr[start:], "]")
		if end == -1 {
			return nil, fmt.Errorf("invalid option definition: unmatched brackets")
		}
		end += start

		option, err := c.parseOptionDefinition(optionsStr[start+1:end], position)
		if err != nil {
			return nil, err
		}

		options = append(options, option)
		position++

		if position > 3 {
			return nil, fmt.Errorf("maximum 3 options allowed")
		}

		optionsStr = optionsStr[end+1:]
	}

	if len(options) == 0 {
		return nil, fmt.Errorf("no valid options found")
	}

	return options, nil
}

// parseOptionDefinition parses a single option definition (Name:Val1;Val2)
func (c *CSVImporter) parseOptionDefinition(optionDef string, position int) (models.ProductOption, error) {
	colonIdx := strings.Index(optionDef, ":")
	if colonIdx == -1 {
		return models.ProductOption{}, fmt.Errorf("invalid option definition: missing ':' separator")
	}

	optionName := strings.TrimSpace(optionDef[:colonIdx])
	valuesStr := optionDef[colonIdx+1:]
	valueStrs := strings.Split(valuesStr, ";")

	values := []models.ProductOptionValue{}
	for pos, val := range valueStrs {
		if val = strings.TrimSpace(val); val != "" {
			values = append(values, models.ProductOptionValue{
				Value:    val,
				Position: pos,
			})
		}
	}

	if len(values) == 0 {
		return models.ProductOption{}, fmt.Errorf("option must have at least one value")
	}

	return models.ProductOption{
		Name:     optionName,
		Values:   values,
		Position: position,
	}, nil
}

// parseVariantDataRows parses variant data rows in format: Val1,Val2,Price,Qty,SKU
func (c *CSVImporter) parseVariantDataRows(variantsStr string, options []models.ProductOption) ([]models.ProductVariant, error) {
	variants := []models.ProductVariant{}
	variantStrs := strings.Split(variantsStr, "|")

	for _, vStr := range variantStrs {
		if vStr = strings.TrimSpace(vStr); vStr == "" {
			continue
		}

		variant, err := c.parseVariantData(vStr, options)
		if err != nil {
			return nil, err
		}

		variants = append(variants, variant)
	}

	return variants, nil
}

// parseVariantData parses a single variant data row
func (c *CSVImporter) parseVariantData(vStr string, options []models.ProductOption) (models.ProductVariant, error) {
	parts := strings.Split(vStr, ",")
	expectedParts := len(options) + 3

	if len(parts) != expectedParts {
		return models.ProductVariant{}, fmt.Errorf("variant data doesn't match option structure: expected %d parts, got %d", expectedParts, len(parts))
	}

	optionValues := make(map[string]string)
	for i, option := range options {
		optionValues[option.Name] = strings.TrimSpace(parts[i])
	}

	price, err := strconv.Atoi(strings.TrimSpace(parts[len(options)]))
	if err != nil {
		return models.ProductVariant{}, fmt.Errorf("invalid price surcharge: %s", parts[len(options)])
	}

	qty, err := strconv.Atoi(strings.TrimSpace(parts[len(options)+1]))
	if err != nil {
		return models.ProductVariant{}, fmt.Errorf("invalid quantity: %s", parts[len(options)+1])
	}

	sku := strings.TrimSpace(parts[len(options)+2])

	return models.ProductVariant{
		OptionValues:   optionValues,
		PriceSurcharge: price,
		Quantity:       qty,
		SKU:            sku,
		Active:         true,
	}, nil
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
