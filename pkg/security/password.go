package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost controls how expensive password hashing is. bcrypt.DefaultCost (=10)
// is recommended for 2020+ hardware and is what the standard library defaults to.
// We deliberately do NOT use bcrypt.MinCost (=4), which finishes in milliseconds
// and would be trivially brute-forceable from a stolen hash.
const bcryptCost = bcrypt.DefaultCost

// HashPassword returns a bcrypt hash of the plaintext password using bcryptCost.
// bcrypt refuses an input longer than 72 bytes rather than silently hashing only
// the first 72 — the refusal is returned as an error, because a caller that
// persisted it would store an error message in the password column and leave an
// account nothing can open.
func HashPassword(p string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(p), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// DummyPasswordHash is a bcrypt hash of an unrelated random password.
//
// A sign-in that has no stored hash to compare against — the address is not
// registered — compares the supplied password against this one instead, so that
// a failed attempt costs the same time whether or not the address exists.
// Without it the two paths differ measurably, and that difference is enough to
// enumerate the shop's accounts.
const DummyPasswordHash = "$2a$10$uo0yzV.HzLdx6hKO4F/4oOM0UQdmaLmZydgeRg9IDdL0lQjZZmoFq"

// ComparePasswords checks if the plaintext password matches the stored hash.
func ComparePasswords(hashedPwd, inputPwd string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(inputPwd)) == nil
}

// NewSigningKey returns a 256-bit cryptographically random signing key. Unlike
// passwords, JWT keys do not need to be derived from user input or slowed by a
// password hash.
func NewSigningKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("generate signing key: %w", err)
	}
	return hex.EncodeToString(key), nil
}

// NewToken returns a deterministic-looking but unpredictable token derived from
// the input. Password-derived JWT signing keys should use NewSigningKey instead;
// this helper remains for callers that intentionally need derivation from input.
//
// Construction: bcrypt(input, DefaultCost) -> hex encoding.
// bcrypt supplies a random salt (128 bits) and produces a cryptographically
// secure hash that is hex-encoded for use as a secret.
func NewToken(text string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return hex.EncodeToString(hash), nil
}
