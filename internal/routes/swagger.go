package routes

import (
	"os"
	"path/filepath"

	"github.com/TDiblik/gofiber-swagger/gofiberswagger"
	"github.com/gofiber/fiber/v3"
	handlers "github.com/shurco/mycart/internal/handlers/public"
)

func RegisterSwagger(app *fiber.App) error {
	baseDir := os.Getenv("LITECART_BASE")
	if baseDir == "" {
		baseDir = "./lc_base"
	}
	swaggerPath := filepath.Join(baseDir, "swagger")
	
	if err := os.MkdirAll(swaggerPath, 0755); err != nil {
		return err
	}
	
	cfg := gofiberswagger.DefaultConfig
	cfg.SwaggerPath = "/_/swagger"
	cfg.SwaggerFilesPath = swaggerPath
	cfg.CreateSwaggerFiles = true
	cfg.Swagger.Info.Title = "myCart API"
	cfg.Swagger.Info.Version = "1.0.0"
	
	if err := gofiberswagger.Register(app, cfg); err != nil {
		return err
	}
	
	app.Get("/openapi.json", handlers.OpenAPISpec)
	app.Get("/swagger.json", handlers.OpenAPISpec)
	app.Get("/.well-known/openapi.json", handlers.OpenAPISpec)
	app.Get("/.well-known/swagger.json", handlers.OpenAPISpec)
	
	return nil
}
