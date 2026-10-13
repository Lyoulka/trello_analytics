package auth

import (
	"fmt"
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost is the fixed cost for password hashing (Architect Decision).
const BcryptCost = 12

// MinPasswordLength is the minimum accepted password length.
const MinPasswordLength = 12

// ValidatePassword checks length (and future rules). Empty → error.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("auth: password too short")
	}
	return nil
}

// HashPassword returns a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword compares plaintext password against a stored bcrypt hash.
func CheckPassword(hash, password string) bool {
	if hash == "" || password == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

var (
	dummyHashOnce sync.Once
	dummyHash     string
)

// DummyPasswordHash returns a bcrypt hash used only to equalize login timing
// when the email does not exist or the user is inactive.
func DummyPasswordHash() string {
	dummyHashOnce.Do(func() {
		h, err := bcrypt.GenerateFromPassword([]byte("timing-dummy-password-not-used"), BcryptCost)
		if err != nil {
			panic("auth: failed to generate dummy password hash: " + err.Error())
		}
		dummyHash = string(h)
	})
	return dummyHash
}
