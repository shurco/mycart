package routes

import (
	"github.com/gofiber/fiber/v3"

	handlers "github.com/shurco/mycart/internal/handlers/private"
	"github.com/shurco/mycart/internal/middleware"
)

// ApiPrivateRoutes sets up private API routes that require authentication.
func ApiPrivateRoutes(c *fiber.App) {
	// Both session-cookie surfaces of the admin: the panel's own API and the
	// sign-in endpoints. Registered before the routes for the same reason
	// middleware always is — a handler that answers first never reaches it.
	c.Use("/api/_/", middleware.CSRFProtect())
	c.Use("/api/sign/", middleware.CSRFProtect())

	c.Get("/api/install/status", handlers.InstallStatus)
	c.Post("/api/install", middleware.AuthLimiter(), handlers.Install)
	c.Post("/api/install/db/test", middleware.AuthLimiter(), handlers.InstallDBTest)

	c.Get("/api/_/version", middleware.JWTProtected(), handlers.Version)

	sign := c.Group("/api/sign")
	sign.Post("/in", middleware.AuthLimiter(), handlers.SignIn)
	sign.Post("/out", middleware.JWTProtected(), handlers.SignOut)

	settings := c.Group("/api/_/settings", middleware.JWTProtected())
	// Registered before the group's own two-segment-free routes: the marks are
	// uploaded and removed rather than written as values, and a route that
	// answers first is the one that is reached.
	settings.Post("/branding/logo", handlers.UploadBrandingLogo)
	settings.Delete("/branding/logo", handlers.DeleteBrandingLogo)
	settings.Post("/branding/favicon", handlers.UploadBrandingFavicon)
	settings.Delete("/branding/favicon", handlers.DeleteBrandingFavicon)
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
	product.Post("/slug/generate", handlers.GenerateSlug)
	product.Post("/import/preview", handlers.ImportPreview)
	product.Post("/import", handlers.ImportProducts)
	product.Get("/export", handlers.ExportProducts)
	product.Get("/:product_id<len(15)>", handlers.Product)
	product.Patch("/:product_id<len(15)>", handlers.UpdateProduct)
	product.Delete("/:product_id<len(15)>", handlers.DeleteProduct)
	product.Patch("/:product_id<len(15)>/active", handlers.UpdateProductActive)

	product.Get("/:product_id<len(15)>/digital", handlers.ProductDigital)
	product.Post("/:product_id<len(15)>/digital", handlers.AddProductDigital)
	product.Get("/:product_id<len(15)>/digital/:digital_id<len(15)>/download", handlers.DownloadProductDigital)
	product.Patch("/:product_id<len(15)>/digital/:digital_id<len(15)>", handlers.UpdateProductDigital)
	product.Delete("/:product_id<len(15)>/digital/:digital_id<len(15)>", handlers.DeleteProductDigital)

	product.Get("/:product_id<len(15)>/image", handlers.ProductImages)
	product.Post("/:product_id<len(15)>/image", handlers.AddProductImage)
	product.Delete("/:product_id<len(15)>/image/:image_id<len(15)>", handlers.DeleteProductImage)
	product.Post("/:product_id<len(15)>/images/reorder", handlers.ReorderProductImages)
	product.Post("/:product_id<len(15)>/images/:image_id<len(15)>/representative", handlers.SetProductRepresentativeImage)

	// carts
	carts := c.Group("/api/_/carts", middleware.JWTProtected())
	carts.Get("/", handlers.Carts)
	carts.Get("/:cart_id<len(15)>", handlers.Cart)
	carts.Post("/:cart_id<len(15)>/mail", handlers.CartSendMail)

	// customers. The list is keyed by email, so the cart lookup is too: most
	// rows in it are buyers who checked out as guests and have no account id
	// to address them by. The account actions below take an id, and only apply
	// to the rows that have one.
	customers := c.Group("/api/_/customers", middleware.JWTProtected())
	customers.Get("/", handlers.Customers)
	customers.Get("/carts", handlers.CustomerCarts)
	customers.Patch("/:customer_id<len(15)>/active", handlers.UpdateCustomerActive)
	customers.Patch("/:customer_id<len(15)>/password", handlers.UpdateCustomerPassword)
	customers.Delete("/:customer_id<len(15)>", handlers.DeleteCustomer)
}
