package jwtutil

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func signClaims(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["typ"] = TokenType
	signed, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign claims: %v", err)
	}
	return signed
}

func validClaims(id string) jwt.MapClaims {
	now := time.Now().Unix()
	return jwt.MapClaims{
		"iss":       Issuer,
		"aud":       Audience,
		"sub":       id,
		"iat":       now,
		"exp":       now + 3600,
		"jti":       id,
		"client_id": ClientID,
	}
}

func TestExtractTokenMetadata_RejectsIncompleteClaims(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		alter   func(jwt.MapClaims)
		wantSub string
	}{
		{"no jti", func(c jwt.MapClaims) { delete(c, "jti") }, "missing jti"},
		{"jti is not a string", func(c jwt.MapClaims) { c["jti"] = 42 }, "missing jti"},
		{"no exp", func(c jwt.MapClaims) { delete(c, "exp") }, "exp claim is required"},
		{"exp is not a number", func(c jwt.MapClaims) { c["exp"] = "tomorrow" }, "exp is invalid"},
		{"subject differs", func(c jwt.MapClaims) { c["sub"] = "another-session" }, "invalid sub"},
		{"expired", func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Minute).Unix() }, "expired"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			claims := validClaims("session-id")
			tt.alter(claims)

			meta, err := parseWithCookie(t, signClaims(t, claims))
			if err == nil {
				t.Fatalf("token with %s was accepted, metadata = %+v", tt.name, meta)
			}
			if meta != nil {
				t.Errorf("metadata returned alongside the error: %+v", meta)
			}
			if !errors.Is(err, ErrInvalidToken) {
				t.Errorf("error %v is not ErrInvalidToken", err)
			}
			if !strings.Contains(err.Error(), tt.wantSub) {
				t.Errorf("error %q does not mention %q", err, tt.wantSub)
			}
		})
	}
}

func TestExtractTokenMetadata_RejectsAnUnusableCookie(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "not-a-jwt", "a.b.c"} {
		meta, err := parseWithCookie(t, raw)
		if err == nil {
			t.Errorf("cookie %q was accepted, metadata = %+v", raw, meta)
		}
	}
}

func TestExtractTokenMetadata_RejectsWrongType(t *testing.T) {
	t.Parallel()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, validClaims("session-id"))
	raw, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	_, err = parseWithCookie(t, raw)
	if err == nil || !strings.Contains(err.Error(), "unexpected token type") {
		t.Fatalf("unexpected wrong-type result: %v", err)
	}
}
