package handlers

import (
	"reflect"
	"testing"

	"github.com/shurco/mycart/internal/models"
)

func TestSettingModelFor_KnownKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		key  string
		want any
	}{
		{"main", &models.Main{}},
		{"social", &models.Social{}},
		{"auth", &models.Auth{}},
		{"jwt", &models.JWT{}},
		{"webhook", &models.Webhook{}},
		{"payment", &models.Payment{}},
		{"stripe", &models.Stripe{}},
		{"paypal", &models.Paypal{}},
		{"spectrocoin", &models.Spectrocoin{}},
		{"coinbase", &models.Coinbase{}},
		{"dummy", &models.Dummy{}},
		{"mail", &models.Mail{}},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.key, func(t *testing.T) {
			t.Parallel()

			got := settingModelFor(tc.key)
			if got == nil {
				t.Fatalf("settingModelFor(%q) returned nil", tc.key)
			}
			if reflect.TypeOf(got) != reflect.TypeOf(tc.want) {
				t.Errorf("type = %T, want %T", got, tc.want)
			}
		})
	}
}

func TestSettingModelFor_UnknownKey(t *testing.T) {
	t.Parallel()

	if got := settingModelFor("not-a-real-group"); got != nil {
		t.Errorf("expected nil for unknown key, got %T", got)
	}
	// "password" is intentionally absent from the registry (write-only).
	if got := settingModelFor("password"); got != nil {
		t.Errorf("password must not be resolvable via the registry, got %T", got)
	}
}

// TestSettingModelFor_FreshInstance guards against the registry ever being
// refactored to cache a shared pointer, which would leak state between
// concurrent requests.
func TestSettingModelFor_FreshInstance(t *testing.T) {
	t.Parallel()

	a := settingModelFor("main")
	b := settingModelFor("main")
	if a == b {
		t.Error("settingModelFor returned the same pointer twice; must return fresh instances")
	}
}
