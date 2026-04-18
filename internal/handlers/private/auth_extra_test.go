package handlers

import (
	"net/http"
	"testing"

	"github.com/shurco/mycart/internal/testutil"
)

func TestSignIn_MalformedJSON(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Post("/api/sign/in", SignIn)
	resp := testutil.DoRequest(t, app, http.MethodPost, "/api/sign/in", "{not json", "")
	testutil.AssertStatus(t, resp, http.StatusBadRequest)
}

func TestSignOut_WithoutCookieReturns500(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Post("/api/sign/out", SignOut)
	// No cookie → ExtractTokenMetadata fails → 500.
	resp := testutil.DoRequest(t, app, http.MethodPost, "/api/sign/out", "", "")
	testutil.AssertStatus(t, resp, http.StatusInternalServerError)
}
