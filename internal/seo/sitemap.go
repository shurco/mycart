package seo

import (
	"encoding/xml"
	"fmt"

	"github.com/shurco/mycart/internal/models"
)

type sitemapURL struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod,omitempty"`
}

type urlSet struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// BuildSitemap generates a sitemap.xml from data, filtered by SEO settings.
func BuildSitemap(d Data, s *models.SEO) ([]byte, error) {
	m := NewMatcher(s.SitemapInclude, s.SitemapExclude)
	set := urlSet{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9"}

	add := func(p, lastmod string) {
		if m.Allow(p) {
			set.URLs = append(set.URLs, sitemapURL{
				Loc:     d.Site.URL + p,
				LastMod: lastmod,
			})
		}
	}

	add("/", "")
	for _, p := range d.Products {
		add("/products/"+p.Slug, p.Updated)
	}
	for _, p := range d.Pages {
		add("/"+p.Slug, p.Updated)
	}

	body, err := xml.MarshalIndent(set, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal sitemap: %w", err)
	}
	return append([]byte(xml.Header), body...), nil
}
