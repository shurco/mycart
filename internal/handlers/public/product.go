package handlers

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
	"github.com/shurco/mycart/web"
)

// Products returns a list of all active products for public access.
//
// @Summary      List active products
// @Description  Get paginated list of active products visible to customers
// @Tags         Public
// @Produce      json
// @Param        page  query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} webutil.HTTPResponse{result=models.Products} "Products list"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/products [get]
func Products(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	p := webutil.ParsePagination(c)

	products, err := db.ListProducts(c.Context(), false, p.Limit, p.Offset, "")
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Products", products)
}

// Product returns a single active product by slug for public access.
//
// The address is the slug and not the row id: the storefront links to
// /products/<slug>, and the public query behind this handler matches on the
// slug. Asking for something the shop does not have is a 404 — the product is
// simply not there — and not the 500 an unclassified error would produce.
//
// @Summary      Get active product
// @Description  Get a single active product by its slug
// @Tags         Public
// @Produce      json
// @Param        product_slug path string true "Product slug"
// @Success      200 {object} webutil.HTTPResponse{result=models.Product} "Product details"
// @Failure      404 {object} webutil.HTTPResponse "Product not found"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/products/{product_slug} [get]
func Product(c fiber.Ctx) error {
	productSlug := c.Params("product_slug")
	db := queries.DB()
	log := logging.New()

	product, err := db.Product(c.Context(), false, productSlug)
	if err != nil {
		if errors.Is(err, errors.ErrProductNotFound) {
			return webutil.StatusNotFound(c)
		}
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Product info", product)
}

// servePlaceholderImage serves the noimage.png placeholder from embedded assets
func servePlaceholderImage(c fiber.Ctx, log *logging.Log) error {
	embedSite := web.EmbedSite()
	placeholderPath := "site/build/assets/img/noimage.png"

	placeholderData, err := fs.ReadFile(embedSite, placeholderPath)
	if err != nil {
		log.ErrorStack(fmt.Errorf("failed to read placeholder image: %w", err))
		return webutil.StatusNotFound(c)
	}

	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "public, max-age=86400") // Cache for 1 day
	return c.Send(placeholderData)
}

// ProductRepresentativeImage serves the representative (first) product image as PNG.
// URL pattern: /products/{slug}.png
// Converts JPEG to PNG if needed.
//
// @Summary      Get product representative image
// @Description  Serves the first/representative product image, converted to PNG format
// @Tags         Public
// @Produce      image/png
// @Param        slug path string true "Product slug (without .png extension)"
// @Success      200 {file} binary "Product image in PNG format"
// @Failure      404 {object} webutil.HTTPResponse "Product or image not found"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /products/{slug}.png [get]
func ProductRepresentativeImage(c fiber.Ctx) error {
	// Extract slug from URL (remove .png extension)
	slug := c.Params("slug")
	slug = strings.TrimSuffix(slug, ".png")

	db := queries.DB()
	log := logging.New()

	// Get product by slug (public access - active products only)
	product, err := db.Product(c.Context(), false, slug)
	if err != nil {
		log.ErrorStack(err)
		return servePlaceholderImage(c, log)
	}

	// Find representative image (first by position or marked as representative)
	if len(product.Images) == 0 {
		return servePlaceholderImage(c, log)
	}

	// Get the first image (position 0)
	var repImage *models.File
	for i := range product.Images {
		if product.Images[i].Position == 0 || product.Images[i].IsRepresentative {
			repImage = &product.Images[i]
			break
		}
	}
	if repImage == nil {
		repImage = &product.Images[0]
	}

	// Build file path
	uploadsDir := "./lc_uploads"
	imagePath := filepath.Join(uploadsDir, fmt.Sprintf("%s.%s", repImage.Name, repImage.Ext))

	// Read the image file
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		log.ErrorStack(fmt.Errorf("failed to read image file %s: %w", imagePath, err))
		return servePlaceholderImage(c, log)
	}

	// If already PNG, serve directly
	if strings.ToLower(repImage.Ext) == "png" {
		c.Set("Content-Type", "image/png")
		c.Set("Cache-Control", "public, max-age=31536000") // Cache for 1 year
		return c.Send(imageData)
	}

	// Convert JPEG to PNG
	if strings.ToLower(repImage.Ext) == "jpg" || strings.ToLower(repImage.Ext) == "jpeg" {
		img, err := jpeg.Decode(bytes.NewReader(imageData))
		if err != nil {
			log.ErrorStack(fmt.Errorf("failed to decode JPEG image: %w", err))
			return webutil.StatusInternalServerError(c)
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			log.ErrorStack(fmt.Errorf("failed to encode PNG image: %w", err))
			return webutil.StatusInternalServerError(c)
		}

		c.Set("Content-Type", "image/png")
		c.Set("Cache-Control", "public, max-age=31536000")
		return c.Send(buf.Bytes())
	}

	// Fallback: try to decode as generic image and encode as PNG
	img, _, err := image.Decode(bytes.NewReader(imageData))
	if err != nil {
		log.ErrorStack(fmt.Errorf("failed to decode image: %w", err))
		return webutil.StatusInternalServerError(c)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.ErrorStack(fmt.Errorf("failed to encode PNG image: %w", err))
		return webutil.StatusInternalServerError(c)
	}

	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "public, max-age=31536000")
	return c.Send(buf.Bytes())
}
