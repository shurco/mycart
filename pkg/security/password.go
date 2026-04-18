package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost controls how expensive password hashing is. bcrypt.DefaultCost (=10)
// is recommended for 2020+ hardware and is what the standard library defaults to.
// We deliberately do NOT use bcrypt.MinCost (=4), which finishes in milliseconds
// and would be trivially brute-forceable from a stolen hash.
const bcryptCost = bcrypt.DefaultCost

// GeneratePassword returns a bcrypt hash of the plaintext password using
// bcryptCost. If hashing fails (extremely unlikely — only on exhausted entropy
// or a bad cost constant), it returns the error string so the caller can store
// it in place of the hash. Callers should still check the result is a valid
// bcrypt-prefixed hash before persisting.
func GeneratePassword(p string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(p), bcryptCost)
	if err != nil {
		return err.Error()
	}
	return string(hash)
}

// ComparePasswords checks if the plaintext password matches the stored hash.
func ComparePasswords(hashedPwd, inputPwd string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(inputPwd)) == nil
}

// NewToken returns a deterministic-looking but unpredictable token derived from
// the input. It is used to materialize non-password secrets (e.g. JWT signing
// keys bootstrapped during install).
//
// Construction: bcrypt(input, DefaultCost) -> SHA-256 hex.
// bcrypt supplies a random salt (64 bits), SHA-256 then compacts the output to
// a fixed-length hex string suitable for use as a secret. We intentionally
// avoid MD5 here: MD5 is collision-broken and should never be used for any
// new security-relevant derivation.
func NewToken(text string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	sum := sha256.Sum256(hash)
	return hex.EncodeToString(sum[:]), nil
}
