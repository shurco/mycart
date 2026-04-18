package jwtutil

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalidToken is returned when the token fails validation (bad signature,
// wrong algorithm, missing/invalid claims, expired, etc).
var ErrInvalidToken = errors.New("invalid token")

// TokenMetadata is the subset of claims extracted from a valid token.
type TokenMetadata struct {
	ID      string
	Expires int64
}

// ExtractTokenMetadata parses and validates the JWT from the request cookie
// and returns its claims. The signing algorithm is restricted to HS* (same
// family used by GenerateNewToken) to prevent "alg confusion" attacks.
func ExtractTokenMetadata(c fiber.Ctx, secret string) (*TokenMetadata, error) {
	token, err := verifyToken(c, secret)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	idStr, ok := claims["id"].(string)
	if !ok {
		return nil, fmt.Errorf("%w: missing id", ErrInvalidToken)
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	exp, ok := claims["expires"].(float64)
	if !ok {
		return nil, fmt.Errorf("%w: missing expires", ErrInvalidToken)
	}

	return &TokenMetadata{
		ID:      id.String(),
		Expires: int64(exp),
	}, nil
}

// verifyToken parses the raw token string from the "token" cookie and returns
// the parsed token. It enforces that the signing method belongs to the HMAC
// family to protect against algorithm substitution (e.g. alg=none, RS256↔HS256).
func verifyToken(c fiber.Ctx, secret string) (*jwt.Token, error) {
	return jwt.Parse(c.Cookies("token"), func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("%w: unexpected signing method: %v", ErrInvalidToken, token.Header["alg"])
		}
		return []byte(secret), nil
	})
}
