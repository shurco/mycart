package queries

import (
	"testing"

	"github.com/shurco/mycart/internal/models"
)

func TestGetPasswordByEmail(t *testing.T) {
	db, ctx := bootstrap(t)

	// Without installation the password is empty -> UserPasswordNotFound.
	if _, err := db.GetPasswordByEmail(ctx, "admin@example.com"); err == nil {
		t.Error("expected error before install")
	}

	if err := db.Install(ctx, &models.Install{
		Email:    "admin@example.com",
		Password: "secret123",
	}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	hash, err := db.GetPasswordByEmail(ctx, "admin@example.com")
	if err != nil {
		t.Fatalf("GetPasswordByEmail: %v", err)
	}
	if hash == "" {
		t.Error("empty hash returned")
	}

	// Unknown email must not leak the stored hash.
	if _, err := db.GetPasswordByEmail(ctx, "someone-else@example.com"); err == nil {
		t.Error("expected error for unknown email")
	}
}
