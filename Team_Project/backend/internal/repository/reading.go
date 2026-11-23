package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"book-metaverse/backend/internal/db"
	"book-metaverse/backend/internal/domain"
)

// Reading covers shelves, reading progress, and history. Writes that change
// status go through the PL/pgSQL functions fn_move_book_to_shelf and
// fn_update_reading_progress, which lock, validate, and write history.
type Reading struct {
	pool *pgxpool.Pool
	conn db.DBTX
	q    *db.Queries
}

func NewReading(pool *pgxpool.Pool) *Reading {
	return &Reading{pool: pool, conn: pool, q: db.New(pool)}
}

// InTx runs fn in one READ COMMITTED transaction (no retry; see services).
func (r *Reading) InTx(ctx context.Context, fn func(*Reading) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(&Reading{pool: r.pool, conn: tx, q: r.q.WithTx(tx)}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Reading) ListShelves(ctx context.Context, userID uuid.UUID) ([]domain.Shelf, error) {
	rows, err := r.q.ListShelves(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Shelf, 0, len(rows))
	for _, s := range rows {
		out = append(out, domain.Shelf{ID: s.ShelfID, Name: s.Name, Type: s.ShelfType, Count: s.BookCount, CreatedAt: timestamp(s.CreatedAt)})
	}
	return out, nil
}

// GetShelf returns the caller's shelf; another user's shelf is ErrNoRows.
func (r *Reading) GetShelf(ctx context.Context, userID, shelfID uuid.UUID) (domain.Shelf, error) {
	s, err := r.q.GetShelf(ctx, db.GetShelfParams{ShelfID: shelfID, UserID: userID})
	if err != nil {
		return domain.Shelf{}, noRows(err)
	}
	return domain.Shelf{ID: s.ShelfID, Name: s.Name, Type: s.ShelfType, Count: s.BookCount, CreatedAt: timestamp(s.CreatedAt)}, nil
}

func (r *Reading) CountCustomShelves(ctx context.Context, userID uuid.UUID) (int32, error) {
	return r.q.CountCustomShelves(ctx, userID)
}

func (r *Reading) CreateShelf(ctx context.Context, userID uuid.UUID, name string) (domain.Shelf, error) {
	s, err := r.q.CreateShelf(ctx, db.CreateShelfParams{UserID: userID, Name: name})
	if err != nil {
		return domain.Shelf{}, err
	}
	return domain.Shelf{ID: s.ShelfID, Name: s.Name, Type: s.ShelfType, CreatedAt: timestamp(s.CreatedAt)}, nil
}

// RenameShelf renames a custom shelf; ErrNoRows if missing or a status shelf.
func (r *Reading) RenameShelf(ctx context.Context, userID, shelfID uuid.UUID, name string) error {
	_, err := r.q.RenameShelf(ctx, db.RenameShelfParams{ShelfID: shelfID, UserID: userID, Name: name})
	return noRows(err)
}

func (r *Reading) DeleteShelf(ctx context.Context, userID, shelfID uuid.UUID) (int64, error) {
	return r.q.DeleteShelf(ctx, db.DeleteShelfParams{ShelfID: shelfID, UserID: userID})
}

// ShelfBooks returns one page of the shelf's books with the caller's progress.
func (r *Reading) ShelfBooks(ctx context.Context, userID, shelfID uuid.UUID, sort string, page domain.Page) ([]domain.Book, int64, error) {
	rows, err := r.q.ListShelfBooks(ctx, db.ListShelfBooksParams{
		ShelfID: shelfID, UserID: userID, Sort: pgtype.Text{String: sort, Valid: true}, Limit: int32(page.Size), Offset: page.Offset(),
	})
	if err != nil {
		return nil, 0, err
	}
	books := make([]domain.Book, 0, len(rows))
	var total int64
	for _, row := range rows {
		total = int64(row.TotalCount)
		book, err := bookRow{row.BookID, row.GoogleBooksID, row.OpenlibraryID, row.Isbn13, row.Title, row.Description, row.PublishedDate, row.Publisher,
			row.PageCount, row.CoverUrl, row.Source, row.PublishedDatePrecision, row.PrimaryAuthor, row.Authors, row.Categories,
			row.AverageRating, row.RatingCount, row.ReviewCount}.book()
		if err != nil {
			return nil, 0, err
		}
		// A book can sit on a custom shelf without progress (LEFT JOIN).
		if row.ProgressStatus.Valid {
			percent, _ := row.PercentComplete.Float64Value()
			book.MyProgress = &domain.ReadingProgress{
				BookID: row.BookID, Status: row.ProgressStatus.String, CurrentPage: row.CurrentPage.Int32, TotalPages: row.TotalPages,
				PercentComplete: percent.Float64, Version: row.ProgressVersion.Int32, UpdatedAt: timestamp(row.ProgressUpdatedAt),
			}
		}
		books = append(books, book)
	}
	return books, total, nil
}

// IsOnShelf reports whether the book is on the caller's shelf.
func (r *Reading) IsOnShelf(ctx context.Context, userID, shelfID, bookID uuid.UUID) (bool, error) {
	return r.q.IsBookOnShelf(ctx, db.IsBookOnShelfParams{ShelfID: shelfID, BookID: bookID, UserID: userID})
}

// RemoveShelfItem removes one membership; 0 means it was not on the shelf.
func (r *Reading) RemoveShelfItem(ctx context.Context, userID, shelfID, bookID uuid.UUID) (int64, error) {
	return r.q.RemoveShelfItem(ctx, db.RemoveShelfItemParams{ShelfID: shelfID, BookID: bookID, UserID: userID})
}

// MoveToShelf calls fn_move_book_to_shelf: it locks the user's status shelves,
// keeps one status shelf per book, updates progress, and writes history.
// Unknown or foreign shelves and unknown books raise P0002.
func (r *Reading) MoveToShelf(ctx context.Context, userID, bookID, shelfID uuid.UUID) error {
	_, err := r.conn.Exec(ctx, "SELECT 1 FROM fn_move_book_to_shelf($1::uuid, $2::uuid, $3::uuid)", userID, bookID, shelfID)
	return err
}

// UpdateProgress calls fn_update_reading_progress. Nil arguments are NULL
// (keep the current value); a version mismatch raises BM001.
func (r *Reading) UpdateProgress(ctx context.Context, userID, bookID uuid.UUID, currentPage *int32, status *string, version *int32) error {
	_, err := r.conn.Exec(ctx, "SELECT 1 FROM fn_update_reading_progress($1::uuid, $2::uuid, $3::integer, $4::text, $5::integer)",
		userID, bookID, currentPage, status, version)
	return err
}

// SetPercent stores a client percentage; 0 rows means the book has a page count.
func (r *Reading) SetPercent(ctx context.Context, userID, bookID uuid.UUID, percent float64) (int64, error) {
	return r.q.SetProgressPercent(ctx, db.SetProgressPercentParams{UserID: userID, BookID: bookID, PercentComplete: percent})
}

func (r *Reading) GetProgress(ctx context.Context, userID, bookID uuid.UUID) (domain.ReadingProgress, error) {
	p, err := r.q.GetReadingProgress(ctx, db.GetReadingProgressParams{UserID: userID, BookID: bookID})
	if err != nil {
		return domain.ReadingProgress{}, noRows(err)
	}
	return domain.ReadingProgress{
		BookID: p.BookID, Status: p.Status, CurrentPage: p.CurrentPage, TotalPages: p.TotalPages, PercentComplete: p.PercentComplete,
		Version: p.Version, UpdatedAt: timestamp(p.UpdatedAt),
		Book: &domain.BookSummary{ID: p.BookID, Title: p.Title, Author: p.PrimaryAuthor, CoverURL: text(p.CoverUrl)},
	}, nil
}

func (r *Reading) ListProgress(ctx context.Context, userID uuid.UUID, status *string, page domain.Page) ([]domain.ReadingProgress, int64, error) {
	rows, err := r.q.ListReadingProgress(ctx, db.ListReadingProgressParams{UserID: userID, Status: pgText(status), Limit: int32(page.Size), Offset: page.Offset()})
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.ReadingProgress, 0, len(rows))
	var total int64
	for _, p := range rows {
		total = int64(p.TotalCount)
		out = append(out, domain.ReadingProgress{
			BookID: p.BookID, Status: p.Status, CurrentPage: p.CurrentPage, TotalPages: p.TotalPages, PercentComplete: p.PercentComplete,
			Version: p.Version, UpdatedAt: timestamp(p.UpdatedAt),
			Book: &domain.BookSummary{ID: p.BookID, Title: p.Title, Author: p.PrimaryAuthor, CoverURL: text(p.CoverUrl)},
		})
	}
	return out, total, nil
}

// DeleteProgress removes the progress row and the book's status-shelf
// memberships (custom shelves are kept). History is append-only and stays.
func (r *Reading) DeleteProgress(ctx context.Context, userID, bookID uuid.UUID) (int64, error) {
	return r.q.DeleteReadingProgress(ctx, db.DeleteReadingProgressParams{UserID: userID, BookID: bookID})
}

func (r *Reading) History(ctx context.Context, userID uuid.UUID, bookID *uuid.UUID, limit int) ([]domain.HistoryEvent, error) {
	rows, err := r.q.ListReadingHistory(ctx, db.ListReadingHistoryParams{UserID: userID, BookID: pgUUID(bookID), Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.HistoryEvent, 0, len(rows))
	for _, h := range rows {
		out = append(out, domain.HistoryEvent{
			ID: h.HistoryID, BookID: h.BookID, BookTitle: h.BookTitle, EventType: h.EventType,
			FromStatus: text(h.FromStatus), ToStatus: text(h.ToStatus), CreatedAt: timestamp(h.CreatedAt),
		})
	}
	return out, nil
}
