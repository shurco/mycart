package jwtutil

import (
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenMetadata struct {
	ID      string
	Expires int64
}

func ExtractTokenMetadata(c fiber.Ctx, secret string) (*TokenMetadata, error) {
	token, err := verifyToken(c, secret)
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		id, err := uuid.Parse(claims["id"].(string))
		if err != nil {
			return nil, err
		}

		expires := int64(claims["expires"].(float64))
		return &TokenMetadata{
			ID:      id.String(),
			Expires: expires,
		}, nil
	}

	return nil, err
}

func verifyToken(c fiber.Ctx, secret string) (*jwt.Token, error) {
	token, err := jwt.Parse(c.Cookies("token"), func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}
