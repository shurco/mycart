package handlers

import (
	"fmt"

	"github.com/disintegration/imaging"
	"github.com/gofiber/fiber/v3"

	"github.com/shurco/litecart/internal/models"
	"github.com/shurco/litecart/internal/queries"
	"github.com/shurco/litecart/pkg/errors"
	"github.com/shurco/litecart/pkg/logging"
	"github.com/shurco/litecart/pkg/webutil"
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

	page := fiber.Query[int](c, "page", 1)
	limit := fiber.Query[int](c, "limit", 20)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	products, err := db.ListProducts(c.Context(), true, limit, offset, "")
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

	// Validation: digital.type field is required when creating a product
	if request.Digital.Type == "" {
		return webutil.StatusBadRequest(c, "digital type is required")
	}

	// Validate model
	if err := request.Validate(); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	product, err := db.AddProduct(c.Context(), request)
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
