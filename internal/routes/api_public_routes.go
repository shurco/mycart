package routes

import (
	"github.com/gofiber/fiber/v3"

	handlers "github.com/shurco/litecart/internal/handlers/public"
)

// ApiPublicRoutes sets up public API routes accessible without authentication.
func ApiPublicRoutes(c *fiber.App) {
	c.Get("/ping", handlers.Ping)

	c.Get("/api/settings", handlers.Settings)
	c.Get("/api/pages/:page_slug", handlers.Page)

	product := c.Group("/api/products")
	product.Get("/", handlers.Products)
	product.Get("/:product_id", handlers.Product)

	cart := c.Group("/cart")
	cart.Post("/payment", handlers.Payment)
	cart.Post("/payment/callback", handlers.PaymentCallback)

	c.Get("/api/cart/payment", handlers.PaymentList)
	c.Get("/api/cart/:cart_id", handlers.GetCart)
}
