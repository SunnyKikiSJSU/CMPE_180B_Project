package services

import (
	"strings"
	"testing"
	"time"

	"book-metaverse/backend/internal/domain"
)

func str(s string) *string { return &s }

func TestValidateBook(t *testing.T) {
	n := int32(320)
	out, err := validateBook(BookFields{
		Title: str("  The Long Road  "), Description: str("Line one\nLine two"), PublishedDate: str("2024-02-29"),
		Publisher: str("Example Press"), ISBN13: str("9780000000001"), CoverURL: str("https://example.com/c.jpg"), PageCount: &n,
	})
	if err != nil {
		t.Fatal(err)
	}
	if *out.Title != "The Long Road" || out.PublishedDate.Time.Format("2006-01-02") != "2024-02-29" || *out.PageCount != 320 {
		t.Fatalf("normalized: %+v", out)
	}

	zero, tooMany := int32(0), int32(100001)
	future := time.Now().AddDate(2, 0, 0).Format("2006-01-02")
	cases := map[string]BookFields{
		"blank title":        {Title: str("   ")},
		"long title":         {Title: str(strings.Repeat("t", 501))},
		"isbn letters":       {ISBN13: str("97800000000X1")},
		"isbn 10 digits":     {ISBN13: str("0123456789")},
		"page count 0":       {PageCount: &zero},
		"page count huge":    {PageCount: &tooMany},
		"bad date":           {PublishedDate: str("2024-02-30")},
		"date format":        {PublishedDate: str("29/02/2024")},
		"far future":         {PublishedDate: str(future)},
		"javascript cover":   {CoverURL: str("javascript:alert(1)")},
		"blank publisher":    {Publisher: str(" ")},
		"control char title": {Title: str("a\x07b")},
	}
	for name, in := range cases {
		if _, err := validateBook(in); err == nil {
			t.Errorf("%s: accepted", name)
		} else if e, ok := err.(*domain.Error); !ok || e.Kind != domain.KindValidation {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestValidateNames(t *testing.T) {
	names, err := validateNames("authors", "Author", []string{" Ann Lee ", "ann lee", "Bo Chen", "ANN LEE"})
	if err != nil || len(names) != 2 || names[0] != "Ann Lee" || names[1] != "Bo Chen" {
		t.Fatalf("dedupe: %v %v", names, err)
	}
	if names, err := validateNames("authors", "Author", nil); err != nil || len(names) != 0 {
		t.Fatal("no authors is allowed")
	}
	if _, err := validateNames("authors", "Author", []string{"ok", " "}); err == nil {
		t.Fatal("blank name accepted")
	}
	many := make([]string, 21)
	for i := range many {
		many[i] = strings.Repeat("a", i+1)
	}
	if _, err := validateNames("categories", "Category", many); err == nil {
		t.Fatal("21 names accepted")
	}
}

func TestOptionalQuery(t *testing.T) {
	if q, err := optionalQuery("q", "   "); err != nil || q != nil {
		t.Fatal("blank query must be nil")
	}
	if q, _ := optionalQuery("q", "  night circus "); *q != "night circus" {
		t.Fatalf("trim: %q", *q)
	}
	if _, err := optionalQuery("q", strings.Repeat("é", 201)); err == nil {
		t.Fatal("201 characters accepted")
	}
}
