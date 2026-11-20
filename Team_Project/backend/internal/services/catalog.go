package services

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/repository"
)

const (
	maxQueryRunes       = 200
	maxTitleRunes       = 500
	maxNameRunes        = 200
	maxNamesPerBook     = 20
	maxDescriptionRunes = 10000
	maxPublisherRunes   = 200
	maxPageCount        = 100000
)

var (
	isbn13Pattern = regexp.MustCompile(`^[0-9]{13}$`)
	searchSorts   = map[string]bool{"relevance": true, "newest": true, "rating": true, "title": true, "popular": true}
)

// CatalogStore is the repository surface the catalog service needs.
type CatalogStore interface {
	InTx(ctx context.Context, fn func(*repository.Catalog) error) error
	Search(ctx context.Context, p repository.SearchParams) ([]domain.Book, int64, error)
	Get(ctx context.Context, id uuid.UUID) (domain.Book, error)
	MyProgress(ctx context.Context, userID, bookID uuid.UUID) (*domain.ReadingProgress, error)
	MyRating(ctx context.Context, userID, bookID uuid.UUID) (*int16, error)
	ListAuthors(ctx context.Context, q *string, page domain.Page) ([]domain.Author, int64, error)
	GetAuthor(ctx context.Context, id uuid.UUID) (domain.Author, error)
	BooksByAuthor(ctx context.Context, id uuid.UUID, page domain.Page) ([]domain.Book, int64, error)
	ListCategories(ctx context.Context, q *string, page domain.Page) ([]domain.Category, int64, error)
	GetCategory(ctx context.Context, id uuid.UUID) (domain.Category, error)
	BooksByCategory(ctx context.Context, id uuid.UUID, page domain.Page) ([]domain.Book, int64, error)
}

type Catalog struct{ store CatalogStore }

func NewCatalog(store CatalogStore) *Catalog { return &Catalog{store: store} }

// SearchInput is GET /api/books after transport parsing.
type SearchInput struct {
	Q, Title             string
	AuthorID, CategoryID *uuid.UUID
	Sort                 string
	Page                 domain.Page
}

// Search is library search over books already in PostgreSQL (q01).
func (s *Catalog) Search(ctx context.Context, in SearchInput) ([]domain.Book, int64, error) {
	q, err := optionalQuery("q", in.Q)
	if err != nil {
		return nil, 0, err
	}
	title, err := optionalQuery("title", in.Title)
	if err != nil {
		return nil, 0, err
	}
	sort := in.Sort
	if sort == "" {
		// Relevance needs a query to rank by; without one, newest first.
		sort = "relevance"
		if q == nil {
			sort = "newest"
		}
	}
	if !searchSorts[sort] {
		return nil, 0, domain.Validation("sort", "sort must be relevance, newest, rating, title, or popular.")
	}
	params := repository.SearchParams{Q: q, Title: title, AuthorID: in.AuthorID, CategoryID: in.CategoryID, Sort: sort, Page: in.Page}
	books, total, err := s.store.Search(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	// The total comes from a window count, so a page past the end has no row
	// to carry it. Ask for the first row to report the real total.
	if len(books) == 0 && in.Page.Number > 1 {
		params.Page = domain.Page{Number: 1, Size: 1}
		_, total, err = s.store.Search(ctx, params)
	}
	return books, total, err
}

// Get returns book detail; with a signed-in caller it adds myProgress and myRating.
func (s *Catalog) Get(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (domain.Book, error) {
	book, err := s.store.Get(ctx, id)
	if errors.Is(err, repository.ErrNoRows) {
		return domain.Book{}, domain.NotFound("Book not found.")
	}
	if err != nil || userID == nil {
		return book, err
	}
	if book.MyProgress, err = s.store.MyProgress(ctx, *userID, id); err != nil {
		return domain.Book{}, err
	}
	if book.MyRating, err = s.store.MyRating(ctx, *userID, id); err != nil {
		return domain.Book{}, err
	}
	return book, nil
}

// BookFields is a create or update request. For update, nil means unchanged
// (the UpdateBook query uses coalesce, so a field cannot be cleared).
type BookFields struct {
	Title, Description, PublishedDate, Publisher, ISBN13, CoverURL *string
	PageCount                                                      *int32
	Authors, Categories                                            []string
}

// Create adds a book by hand (source = 'user') with its authors and
// categories in one transaction.
func (s *Catalog) Create(ctx context.Context, in BookFields) (domain.Book, error) {
	if in.Title == nil {
		return domain.Book{}, domain.Validation("title", "Title is required.")
	}
	fields, err := validateBook(in)
	if err != nil {
		return domain.Book{}, err
	}
	authors, err := validateNames("authors", "Author", in.Authors)
	if err != nil {
		return domain.Book{}, err
	}
	categories, err := validateNames("categories", "Category", in.Categories)
	if err != nil {
		return domain.Book{}, err
	}

	var id uuid.UUID
	err = s.store.InTx(ctx, func(tx *repository.Catalog) error {
		if id, err = tx.Create(ctx, fields); err != nil {
			return err
		}
		if err := tx.SetAuthors(ctx, id, authors); err != nil {
			return err
		}
		return tx.SetCategories(ctx, id, categories)
	})
	if err := isbnConflict(err); err != nil {
		return domain.Book{}, err
	}
	return s.store.Get(ctx, id)
}

// Update corrects seed or user-created metadata. Google rows are read-only.
func (s *Catalog) Update(ctx context.Context, id uuid.UUID, in BookFields) (domain.Book, error) {
	fields, err := validateBook(in)
	if err != nil {
		return domain.Book{}, err
	}
	current, err := s.store.Get(ctx, id)
	if errors.Is(err, repository.ErrNoRows) {
		return domain.Book{}, domain.NotFound("Book not found.")
	}
	if err != nil {
		return domain.Book{}, err
	}
	if current.Source == "google" {
		return domain.Book{}, &domain.Error{Kind: domain.KindForbidden, Message: "Books imported from Google Books cannot be edited."}
	}
	err = s.store.InTx(ctx, func(tx *repository.Catalog) error { return tx.Update(ctx, id, fields) })
	if errors.Is(err, repository.ErrNoRows) {
		// Deleted or turned into a Google row since the read above.
		return domain.Book{}, domain.NotFound("Book not found.")
	}
	if err := isbnConflict(err); err != nil {
		return domain.Book{}, err
	}
	return s.store.Get(ctx, id)
}

func (s *Catalog) ListAuthors(ctx context.Context, q string, page domain.Page) ([]domain.Author, int64, error) {
	query, err := optionalQuery("q", q)
	if err != nil {
		return nil, 0, err
	}
	authors, total, err := s.store.ListAuthors(ctx, query, page)
	if err == nil && len(authors) == 0 && page.Number > 1 {
		_, total, err = s.store.ListAuthors(ctx, query, domain.Page{Number: 1, Size: 1})
	}
	return authors, total, err
}

// Author returns the author with one page of their books (total in X-Total-Count).
func (s *Catalog) Author(ctx context.Context, id uuid.UUID, page domain.Page) (domain.Author, int64, error) {
	author, err := s.store.GetAuthor(ctx, id)
	if errors.Is(err, repository.ErrNoRows) {
		return domain.Author{}, 0, domain.NotFound("Author not found.")
	}
	if err != nil {
		return domain.Author{}, 0, err
	}
	books, total, err := s.store.BooksByAuthor(ctx, id, page)
	if err == nil && len(books) == 0 && page.Number > 1 {
		_, total, err = s.store.BooksByAuthor(ctx, id, domain.Page{Number: 1, Size: 1})
	}
	author.Books, author.BookCount = books, ptr(int32(total))
	return author, total, err
}

func (s *Catalog) ListCategories(ctx context.Context, q string, page domain.Page) ([]domain.Category, int64, error) {
	query, err := optionalQuery("q", q)
	if err != nil {
		return nil, 0, err
	}
	categories, total, err := s.store.ListCategories(ctx, query, page)
	if err == nil && len(categories) == 0 && page.Number > 1 {
		_, total, err = s.store.ListCategories(ctx, query, domain.Page{Number: 1, Size: 1})
	}
	return categories, total, err
}

func (s *Catalog) Category(ctx context.Context, id uuid.UUID, page domain.Page) (domain.Category, int64, error) {
	category, err := s.store.GetCategory(ctx, id)
	if errors.Is(err, repository.ErrNoRows) {
		return domain.Category{}, 0, domain.NotFound("Category not found.")
	}
	if err != nil {
		return domain.Category{}, 0, err
	}
	books, total, err := s.store.BooksByCategory(ctx, id, page)
	if err == nil && len(books) == 0 && page.Number > 1 {
		_, total, err = s.store.BooksByCategory(ctx, id, domain.Page{Number: 1, Size: 1})
	}
	category.Books, category.BookCount = books, ptr(int32(total))
	return category, total, err
}

func optionalQuery(field, value string) (*string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(value) > maxQueryRunes {
		return nil, domain.Validation(field, field+" must be at most 200 characters.")
	}
	return &value, nil
}

func validateBook(in BookFields) (repository.BookInput, error) {
	var out repository.BookInput
	var err error
	if in.Title != nil {
		title, verr := validateText("title", "Title", *in.Title, 1, maxTitleRunes, false)
		if verr != nil {
			return out, verr
		}
		out.Title = &title
	}
	if in.Description != nil {
		value, verr := validateText("description", "Description", *in.Description, 0, maxDescriptionRunes, true)
		if verr != nil {
			return out, verr
		}
		out.Description = &value
	}
	if in.Publisher != nil {
		value, verr := validateText("publisher", "Publisher", *in.Publisher, 1, maxPublisherRunes, false)
		if verr != nil {
			return out, verr
		}
		out.Publisher = &value
	}
	if in.ISBN13 != nil {
		value := strings.TrimSpace(*in.ISBN13)
		if !isbn13Pattern.MatchString(value) {
			return out, domain.Validation("isbn13", "ISBN-13 must be exactly 13 digits.")
		}
		out.ISBN13 = &value
	}
	if in.CoverURL != nil {
		value, verr := validateURL("coverUrl", "Cover URL", *in.CoverURL)
		if verr != nil {
			return out, verr
		}
		out.CoverURL = &value
	}
	if in.PageCount != nil {
		if *in.PageCount < 1 || *in.PageCount > maxPageCount {
			return out, domain.Validation("pageCount", "Page count must be between 1 and 100000.")
		}
		out.PageCount = in.PageCount
	}
	if in.PublishedDate != nil {
		date, perr := time.Parse("2006-01-02", strings.TrimSpace(*in.PublishedDate))
		if perr != nil || date.Year() < 1 || date.After(time.Now().AddDate(1, 0, 0)) {
			return out, domain.Validation("publishedDate", "Published date must be a date in YYYY-MM-DD format, at most one year ahead.")
		}
		out.PublishedDate = &pgtype.Date{Time: date, Valid: true}
	}
	return out, err
}

// validateNames trims, drops case-insensitive duplicates, and bounds author
// or category names (the database dedupes by lower(btrim(name))).
func validateNames(field, label string, names []string) ([]string, error) {
	if len(names) > maxNamesPerBook {
		return nil, domain.Validation(field, "At most 20 "+field+" per book.")
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(names))
	for _, name := range names {
		value, err := validateText(field, label+" name", name, 1, maxNameRunes, false)
		if err != nil {
			return nil, err
		}
		key := strings.ToLower(value)
		if !seen[key] {
			seen[key] = true
			out = append(out, value)
		}
	}
	return out, nil
}

// isbnConflict turns a duplicate ISBN-13 into a 409 the client can act on.
func isbnConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "books_isbn13_unique" {
		return domain.Conflict("A book with this ISBN-13 is already in the library.", map[string]any{"field": "isbn13", "constraint": pgErr.ConstraintName})
	}
	return err
}

func ptr[T any](v T) *T { return &v }
