package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	// CookieName is the HTTP-only session cookie.
	CookieName = "tta_session"
	// TokenBytes is the entropy of the raw cookie value before hashing.
	TokenBytes = 32
)

// HashToken returns hex-encoded SHA-256 of the raw cookie value (stored in DB).
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// NewSessionToken generates a cryptographically random cookie value.
func NewSessionToken() (raw string, err error) {
	b := make([]byte, TokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: session token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
