// Package services holds business rules, validation, and transactions.
package services

import (
	"net/mail"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"book-metaverse/backend/internal/domain"
)

const (
	maxEmailBytes       = 254
	maxDisplayNameRunes = 80
	maxAvatarURLBytes   = 2048
	maxBioRunes         = 2000
	maxGenreRunes       = 80
)

// NormalizeEmail matches the users_email_normalized check (lower(btrim(email))).
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateEmail(email string) error {
	if email == "" {
		return domain.Validation("email", "Email is required.")
	}
	addr, err := mail.ParseAddress(email)
	if len(email) > maxEmailBytes || err != nil || addr.Address != email || !strings.Contains(email[strings.LastIndex(email, "@"):], ".") {
		return domain.Validation("email", "Enter a valid email address.")
	}
	return nil
}

// validateText trims s and checks its length in characters and that it has no
// control characters (newlines are allowed only when multiline is true).
func validateText(field, label, s string, minRunes, maxRunes int, multiline bool) (string, error) {
	s = strings.TrimSpace(s)
	n := utf8.RuneCountInString(s)
	if n < minRunes {
		return "", domain.Validation(field, label+" is required.")
	}
	if n > maxRunes {
		return "", domain.Validation(field, label+" is too long.")
	}
	for _, r := range s {
		if unicode.IsControl(r) && !(multiline && (r == '\n' || r == '\r' || r == '\t')) {
			return "", domain.Validation(field, label+" contains characters that are not allowed.")
		}
	}
	return s, nil
}

func validateAvatarURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if raw == "" || len(raw) > maxAvatarURLBytes || err != nil || parsed.Host == "" ||
		(parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.User != nil {
		return "", domain.Validation("avatarUrl", "Avatar URL must be an http or https address.")
	}
	return raw, nil
}
