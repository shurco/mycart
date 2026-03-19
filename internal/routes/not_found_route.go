package routes

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/litecart/pkg/webutil"
)

// NotFoundRoute handles 404 errors and routes to appropriate pages.
func NotFoundRoute(a *fiber.App, noSite bool) {
	a.Use(func(c fiber.Ctx) error {
		if strings.HasPrefix(c.Path(), "/api") {
			return webutil.StatusNotFound(c)
		}
		if strings.HasPrefix(c.Path(), "/_") {
			return c.Next()
		}

		// For SPA, let the frontend handle 404s
		// The SPA router will handle routing
		return c.Next()
	})
}
