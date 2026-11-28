package routes

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
	handlers "github.com/shurco/mycart/internal/handlers/public"
)

func RegisterSwagger(app *fiber.App) error {
	// Serve Swagger UI and docs from docs/swagger at /_/swagger/
	app.Get("/_/swagger/*", static.New("./docs/swagger", static.Config{
		Browse: true,
	}))

	// OpenAPI spec endpoints (filtered via settings)
	app.Get("/openapi.json", handlers.OpenAPISpec)
	app.Get("/swagger.json", handlers.OpenAPISpec)
	app.Get("/.well-known/openapi.json", handlers.OpenAPISpec)
	app.Get("/.well-known/swagger.json", handlers.OpenAPISpec)

	return nil
}
