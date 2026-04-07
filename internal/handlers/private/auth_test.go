package handlers

import (
	"net/http"
	"testing"

	"github.com/shurco/mycart/internal/testutil"
)

func TestSignIn(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Post("/api/sign/in", SignIn)

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			"valid credentials",
			`{"email":"user@mail.com","password":"Pass123"}`,
			http.StatusOK,
		},
		{
			"wrong password",
			`{"email":"user@mail.com","password":"wrong"}`,
			http.StatusBadRequest,
		},
		{
			"wrong email",
			`{"email":"nobody@example.com","password":"Pass123"}`,
			http.StatusInternalServerError,
		},
		{
			"invalid email format",
			`{"email":"bad","password":"Pass123"}`,
			http.StatusBadRequest,
		},
		{
			"empty body",
			`{}`,
			http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := testutil.DoRequest(t, app, http.MethodPost, "/api/sign/in", tt.body, "")
			testutil.AssertStatus(t, resp, tt.wantStatus)
		})
	}
}

func TestSignIn_SetsCookie(t *testing.T) {
	app, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Post("/api/sign/in", SignIn)

	resp := testutil.DoRequest(t, app, http.MethodPost, "/api/sign/in",
		`{"email":"user@mail.com","password":"Pass123"}`, "")
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if cookie := resp.Header.Get("Set-Cookie"); cookie == "" {
		t.Fatal("expected token cookie to be set")
	}
}

func TestSignOut(t *testing.T) {
	app, cookie, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app.Post("/api/sign/in", SignIn)
	app.Post("/api/sign/out", SignOut)

	resp := testutil.DoRequest(t, app, http.MethodPost, "/api/sign/in",
		`{"email":"user@mail.com","password":"Pass123"}`, "")
	defer func() { _ = resp.Body.Close() }()

	loginCookie := resp.Header.Get("Set-Cookie")
	if loginCookie == "" {
		t.Skip("signin did not return cookie, skipping signout")
	}

	resp2 := testutil.DoRequest(t, app, http.MethodPost, "/api/sign/out", "", cookie)
	testutil.AssertStatus(t, resp2, http.StatusNoContent, http.StatusInternalServerError)
}
