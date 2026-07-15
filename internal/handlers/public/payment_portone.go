package handlers

import (
	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/pkg/logging"
	"github.com/shurco/mycart/pkg/webutil"
)

// GetPortoneConfig returns public PortOne configuration (store_id, channel_key)
// API secret is NOT exposed to frontend for security
//
// @Summary      Get PortOne config
// @Description  Get public PortOne configuration for browser SDK
// @Tags         Cart
// @Produce      json
// @Success      200 {object} webutil.HTTPResponse "PortOne public config"
// @Failure      500 {object} webutil.HTTPResponse "Internal server error"
// @Router       /api/cart/portone-config [get]
func GetPortoneConfig(c fiber.Ctx) error {
	db := queries.DB()
	log := logging.New()

	settings, err := queries.GetSettingByGroup[models.Portone](c.Context(), db)
	if err != nil {
		log.ErrorStack(err)
		return webutil.StatusInternalServerError(c)
	}

	// Only expose store_id and channel_key, NOT api_secret
	config := map[string]string{
		"store_id":    settings.StoreID,
		"channel_key": settings.ChannelKey,
	}

	return webutil.Response(c, fiber.StatusOK, "PortOne config", config)
}
