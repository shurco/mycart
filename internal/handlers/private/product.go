package handlers

import (
	"fmt"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/csvimport"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/security"
	"github.com/shurco/mycart/pkg/webutil"
)

// Products returns a list of all products.
//
// @Summary      List products
// @Description  Get paginated list of all products (including inactive)
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        page  query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} webutil.HTTPResponse{result=models.Products} "Products list"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products [get]
func Products(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	p := webutil.ParsePagination(c)

	products, err := db.ListProducts(c.Context(), true, p.Limit, p.Offset, "")
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Products", products)
}

// AddProduct creates a new product.
//
// @Summary      Create product
// @Description  Create a new product with name, slug, price, and digital type
// @Tags         Products
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body models.Product true "Product data"
// @Success      200 {object} webutil.HTTPResponse{result=models.Product} "Created product"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products [post]
func AddProduct(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()
	request := &models.Product{}

	if err := c.Bind().Body(request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	// Generate ID if not provided
	if request.ID == "" {
		request.ID = security.RandomString()
	}

	// Generate IDs for options and variants
	for i := range request.Options {
		if request.Options[i].ID == "" {
			request.Options[i].ID = security.RandomString()
		}
		request.Options[i].ProductID = request.ID

		for j := range request.Options[i].Values {
			if request.Options[i].Values[j].ID == "" {
				request.Options[i].Values[j].ID = security.RandomString()
			}
			request.Options[i].Values[j].OptionID = request.Options[i].ID
		}
	}

	for i := range request.Variants {
		if request.Variants[i].ID == "" {
			request.Variants[i].ID = security.RandomString()
		}
		request.Variants[i].ProductID = request.ID

		for j := range request.Variants[i].Images {
			if request.Variants[i].Images[j].ID == "" {
				request.Variants[i].Images[j].ID = security.RandomString()
			}
		}
	}

	// Validation: digital.type field is required when creating a product
	if request.Digital.Type == "" {
		return webutil.StatusBadRequest(c, "digital type is required")
	}

	// Validate model
	if err := request.Validate(); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	// Use appropriate add method based on has_variants
	var product *models.Product
	var err error

	if request.HasVariants {
		product, err = db.AddProductWithVariants(c.Context(), request)
	} else {
		product, err = db.AddProduct(c.Context(), request)
	}

	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	return webutil.Response(c, fiber.StatusOK, "Product added", product)
}

// Product returns a single product by ID.
//
// @Summary      Get product
// @Description  Get a single product by its ID (including inactive)
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Success      200 {object} webutil.HTTPResponse{result=models.Product} "Product details"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id} [get]
func Product(c fiber.Ctx) error {
	productID := c.Params("product_id")
	db := queries.DB()
	log := logging.New()

	product, err := db.Product(c.Context(), true, productID)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Product info", product)
}

// UpdateProduct updates an existing product.
//
// @Summary      Update product
// @Description  Update product metadata and return the updated product
// @Tags         Products
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Param        request body models.Product true "Product data"
// @Success      200 {object} webutil.HTTPResponse{result=models.Product} "Updated product"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id} [patch]
func UpdateProduct(c fiber.Ctx) error {
	productID := c.Params("product_id")
	db := queries.DB()
	log := logging.New()
	request := new(models.Product)
	request.ID = productID

	if err := c.Bind().Body(request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	if err := db.UpdateProduct(c.Context(), request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Return updated product
	product, err := db.Product(c.Context(), true, productID)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Product updated", product)
}

// DeleteProduct deletes a product by ID.
//
// @Summary      Delete product
// @Description  Delete a product by its ID
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Success      200 {object} webutil.HTTPResponse "Product deleted"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id} [delete]
func DeleteProduct(c fiber.Ctx) error {
	productID := c.Params("product_id")
	db := queries.DB()
	log := logging.New()

	if err := db.DeleteProduct(c.Context(), productID); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Product deleted", nil)
}

// UpdateProductActive updates the active status of a product.
//
// @Summary      Toggle product active
// @Description  Toggle the active/inactive status of a product
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Success      200 {object} webutil.HTTPResponse "Product active updated"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/active [patch]
func UpdateProductActive(c fiber.Ctx) error {
	productID := c.Params("product_id")
	db := queries.DB()
	log := logging.New()

	if err := db.UpdateActive(c.Context(), productID); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Product active updated", nil)
}

// ProductImages returns a list of images for a product.
//
// @Summary      List product images
// @Description  Get all images for a product
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Success      200 {object} webutil.HTTPResponse{result=[]models.File} "Product images"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/image [get]
func ProductImages(c fiber.Ctx) error {
	productID := c.Params("product_id")
	db := queries.DB()
	log := logging.New()

	images, err := db.ProductImages(c.Context(), productID)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Product images", images)
}

// AddProductImage uploads and adds an image to a product.
//
// @Summary      Upload product image
// @Description  Upload an image (PNG/JPEG) and create sm (147x147) and md (400x400) variants
// @Tags         Products
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Param        document   formData file true "Image file (PNG or JPEG)"
// @Success      200 {object} webutil.HTTPResponse{result=models.File} "Added image"
// @Failure      400 {object} webutil.HTTPResponse "Invalid file format"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/image [post]
func AddProductImage(c fiber.Ctx) error {
	productID := c.Params("product_id")
	db := queries.DB()
	log := logging.New()

	file, err := c.FormFile("document")
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	mimeType := file.Header["Content-Type"][0]
	if !validateImageMIME(mimeType) {
		return webutil.StatusBadRequest(c, "file format not supported")
	}

	fileUUID, fileExt, fileName := generateFileName(file.Filename)
	filePath := fmt.Sprintf("%s/%s", dirUploads, fileName)
	fileOrigName := file.Filename

	if err := saveFile(file, filePath); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	fileSource, err := imaging.Open(filePath)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	sizes := []struct {
		size string
		dim  int
	}{
		{"sm", 147},
		{"md", 400},
	}

	for _, s := range sizes {
		resizedImage := imaging.Fill(fileSource, s.dim, s.dim, imaging.Center, imaging.Lanczos)
		resizedPath := fmt.Sprintf("%s/%s_%s.%s", dirUploads, fileUUID, s.size, fileExt)
		if err := imaging.Save(resizedImage, resizedPath); err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
	}

	addedImage, err := db.AddImage(c.Context(), productID, fileUUID, fileExt, fileOrigName)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Image added", addedImage)
}

// DeleteProductImage deletes an image from a product.
//
// @Summary      Delete product image
// @Description  Delete a product image by product and image ID
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Param        image_id   path string true "Image ID"
// @Success      200 {object} webutil.HTTPResponse "Image deleted"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/image/{image_id} [delete]
func DeleteProductImage(c fiber.Ctx) error {
	productID := c.Params("product_id")
	imageID := c.Params("image_id")
	db := queries.DB()
	log := logging.New()

	if err := db.DeleteImage(c.Context(), productID, imageID); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Image deleted", nil)
}

// ProductDigital returns digital content for a product.
//
// @Summary      Get product digital content
// @Description  Get digital files or license keys for a product
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Success      200 {object} webutil.HTTPResponse{result=models.Digital} "Digital content"
// @Failure      404 {object} webutil.HTTPResponse "Product not found"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/digital [get]
func ProductDigital(c fiber.Ctx) error {
	productID := c.Params("product_id")
	db := queries.DB()
	log := logging.New()

	digital, err := db.ProductDigital(c.Context(), productID)
	if err != nil {
		if errors.Is(err, errors.ErrProductNotFound) {
			return webutil.StatusNotFound(c)
		}
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Product digital", digital)
}

// AddProductDigital adds digital content (file or data) to a product.
//
// @Summary      Add product digital content
// @Description  Upload a digital file or create a license key entry
// @Tags         Products
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Param        document   formData file false "Digital file"
// @Success      200 {object} webutil.HTTPResponse "Digital content added"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/digital [post]
func AddProductDigital(c fiber.Ctx) error {
	productID := c.Params("product_id")
	db := queries.DB()
	log := logging.New()

	fileTmp, _ := c.FormFile("document")
	if fileTmp != nil {
		fileUUID, fileExt, fileName := generateFileName(fileTmp.Filename)
		filePath := fmt.Sprintf("%s/%s", dirDigitals, fileName)
		fileOrigName := fileTmp.Filename

		if err := saveFile(fileTmp, filePath); err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}

		file, err := db.AddDigitalFile(c.Context(), productID, fileUUID, fileExt, fileOrigName)
		if err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}

		return webutil.Response(c, fiber.StatusOK, "Digital added", file)
	}

	data, err := db.AddDigitalData(c.Context(), productID, "")
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Digital added", data)
}

// UpdateProductDigital updates digital content for a product.
//
// @Summary      Update product digital content
// @Description  Update the content of a digital data entry (license key)
// @Tags         Products
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Param        digital_id path string true "Digital ID"
// @Param        request    body models.Data true "Digital data"
// @Success      200 {object} webutil.HTTPResponse "Digital content updated"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/digital/{digital_id} [patch]
func UpdateProductDigital(c fiber.Ctx) error {
	request := new(models.Data)
	request.ID = c.Params("digital_id")
	// request.Content = c.Params("digital_id")
	db := queries.DB()
	log := logging.New()

	if err := c.Bind().Body(request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	if err := db.UpdateDigital(c.Context(), request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Digital updated", nil)
}

// DeleteProductDigital deletes digital content from a product.
//
// @Summary      Delete product digital content
// @Description  Delete a digital content entry by product and digital ID
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Param        digital_id path string true "Digital ID"
// @Success      200 {object} webutil.HTTPResponse "Digital content deleted"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/digital/{digital_id} [delete]
func DeleteProductDigital(c fiber.Ctx) error {
	productID := c.Params("product_id")
	digitalID := c.Params("digital_id")
	db := queries.DB()
	log := logging.New()

	if err := db.DeleteDigital(c.Context(), productID, digitalID); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Digital deleted", nil)
}

// GenerateSlug generates a unique slug from product name
//
// @Summary      Generate product slug
// @Description  Generate URL-friendly slug from product name with uniqueness check
// @Tags         Products
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body object{name=string,exclude_id=string} true "Slug generation request"
// @Success      200 {object} webutil.HTTPResponse{result=object{slug=string}} "Generated slug"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Router       /api/_/products/slug/generate [post]
func GenerateSlug(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	var request struct {
		Name      string `json:"name"`
		ExcludeID string `json:"exclude_id"`
	}

	if err := c.Bind().Body(&request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	if request.Name == "" {
		return webutil.StatusBadRequest(c, "name is required")
	}

	slug, err := db.GenerateUniqueSlug(c.Context(), request.Name, request.ExcludeID)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Slug generated", map[string]string{
		"slug": slug,
	})
}

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

	// Build options part: [OptionName:Value1;Value2]
	var optionsParts []string
	for _, option := range product.Options {
		if len(option.Values) == 0 {
			continue
		}

		valueStrs := make([]string, len(option.Values))
		for i, val := range option.Values {
			valueStrs[i] = val.Value
		}

		optionsParts = append(optionsParts, fmt.Sprintf("[%s:%s]", option.Name, strings.Join(valueStrs, ";")))
	}

	if len(optionsParts) == 0 {
		return ""
	}

	optionsStr := strings.Join(optionsParts, "")

	// Build variants part: OptionValue1,OptionValue2,Price,Qty,SKU
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

	if len(variantParts) == 0 {
		return optionsStr + "=>"
	}

	return optionsStr + "=>" + strings.Join(variantParts, "|")
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

		active := "false"
		if product.Active {
			active = "true"
		}

		row := fmt.Sprintf("%s,%s,%s,%s,%s,%s,%d,%d,%s,%s,%s,%s\n",
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

		if _, err := c.Write([]byte(row)); err != nil {
			log.ErrorStack(err)
			return webutil.StatusInternalServerError(c)
		}
	}

	return nil
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
