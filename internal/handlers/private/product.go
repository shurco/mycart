package handlers

import (
	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
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

// assignProductIDs fills in missing IDs for the product, its options, option
// values, variants and variant images, and links children to their parents.
func assignProductIDs(request *models.Product) {
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

	assignProductIDs(request)

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
		if errors.Is(err, errors.ErrProductSold) {
			return webutil.StatusBadRequest(c, errors.MsgProductSold)
		}
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
