package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/shurco/litecart/internal/testutil"
)

func TestPing(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/ping", Ping)

	resp := testutil.DoRequest(t, app, http.MethodGet, "/ping", "", "")
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var res struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)

	if !res.Success {
		t.Error("expected success=true")
	}
	if res.Message != "Pong" {
		t.Errorf("message = %q, want Pong", res.Message)
	}
}

func TestSettings(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Get("/api/settings", Settings)

	resp := testutil.DoRequest(t, app, http.MethodGet, "/api/settings", "", "")
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var res struct {
		Success bool `json:"success"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)

	if !res.Success {
		t.Error("expected success=true")
	}
}
