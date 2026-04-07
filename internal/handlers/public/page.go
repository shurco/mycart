package handlers

import (
	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/errors"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// Page returns a page by slug for public access.
//
// @Summary      Get page by slug
// @Description  Get an active page by its URL slug
// @Tags         Public
// @Produce      json
// @Param        page_slug path string true "Page slug"
// @Success      200 {object} webutil.HTTPResponse{result=models.Page} "Page content"
// @Failure      404 {object} webutil.HTTPResponse "Page not found"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/pages/{page_slug} [get]
func Page(c fiber.Ctx) error {
	pageSlug := c.Params("page_slug")
	log := logging.New()
	db := queries.DB()

	page, err := db.Page(c.Context(), pageSlug)
	if err != nil {
		if errors.Is(err, errors.ErrPageNotFound) {
			return webutil.StatusNotFound(c)
		}
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Page content", page)
}
