package handlers

import (
	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// Ping returns a pong response for health checks.
//
// @Summary      Health check
// @Description  Returns pong for liveness probes
// @Tags         Public
// @Produce      json
// @Success      200 {object} webutil.HTTPResponse "Pong"
// @Router       /ping [get]
func Ping(c fiber.Ctx) error {
	return webutil.Response(c, fiber.StatusOK, "Pong", nil)
}

// Settings returns public settings including main, social, payment, and pages.
//
// @Summary      Get public settings
// @Description  Get site name, domain, currency, social links, and published pages
// @Tags         Public
// @Produce      json
// @Success      200 {object} webutil.HTTPResponse "Public settings"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/settings [get]
func Settings(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	settingMain, err := queries.GetSettingByGroup[models.Main](c.Context(), db)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	settingSocial, err := queries.GetSettingByGroup[models.Social](c.Context(), db)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	settingPayment, err := queries.GetSettingByGroup[models.Payment](c.Context(), db)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	pages, _, err := db.ListPages(c.Context(), false, 0, 0)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	return webutil.Response(c, fiber.StatusOK, "Settings", map[string]any{
		"main": map[string]string{
			"site_name": settingMain.SiteName,
			"domain":    settingMain.Domain,
			"currency":  settingPayment.Currency,
		},
		"socials": settingSocial,
		"pages":   pages,
	})
}
