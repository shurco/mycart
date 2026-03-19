package routes

import (
	"io/fs"
	"strings"

	"github.com/gofiber/fiber/v3"

	handlers "github.com/shurco/litecart/internal/handlers/public"
	"github.com/shurco/litecart/web"
)

func SiteRoutes(c *fiber.App) {
	embedSite, _ := fs.Sub(web.EmbedSite(), web.SiteBuildPath)

	c.Use("/cart/payment/success", handlers.PaymentSuccess)
	c.Use("/cart/payment/cancel", handlers.PaymentCancel)

	skipPaths := func(path string) bool {
		return strings.HasPrefix(path, "/api") ||
			(path == "/_" || (strings.HasPrefix(path, "/_/") && !strings.HasPrefix(path, "/_app"))) ||
			strings.HasPrefix(path, "/uploads")
	}

	c.Use("/", setupSPAHandler(embedSite, skipPaths, ""))
}
