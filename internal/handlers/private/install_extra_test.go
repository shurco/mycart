package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/shurco/mycart/internal/testutil"
)

func TestInstall_MalformedJSON(t *testing.T) {
	app, cleanup := setupCleanDB(t)
	defer cleanup()
	app.Post("/api/install", Install)

	resp := testutil.DoRequest(t, app, http.MethodPost,
		"/api/install", "{not json", "")
	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestInstallStatus_NotInstalled(t *testing.T) {
	app, cleanup := setupCleanDB(t)
	defer cleanup()
	app.Get("/api/install/status", InstallStatus)

	resp := testutil.DoRequest(t, app, http.MethodGet, "/api/install/status", "", "")
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body struct {
		Success bool `json:"success"`
		Result  struct {
			Installed bool `json:"installed"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Result.Installed {
		t.Fatal("expected installed=false before setup")
	}
}

func TestInstallStatus_Installed(t *testing.T) {
	app, cleanup := setupCleanDB(t)
	defer cleanup()
	app.Post("/api/install", Install)
	app.Get("/api/install/status", InstallStatus)

	body := `{"email":"admin@example.com","password":"secret","domain":"example.com"}`
	resp := testutil.DoRequest(t, app, http.MethodPost, "/api/install", body, "")
	testutil.AssertStatus(t, resp, http.StatusOK)

	resp = testutil.DoRequest(t, app, http.MethodGet, "/api/install/status", "", "")
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var status struct {
		Result struct {
			Installed bool `json:"installed"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !status.Result.Installed {
		t.Fatal("expected installed=true after setup")
	}
}
