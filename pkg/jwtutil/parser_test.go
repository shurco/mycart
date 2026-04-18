package jwtutil

import (
	"errors"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "s3cret-key-for-tests"

// parseWithCookie spins up a throwaway Fiber app, drives a request with the
// given "token" cookie through it and returns whatever ExtractTokenMetadata
// would have seen. It is the simplest way to exercise the real Fiber context
// without pulling in internal/testutil (which expects a DB).
func parseWithCookie(t *testing.T, rawToken string) (*TokenMetadata, error) {
	t.Helper()

	app := fiber.New()
	var (
		meta *TokenMetadata
		err  error
	)
	app.Get("/", func(c fiber.Ctx) error {
		meta, err = ExtractTokenMetadata(c, testSecret)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest(fiber.MethodGet, "/", nil)
	req.Header.Set("Cookie", "token="+rawToken)

	resp, tErr := app.Test(req)
	if tErr != nil {
		t.Fatalf("app.Test: %v", tErr)
	}
	t.Cleanup(func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	})

	return meta, err
}

func TestExtractTokenMetadata_HappyPath(t *testing.T) {
	t.Parallel()

	id := uuid.NewString()
	expires := time.Now().Add(time.Hour).Unix()
	tok, err := GenerateNewToken(testSecret, id, expires, nil)
	if err != nil {
		t.Fatalf("GenerateNewToken: %v", err)
	}

	meta, err := parseWithCookie(t, tok)
	if err != nil {
		t.Fatalf("ExtractTokenMetadata: %v", err)
	}
	if meta == nil {
		t.Fatal("meta is nil")
	}
	if meta.ID != id {
		t.Errorf("ID: got %q, want %q", meta.ID, id)
	}
	if meta.Expires != expires {
		t.Errorf("Expires: got %d, want %d", meta.Expires, expires)
	}
}

// TestExtractTokenMetadata_RejectsAlgNone is the key security regression test:
// an attacker-crafted `alg: none` token must NOT be accepted, even though
// jwt-go permits `alg: none` when the Keyfunc returns the unsafe sentinel.
func TestExtractTokenMetadata_RejectsAlgNone(t *testing.T) {
	t.Parallel()

	claims := jwt.MapClaims{
		"id":      uuid.NewString(),
		"expires": float64(time.Now().Add(time.Hour).Unix()),
	}
	// jwt.UnsafeAllowNoneSignatureType is a marker value the library
	// requires a test/attacker to pass. Our Keyfunc must reject this path.
	unsafe := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	raw, err := unsafe.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("SignedString(none): %v", err)
	}

	_, err = parseWithCookie(t, raw)
	if err == nil {
		t.Fatal("expected error for alg=none token, got nil")
	}
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("error does not wrap ErrInvalidToken: %v", err)
	}
}

func TestExtractTokenMetadata_WrongSecret(t *testing.T) {
	t.Parallel()

	tok, err := GenerateNewToken("other-secret", uuid.NewString(), time.Now().Unix()+60, nil)
	if err != nil {
		t.Fatalf("GenerateNewToken: %v", err)
	}

	_, err = parseWithCookie(t, tok)
	if err == nil {
		t.Fatal("expected signature-mismatch error, got nil")
	}
}
