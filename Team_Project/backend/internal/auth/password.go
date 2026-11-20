// Package auth hashes passwords and issues and parses session tokens.
package auth

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// Cost matches the seeded demo accounts (pgcrypto, cost 12).
const Cost = 12

const (
	MinPasswordBytes = 8
	// bcrypt ignores everything after byte 72, so longer passwords are rejected
	// rather than silently truncated.
	MaxPasswordBytes = 72
)

var (
	ErrPasswordLength  = errors.New("password must be 8 to 72 bytes")
	ErrPasswordControl = errors.New("password must not contain control characters")
)

// dummyHash is compared when a login email does not exist, so a missing
// account takes about as long as a wrong password (BACKEND_PLAN §6.3).
var dummyHash = func() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), Cost)
	if err != nil {
		panic(err)
	}
	return hash
}()

// ValidatePassword checks the rules for a new password.
func ValidatePassword(password string) error {
	if len(password) < MinPasswordBytes || len(password) > MaxPasswordBytes {
		return ErrPasswordLength
	}
	for _, r := range password {
		if unicode.IsControl(r) {
			return ErrPasswordControl
		}
	}
	return nil
}

// HashPassword hashes with bcrypt at Cost.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), Cost)
	return string(hash), err
}

// CheckPassword compares in constant time; it accepts $2a$ (pgcrypto) and
// $2b$ hashes alike.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// CheckDummy spends the same work as CheckPassword for a missing account.
func CheckDummy(password string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
}
