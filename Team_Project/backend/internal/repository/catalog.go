package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"book-metaverse/backend/internal/db"
	"book-metaverse/backend/internal/domain"
)

const dateLayout = "2006-01-02"

// Catalog maps the catalog queries (database/queries/books.sql, authors.sql,
// categories.sql) to domain types.
type Catalog struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewCatalog(pool *pgxpool.Pool) *Catalog {
	return &Catalog{pool: pool, q: db.New(pool)}
}

// InTx runs fn in one transaction, committing only if fn returns nil.
func (c *Catalog) InTx(ctx context.Context, fn func(*Catalog) error) error {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(&Catalog{pool: c.pool, q: c.q.WithTx(tx)}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// bookRow is the column set every book query returns.
type bookRow struct {
	BookID                 uuid.UUID
	GoogleBooksID          pgtype.Text
	Isbn13                 pgtype.Text
	Title                  string
	Description            pgtype.Text
	PublishedDate          pgtype.Date
	Publisher              pgtype.Text
	PageCount              pgtype.Int4
	CoverUrl               pgtype.Text
	Source                 string
	PublishedDatePrecision pgtype.Text
	PrimaryAuthor          string
	Authors                any
	Categories             any
	AverageRating          float64
	RatingCount            int32
	ReviewCount            int32
}

func (r bookRow) book() (domain.Book, error) {
	authors, err := refs(r.Authors)
	if err != nil {
		return domain.Book{}, fmt.Errorf("decode authors: %w", err)
	}
	categories, err := refs(r.Categories)
	if err != nil {
		return domain.Book{}, fmt.Errorf("decode categories: %w", err)
	}
	b := domain.Book{
		ID: r.BookID, Title: r.Title, Author: r.PrimaryAuthor, Authors: authors, Categories: categories,
		Source: r.Source, Description: text(r.Description), PublishedDatePrecision: text(r.PublishedDatePrecision),
		Publisher: text(r.Publisher), ISBN13: text(r.Isbn13), CoverURL: text(r.CoverUrl),
		AverageRating: r.AverageRating, RatingCount: r.RatingCount, ReviewCount: r.ReviewCount,
		GoogleBooksID: text(r.GoogleBooksID),
	}
	if r.PublishedDate.Valid {
		d := r.PublishedDate.Time.Format(dateLayout)
		b.PublishedDate = &d
	}
	if r.PageCount.Valid {
		n := r.PageCount.Int32
		b.PageCount = &n
	}
	// Seed rows hold synthetic volume IDs, so only real Google rows get a link.
	if r.Source == "google" && b.GoogleBooksID != nil {
		link := "https://books.google.com/books?id=" + *b.GoogleBooksID
		b.InfoLink = &link
	}
	return b, nil
}

// refs decodes a json_agg column ([{"id": ..., "name": ...}]) that pgx returns
// as []any.
func refs(value any) ([]domain.Ref, error) {
	out := []domain.Ref{}
	if value == nil {
		return out, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SearchParams mirrors GET /api/books after validation.
type SearchParams struct {
	Q, Title             *string
	AuthorID, CategoryID *uuid.UUID
	Sort                 string
	Page                 domain.Page
}

// Search returns one page and the total number of matches.
func (c *Catalog) Search(ctx context.Context, p SearchParams) ([]domain.Book, int64, error) {
	rows, err := c.q.SearchBooks(ctx, db.SearchBooksParams{
		Q: pgText(p.Q), Title: pgText(p.Title), AuthorID: pgUUID(p.AuthorID), CategoryID: pgUUID(p.CategoryID),
		Sort: pgtype.Text{String: p.Sort, Valid: true}, Limit: int32(p.Page.Size), Offset: p.Page.Offset(),
	})
	if err != nil {
		return nil, 0, err
	}
	books := make([]domain.Book, 0, len(rows))
	var total int64
	for _, r := range rows {
		total = int64(r.TotalCount)
		b, err := bookRow{r.BookID, r.GoogleBooksID, r.Isbn13, r.Title, r.Description, r.PublishedDate, r.Publisher, r.PageCount, r.CoverUrl,
			r.Source, r.PublishedDatePrecision, r.PrimaryAuthor, r.Authors, r.Categories, r.AverageRating, r.RatingCount, r.ReviewCount}.book()
		if err != nil {
			return nil, 0, err
		}
		books = append(books, b)
	}
	return books, total, nil
}

func (c *Catalog) Get(ctx context.Context, id uuid.UUID) (domain.Book, error) {
	r, err := c.q.GetBookByID(ctx, id)
	if err != nil {
		return domain.Book{}, noRows(err)
	}
	return bookRow(r).book()
}

// MyProgress returns the user's progress on the book, or nil.
func (c *Catalog) MyProgress(ctx context.Context, userID, bookID uuid.UUID) (*domain.ReadingProgress, error) {
	r, err := c.q.GetReadingProgress(ctx, db.GetReadingProgressParams{UserID: userID, BookID: bookID})
	if errors.Is(noRows(err), ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.ReadingProgress{
		BookID: r.BookID, Status: r.Status, CurrentPage: r.CurrentPage, TotalPages: r.TotalPages,
		PercentComplete: r.PercentComplete, Version: r.Version, UpdatedAt: timestamp(r.UpdatedAt),
	}, nil
}

// MyRating returns the user's 1-5 rating of the book, or nil.
func (c *Catalog) MyRating(ctx context.Context, userID, bookID uuid.UUID) (*int16, error) {
	r, err := c.q.GetMyRating(ctx, db.GetMyRatingParams{UserID: userID, BookID: bookID})
	if errors.Is(noRows(err), ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r.RatingValue, nil
}

// BookInput is a validated create or update. Nil fields are not set.
type BookInput struct {
	Title, Description, Publisher, ISBN13, CoverURL *string
	PublishedDate                                   *pgtype.Date
	PageCount                                       *int32
}

// Create inserts a source = 'user' book. Call inside InTx with SetRefs.
func (c *Catalog) Create(ctx context.Context, in BookInput) (uuid.UUID, error) {
	row, err := c.q.CreateBook(ctx, db.CreateBookParams{
		Title: *in.Title, Description: pgText(in.Description), Publisher: pgText(in.Publisher), Isbn13: pgText(in.ISBN13),
		CoverUrl: pgText(in.CoverURL), PublishedDate: pgDate(in.PublishedDate), PageCount: pgInt4(in.PageCount),
	})
	return row.BookID, err
}

// Update changes the given fields of a non-Google book. ErrNoRows means the
// book is missing or is a Google row (UpdateBook filters source <> 'google').
func (c *Catalog) Update(ctx context.Context, id uuid.UUID, in BookInput) error {
	_, err := c.q.UpdateBook(ctx, db.UpdateBookParams{
		BookID: id, Title: pgText(in.Title), Description: pgText(in.Description), Publisher: pgText(in.Publisher),
		Isbn13: pgText(in.ISBN13), CoverUrl: pgText(in.CoverURL), PublishedDate: pgDate(in.PublishedDate), PageCount: pgInt4(in.PageCount),
	})
	return noRows(err)
}

// SetAuthors upserts authors by normalized name and links them to the book.
func (c *Catalog) SetAuthors(ctx context.Context, bookID uuid.UUID, names []string) error {
	if err := c.q.ClearBookAuthors(ctx, bookID); err != nil {
		return err
	}
	for _, name := range names {
		author, err := c.q.UpsertAuthor(ctx, name)
		if err != nil {
			return err
		}
		if err := c.q.AddBookAuthor(ctx, db.AddBookAuthorParams{BookID: bookID, AuthorID: author.AuthorID}); err != nil {
			return err
		}
	}
	return nil
}

// SetCategories upserts categories by normalized name and links them.
func (c *Catalog) SetCategories(ctx context.Context, bookID uuid.UUID, names []string) error {
	if err := c.q.ClearBookCategories(ctx, bookID); err != nil {
		return err
	}
	for _, name := range names {
		category, err := c.q.UpsertCategory(ctx, name)
		if err != nil {
			return err
		}
		if err := c.q.AddBookCategory(ctx, db.AddBookCategoryParams{BookID: bookID, CategoryID: category.CategoryID}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Catalog) ListAuthors(ctx context.Context, q *string, page domain.Page) ([]domain.Author, int64, error) {
	rows, err := c.q.ListAuthors(ctx, db.ListAuthorsParams{Q: pgText(q), Limit: int32(page.Size), Offset: page.Offset()})
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Author, 0, len(rows))
	var total int64
	for _, r := range rows {
		total = int64(r.TotalCount)
		count := r.BookCount
		out = append(out, domain.Author{ID: r.AuthorID, Name: r.Name, BookCount: &count})
	}
	return out, total, nil
}

func (c *Catalog) GetAuthor(ctx context.Context, id uuid.UUID) (domain.Author, error) {
	a, err := c.q.GetAuthorByID(ctx, id)
	if err != nil {
		return domain.Author{}, noRows(err)
	}
	return domain.Author{ID: a.AuthorID, Name: a.Name}, nil
}

func (c *Catalog) BooksByAuthor(ctx context.Context, id uuid.UUID, page domain.Page) ([]domain.Book, int64, error) {
	rows, err := c.q.ListBooksByAuthor(ctx, db.ListBooksByAuthorParams{AuthorID: id, Limit: int32(page.Size), Offset: page.Offset()})
	if err != nil {
		return nil, 0, err
	}
	books := make([]domain.Book, 0, len(rows))
	var total int64
	for _, r := range rows {
		total = int64(r.TotalCount)
		b, err := bookRow{r.BookID, r.GoogleBooksID, r.Isbn13, r.Title, r.Description, r.PublishedDate, r.Publisher, r.PageCount, r.CoverUrl,
			r.Source, r.PublishedDatePrecision, r.PrimaryAuthor, r.Authors, r.Categories, r.AverageRating, r.RatingCount, r.ReviewCount}.book()
		if err != nil {
			return nil, 0, err
		}
		books = append(books, b)
	}
	return books, total, nil
}

func (c *Catalog) ListCategories(ctx context.Context, q *string, page domain.Page) ([]domain.Category, int64, error) {
	rows, err := c.q.ListCategories(ctx, db.ListCategoriesParams{Q: pgText(q), Limit: int32(page.Size), Offset: page.Offset()})
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Category, 0, len(rows))
	var total int64
	for _, r := range rows {
		total = int64(r.TotalCount)
		count := r.BookCount
		out = append(out, domain.Category{ID: r.CategoryID, Name: r.Name, BookCount: &count})
	}
	return out, total, nil
}

func (c *Catalog) GetCategory(ctx context.Context, id uuid.UUID) (domain.Category, error) {
	cat, err := c.q.GetCategoryByID(ctx, id)
	if err != nil {
		return domain.Category{}, noRows(err)
	}
	return domain.Category{ID: cat.CategoryID, Name: cat.Name}, nil
}

func (c *Catalog) BooksByCategory(ctx context.Context, id uuid.UUID, page domain.Page) ([]domain.Book, int64, error) {
	rows, err := c.q.ListBooksByCategory(ctx, db.ListBooksByCategoryParams{CategoryID: id, Limit: int32(page.Size), Offset: page.Offset()})
	if err != nil {
		return nil, 0, err
	}
	books := make([]domain.Book, 0, len(rows))
	var total int64
	for _, r := range rows {
		total = int64(r.TotalCount)
		b, err := bookRow{r.BookID, r.GoogleBooksID, r.Isbn13, r.Title, r.Description, r.PublishedDate, r.Publisher, r.PageCount, r.CoverUrl,
			r.Source, r.PublishedDatePrecision, r.PrimaryAuthor, r.Authors, r.Categories, r.AverageRating, r.RatingCount, r.ReviewCount}.book()
		if err != nil {
			return nil, 0, err
		}
		books = append(books, b)
	}
	return books, total, nil
}

func pgUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func pgDate(d *pgtype.Date) pgtype.Date {
	if d == nil {
		return pgtype.Date{}
	}
	return *d
}

func pgInt4(n *int32) pgtype.Int4 {
	if n == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *n, Valid: true}
}
