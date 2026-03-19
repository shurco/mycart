package routes

import (
	"github.com/gofiber/fiber/v3"

	handlers "github.com/shurco/litecart/internal/handlers/private"
	"github.com/shurco/litecart/internal/middleware"
)

// ApiPrivateRoutes sets up private API routes that require authentication.
func ApiPrivateRoutes(c *fiber.App) {
	c.Post("/api/install", handlers.Install)

	c.Get("/api/_/version", middleware.JWTProtected(), handlers.Version)

	sign := c.Group("/api/sign")
	sign.Post("/in", handlers.SignIn)
	sign.Post("/out", middleware.JWTProtected(), handlers.SignOut)

	settings := c.Group("/api/_/settings", middleware.JWTProtected())
	settings.Get("/:setting_key", handlers.GetSetting)
	settings.Patch("/:setting_key", handlers.UpdateSetting)

	test := c.Group("/api/_/test", middleware.JWTProtected())
	test.Get("/letter/:letter_name", handlers.TestLetter)

	pages := c.Group("/api/_/pages", middleware.JWTProtected())
	pages.Get("/", handlers.Pages)
	pages.Get("/:page_id<len(15)>", handlers.GetPage)
	pages.Post("/", handlers.AddPage)
	pages.Patch("/:page_id<len(15)>", handlers.UpdatePage)
	pages.Delete("/:page_id<len(15)>", handlers.DeletePage)
	pages.Patch("/:page_id<len(15)>/content", handlers.UpdatePageContent)
	pages.Patch("/:page_id<len(15)>/active", handlers.UpdatePageActive)

	product := c.Group("/api/_/products", middleware.JWTProtected())
	product.Get("/", handlers.Products)
	product.Post("/", handlers.AddProduct)
	product.Get("/:product_id<len(15)>", handlers.Product)
	product.Patch("/:product_id<len(15)>", handlers.UpdateProduct)
	product.Delete("/:product_id<len(15)>", handlers.DeleteProduct)
	product.Patch("/:product_id<len(15)>/active", handlers.UpdateProductActive)

	product.Get("/:product_id<len(15)>/digital", handlers.ProductDigital)
	product.Post("/:product_id<len(15)>/digital", handlers.AddProductDigital)
	product.Patch("/:product_id<len(15)>/digital/:digital_id<len(15)>", handlers.UpdateProductDigital)
	product.Delete("/:product_id<len(15)>/digital/:digital_id<len(15)>", handlers.DeleteProductDigital)

	product.Get("/:product_id<len(15)>/image", handlers.ProductImages)
	product.Post("/:product_id<len(15)>/image", handlers.AddProductImage)
	product.Delete("/:product_id<len(15)>/image/:image_id<len(15)>", handlers.DeleteProductImage)

	// carts
	carts := c.Group("/api/_/carts", middleware.JWTProtected())
	carts.Get("/", handlers.Carts)
	carts.Get("/:cart_id<len(15)>", handlers.Cart)
	carts.Post("/:cart_id<len(15)>/mail", handlers.CartSendMail)
}
