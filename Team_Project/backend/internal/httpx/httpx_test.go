package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"book-metaverse/backend/internal/domain"
)

func TestDecode(t *testing.T) {
	type body struct {
		Name string `json:"name"`
	}
	cases := map[string]string{
		"empty":         "",
		"syntax":        `{"name":`,
		"unknown field": `{"name":"a","admin":true}`,
		"wrong type":    `{"name":5}`,
		"two objects":   `{"name":"a"}{"name":"b"}`,
		"too large":     `{"name":"` + strings.Repeat("a", MaxBodyBytes) + `"}`,
	}
	for name, input := range cases {
		var dst body
		err := Decode(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(input)), &dst)
		if Classify(err).Kind != domain.KindValidation {
			t.Errorf("%s: got %v", name, err)
		}
	}
	var ok body
	if err := Decode(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"a"}`)), &ok); err != nil || ok.Name != "a" {
		t.Fatalf("valid body: %v", err)
	}
}

func TestClassifyPostgres(t *testing.T) {
	cases := map[string]domain.Kind{
		"23505": domain.KindConflict, "23503": domain.KindValidation, "23514": domain.KindValidation,
		"22023": domain.KindValidation, "P0002": domain.KindNotFound, "BM001": domain.KindStaleVersion,
		"40001": domain.KindConflict, "40P01": domain.KindConflict, "55000": domain.KindConflict,
		"08006": domain.KindDatabase, "57P01": domain.KindDatabase, "42601": domain.KindInternal,
	}
	for code, kind := range cases {
		if got := Classify(&pgconn.PgError{Code: code}).Kind; got != kind {
			t.Errorf("%s: got %s want %s", code, got, kind)
		}
	}
}

func TestErrorEnvelopeHidesInternals(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(WithRequestID(req.Context(), "req-12345678"))
	rec := httptest.NewRecorder()
	Error(rec, req, errors.New("pq: relation users does not exist at 10.0.0.5"))
	if rec.Code != 500 {
		t.Fatalf("status %d", rec.Code)
	}
	var env struct {
		Error struct{ Code, Message, RequestID string }
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env.Error.Code != "INTERNAL" || env.Error.RequestID != "req-12345678" || strings.Contains(rec.Body.String(), "10.0.0.5") {
		t.Fatalf("envelope: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	Error(rec, req, &pgconn.PgError{Code: "23505", ConstraintName: "reviews_user_book_unique", Message: "duplicate key value violates unique constraint"})
	if rec.Code != 409 || !strings.Contains(rec.Body.String(), `"constraint":"reviews_user_book_unique"`) || strings.Contains(rec.Body.String(), "duplicate key") {
		t.Fatalf("conflict: %d %s", rec.Code, rec.Body.String())
	}
}
