package csvimport

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/pkg/security"
)

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
