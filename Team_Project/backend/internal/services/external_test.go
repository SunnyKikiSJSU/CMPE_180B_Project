package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/openlibrary"
	"book-metaverse/backend/internal/repository"
)

type fakeSource struct {
	params openlibrary.SearchParams
	books  []domain.ExternalBook
	err    error
}

func (f *fakeSource) Search(_ context.Context, p openlibrary.SearchParams) (int64, []domain.ExternalBook, error) {
	f.params = p
	return 9, f.books, f.err
}

func (f *fakeSource) Work(context.Context, string) (domain.ExternalBook, error) {
	return domain.ExternalBook{}, f.err
}

type fakeExternalStore struct {
	ExternalStore
	matches repository.ExternalMatches
}

func (f *fakeExternalStore) MatchExternal(context.Context, []string, []string) (repository.ExternalMatches, error) {
	return f.matches, nil
}

func TestExternalSearch(t *testing.T) {
	seed, byISBN := uuid.New(), uuid.New()
	src := &fakeSource{books: []domain.ExternalBook{
		{ExternalID: "OL1W", ISBN13s: []string{"9780000000002"}},
		{ExternalID: "OL2W", ISBN13s: []string{"9780441013593", "9780000000001"}},
		{ExternalID: "OL3W"},
	}}
	store := &fakeExternalStore{matches: repository.ExternalMatches{
		ByWorkID: map[string]uuid.UUID{"OL1W": seed},
		ByISBN13: map[string]uuid.UUID{"9780000000001": byISBN, "9780000000002": uuid.New()},
	}}
	s := NewExternal(src, store)
	books, total, err := s.Search(context.Background(), ExternalSearchInput{Title: "  dune ", Category: "Fantasy", Sort: "newest", Page: domain.Page{Number: 3, Size: 40}})
	if err != nil || total != 9 {
		t.Fatal(err)
	}
	if src.params != (openlibrary.SearchParams{Title: "dune", Subject: "Fantasy", Newest: true, Page: 3, Limit: 40}) {
		t.Fatalf("params: %+v", src.params)
	}
	// The work ID wins over an ISBN match; then any ISBN; otherwise nil.
	if *books[0].BookID != seed || *books[1].BookID != byISBN || books[2].BookID != nil {
		t.Fatalf("marking: %v %v %v", books[0].BookID, books[1].BookID, books[2].BookID)
	}

	for name, in := range map[string]ExternalSearchInput{
		"nothing to search": {Q: "  ", Page: domain.Page{Number: 1, Size: 20}},
		"bad isbn":          {ISBN: "12345", Page: domain.Page{Number: 1, Size: 20}},
		"bad sort":          {Q: "x", Sort: "rating", Page: domain.Page{Number: 1, Size: 20}},
		"page size":         {Q: "x", Page: domain.Page{Number: 1, Size: 41}},
		"deep page":         {Q: "x", Page: domain.Page{Number: 101, Size: 20}},
	} {
		var e *domain.Error
		if _, _, err := s.Search(context.Background(), in); !errors.As(err, &e) || e.Kind != domain.KindValidation {
			t.Errorf("%s: %v", name, err)
		}
	}

	src.err = &openlibrary.UpstreamError{Op: "search", Status: 503}
	var e *domain.Error
	if _, _, err := s.Search(context.Background(), ExternalSearchInput{Q: "x", Page: domain.Page{Number: 1, Size: 1}}); !errors.As(err, &e) || e.Kind != domain.KindUpstream {
		t.Fatalf("upstream: %v", err)
	}
}

func TestImportValidatesBeforeCallingOut(t *testing.T) {
	s := NewExternal(&fakeSource{err: errors.New("must not be called")}, &fakeExternalStore{})
	for _, c := range []struct{ source, id, field string }{
		{"google", "OL1W", "source"}, {"", "OL1M", "externalId"}, {"openlibrary", "", "externalId"}, {"", "/works/OL1W", "externalId"},
	} {
		if _, _, err := s.Import(context.Background(), c.source, c.id); validationField(t, err) != c.field {
			t.Errorf("%+v: %v", c, err)
		}
	}
}
