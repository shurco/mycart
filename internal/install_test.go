package app

import (
	"context"
	"os"
	"testing"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/internal/queries"
)

func TestInstallAdmin_CreatesAdminAccount(t *testing.T) {
	prev, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	ctx := context.Background()
	install := &models.Install{
		Email:    "admin@example.com",
		Password: "secret12",
		Domain:   "example.com",
	}

	if err := InstallAdmin(ctx, install); err != nil {
		t.Fatalf("InstallAdmin: %v", err)
	}

	installed, err := queries.DB().IsInstalled(ctx)
	if err != nil {
		t.Fatalf("IsInstalled: %v", err)
	}
	if !installed {
		t.Fatal("expected installed=true")
	}

	if err := InstallAdmin(ctx, install); err == nil {
		t.Fatal("expected second InstallAdmin to fail")
	}
}

func TestInstallAdmin_ValidationError(t *testing.T) {
	prev, _ := os.Getwd()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	if err := InstallAdmin(context.Background(), &models.Install{
		Email:    "bad",
		Password: "secret12",
	}); err == nil {
		t.Fatal("expected validation error")
	}

	if _, err := os.Stat("lc_base/data.db"); err == nil {
		t.Fatal("did not expect database file on validation failure")
	}
}
