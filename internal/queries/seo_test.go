package queries

import (
	"testing"

	"github.com/shurco/mycart/pkg/security"
)

func TestActiveProductsForSEO(t *testing.T) {
	db, ctx := bootstrap(t)

	// Bootstrap creates a fresh DB; insert test products.
	_, err := Conn().ExecContext(ctx, `
		INSERT INTO product (id, name, slug, brief, "desc", amount, active)
		VALUES
		('p1', 'Product A', 'product-a', 'Brief A', 'Description A', 1000, TRUE),
		('p2', 'Product B', 'product-b', 'Brief B', 'Description B', 2000, TRUE),
		('p3', 'Product C', 'product-c', 'Brief C', 'Description C', 3000, FALSE)
	`)
	if err != nil {
		t.Fatalf("insert products: %v", err)
	}

	entries, err := db.SEOQueries.ActiveProductsForSEO(ctx, 100)
	if err != nil {
		t.Fatalf("ActiveProductsForSEO: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 active products, got %d", len(entries))
	}

	// Sorted by name, so A then B.
	if entries[0].Name != "Product A" || entries[0].Slug != "product-a" ||
		entries[0].Brief != "Brief A" || entries[0].Description != "Description A" {
		t.Errorf("first product wrong: %+v", entries[0])
	}
	if entries[0].Updated <= 0 {
		t.Errorf("expected positive unix timestamp, got %d", entries[0].Updated)
	}
}

func TestActivePagesForSEO(t *testing.T) {
	db, ctx := bootstrap(t)

	// Insert test pages (some bootstrap data already exists).
	_, err := Conn().ExecContext(ctx, `
		INSERT INTO page (id, name, slug, position, active)
		VALUES
		('page1', 'About Us', 'about', 'header', TRUE),
		('page2', 'Contact', 'contact', 'footer', TRUE),
		('page3', 'Hidden', 'hidden', 'footer', FALSE)
	`)
	if err != nil {
		t.Fatalf("insert pages: %v", err)
	}

	entries, err := db.SEOQueries.ActivePagesForSEO(ctx, 100)
	if err != nil {
		t.Fatalf("ActivePagesForSEO: %v", err)
	}

	// At least 2 from above, plus bootstrap defaults (terms, privacy if active).
	if len(entries) < 2 {
		t.Fatalf("expected at least 2 active pages, got %d", len(entries))
	}

	// Find our inserted pages.
	found := 0
	for _, e := range entries {
		if e.Name == "About Us" || e.Name == "Contact" {
			found++
			if e.Updated <= 0 {
				t.Errorf("expected positive unix timestamp for %q, got %d", e.Name, e.Updated)
			}
		}
		if e.Name == "Hidden" {
			t.Errorf("inactive page %q should not be returned", e.Name)
		}
	}
	if found != 2 {
		t.Errorf("expected to find 2 test pages, found %d", found)
	}
}

func TestActiveProductsForSEO_Limit(t *testing.T) {
	db, ctx := bootstrap(t)

	// Insert 5 products.
	for i := 1; i <= 5; i++ {
		_, err := Conn().ExecContext(ctx, `
			INSERT INTO product (id, name, slug, brief, "desc", amount, active)
			VALUES (?, ?, ?, ?, ?, ?, TRUE)
		`, security.RandomString(), "Product "+string(rune('0'+i)), "product-"+string(rune('0'+i)), "", "", 1000)
		if err != nil {
			t.Fatalf("insert product %d: %v", i, err)
		}
	}

	entries, err := db.SEOQueries.ActiveProductsForSEO(ctx, 3)
	if err != nil {
		t.Fatalf("ActiveProductsForSEO: %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected limit 3, got %d", len(entries))
	}
}
