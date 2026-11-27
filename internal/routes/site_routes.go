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

	// SEO routes (registered before SPA catch-all)
	c.Get("/sitemap.xml", handlers.Sitemap)
	c.Get("/robots.txt", handlers.SEOText("robots"))
	c.Get("/llms.txt", handlers.SEOText("llms"))
	c.Get("/.well-known/llms.txt", handlers.SEOText("llms"))
	c.Get("/ai.txt", handlers.SEOText("ai"))
	c.Get("/.well-known/ai.txt", handlers.SEOText("ai"))

	c.Use("/cart/payment/success", handlers.PaymentSuccess)
	c.Use("/cart/payment/cancel", handlers.PaymentCancel)

	skipPaths := func(path string) bool {
		return strings.HasPrefix(path, "/api") ||
			(path == "/_" || (strings.HasPrefix(path, "/_/") && !strings.HasPrefix(path, "/_app"))) ||
			strings.HasPrefix(path, "/uploads") ||
			(strings.HasPrefix(path, "/products/") && strings.HasSuffix(path, ".png")) ||
			path == "/sitemap.xml" || path == "/robots.txt" || path == "/llms.txt" ||
			path == "/ai.txt" || strings.HasPrefix(path, "/.well-known/")
	}

	c.Use("/", setupSPAHandler(embedSite, skipPaths, ""))
}
