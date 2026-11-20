package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestPasswordHashAndCheck(t *testing.T) {
	hash, err := HashPassword("correct horse battery")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$2a$12$") {
		t.Fatalf("expected bcrypt cost 12, got %s", hash[:7])
	}
	if !CheckPassword(hash, "correct horse battery") || CheckPassword(hash, "wrong") {
		t.Fatal("check failed")
	}
}

func TestPasswordRules(t *testing.T) {
	if ValidatePassword("short") != ErrPasswordLength || ValidatePassword(strings.Repeat("a", 73)) != ErrPasswordLength {
		t.Fatal("length rule")
	}
	if ValidatePassword("abc\x00defgh") != ErrPasswordControl {
		t.Fatal("control rule")
	}
	if ValidatePassword(strings.Repeat("a", 72)) != nil {
		t.Fatal("72 bytes is allowed")
	}
}

func newTokens(now time.Time) *Tokens {
	t := NewTokens([]byte("test-secret-0123456789abcdef0123"))
	t.now = func() time.Time { return now }
	return t
}

func TestTokenRoundTrip(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	tokens := newTokens(now)
	id := uuid.New()
	signed, expires, err := tokens.Issue(id, "reader@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !expires.Equal(now.Add(60 * time.Minute)) {
		t.Fatalf("expiry %v", expires)
	}
	got, claims, err := tokens.Parse(signed)
	if err != nil || got != id || claims.Email != "reader@example.com" || claims.Issuer != Issuer {
		t.Fatalf("parse: %v %v %+v", err, got, claims)
	}
}

func TestTokenRejections(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	tokens := newTokens(now)
	id := uuid.New()
	signed, _, _ := tokens.Issue(id, "a@example.com")

	// Expired, beyond the 30-second leeway.
	if _, _, err := newTokens(now.Add(61 * time.Minute)).Parse(signed); err != ErrTokenExpired {
		t.Fatalf("expired: %v", err)
	}
	// Within the leeway.
	if _, _, err := newTokens(now.Add(60*time.Minute + 20*time.Second)).Parse(signed); err != nil {
		t.Fatalf("leeway: %v", err)
	}
	// Different secret.
	other := NewTokens([]byte("another-secret-0123456789abcdef01"))
	other.now = tokens.now
	if _, _, err := other.Parse(signed); err != ErrTokenInvalid {
		t.Fatalf("wrong key: %v", err)
	}

	claims := Claims{Email: "a@example.com", RegisteredClaims: jwt.RegisteredClaims{
		Subject: id.String(), Issuer: Issuer, IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
	}}
	// alg "none".
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, _, err := tokens.Parse(none); err != ErrTokenInvalid {
		t.Fatalf("alg none accepted: %v", err)
	}
	// HS512 with the right key is still rejected (HS256 only).
	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString(tokens.secret)
	if _, _, err := tokens.Parse(hs512); err != ErrTokenInvalid {
		t.Fatalf("HS512 accepted: %v", err)
	}
	// Wrong issuer.
	claims.Issuer = "someone-else"
	foreign, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(tokens.secret)
	if _, _, err := tokens.Parse(foreign); err != ErrTokenInvalid {
		t.Fatalf("foreign issuer accepted: %v", err)
	}
	// Missing expiry.
	claims.Issuer, claims.ExpiresAt = Issuer, nil
	noExp, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(tokens.secret)
	if _, _, err := tokens.Parse(noExp); err != ErrTokenInvalid {
		t.Fatalf("token without exp accepted: %v", err)
	}
	if _, _, err := tokens.Parse("garbage"); err != ErrTokenInvalid {
		t.Fatalf("garbage: %v", err)
	}
}
