package queries

import (
	"context"
	"testing"
	"time"

	"github.com/shurco/mycart/internal/models"
	"github.com/shurco/mycart/migrations"
)

// bootstrap initialises a fresh DB in a temp workdir and returns the *Base.
// The returned context has a 5s deadline so hung queries surface as test
// failures rather than timeouts.
func bootstrap(t *testing.T) (*Base, context.Context) {
	t.Helper()
	cleanup := withTempBase(t)
	t.Cleanup(cleanup)
	if err := New(migrations.Embed()); err != nil {
		t.Fatalf("init queries: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return DB(), ctx
}

func TestGroupFieldMap_AllKnownTypes(t *testing.T) {
	// GroupFieldMap is pure, no DB needed.
	q := SettingQueries{}
	types := []any{
		&models.Main{}, &models.Auth{}, &models.JWT{}, &models.Social{},
		&models.Payment{}, &models.Stripe{}, &models.Paypal{}, &models.Spectrocoin{},
		&models.Coinbase{}, &models.Dummy{}, &models.Webhook{}, &models.Mail{},
	}
	for _, v := range types {
		v := v
		t.Run("known", func(t *testing.T) {
			if m := q.GroupFieldMap(v); len(m) == 0 {
				t.Errorf("empty map for %T", v)
			}
		})
	}

	if q.GroupFieldMap(&struct{}{}) != nil {
		t.Error("unknown type should return nil")
	}
}

func TestGetSettingByKey_EmptyKeys(t *testing.T) {
	// Explicit empty-keys guard avoids building an invalid `IN ()` clause.
	db, ctx := bootstrap(t)
	if _, err := db.GetSettingByKey(ctx); err == nil {
		t.Error("expected error for empty keys")
	}
}

func TestGetSettingByKey_UnknownReturnsEmptyMap(t *testing.T) {
	db, ctx := bootstrap(t)
	got, err := db.GetSettingByKey(ctx, "totally-missing-key")
	if err != nil {
		t.Fatalf("GetSettingByKey: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestGetAndUpdateSettingByGroup_Stripe(t *testing.T) {
	db, ctx := bootstrap(t)

	newKey := "sk_live_" + longString(100)
	err := db.UpdateSettingByGroup(ctx, &models.Stripe{SecretKey: newKey, Active: true})
	if err != nil {
		t.Fatalf("UpdateSettingByGroup: %v", err)
	}

	out, err := GetSettingByGroup[models.Stripe](ctx, db)
	if err != nil {
		t.Fatalf("GetSettingByGroup[Stripe]: %v", err)
	}
	if out.SecretKey != newKey || !out.Active {
		t.Errorf("persisted wrong values: %+v", out)
	}
}

func TestGetSettingByGroup_UnsupportedType(t *testing.T) {
	db, ctx := bootstrap(t)
	if _, err := db.GetSettingByGroup(ctx, &struct{}{}); err == nil {
		t.Error("expected ErrSettingNotFound for unsupported type")
	}
}

func TestUpdateSettingByKey_Roundtrip(t *testing.T) {
	db, ctx := bootstrap(t)

	err := db.UpdateSettingByKey(ctx, &models.SettingName{Key: "site_name", Value: "litecart-test"})
	if err != nil {
		t.Fatalf("UpdateSettingByKey: %v", err)
	}

	got, err := db.GetSettingByKey(ctx, "site_name")
	if err != nil {
		t.Fatalf("GetSettingByKey: %v", err)
	}
	if got["site_name"].Value.(string) != "litecart-test" {
		t.Errorf("site_name = %v, want litecart-test", got["site_name"].Value)
	}
}

func TestUpdatePassword_UserNotFound_WrongPassword_Success(t *testing.T) {
	db, ctx := bootstrap(t)

	// Empty password setting = UserNotFound path.
	if err := db.UpdatePassword(ctx, &models.Password{Old: "any", New: "newpass"}); err == nil {
		t.Error("expected error when password is empty (user not initialised)")
	}

	// Install gives us a password hash to compare against.
	if err := db.Install(ctx, &models.Install{
		Email:    "admin@example.com",
		Password: "initialpass",
	}); err != nil {
		t.Fatalf("Install: %v", err)
	}

	// Wrong old password path.
	err := db.UpdatePassword(ctx, &models.Password{Old: "wrong", New: "newpass"})
	if err == nil {
		t.Error("expected wrong-password error")
	}

	// Happy path.
	err = db.UpdatePassword(ctx, &models.Password{Old: "initialpass", New: "newpassword"})
	if err != nil {
		t.Fatalf("UpdatePassword happy path: %v", err)
	}
}

func longString(n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = 'x'
	}
	return string(out)
}
