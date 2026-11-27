package jwtutil

import (
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// TokenType identifies JWTs used as access tokens. It is deliberately not
	// the generic default "JWT" emitted by jwt-go.
	TokenType = "at+jwt"
	Issuer    = "mycart"
	Audience  = "mycart-api"
	ClientID  = "mycart"
)

type Setting struct {
	Secret            string
	SecretExpireHours int
}

// GenerateNewToken generates an HS256 access token. id is the server-side
// session identifier: keeping it in jti makes revocation unambiguous, while
// sub identifies the authenticated session subject for this first-party API.
func GenerateNewToken(secret, id string, expires int64, credentials []string) (string, error) {
	claims := jwt.MapClaims{
		"iss":       Issuer,
		"aud":       Audience,
		"sub":       id,
		"iat":       time.Now().Unix(),
		"exp":       expires,
		"jti":       id,
		"client_id": ClientID,
	}
	if len(credentials) > 0 {
		claims["scope"] = strings.Join(credentials, " ")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["typ"] = TokenType
	return token.SignedString([]byte(secret))
}
