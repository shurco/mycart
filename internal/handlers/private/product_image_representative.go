package handlers

import (
	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// SetProductRepresentativeImage marks a product image as the representative image.
//
// @Summary      Set representative product image
// @Description  Mark a specific product image as the representative (primary) image
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Param        image_id path string true "Image ID"
// @Success      200 {object} webutil.HTTPResponse "Image marked as representative"
// @Failure      404 {object} webutil.HTTPResponse "Image not found"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/images/{image_id}/representative [post]
func SetProductRepresentativeImage(c fiber.Ctx) error {
	productID := c.Params("product_id")
	imageID := c.Params("image_id")
	db := queries.DB()
	log := logging.New()

	// Call query function
	if err := db.SetRepresentativeImage(c.Context(), productID, imageID); err != nil {
		log.ErrorStack(err)
		// Check if it's a "not found" error
		errMsg := err.Error()
		if errMsg == "image not found: "+imageID ||
		   errMsg == "image does not belong to product: image_id="+imageID+", product_id="+productID {
			return webutil.StatusNotFound(c)
		}
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Image marked as representative", nil)
}
