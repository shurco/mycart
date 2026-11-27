package models

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// maxSEOText caps each template and rule list; they are rendered per request.
const maxSEOText = 64 << 10

// SEO is the "seo" setting group: crawler-facing files and CDN cache rules.
// Every field is always sent, empty strings included (see Branding).
type SEO struct {
	SitemapEnabled bool   `json:"sitemap_enabled"`
	SitemapInclude string `json:"sitemap_include"` // one rule per line
	SitemapExclude string `json:"sitemap_exclude"`

	LlmsEnabled    bool   `json:"llms_enabled"`
	LlmsTemplate   string `json:"llms_template"`
	AiEnabled      bool   `json:"ai_enabled"`
	AiTemplate     string `json:"ai_template"`
	RobotsEnabled  bool   `json:"robots_enabled"`
	RobotsTemplate string `json:"robots_template"`

	OpenAPIEnabled bool   `json:"openapi_enabled"`
	OpenAPIInclude string `json:"openapi_include"`
	OpenAPIExclude string `json:"openapi_exclude"`

	CDNEnabled bool   `json:"cdn_enabled"`
	CDNRules   string `json:"cdn_rules"` // "/products/*|24h" per line or comma
}

func (v SEO) Validate() error {
	text := validation.Length(0, maxSEOText)
	return validation.ValidateStruct(&v,
		validation.Field(&v.SitemapInclude, text, validation.By(validGlobs)),
		validation.Field(&v.SitemapExclude, text, validation.By(validGlobs)),
		validation.Field(&v.LlmsTemplate, text),
		validation.Field(&v.AiTemplate, text),
		validation.Field(&v.RobotsTemplate, text),
		validation.Field(&v.OpenAPIInclude, text, validation.By(validGlobs)),
		validation.Field(&v.OpenAPIExclude, text, validation.By(validGlobs)),
		validation.Field(&v.CDNRules, text, validation.By(validCDNRules)),
	)
}

// Lines splits a rule list on newlines and commas, dropping blanks and # comments.
func Lines(s string) []string {
	var out []string
	for _, l := range strings.FieldsFunc(s, func(r rune) bool { return r == '\n' || r == ',' }) {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "#") {
			out = append(out, l)
		}
	}
	return out
}

func validGlobs(value any) error {
	for _, g := range Lines(value.(string)) {
		if !strings.HasPrefix(g, "/") {
			return fmt.Errorf("rule %q must start with /", g)
		}
		if _, err := path.Match(g, "/"); err != nil {
			return fmt.Errorf("rule %q: %w", g, err)
		}
	}
	return nil
}

// CDNRule is a cache rule: requests under Prefix get TTL.
type CDNRule struct {
	Prefix string
	TTL    time.Duration
}

// ParseCDNRules reads "/products/|24h, /sitemap.xml|10d"; the longest prefix wins at lookup.
func ParseCDNRules(s string) ([]CDNRule, error) {
	var rules []CDNRule
	for _, l := range Lines(s) {
		prefix, dur, ok := strings.Cut(l, "|")
		prefix = strings.TrimSpace(prefix)
		if !ok || !strings.HasPrefix(prefix, "/") {
			return nil, fmt.Errorf("cdn rule %q: want /path|duration", l)
		}
		ttl, err := parseTTL(strings.TrimSpace(dur))
		if err != nil {
			return nil, fmt.Errorf("cdn rule %q: %w", l, err)
		}
		rules = append(rules, CDNRule{strings.TrimSuffix(prefix, "*"), ttl})
	}
	sort.Slice(rules, func(i, j int) bool { return len(rules[i].Prefix) > len(rules[j].Prefix) })
	return rules, nil
}

func validCDNRules(value any) error {
	_, err := ParseCDNRules(value.(string))
	return err
}

// parseTTL adds a "d" (day) unit on top of time.ParseDuration.
func parseTTL(s string) (time.Duration, error) {
	if n, ok := strings.CutSuffix(s, "d"); ok {
		days, err := strconv.Atoi(n)
		if err != nil || days < 0 {
			return 0, fmt.Errorf("bad days %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 0, fmt.Errorf("bad duration %q", s)
	}
	return d, nil
}
