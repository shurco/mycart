package queries

import (
	"context"
	"fmt"

	"github.com/shurco/mycart/internal/database"
)

// SEOQueries holds a dialect-aware database handle for SEO-related queries.
type SEOQueries struct {
	DB *database.Conn
}

// SEOEntry represents a product or page for crawler-facing output.
type SEOEntry struct {
	Name        string
	Slug        string
	Brief       string
	Description string
	Updated     int64 // unix seconds
}

// ActiveProductsForSEO lists active products for sitemaps and crawlers.
// Capped to avoid unbounded memory growth on one request.
func (q *SEOQueries) ActiveProductsForSEO(ctx context.Context, limit int) ([]SEOEntry, error) {
	query := fmt.Sprintf(`SELECT name, slug, brief, "desc",
		%s FROM product WHERE active = TRUE ORDER BY name LIMIT ?`,
		q.DB.Dialect().Epoch("COALESCE(updated, created)"))

	rows, err := q.DB.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("seo products: %w", err)
	}
	defer rows.Close()

	var entries []SEOEntry
	for rows.Next() {
		var e SEOEntry
		if err := rows.Scan(&e.Name, &e.Slug, &e.Brief, &e.Description, &e.Updated); err != nil {
			return nil, fmt.Errorf("scan product: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate products: %w", err)
	}
	return entries, nil
}

// ActivePagesForSEO lists active pages for sitemaps and crawlers.
func (q *SEOQueries) ActivePagesForSEO(ctx context.Context, limit int) ([]SEOEntry, error) {
	query := fmt.Sprintf(`SELECT name, slug,
		%s FROM page WHERE active = TRUE ORDER BY name LIMIT ?`,
		q.DB.Dialect().Epoch("COALESCE(updated, created)"))

	rows, err := q.DB.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("seo pages: %w", err)
	}
	defer rows.Close()

	var entries []SEOEntry
	for rows.Next() {
		var e SEOEntry
		if err := rows.Scan(&e.Name, &e.Slug, &e.Updated); err != nil {
			return nil, fmt.Errorf("scan page: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pages: %w", err)
	}
	return entries, nil
}
