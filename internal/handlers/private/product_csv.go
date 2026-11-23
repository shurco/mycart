package handlers

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/csvimport"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// ImportPreview validates and previews CSV import without executing
//
// @Summary      Preview CSV import
// @Description  Validate CSV file and preview what will be imported
// @Tags         Products
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "CSV file"
// @Success      200 {object} webutil.HTTPResponse{result=csvimport.ImportResult} "Preview results"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Router       /api/_/products/import/preview [post]
func ImportPreview(c fiber.Ctx) error {
	log := logging.New()
	db := queries.DB()

	// Get uploaded file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return webutil.StatusBadRequest(c, "file is required")
	}

	file, err := fileHeader.Open()
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, "failed to open file")
	}
	defer file.Close()

	// Create importer and validate
	importer := csvimport.NewCSVImporter(db.ProductQueries.DB)
	result, _, err := importer.ValidateAndPreview(file)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	return webutil.Response(c, fiber.StatusOK, "Preview generated", result)
}

// ImportProducts executes CSV import
//
// @Summary      Import products from CSV
// @Description  Execute CSV import after preview validation
// @Tags         Products
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "CSV file"
// @Success      200 {object} webutil.HTTPResponse{result=csvimport.ImportResult} "Import results"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Router       /api/_/products/import [post]
func ImportProducts(c fiber.Ctx) error {
	log := logging.New()
	db := queries.DB()

	// Get uploaded file
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return webutil.StatusBadRequest(c, "file is required")
	}

	file, err := fileHeader.Open()
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, "failed to open file")
	}
	defer file.Close()

	// Create importer and validate
	importer := csvimport.NewCSVImporter(db.ProductQueries.DB)
	_, products, err := importer.ValidateAndPreview(file)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	// Execute import
	result, err := importer.Import(c.Context(), products)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Import completed", result)
}

// buildImageString joins image filenames with pipe separator
func buildImageString(images []models.File) string {
	if len(images) == 0 {
		return ""
	}

	filenames := make([]string, len(images))
	for i, img := range images {
		// Use original name if available, otherwise construct from name + ext
		if img.OrigName != "" {
			filenames[i] = img.OrigName
		} else if img.Ext != "" {
			filenames[i] = img.Name + "." + img.Ext
		} else {
			filenames[i] = img.Name
		}
	}

	return strings.Join(filenames, "|")
}

// buildAttributeString joins attributes with pipe separator
func buildAttributeString(attributes []string) string {
	if len(attributes) == 0 {
		return ""
	}
	return strings.Join(attributes, "|")
}

// buildVariantsB3String generates B3 format variant string
func buildVariantsB3String(product models.Product) string {
	if !product.HasVariants || len(product.Options) == 0 {
		return ""
	}

	optionsParts := buildB3OptionParts(product.Options)
	if len(optionsParts) == 0 {
		return ""
	}

	optionsStr := strings.Join(optionsParts, "")

	variantParts := buildB3VariantParts(product)
	if len(variantParts) == 0 {
		return optionsStr + "=>"
	}

	return optionsStr + "=>" + strings.Join(variantParts, "|")
}

// buildB3OptionParts builds the options part: [OptionName:Value1;Value2]
func buildB3OptionParts(options []models.ProductOption) []string {
	var optionsParts []string
	for _, option := range options {
		if len(option.Values) == 0 {
			continue
		}

		valueStrs := make([]string, len(option.Values))
		for i, val := range option.Values {
			valueStrs[i] = val.Value
		}

		optionsParts = append(optionsParts, fmt.Sprintf("[%s:%s]", option.Name, strings.Join(valueStrs, ";")))
	}
	return optionsParts
}

// buildB3VariantParts builds the variants part: OptionValue1,OptionValue2,Price,Qty,SKU
func buildB3VariantParts(product models.Product) []string {
	var variantParts []string
	for _, variant := range product.Variants {
		var parts []string

		// Add option values in order
		for _, option := range product.Options {
			if val, ok := variant.OptionValues[option.Name]; ok {
				parts = append(parts, val)
			}
		}

		// Add price, quantity, SKU
		parts = append(parts, fmt.Sprintf("%d", variant.PriceSurcharge))
		parts = append(parts, fmt.Sprintf("%d", variant.Quantity))
		parts = append(parts, variant.SKU)

		variantParts = append(variantParts, strings.Join(parts, ","))
	}
	return variantParts
}

// ExportProducts exports products to CSV
//
// @Summary      Export products to CSV
// @Description  Download all products as CSV file
// @Tags         Products
// @Security     BearerAuth
// @Produce      text/csv
// @Success      200 {file} file "CSV file"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/export [get]
func ExportProducts(c fiber.Ctx) error {
	log := logging.New()
	db := queries.DB()

	// Get all products
	products, err := db.ListProducts(c.Context(), true, 0, 0, "")
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Set headers for CSV download
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", "attachment; filename=products.csv")

	// Write CSV header
	header := "name,slug,brief,description,images,attributes,amount,quantity,sku,variants,digital,active\n"
	if _, err := c.Write([]byte(header)); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Write product rows
	for _, product := range products.Products {
		// For products with variants, load full product data to get options and variants
		if product.HasVariants {
			fullProduct, err := db.Product(c.Context(), true, product.ID)
			if err != nil {
				log.ErrorStack(err)
				// Continue with partial data if we can't load full product
			} else {
				product = *fullProduct
			}
		}

		row := productCSVRow(product)

		if _, err := c.Write([]byte(row)); err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
	}

	return nil
}

// productCSVRow renders one product as a CSV line.
func productCSVRow(product models.Product) string {
	active := "false"
	if product.Active {
		active = "true"
	}

	return fmt.Sprintf("%s,%s,%s,%s,%s,%s,%d,%d,%s,%s,%s,%s\n",
		escapeCSV(product.Name),
		escapeCSV(product.Slug),
		escapeCSV(product.Brief),
		escapeCSV(product.Description),
		escapeCSV(buildImageString(product.Images)),
		escapeCSV(buildAttributeString(product.Attributes)),
		product.Amount,
		product.Quantity,
		escapeCSV(product.SKU),
		escapeCSV(buildVariantsB3String(product)),
		product.Digital.Type,
		active,
	)
}

// escapeCSV escapes a string for CSV output
func escapeCSV(s string) string {
	// If contains comma, quote, or newline, wrap in quotes and escape quotes
	needsQuotes := false
	for _, r := range s {
		if r == ',' || r == '"' || r == '\n' || r == '\r' {
			needsQuotes = true
			break
		}
	}

	if !needsQuotes {
		return s
	}

	// Escape quotes by doubling them
	escaped := ""
	for _, r := range s {
		if r == '"' {
			escaped += "\"\""
		} else {
			escaped += string(r)
		}
	}

	return "\"" + escaped + "\""
}
