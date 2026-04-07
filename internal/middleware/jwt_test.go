package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/shurco/mycart/internal/testutil"
	"github.com/shurco/mycart/pkg/jwtutil"
)

func TestJWTProtected(t *testing.T) {
	_, _, cleanup := testutil.SetupTestApp(t)
	defer cleanup()

	app := fiber.New()
	app.Use(JWTProtected())
	app.Get("/api/test", func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

	validTok := mustGenerateToken(t, testutil.FixtureJWTSecret)
	wrongTok := mustGenerateToken(t, "wrongsecret")

	tests := []struct {
		name        string
		tokenHeader string
		wantStatus  []int
	}{
		{"no token", "", []int{http.StatusUnauthorized, http.StatusBadRequest}},
		{"valid token", "Bearer " + validTok, []int{http.StatusOK}},
		{"invalid token", "Bearer badtoken", []int{http.StatusUnauthorized, http.StatusBadRequest}},
		{"wrong secret", "Bearer " + wrongTok, []int{http.StatusUnauthorized, http.StatusBadRequest}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
			if tt.tokenHeader != "" {
				req.Header.Set("Authorization", tt.tokenHeader)
			}
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test: %v", err)
			}
			testutil.AssertStatus(t, resp, tt.wantStatus...)
		})
	}
}

func mustGenerateToken(t *testing.T, secret string) string {
	t.Helper()
	tok, err := jwtutil.GenerateNewToken(secret, uuid.NewString(), time.Now().Add(time.Hour).Unix(), nil)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return tok
}
