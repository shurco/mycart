package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/shurco/mycart/internal/testutil"
)

func TestGetSetting(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/_/settings/:setting_key", GetSetting)

	tests := []struct {
		name       string
		key        string
		wantStatus int
	}{
		{"main", "main", http.StatusOK},
		{"social", "social", http.StatusOK},
		{"payment", "payment", http.StatusOK},
		{"stripe", "stripe", http.StatusOK},
		{"paypal", "paypal", http.StatusOK},
		{"spectrocoin", "spectrocoin", http.StatusOK},
		{"coinbase", "coinbase", http.StatusOK},
		{"dummy", "dummy", http.StatusOK},
		{"jwt", "jwt", http.StatusOK},
		{"webhook", "webhook", http.StatusOK},
		{"mail", "mail", http.StatusOK},
		{"auth", "auth", http.StatusOK},
		{"password is blocked", "password", http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodGet, "/api/_/settings/"+tt.key, "", cookie)
			testutil.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestUpdateSetting(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Patch("/api/_/settings/:setting_key", UpdateSetting)

	tests := []struct {
		name       string
		key        string
		body       string
		wantStatus int
	}{
		{"update main", "main", `{"site_name":"Updated","domain":"u.com"}`, http.StatusOK},
		{"update social", "social", `{"facebook":"fb","github":"gh"}`, http.StatusOK},
		{"update webhook", "webhook", `{"url":"https://example.com/wh"}`, http.StatusOK},
		{"update payment currency", "payment", `{"currency":"EUR"}`, http.StatusOK},
		{"update password", "password", `{"old":"Pass123","new":"NewPass456"}`, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodPatch, "/api/_/settings/"+tt.key, tt.body, cookie)
			testutil.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: Version handler requires network access")
	}

	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/_/version", Version)

	req := httptest.NewRequest(http.MethodGet, "/api/_/version", nil)
	req.Header.Set("Cookie", cookie)
	resp, err := app.Test(req, fiber.TestConfig{Timeout: 30 * time.Second})
	if err != nil {
		t.Skipf("network issue: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 200 or 500", resp.StatusCode)
	}
}

func TestTestLetter(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/_/test/letter/:letter_name", TestLetter)

	tests := []struct {
		name string
		path string
	}{
		{"smtp test", "/api/_/test/letter/smtp"},
		{"letter template", "/api/_/test/letter/mail_letter_payment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodGet, tt.path, "", cookie)
			testutil.AssertStatus(t, resp, http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError)
		})
	}
}
