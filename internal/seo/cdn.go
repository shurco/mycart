package seo

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/gofiber/fiber/v3"
	"github.com/shurco/mycart/internal/models"
)

// neverCache: per-user or admin responses that must not leak across visitors.
var neverCache = []string{"/_", "/api/_", "/api/customer", "/api/cart", "/cart"}

var cdnRules atomic.Pointer[[]models.CDNRule]

// StoreCDN swaps the live CDN rules. Pass nil or disabled SEO to turn middleware off.
func StoreCDN(s *models.SEO) {
	if s == nil || !s.CDNEnabled {
		cdnRules.Store(nil)
		return
	}
	rules, err := models.ParseCDNRules(s.CDNRules)
	if err != nil {
		// Validation happened on save; only hand-edited DB reaches here.
		cdnRules.Store(nil)
		return
	}
	cdnRules.Store(&rules)
}

// CDNMiddleware sets Cache-Control headers for matching GET/HEAD requests.
func CDNMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		err := c.Next()
		rules := cdnRules.Load()

		// Skip if: error, no rules, not GET/HEAD, not 200, private path, or has Set-Cookie.
		if err != nil || rules == nil ||
			(c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead) ||
			c.Response().StatusCode() != fiber.StatusOK ||
			hasAnyPrefix(c.Path(), neverCache) ||
			len(c.Response().Header.Peek(fiber.HeaderSetCookie)) > 0 {
			return err
		}

		// Find matching rule (longest prefix wins).
		// Match path exactly or as prefix: "/products/" matches "/products" and "/products/item"
		path := c.Path()
		for _, r := range *rules {
			pfx := strings.TrimSuffix(r.Prefix, "/")
			if path == pfx || strings.HasPrefix(path, r.Prefix) {
				s := int(r.TTL.Seconds())
				// Edge caches for TTL; browsers revalidate after 60s max.
				c.Set(fiber.HeaderCacheControl, fmt.Sprintf("public, max-age=%d, s-maxage=%d", min(s, 60), s))
				break
			}
		}
		return nil
	}
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}
