package seo

import (
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/shurco/mycart/internal/models"
)

func TestDefault(t *testing.T) {
	t.Parallel()
	tests := []struct {
		file string
		want string
	}{
		{"robots", "User-agent: *"},
		{"llms", "# {{ site.name }}"},
		{"ai", "# ai.txt for"},
		{"nonexistent", ""},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			got := Default(tt.file)
			if tt.want == "" && got != "" {
				t.Errorf("Default(%q) = %q, want empty", tt.file, got)
			}
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Errorf("Default(%q) missing %q", tt.file, tt.want)
			}
		})
	}
}

func TestMatcher_Allow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		include string
		exclude string
		path    string
		want    bool
	}{
		{"empty include allows all", "", "", "/anything", true},
		{"explicit include", "/products/*", "", "/products/item", true},
		{"exclude wins", "/products/*", "/products/hidden", "/products/hidden", false},
		{"prefix match", "/api/*", "", "/api/v1/products", true},
		{"no match", "/api/*", "", "/pages/about", false},
		{"glob pattern", "/p[ag]*", "", "/pages", true},
		{"multiple rules", "/a\n/b", "", "/b", true},
		{"root", "/", "", "/", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMatcher(tt.include, tt.exclude)
			if got := m.Allow(tt.path); got != tt.want {
				t.Errorf("Allow(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRender(t *testing.T) {
	t.Parallel()
	data := Data{
		Site: SiteInfo{Name: "Test Shop", URL: "https://example.com"},
		Products: []ProductInfo{
			{Name: "Widget", Slug: "widget", URL: "https://example.com/products/widget"},
		},
	}

	tpl := "# {{ site.name }}\n{% for product in products %}- {{ product.name }}\n{% endfor %}"
	out, err := Render(tpl, data)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "Test Shop") || !strings.Contains(out, "Widget") {
		t.Errorf("Render output missing expected content: %q", out)
	}
}

func TestBuildSitemap(t *testing.T) {
	t.Parallel()
	data := Data{
		Site: SiteInfo{URL: "https://example.com"},
		Products: []ProductInfo{
			{Slug: "product-a", Updated: "2024-01-01"},
			{Slug: "product-b", Updated: "2024-01-02"},
		},
		Pages: []PageInfo{
			{Slug: "about", Updated: "2024-01-03"},
		},
	}

	seo := &models.SEO{
		SitemapInclude: "/\n/products/*\n/about",
		SitemapExclude: "/products/product-b",
	}

	out, err := BuildSitemap(data, seo)
	if err != nil {
		t.Fatalf("BuildSitemap: %v", err)
	}

	// Parse XML to verify structure.
	var set urlSet
	if err := xml.Unmarshal(out, &set); err != nil {
		t.Fatalf("parse sitemap XML: %v", err)
	}

	if set.Xmlns != "http://www.sitemaps.org/schemas/sitemap/0.9" {
		t.Errorf("xmlns = %q, want sitemaps.org namespace", set.Xmlns)
	}

	// Should have: /, /products/product-a, /about (product-b excluded).
	wantURLs := []string{
		"https://example.com/",
		"https://example.com/products/product-a",
		"https://example.com/about",
	}
	if len(set.URLs) != len(wantURLs) {
		t.Fatalf("got %d URLs, want %d", len(set.URLs), len(wantURLs))
	}
	for i, want := range wantURLs {
		if set.URLs[i].Loc != want {
			t.Errorf("URL[%d] = %q, want %q", i, set.URLs[i].Loc, want)
		}
	}
}

func TestFilterSpec(t *testing.T) {
	t.Parallel()
	spec := `{
		"openapi": "3.0.0",
		"paths": {
			"/api/products": {},
			"/api/_/settings": {},
			"/api/cart": {}
		}
	}`

	m := NewMatcher("/api/products\n/api/cart", "/api/_/*")
	out, err := FilterSpec([]byte(spec), m)
	if err != nil {
		t.Fatalf("FilterSpec: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("parse filtered spec: %v", err)
	}
	// Should not contain the excluded path.
	if strings.Contains(string(out), "/api/_/settings") {
		t.Error("filtered spec still contains excluded path /api/_/settings")
	}
	if !strings.Contains(string(out), "/api/products") {
		t.Error("filtered spec missing included path /api/products")
	}
}

func TestStoreCDN(t *testing.T) {
	t.Parallel()
	// Nil clears.
	StoreCDN(nil)
	if cdnRules.Load() != nil {
		t.Error("StoreCDN(nil) should clear rules")
	}

	// Disabled clears.
	StoreCDN(&models.SEO{CDNEnabled: false, CDNRules: "/test|1h"})
	if cdnRules.Load() != nil {
		t.Error("StoreCDN with disabled should clear rules")
	}

	// Valid rules store.
	StoreCDN(&models.SEO{CDNEnabled: true, CDNRules: "/products/*|24h"})
	rules := cdnRules.Load()
	if rules == nil || len(*rules) != 1 {
		t.Fatalf("StoreCDN should store 1 rule, got %v", rules)
	}
	if (*rules)[0].Prefix != "/products/" {
		t.Errorf("rule prefix = %q, want /products/", (*rules)[0].Prefix)
	}
}
