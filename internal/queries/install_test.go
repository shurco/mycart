package queries

import (
	"errors"
	"testing"

	"github.com/shurco/mycart/internal/models"
)

func TestInstall_HappyPath(t *testing.T) {
	db, ctx := bootstrap(t)
	err := db.Install(ctx, &models.Install{
		Email:    "admin@example.com",
		Password: "strongpass",
		Domain:   "example.com",
	})
	if err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Second install must be idempotent-blocking.
	err = db.Install(ctx, &models.Install{Email: "admin@example.com", Password: "other"})
	if !errors.Is(err, ErrAlreadyInstalled) {
		t.Fatalf("expected ErrAlreadyInstalled, got %v", err)
	}
}
