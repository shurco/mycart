package handlers

import (
	"github.com/gofiber/fiber/v3"

	"github.com/shurco/litecart/internal/models"
	"github.com/shurco/litecart/internal/queries"
	"github.com/shurco/litecart/pkg/errors"
	"github.com/shurco/litecart/pkg/logging"
	"github.com/shurco/litecart/pkg/webutil"
)

// Pages returns a list of all pages.
//
// @Summary      List pages
// @Description  Get paginated list of all pages (including inactive)
// @Tags         Pages
// @Security     BearerAuth
// @Produce      json
// @Param        page  query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(20)
// @Success      200 {object} webutil.HTTPResponse "Pages list with pagination"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/pages [get]
func Pages(c fiber.Ctx) error {
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

	pages, total, err := db.ListPages(c.Context(), true, limit, offset)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Pages", map[string]any{
		"pages": pages,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetPage returns a single page by ID.
//
// @Summary      Get page
// @Description  Get a single page by its ID
// @Tags         Pages
// @Security     BearerAuth
// @Produce      json
// @Param        page_id path string true "Page ID"
// @Success      200 {object} webutil.HTTPResponse{result=models.Page} "Page details"
// @Failure      404 {object} webutil.HTTPResponse "Page not found"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/pages/{page_id} [get]
func GetPage(c fiber.Ctx) error {
	pageID := c.Params("page_id")
	db := queries.DB()
	log := logging.New()

	page, err := db.PageByID(c.Context(), pageID)
	if err != nil {
		if errors.Is(err, errors.ErrPageNotFound) {
			return webutil.StatusNotFound(c)
		}
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Page", page)
}

// AddPage creates a new page.
//
// @Summary      Create page
// @Description  Create a new page with name, slug, and optional content
// @Tags         Pages
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body models.Page true "Page data"
// @Success      200 {object} webutil.HTTPResponse{result=models.Page} "Created page"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/pages [post]
func AddPage(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()
	request := new(models.Page)

	if err := c.Bind().Body(request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	page, err := db.AddPage(c.Context(), request)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Page added", page)
}

// UpdatePage updates an existing page.
//
// @Summary      Update page
// @Description  Update page metadata (name, slug, position, SEO)
// @Tags         Pages
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        page_id path string true "Page ID"
// @Param        request body models.Page true "Page data"
// @Success      200 {object} webutil.HTTPResponse "Page updated"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/pages/{page_id} [patch]
func UpdatePage(c fiber.Ctx) error {
	pageID := c.Params("page_id")
	db := queries.DB()
	log := logging.New()
	request := new(models.Page)
	request.ID = pageID

	if err := c.Bind().Body(request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	if err := db.UpdatePage(c.Context(), request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Page updated", nil)
}

// DeletePage deletes a page by ID.
//
// @Summary      Delete page
// @Description  Delete a page by its ID
// @Tags         Pages
// @Security     BearerAuth
// @Produce      json
// @Param        page_id path string true "Page ID"
// @Success      200 {object} webutil.HTTPResponse "Page deleted"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/pages/{page_id} [delete]
func DeletePage(c fiber.Ctx) error {
	pageID := c.Params("page_id")
	db := queries.DB()
	log := logging.New()

	if err := db.DeletePage(c.Context(), pageID); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Page deleted", nil)
}

// UpdatePageContent updates the content of a page.
//
// @Summary      Update page content
// @Description  Update only the HTML content of a page
// @Tags         Pages
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        page_id path string true "Page ID"
// @Param        request body object{content=string} true "Content"
// @Success      200 {object} webutil.HTTPResponse "Content updated"
// @Failure      400 {object} webutil.HTTPResponse "Validation error"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/pages/{page_id}/content [patch]
func UpdatePageContent(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()
	pageID := c.Params("page_id")

	request := &models.Page{
		Core: models.Core{
			ID: pageID,
		},
	}

	if err := c.Bind().Body(request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusBadRequest(c, err.Error())
	}

	if err := db.UpdatePageContent(c.Context(), request); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Page content updated", nil)
}

// UpdatePageActive updates the active status of a page.
//
// @Summary      Toggle page active
// @Description  Toggle the active/inactive status of a page
// @Tags         Pages
// @Security     BearerAuth
// @Produce      json
// @Param        page_id path string true "Page ID"
// @Success      200 {object} webutil.HTTPResponse{result=models.Page} "Updated page"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/_/pages/{page_id}/active [patch]
func UpdatePageActive(c fiber.Ctx) error {
	pageID := c.Params("page_id")
	db := queries.DB()
	log := logging.New()

	if err := db.UpdatePageActive(c.Context(), pageID); err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Get updated page to return with updated timestamp
	page, err := db.PageByID(c.Context(), pageID)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Page active updated", page)
}
