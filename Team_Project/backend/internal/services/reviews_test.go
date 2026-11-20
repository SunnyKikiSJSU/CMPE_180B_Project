package services

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"book-metaverse/backend/internal/domain"
)

func intp(v int) *int { return &v }

func validationField(t *testing.T, err error) string {
	t.Helper()
	var e *domain.Error
	if !errors.As(err, &e) || e.Kind != domain.KindValidation {
		t.Fatalf("want validation error, got %v", err)
	}
	return e.Details["field"].(string)
}

func TestRatingValue(t *testing.T) {
	if v, err := ratingValue("value", intp(5), true); err != nil || *v != 5 {
		t.Fatalf("5: %v %v", v, err)
	}
	if v, err := ratingValue("rating", nil, false); err != nil || v != nil {
		t.Fatalf("optional nil: %v %v", v, err)
	}
	if _, err := ratingValue("value", nil, true); validationField(t, err) != "value" {
		t.Fatal("required nil")
	}
	for _, bad := range []int{0, 6, -1} {
		if _, err := ratingValue("rating", intp(bad), false); validationField(t, err) != "rating" {
			t.Fatalf("%d accepted", bad)
		}
	}
}

func TestValidateReview(t *testing.T) {
	out, err := validateReview(ReviewInput{Title: str("  Good  "), Body: "  Fine book.\nRecommended.  "})
	if err != nil || *out.Title != "Good" || out.Body != "Fine book.\nRecommended." {
		t.Fatalf("normalized: %+v %v", out, err)
	}
	if out, err := validateReview(ReviewInput{Title: str("   "), Body: "ok"}); err != nil || out.Title != nil {
		t.Fatalf("blank title should become nil: %+v %v", out, err)
	}
	cases := map[string]struct {
		in    ReviewInput
		field string
	}{
		"empty body":   {ReviewInput{Body: ""}, "body"},
		"blank body":   {ReviewInput{Body: " \n "}, "body"},
		"long body":    {ReviewInput{Body: strings.Repeat("b", 5001)}, "body"},
		"long title":   {ReviewInput{Title: str(strings.Repeat("t", 201)), Body: "ok"}, "title"},
		"control char": {ReviewInput{Body: "a\x00b"}, "body"},
	}
	for name, c := range cases {
		if _, err := validateReview(c.in); err == nil {
			t.Errorf("%s: accepted", name)
		} else if got := validationField(t, err); got != c.field {
			t.Errorf("%s: field %q, want %q", name, got, c.field)
		}
	}
	if _, err := validateReview(ReviewInput{Body: strings.Repeat("é", 5000)}); err != nil {
		t.Fatalf("5000 runes should pass: %v", err)
	}
}

func TestReviewErrorMapping(t *testing.T) {
	missing := bookMissing(&pgconn.PgError{Code: "23503", ConstraintName: "ratings_book_fk"})
	if e, ok := missing.(*domain.Error); !ok || e.Kind != domain.KindNotFound {
		t.Fatalf("book fk: %v", missing)
	}
	userFK := &pgconn.PgError{Code: "23503", ConstraintName: "ratings_user_fk"}
	if got := bookMissing(userFK); got != error(userFK) {
		t.Fatalf("user fk must pass through: %v", got)
	}
	dup := reviewErrors(&pgconn.PgError{Code: "23505", ConstraintName: "reviews_user_book_unique"})
	if e, ok := dup.(*domain.Error); !ok || e.Kind != domain.KindConflict {
		t.Fatalf("duplicate review: %v", dup)
	}
	if bookMissing(nil) != nil || reviewErrors(nil) != nil {
		t.Fatal("nil must stay nil")
	}
}
