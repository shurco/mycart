package handlers

import (
	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// ReorderProductImages reorders product images by updating their positions.
//
// @Summary      Reorder product images
// @Description  Update the position of multiple product images in a batch operation
// @Tags         Products
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        product_id path string true "Product ID"
// @Param        request body object{updates=[]object{imageId=string,position=int}} true "Position updates"
// @Success      200 {object} webutil.HTTPResponse "Positions updated"
// @Failure      400 {object} webutil.HTTPResponse "Invalid request"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/products/{product_id}/images/reorder [post]
func ReorderProductImages(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	// Parse request body
	var request struct {
		Updates []struct {
			ImageID  string `json:"imageId"`
			Position int    `json:"position"`
		} `json:"updates"`
	}

	if err := c.Bind().Body(&request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	// Validate: updates array not empty
	if len(request.Updates) == 0 {
		return webutil.StatusBadRequest(c, "updates array cannot be empty")
	}

	// Validate: each update has valid imageId and position
	for _, update := range request.Updates {
		if update.ImageID == "" {
			return webutil.StatusBadRequest(c, "imageId is required for each update")
		}
		if update.Position < 0 {
			return webutil.StatusBadRequest(c, "position must be non-negative")
		}
	}

	// Convert request updates to query updates
	updates := make([]queries.ImagePositionUpdate, len(request.Updates))
	for i, u := range request.Updates {
		updates[i] = queries.ImagePositionUpdate{
			ImageID:  u.ImageID,
			Position: u.Position,
		}
	}

	// Call query function
	if err := db.UpdateProductImagePositions(c.Context(), updates); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Images reordered", nil)
}
