package services

import (
	"strings"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	for _, ok := range []string{"reader@example.com", "a.b+c@sub.example.org"} {
		if err := validateEmail(NormalizeEmail(ok)); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "no-at-sign", "a@b", "Name <a@b.com>", "a@b.com x", strings.Repeat("a", 250) + "@b.com"} {
		if validateEmail(NormalizeEmail(bad)) == nil {
			t.Errorf("%q accepted", bad)
		}
	}
	if NormalizeEmail("  Reader@Example.COM ") != "reader@example.com" {
		t.Fatal("normalization must match users_email_normalized")
	}
}

func TestValidateText(t *testing.T) {
	if v, err := validateText("displayName", "Display name", "  Ann  ", 1, 80, false); err != nil || v != "Ann" {
		t.Fatalf("trim: %q %v", v, err)
	}
	if _, err := validateText("displayName", "Display name", "   ", 1, 80, false); err == nil {
		t.Fatal("blank accepted")
	}
	if _, err := validateText("displayName", "Display name", strings.Repeat("é", 81), 1, 80, false); err == nil {
		t.Fatal("81 characters accepted")
	}
	if _, err := validateText("displayName", "Display name", "a\nb", 1, 80, false); err == nil {
		t.Fatal("newline in a single-line field accepted")
	}
	if _, err := validateText("bio", "Bio", "line one\nline two", 0, 2000, true); err != nil {
		t.Fatal("newline in bio rejected")
	}
	if _, err := validateText("bio", "Bio", "bell\a", 0, 2000, true); err == nil {
		t.Fatal("control character accepted")
	}
}

func TestValidateAvatarURL(t *testing.T) {
	if _, err := validateAvatarURL("https://example.com/me.png"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"", "javascript:alert(1)", "ftp://x/y", "https://user:pw@example.com/a", "/relative.png", "https://" + strings.Repeat("a", 2050)} {
		if _, err := validateAvatarURL(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
