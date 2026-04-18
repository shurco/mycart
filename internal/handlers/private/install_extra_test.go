package handlers

import (
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

func TestInstall_SecondAttemptFails(t *testing.T) {
	app, cleanup := setupCleanDB(t)
	defer cleanup()
	app.Post("/api/install", Install)

	body := `{"email":"admin@example.com","password":"secret","domain":"example.com"}`
	resp := testutil.DoRequest(t, app, http.MethodPost, "/api/install", body, "")
	testutil.AssertStatus(t, resp, http.StatusOK)

	// Second install should fail with ErrAlreadyInstalled → 400.
	resp = testutil.DoRequest(t, app, http.MethodPost, "/api/install", body, "")
	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}
