package handlers

import (
	"fmt"
	"os"
	"path/filepath"
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
		baseDir := os.Getenv("LITECART_BASE")
		if baseDir == "" {
			baseDir = "./lc_base"
		}
		spec, err := os.ReadFile(filepath.Join(baseDir, "swagger", "swagger.json"))
		if err != nil {
			content = `{"error": "Spec not generated yet"}`
		} else {
			m := seo.NewMatcher(s.OpenAPIInclude, s.OpenAPIExclude)
			filtered, _ := seo.FilterSpec(spec, m)
			content = string(filtered)
		}
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

	// Generate default sitemap include rules
	sitemapInclude := []string{"/products/*", "/"}
	for _, p := range data.Pages {
		sitemapInclude = append(sitemapInclude, "/"+p.Slug)
	}

	// Sitemap exclude: admin, cart, APIs
	sitemapExclude := []string{
		"/_/swagger",
		"/cart/*",
		"/api/_/*",
		"/api/*",
		"/_/*",
	}

	// Default CDN rules: zero-cache dynamic, 24h static
	cdnRules := []string{
		"/products/*|24h",
		"/|24h",
		"/_/swagger|0",
		"/cart/*|0",
		"/api/_/*|0",
		"/api/*|0",
		"/_/*|0",
	}

	defaults := fiber.Map{
		"llms":             seo.Default("llms"),
		"ai":               seo.Default("ai"),
		"robots":           seo.Default("robots"),
		"sitemap_include":  strings.Join(sitemapInclude, "\n"),
		"sitemap_exclude":  strings.Join(sitemapExclude, "\n"),
		"openapi_exclude":  "/api/_/*\n/api/install*\n/api/customer/*\n/cart/*",
		"cdn_rules":        strings.Join(cdnRules, "\n"),
	}

	return webutil.Response(c, fiber.StatusOK, "defaults", defaults)
}
