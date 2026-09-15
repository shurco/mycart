package routes

import (
	"io/fs"
	"strings"

	"github.com/gofiber/fiber/v3"

	handlers "github.com/shurco/mycart/internal/handlers/public"
	"github.com/shurco/mycart/web"
)

func SiteRoutes(c *fiber.App) {
	embedSite, _ := fs.Sub(web.EmbedSite(), web.SiteBuildPath)

	// Product representative image route
	c.Get("/products/:slug.png", handlers.ProductRepresentativeImage)

	c.Use("/cart/payment/success", handlers.PaymentSuccess)
	c.Use("/cart/payment/cancel", handlers.PaymentCancel)

	skipPaths := func(path string) bool {
		return strings.HasPrefix(path, "/api") ||
			(path == "/_" || (strings.HasPrefix(path, "/_/") && !strings.HasPrefix(path, "/_app"))) ||
			strings.HasPrefix(path, "/uploads") ||
			(strings.HasPrefix(path, "/products/") && strings.HasSuffix(path, ".png"))
	}

	c.Use("/", setupSPAHandler(embedSite, skipPaths, ""))
}
