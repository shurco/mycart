package handlers

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/internal/seo"
)

var (
	specOnce sync.Once
	specData []byte
)

func OpenAPISpec(c fiber.Ctx) error {
	db := queries.DB()
	s, err := queries.GetSettingByGroup[models.SEO](c.Context(), db)
	if err != nil || !s.OpenAPIEnabled {
		return c.SendStatus(fiber.StatusNotFound)
	}
	
	specOnce.Do(func() {
		baseDir := os.Getenv("LITECART_BASE")
		if baseDir == "" {
			baseDir = "./lc_base"
		}
		specPath := filepath.Join(baseDir, "swagger", "swagger.json")
		specData, _ = os.ReadFile(specPath)
	})
	
	if len(specData) == 0 {
		return c.SendStatus(fiber.StatusNotFound)
	}
	
	m := seo.NewMatcher(s.OpenAPIInclude, s.OpenAPIExclude)
	filtered, err := seo.FilterSpec(specData, m)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	
	c.Set(fiber.HeaderContentType, "application/json")
	return c.Send(filtered)
}
