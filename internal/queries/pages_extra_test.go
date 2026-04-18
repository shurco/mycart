package queries

import (
	"testing"

	"github.com/shurco/mycart/internal/models"
)

func TestPage_FullLifecycle(t *testing.T) {
	db, ctx := bootstrap(t)

	created, err := db.AddPage(ctx, &models.Page{Name: "About", Slug: "about", Position: "footer"})
	if err != nil {
		t.Fatalf("AddPage: %v", err)
	}

	if !db.IsPage(ctx, "about") {
		t.Error("IsPage must find the just-created page")
	}
	if db.IsPage(ctx, "does-not-exist") {
		t.Error("IsPage must return false for missing slug")
	}

	// ListPages(private=true) should include the inactive page.
	pages, total, err := db.ListPages(ctx, true, 10, 0)
	if err != nil {
		t.Fatalf("ListPages private: %v", err)
	}
	if total == 0 || len(pages) == 0 {
		t.Error("expected at least one page in private listing")
	}

	// Activate the page — UpdatePageActive toggles the flag.
	if err := db.UpdatePageActive(ctx, created.ID); err != nil {
		t.Fatalf("UpdatePageActive: %v", err)
	}
	publicPages, _, err := db.ListPages(ctx, false, 10, 0)
	if err != nil {
		t.Fatalf("ListPages public: %v", err)
	}
	found := false
	for _, p := range publicPages {
		if p.ID == created.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("public listing should include the activated page")
	}

	// PageByID must return the updated fields.
	fetched, err := db.PageByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("PageByID: %v", err)
	}
	if !fetched.Active {
		t.Error("page should be active after toggle")
	}

	// UpdatePage partial: only change name, everything else must survive.
	if err := db.UpdatePage(ctx, &models.Page{
		Core: models.Core{ID: created.ID},
		Name: "About Us",
	}); err != nil {
		t.Fatalf("UpdatePage: %v", err)
	}
	fetched, _ = db.PageByID(ctx, created.ID)
	if fetched.Name != "About Us" {
		t.Errorf("name = %s, want About Us", fetched.Name)
	}
	if fetched.Slug != "about" {
		t.Errorf("slug changed unexpectedly: %s", fetched.Slug)
	}

	// Full UpdatePage with SEO, content and position.
	content := "body"
	seo := &models.Seo{Title: "T", Description: "D", Keywords: "K"}
	if err := db.UpdatePage(ctx, &models.Page{
		Core:     models.Core{ID: created.ID},
		Name:     "About Us",
		Slug:     "about",
		Position: "header",
		Content:  &content,
		Seo:      seo,
	}); err != nil {
		t.Fatalf("UpdatePage full: %v", err)
	}

	full, err := db.PageByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("PageByID: %v", err)
	}
	if full.Position != "header" || full.Seo == nil || full.Seo.Title != "T" {
		t.Errorf("SEO/position not persisted: %+v", full)
	}

	if err := db.DeletePage(ctx, created.ID); err != nil {
		t.Fatalf("DeletePage: %v", err)
	}
	if _, err := db.PageByID(ctx, created.ID); err == nil {
		t.Error("expected error for deleted page")
	}
}

func TestPage_PageBySlug_NotFound(t *testing.T) {
	db, ctx := bootstrap(t)
	if _, err := db.Page(ctx, "missing"); err == nil {
		t.Error("expected ErrPageNotFound")
	}
}
