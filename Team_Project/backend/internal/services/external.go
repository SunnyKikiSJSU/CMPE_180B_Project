package services

import (
	"context"
	"errors"
	"regexp"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/openlibrary"
	"book-metaverse/backend/internal/repository"
)

const (
	maxExternalPageSize = 40
	// Open Library answers deep pages slowly; nobody browses past page 100.
	maxExternalPage = 100
)

var isbnQueryPattern = regexp.MustCompile(`^([0-9]{9}[0-9Xx]|[0-9]{13})$`)

// ExternalSource is the outside catalog (Open Library).
type ExternalSource interface {
	Search(ctx context.Context, p openlibrary.SearchParams) (int64, []domain.ExternalBook, error)
	Work(ctx context.Context, id string) (domain.ExternalBook, error)
}

// ExternalStore is the repository surface for search marking and import.
type ExternalStore interface {
	InTx(ctx context.Context, fn func(*repository.Catalog) error) error
	Get(ctx context.Context, id uuid.UUID) (domain.Book, error)
	MatchExternal(ctx context.Context, workIDs, isbn13s []string) (repository.ExternalMatches, error)
	FindImported(ctx context.Context, workID string, isbn13s []string) (uuid.UUID, error)
}

// External is outside search (GET /api/external/books) and import
// (POST /api/books/import).
type External struct {
	source ExternalSource
	store  ExternalStore
}

func NewExternal(source ExternalSource, store ExternalStore) *External {
	return &External{source: source, store: store}
}

// ExternalSearchInput is GET /api/external/books after transport parsing.
type ExternalSearchInput struct {
	Q, Title, Author, Category, ISBN, Sort string
	Page                                   domain.Page
}

// Search queries Open Library and marks results already in the library.
// Nothing is written.
func (s *External) Search(ctx context.Context, in ExternalSearchInput) ([]domain.ExternalBook, int64, error) {
	p := openlibrary.SearchParams{Page: in.Page.Number, Limit: in.Page.Size}
	for _, f := range []struct {
		name, label, value string
		max                int
		dst                *string
	}{
		{"q", "Search words", in.Q, maxQueryRunes, &p.Q},
		{"title", "Title", in.Title, maxQueryRunes, &p.Title},
		{"author", "Author", in.Author, maxQueryRunes, &p.Author},
		{"category", "Category", in.Category, 100, &p.Subject},
	} {
		v, err := validateText(f.name, f.label, f.value, 0, f.max, false)
		if err != nil {
			return nil, 0, err
		}
		*f.dst = v
	}
	if in.ISBN != "" {
		if !isbnQueryPattern.MatchString(in.ISBN) {
			return nil, 0, domain.Validation("isbn", "isbn must be an ISBN-10 or ISBN-13, digits only.")
		}
		p.ISBN = in.ISBN
	}
	if p.Q == "" && p.Title == "" && p.Author == "" && p.Subject == "" && p.ISBN == "" {
		return nil, 0, domain.Validation("q", "Enter search words, a title, an author, a category, or an ISBN.")
	}
	switch in.Sort {
	case "", "relevance":
	case "newest":
		p.Newest = true
	default:
		return nil, 0, domain.Validation("sort", "sort must be relevance or newest.")
	}
	if in.Page.Size > maxExternalPageSize {
		return nil, 0, domain.Validation("pageSize", "pageSize must be a whole number from 1 to 40.")
	}
	if in.Page.Number > maxExternalPage {
		return nil, 0, domain.Validation("page", "page must be a whole number from 1 to 100.")
	}

	total, books, err := s.source.Search(ctx, p)
	if err != nil {
		return nil, 0, upstream(err)
	}
	ids, isbns := []string{}, []string{}
	for _, b := range books {
		ids = append(ids, b.ExternalID)
		isbns = append(isbns, b.ISBN13s...)
	}
	if len(books) > 0 {
		matches, err := s.store.MatchExternal(ctx, ids, isbns)
		if err != nil {
			return nil, 0, err
		}
		for i := range books {
			books[i].BookID = match(matches, books[i])
		}
	}
	return books, total, nil
}

// match applies the import rule: the work ID wins, then any ISBN-13.
func match(m repository.ExternalMatches, b domain.ExternalBook) *uuid.UUID {
	if id, ok := m.ByWorkID[b.ExternalID]; ok {
		return &id
	}
	for _, isbn := range b.ISBN13s {
		if id, ok := m.ByISBN13[isbn]; ok {
			return &id
		}
	}
	return nil
}

// Import stores an Open Library work as a local book. created is false when
// the work, or a book with one of its ISBN-13s, is already in the library;
// that book is returned and nothing is written. Only the work ID comes from
// the client; all metadata is read from Open Library.
func (s *External) Import(ctx context.Context, source, id string) (domain.Book, bool, error) {
	if source != "" && source != openlibrary.Source {
		return domain.Book{}, false, domain.Validation("source", "source must be openlibrary.")
	}
	if !openlibrary.WorkIDPattern.MatchString(id) {
		return domain.Book{}, false, domain.Validation("externalId", "externalId must be an Open Library work ID such as OL893414W.")
	}
	if book, ok, err := s.existing(ctx, id, nil); ok || err != nil {
		return book, false, err
	}
	work, err := s.source.Work(ctx, id)
	if err != nil {
		return domain.Book{}, false, upstream(err)
	}
	// The work may have moved to a new ID, or match a book by ISBN.
	if book, ok, err := s.existing(ctx, work.ExternalID, work.ISBN13s); ok || err != nil {
		return book, false, err
	}

	var bookID uuid.UUID
	var created bool
	err = s.store.InTx(ctx, func(tx *repository.Catalog) error {
		var err error
		if bookID, created, err = tx.ImportOpenLibrary(ctx, work); err != nil || !created {
			return err
		}
		if err := tx.SetAuthors(ctx, bookID, work.Authors); err != nil {
			return err
		}
		return tx.SetCategories(ctx, bookID, work.Categories)
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "books_isbn13_unique" {
		// Another book with this ISBN was added since the check above.
		if book, ok, err := s.existing(ctx, work.ExternalID, work.ISBN13s); ok || err != nil {
			return book, false, err
		}
	}
	if err != nil {
		return domain.Book{}, false, err
	}
	book, err := s.store.Get(ctx, bookID)
	return book, created, err
}

func (s *External) existing(ctx context.Context, id string, isbns []string) (domain.Book, bool, error) {
	bookID, err := s.store.FindImported(ctx, id, isbns)
	if errors.Is(err, repository.ErrNoRows) {
		return domain.Book{}, false, nil
	}
	if err != nil {
		return domain.Book{}, false, err
	}
	book, err := s.store.Get(ctx, bookID)
	return book, err == nil, err
}

// upstream turns Open Library failures into API errors: a missing work is 404,
// everything else (timeout, rate limit, 5xx, bad JSON) is 502.
func upstream(err error) error {
	if errors.Is(err, openlibrary.ErrNotFound) {
		return domain.NotFound("Book not found in Open Library.")
	}
	return &domain.Error{Kind: domain.KindUpstream, Message: "Open Library is not responding right now. Try again in a moment.", Err: err}
}
