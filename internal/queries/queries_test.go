package queries

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/migrations"
)

func withTempBase(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	oldwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to temp: %v", err)
	}
	_ = os.MkdirAll("lc_base", 0o775)
	_ = os.MkdirAll("lc_uploads", 0o775)
	_ = os.MkdirAll("lc_digitals", 0o775)
	return func() { _ = os.Chdir(oldwd) }
}

func Test_queries_init_and_settings(t *testing.T) {
	cleanup := withTempBase(t)
	defer cleanup()

	if err := New(migrations.Embed()); err != nil {
		t.Fatalf("init queries: %v", err)
	}

	db := DB()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	setting, err := db.GetSettingByKey(ctx, "installed")
	if err != nil {
		t.Fatalf("get setting: %v", err)
	}
	if _, ok := setting["installed"]; !ok {
		t.Fatalf("installed key not found")
	}
}

func Test_queries_page_crud(t *testing.T) {
	cleanup := withTempBase(t)
	defer cleanup()
	if err := New(migrations.Embed()); err != nil {
		t.Fatalf("init queries: %v", err)
	}
	db := DB()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	page, err := db.AddPage(ctx, &models.Page{Name: "Test", Slug: "test", Position: "footer"})
	if err != nil {
		t.Fatalf("add page: %v", err)
	}
	if page.Created == 0 {
		t.Fatalf("expected created timestamp")
	}

	list, _, err := db.ListPages(ctx, true, 0, 0)
	if err != nil {
		t.Fatalf("list pages: %v", err)
	}
	if len(list) == 0 {
		t.Fatalf("expected pages > 0")
	}

	if err := db.UpdatePageContent(ctx, &models.Page{Core: models.Core{ID: page.ID}, Content: ptr("content")}); err != nil {
		t.Fatalf("update content: %v", err)
	}
}

func ptr[T any](v T) *T { return &v }
