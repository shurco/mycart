package handlers

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/internal/seo"
	"github.com/shurco/mycart/pkg/webutil"
)

// SEOPreview renders a template with current settings for admin preview.
func SEOPreview(c fiber.Ctx) error {
	s := new(models.SEO)
	if err := c.Bind().Body(s); err != nil {
		return webutil.StatusBadRequest(c, err.Error())
	}
	if err := s.Validate(); err != nil {
		return webutil.StatusBadRequest(c, err)
	}

	file := c.Params("file") // sitemap.xml, llms.txt, ai.txt, robots.txt, openapi.json
	data, err := seo.Load(c.Context(), queries.Conn(), c.BaseURL())
	if err != nil {
		return webutil.StatusInternalServerError(c)
	}

	var content string
	switch file {
	case "sitemap.xml":
		body, err := seo.BuildSitemap(data, s)
		if err != nil {
			return webutil.StatusBadRequest(c, fmt.Sprintf("sitemap error: %v", err))
		}
		content = string(body)
	case "llms.txt", "ai.txt", "robots.txt":
		name := strings.TrimSuffix(file, ".txt")
		tpl := ""
		switch name {
		case "llms":
			tpl = s.LlmsTemplate
		case "ai":
			tpl = s.AiTemplate
		case "robots":
			tpl = s.RobotsTemplate
		}
		if tpl == "" {
			tpl = seo.Default(name)
		}
		out, err := seo.Render(tpl, data)
		if err != nil {
			// Admin may see the Liquid error.
			return webutil.StatusBadRequest(c, fmt.Sprintf("template error: %v", err))
		}
		content = out
	case "openapi.json":
		// Placeholder for task 10 (OpenAPI filtering).
		content = `{"note": "OpenAPI preview not yet implemented"}`
	default:
		return webutil.StatusBadRequest(c, "unknown file")
	}

	return webutil.Response(c, fiber.StatusOK, "preview", fiber.Map{"content": content})
}

// SEODefaults returns default templates and generated rules.
func SEODefaults(c fiber.Ctx) error {
	data, err := seo.Load(c.Context(), queries.Conn(), c.BaseURL())
	if err != nil {
		return webutil.StatusInternalServerError(c)
	}

	// Generate default sitemap include rules: "/", "/products/*", plus one per active page.
	sitemapInclude := []string{"/", "/products/*"}
	for _, p := range data.Pages {
		sitemapInclude = append(sitemapInclude, "/"+p.Slug)
	}

	defaults := fiber.Map{
		"llms":             seo.Default("llms"),
		"ai":               seo.Default("ai"),
		"robots":           seo.Default("robots"),
		"sitemap_include":  strings.Join(sitemapInclude, "\n"),
		"openapi_exclude":  "/api/_/*\n/api/install*\n/api/customer/*\n/cart/*",
	}

	return webutil.Response(c, fiber.StatusOK, "defaults", defaults)
}
