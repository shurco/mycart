package jwtutil

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken is returned when the token fails validation (bad signature,
// wrong algorithm, missing/invalid claims, expired, etc).
var ErrInvalidToken = errors.New("invalid token")

// TokenMetadata is the subset of claims extracted from a valid token.
type TokenMetadata struct {
	ID      string
	Expires int64
}

// ParserOptions pins both the token format and the issuer/audience. These are
// shared by every verifier so a token cannot be valid on one protected route
// and invalid on another.
func ParserOptions() []jwt.ParserOption {
	return []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithAudience(Audience),
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	}
}

// ValidateHeader rejects tokens with the generic JWT type or another media
// type, even if their signature and registered claims are otherwise valid.
func ValidateHeader(token *jwt.Token) error {
	typ, _ := token.Header["typ"].(string)
	if typ != TokenType {
		return fmt.Errorf("%w: unexpected token type", ErrInvalidToken)
	}
	return nil
}

// ValidateSessionClaims requires the identity claims used by the revocation
// model. jti is the DB session key; sub must identify that same session.
func ValidateSessionClaims(claims jwt.MapClaims) (string, error) {
	id, _ := claims["jti"].(string)
	if id == "" {
		return "", fmt.Errorf("%w: missing jti", ErrInvalidToken)
	}
	sub, err := claims.GetSubject()
	if err != nil || sub != id {
		return "", fmt.Errorf("%w: missing or invalid sub", ErrInvalidToken)
	}
	clientID, _ := claims["client_id"].(string)
	if clientID != ClientID {
		return "", fmt.Errorf("%w: missing or invalid client_id", ErrInvalidToken)
	}
	return id, nil
}

// SessionID is retained as a concise name for callers that need the revocation
// identifier from fully validated claims.
func SessionID(claims jwt.MapClaims) (string, error) {
	return ValidateSessionClaims(claims)
}

// ExtractTokenMetadata parses and validates the JWT from the request cookie
// and returns its claims.
func ExtractTokenMetadata(c fiber.Ctx, secret string) (*TokenMetadata, error) {
	token, err := verifyToken(c, secret)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	id, err := SessionID(claims)
	if err != nil {
		return nil, err
	}
	exp, err := claims.GetExpirationTime()
	if err != nil || exp == nil {
		return nil, fmt.Errorf("%w: missing exp", ErrInvalidToken)
	}
	return &TokenMetadata{ID: id, Expires: exp.Unix()}, nil
}

// verifyToken enforces the exact HS256 method used by GenerateNewToken. The
// method check in the Keyfunc protects against algorithm substitution before
// a key is returned; ParserOptions applies it again to the parser itself.
func verifyToken(c fiber.Ctx, secret string) (*jwt.Token, error) {
	return jwt.Parse(c.Cookies("token"), func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		if err := ValidateHeader(token); err != nil {
			return nil, err
		}
		return []byte(secret), nil
	}, ParserOptions()...)
}
