package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
	"github.com/shurco/mycart/internal/seo"
	"github.com/shurco/mycart/pkg/logging"
)

// loadSEO fetches SEO settings and data for rendering.
func loadSEO(c fiber.Ctx) (*models.SEO, seo.Data, error) {
	db := queries.DB()
	ctx := c.Context()

	s, err := queries.GetSettingByGroup[models.SEO](ctx, db)
	if err != nil {
		return nil, seo.Data{}, err
	}

	data, err := seo.Load(ctx, queries.Conn(), c.BaseURL())
	if err != nil {
		return nil, seo.Data{}, err
	}

	return s, data, nil
}

// Sitemap serves /sitemap.xml with active products and pages.
func Sitemap(c fiber.Ctx) error {
	log := logging.New()
	s, d, err := loadSEO(c)
	if err != nil {
		log.ErrorStack(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	if !s.SitemapEnabled {
		return c.SendStatus(fiber.StatusNotFound)
	}

	body, err := seo.BuildSitemap(d, s)
	if err != nil {
		log.ErrorStack(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	c.Set(fiber.HeaderContentType, "application/xml; charset=utf-8")
	return c.Send(body)
}

// seoTextFiles maps file keys to their MIME types and accessors.
var seoTextFiles = map[string]struct {
	mime     string
	enabled  func(*models.SEO) bool
	template func(*models.SEO) string
}{
	"llms":   {"text/markdown; charset=utf-8", func(s *models.SEO) bool { return s.LlmsEnabled }, func(s *models.SEO) string { return s.LlmsTemplate }},
	"ai":     {"text/plain; charset=utf-8", func(s *models.SEO) bool { return s.AiEnabled }, func(s *models.SEO) string { return s.AiTemplate }},
	"robots": {"text/plain; charset=utf-8", func(s *models.SEO) bool { return s.RobotsEnabled }, func(s *models.SEO) string { return s.RobotsTemplate }},
}

// SEOText returns a handler for llms.txt, ai.txt, or robots.txt.
func SEOText(file string) fiber.Handler {
	f := seoTextFiles[file]
	return func(c fiber.Ctx) error {
		log := logging.New()
		s, d, err := loadSEO(c)
		if err != nil {
			log.ErrorStack(err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		tpl := f.template(s)
		if !f.enabled(s) {
			// User decision: robots.txt returns 404 when disabled (not always-on).
			return c.SendStatus(fiber.StatusNotFound)
		}

		if tpl == "" {
			tpl = seo.Default(file)
		}

		out, err := seo.Render(tpl, d)
		if err != nil {
			log.ErrorStack(err) // Never echo template errors publicly.
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		c.Set(fiber.HeaderContentType, f.mime)
		c.Set("X-Content-Type-Options", "nosniff")
		return c.SendString(out)
	}
}
