package seo

import (
	"context"
	"fmt"
	"time"

	"github.com/shurco/mycart/internal/database"
	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
)

// Data holds the bindings for Liquid template rendering.
type Data struct {
	Site     SiteInfo
	Products []ProductInfo
	Pages    []PageInfo
}

// SiteInfo provides {{ site.* }} variables.
type SiteInfo struct {
	Name    string
	Email   string
	Tagline string
	URL     string
}

// ProductInfo provides {{ product.* }} variables.
type ProductInfo struct {
	Name        string
	Slug        string
	URL         string
	Brief       string
	Description string
	Updated     string // YYYY-MM-DD
}

// PageInfo provides {{ page.* }} variables.
type PageInfo struct {
	Name    string
	Slug    string
	URL     string
	Updated string // YYYY-MM-DD
}

// Load fetches site settings and active products/pages from the database.
// baseURL is the site's public URL (from c.BaseURL() or settings).
func Load(ctx context.Context, conn *database.Conn, baseURL string) (Data, error) {
	db := queries.NewBase(conn)

	// Load site settings.
	main, err := queries.GetSettingByGroup[models.Main](ctx, db)
	if err != nil {
		return Data{}, fmt.Errorf("load main settings: %w", err)
	}
	branding, err := queries.GetSettingByGroup[models.Branding](ctx, db)
	if err != nil {
		return Data{}, fmt.Errorf("load branding: %w", err)
	}

	siteURL := baseURL
	if main.Domain != "" {
		siteURL = "https://" + main.Domain
	}

	data := Data{
		Site: SiteInfo{
			Name:    main.SiteName,
			Email:   main.Email,
			Tagline: branding.Tagline,
			URL:     siteURL,
		},
	}

	// Load active products (capped at 50k per sitemap spec).
	products, err := db.SEOQueries.ActiveProductsForSEO(ctx, 50000)
	if err != nil {
		return Data{}, fmt.Errorf("load products: %w", err)
	}
	for _, p := range products {
		data.Products = append(data.Products, ProductInfo{
			Name:        p.Name,
			Slug:        p.Slug,
			URL:         siteURL + "/products/" + p.Slug,
			Brief:       p.Brief,
			Description: p.Description,
			Updated:     formatDate(p.Updated),
		})
	}

	// Load active pages.
	pages, err := db.SEOQueries.ActivePagesForSEO(ctx, 50000)
	if err != nil {
		return Data{}, fmt.Errorf("load pages: %w", err)
	}
	for _, pg := range pages {
		data.Pages = append(data.Pages, PageInfo{
			Name:    pg.Name,
			Slug:    pg.Slug,
			URL:     siteURL + "/" + pg.Slug,
			Updated: formatDate(pg.Updated),
		})
	}

	return data, nil
}

// Bindings returns a map suitable for liquid.Engine.ParseAndRenderString.
func (d Data) Bindings() map[string]any {
	products := make([]map[string]any, len(d.Products))
	for i, p := range d.Products {
		products[i] = map[string]any{
			"name":        p.Name,
			"slug":        p.Slug,
			"url":         p.URL,
			"brief":       p.Brief,
			"description": p.Description,
			"updated":     p.Updated,
		}
	}

	pages := make([]map[string]any, len(d.Pages))
	for i, p := range d.Pages {
		pages[i] = map[string]any{
			"name":    p.Name,
			"slug":    p.Slug,
			"url":     p.URL,
			"updated": p.Updated,
		}
	}

	return map[string]any{
		"site": map[string]any{
			"name":    d.Site.Name,
			"email":   d.Site.Email,
			"tagline": d.Site.Tagline,
			"url":     d.Site.URL,
		},
		"products":    products,
		"pages":       pages,
		"sitemap_url": d.Site.URL + "/sitemap.xml",
		"now":         time.Now().UTC().Format("2006-01-02"),
	}
}

func formatDate(unixSec int64) string {
	if unixSec == 0 {
		return ""
	}
	return time.Unix(unixSec, 0).UTC().Format("2006-01-02")
}
