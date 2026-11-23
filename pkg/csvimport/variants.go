package csvimport

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shurco/mycart/internal/models"
)

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
