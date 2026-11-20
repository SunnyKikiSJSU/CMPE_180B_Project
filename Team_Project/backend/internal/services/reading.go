package services

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/repository"
)

const (
	maxShelfNameRunes = 80
	maxCustomShelves  = 50
	maxCurrentPage    = 100000
	maxHistoryLimit   = 100
)

var readingStatuses = map[string]bool{"want_to_read": true, "reading": true, "completed": true, "paused": true}

// ReadingStore is the repository surface for shelves, progress, and history.
type ReadingStore interface {
	InTx(ctx context.Context, fn func(*repository.Reading) error) error
	ListShelves(ctx context.Context, userID uuid.UUID) ([]domain.Shelf, error)
	GetShelf(ctx context.Context, userID, shelfID uuid.UUID) (domain.Shelf, error)
	ShelfBooks(ctx context.Context, userID, shelfID uuid.UUID, sort string, page domain.Page) ([]domain.Book, int64, error)
	GetProgress(ctx context.Context, userID, bookID uuid.UUID) (domain.ReadingProgress, error)
	ListProgress(ctx context.Context, userID uuid.UUID, status *string, page domain.Page) ([]domain.ReadingProgress, int64, error)
	History(ctx context.Context, userID uuid.UUID, bookID *uuid.UUID, limit int) ([]domain.HistoryEvent, error)
}

type Reading struct{ store ReadingStore }

func NewReading(store ReadingStore) *Reading { return &Reading{store: store} }

// inTx runs fn in a transaction and retries once on a serialization failure
// or deadlock (40001, 40P01). A stale version (BM001) is never retried.
func (s *Reading) inTx(ctx context.Context, fn func(*repository.Reading) error) error {
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		err = s.store.InTx(ctx, fn)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || (pgErr.Code != "40001" && pgErr.Code != "40P01") {
			return err
		}
	}
	return err
}

// ---- Shelves ----

func (s *Reading) Shelves(ctx context.Context, userID uuid.UUID) ([]domain.Shelf, error) {
	return s.store.ListShelves(ctx, userID)
}

// CreateShelf adds a custom shelf; status shelves come from registration.
func (s *Reading) CreateShelf(ctx context.Context, userID uuid.UUID, name, shelfType string) (domain.Shelf, error) {
	if shelfType != "" && shelfType != "custom" {
		return domain.Shelf{}, domain.Validation("type", "Only custom shelves can be created; status shelves exist for every account.")
	}
	name, err := validateText("name", "Shelf name", name, 1, maxShelfNameRunes, false)
	if err != nil {
		return domain.Shelf{}, err
	}
	var shelf domain.Shelf
	err = s.inTx(ctx, func(tx *repository.Reading) error {
		count, err := tx.CountCustomShelves(ctx, userID)
		if err != nil {
			return err
		}
		if count >= maxCustomShelves {
			return domain.Conflict("You can have at most 50 custom shelves.", map[string]any{"limit": maxCustomShelves})
		}
		shelf, err = tx.CreateShelf(ctx, userID, name)
		return err
	})
	return shelf, shelfNameConflict(err)
}

func (s *Reading) RenameShelf(ctx context.Context, userID, shelfID uuid.UUID, name string) (domain.Shelf, error) {
	name, err := validateText("name", "Shelf name", name, 1, maxShelfNameRunes, false)
	if err != nil {
		return domain.Shelf{}, err
	}
	shelf, err := s.ownShelf(ctx, userID, shelfID)
	if err != nil {
		return domain.Shelf{}, err
	}
	if shelf.Type != "custom" {
		return domain.Shelf{}, forbidden("Status shelves cannot be renamed.")
	}
	err = s.inTx(ctx, func(tx *repository.Reading) error { return tx.RenameShelf(ctx, userID, shelfID, name) })
	if errors.Is(err, repository.ErrNoRows) {
		return domain.Shelf{}, domain.NotFound("Shelf not found.")
	}
	if err := shelfNameConflict(err); err != nil {
		return domain.Shelf{}, err
	}
	return s.ownShelf(ctx, userID, shelfID)
}

// DeleteShelf removes a custom shelf and its memberships (not the books or
// progress). Status shelves are permanent.
func (s *Reading) DeleteShelf(ctx context.Context, userID, shelfID uuid.UUID) error {
	shelf, err := s.ownShelf(ctx, userID, shelfID)
	if err != nil {
		return err
	}
	if shelf.Type != "custom" {
		return forbidden("Status shelves cannot be deleted.")
	}
	return s.inTx(ctx, func(tx *repository.Reading) error {
		n, err := tx.DeleteShelf(ctx, userID, shelfID)
		if err == nil && n == 0 {
			return domain.NotFound("Shelf not found.")
		}
		return err
	})
}

func (s *Reading) ShelfBooks(ctx context.Context, userID, shelfID uuid.UUID, sort string, page domain.Page) ([]domain.Book, int64, error) {
	if sort == "" {
		sort = "recent"
	}
	if sort != "recent" && sort != "title" {
		return nil, 0, domain.Validation("sort", "sort must be recent or title.")
	}
	if _, err := s.ownShelf(ctx, userID, shelfID); err != nil {
		return nil, 0, err
	}
	books, total, err := s.store.ShelfBooks(ctx, userID, shelfID, sort, page)
	if err == nil && len(books) == 0 && page.Number > 1 {
		_, total, err = s.store.ShelfBooks(ctx, userID, shelfID, sort, domain.Page{Number: 1, Size: 1})
	}
	return books, total, err
}

// AddBook puts a book on a shelf. On a status shelf it also sets the reading
// status (and leaves every other status shelf); custom shelves only tag.
func (s *Reading) AddBook(ctx context.Context, userID, shelfID, bookID uuid.UUID) (domain.ShelfBookResult, error) {
	var result domain.ShelfBookResult
	err := s.inTx(ctx, func(tx *repository.Reading) error {
		if err := tx.MoveToShelf(ctx, userID, bookID, shelfID); err != nil {
			return err
		}
		progress, err := tx.GetProgress(ctx, userID, bookID)
		result = domain.ShelfBookResult{ShelfID: shelfID, BookID: bookID, Progress: progress, HistoryEvent: "shelf_moved"}
		return err
	})
	return result, functionError(err)
}

// MoveBook moves a book from one of the caller's shelves to another. The book
// must be on the source shelf. Leaving a custom shelf removes that tag;
// status shelves are handled by fn_move_book_to_shelf (one status per book).
func (s *Reading) MoveBook(ctx context.Context, userID, fromShelfID, bookID, toShelfID uuid.UUID) (domain.ShelfBookResult, error) {
	from, err := s.ownShelf(ctx, userID, fromShelfID)
	if err != nil {
		return domain.ShelfBookResult{}, err
	}
	var result domain.ShelfBookResult
	err = s.inTx(ctx, func(tx *repository.Reading) error {
		if err := onShelf(ctx, tx, userID, from, bookID); err != nil {
			return err
		}
		if err := tx.MoveToShelf(ctx, userID, bookID, toShelfID); err != nil {
			return err
		}
		if from.Type == "custom" && fromShelfID != toShelfID {
			if _, err := tx.RemoveShelfItem(ctx, userID, fromShelfID, bookID); err != nil {
				return err
			}
		}
		progress, err := tx.GetProgress(ctx, userID, bookID)
		result = domain.ShelfBookResult{ShelfID: toShelfID, BookID: bookID, Progress: progress, HistoryEvent: "shelf_moved"}
		return err
	})
	return result, functionError(err)
}

// RemoveBook takes a book off a shelf. Removing it from a status shelf also
// removes its reading progress (as DELETE /api/reading-progress does), so no
// progress row ever claims a status without its shelf. History is kept.
func (s *Reading) RemoveBook(ctx context.Context, userID, shelfID, bookID uuid.UUID) error {
	shelf, err := s.ownShelf(ctx, userID, shelfID)
	if err != nil {
		return err
	}
	return s.inTx(ctx, func(tx *repository.Reading) error {
		if shelf.Type == "custom" {
			n, err := tx.RemoveShelfItem(ctx, userID, shelfID, bookID)
			if err == nil && n == 0 {
				return notOnShelf()
			}
			return err
		}
		if err := onShelf(ctx, tx, userID, shelf, bookID); err != nil {
			return err
		}
		_, err := tx.DeleteProgress(ctx, userID, bookID)
		return err
	})
}

// ---- Progress and history ----

func (s *Reading) Progress(ctx context.Context, userID, bookID uuid.UUID) (domain.ReadingProgress, error) {
	progress, err := s.store.GetProgress(ctx, userID, bookID)
	if errors.Is(err, repository.ErrNoRows) {
		return domain.ReadingProgress{}, domain.NotFound("No reading progress for this book.")
	}
	return progress, err
}

func (s *Reading) ListProgress(ctx context.Context, userID uuid.UUID, status string, page domain.Page) ([]domain.ReadingProgress, int64, error) {
	var filter *string
	if status != "" {
		if !readingStatuses[status] {
			return nil, 0, domain.Validation("status", "status must be want_to_read, reading, completed, or paused.")
		}
		filter = &status
	}
	items, total, err := s.store.ListProgress(ctx, userID, filter, page)
	if err == nil && len(items) == 0 && page.Number > 1 {
		_, total, err = s.store.ListProgress(ctx, userID, filter, domain.Page{Number: 1, Size: 1})
	}
	return items, total, err
}

// ProgressUpdate is PATCH /api/reading-progress/{bookId}.
type ProgressUpdate struct {
	CurrentPage     *int32
	PercentComplete *float64
	Status          *string
	Version         *int32
}

// UpdateProgress calls fn_update_reading_progress, which validates pages,
// completes the book on its last page, moves status shelves, and writes
// history. percentComplete is stored only for books without a page count.
func (s *Reading) UpdateProgress(ctx context.Context, userID, bookID uuid.UUID, in ProgressUpdate) (domain.ReadingProgress, error) {
	if in.CurrentPage == nil && in.PercentComplete == nil && in.Status == nil {
		return domain.ReadingProgress{}, domain.Validation("currentPage", "Send currentPage, percentComplete, or status.")
	}
	if in.CurrentPage != nil && (*in.CurrentPage < 0 || *in.CurrentPage > maxCurrentPage) {
		return domain.ReadingProgress{}, domain.Validation("currentPage", "currentPage must be between 0 and 100000.")
	}
	if in.PercentComplete != nil && (*in.PercentComplete < 0 || *in.PercentComplete > 100) {
		return domain.ReadingProgress{}, domain.Validation("percentComplete", "percentComplete must be between 0 and 100.")
	}
	if in.Status != nil && !readingStatuses[*in.Status] {
		return domain.ReadingProgress{}, domain.Validation("status", "status must be want_to_read, reading, completed, or paused.")
	}
	if in.Version != nil && *in.Version < 1 {
		return domain.ReadingProgress{}, domain.Validation("version", "version must be 1 or more.")
	}

	var progress domain.ReadingProgress
	err := s.inTx(ctx, func(tx *repository.Reading) error {
		if err := tx.UpdateProgress(ctx, userID, bookID, in.CurrentPage, in.Status, in.Version); err != nil {
			return err
		}
		if in.PercentComplete != nil {
			n, err := tx.SetPercent(ctx, userID, bookID, *in.PercentComplete)
			if err != nil {
				return err
			}
			if n == 0 {
				return domain.Validation("percentComplete", "This book has a page count; send currentPage and the percentage is calculated.")
			}
		}
		var err error
		progress, err = tx.GetProgress(ctx, userID, bookID)
		return err
	})
	return progress, functionError(err)
}

func (s *Reading) DeleteProgress(ctx context.Context, userID, bookID uuid.UUID) error {
	return s.inTx(ctx, func(tx *repository.Reading) error {
		n, err := tx.DeleteProgress(ctx, userID, bookID)
		if err == nil && n == 0 {
			return domain.NotFound("No reading progress for this book.")
		}
		return err
	})
}

func (s *Reading) History(ctx context.Context, userID uuid.UUID, bookID *uuid.UUID, limit int) ([]domain.HistoryEvent, error) {
	if limit < 1 || limit > maxHistoryLimit {
		return nil, domain.Validation("limit", "limit must be between 1 and 100.")
	}
	return s.store.History(ctx, userID, bookID, limit)
}

// ---- helpers ----

// ownShelf returns the caller's shelf; a missing or foreign shelf is a 404, so
// other users' shelf IDs are not revealed.
func (s *Reading) ownShelf(ctx context.Context, userID, shelfID uuid.UUID) (domain.Shelf, error) {
	shelf, err := s.store.GetShelf(ctx, userID, shelfID)
	if errors.Is(err, repository.ErrNoRows) {
		return domain.Shelf{}, domain.NotFound("Shelf not found.")
	}
	return shelf, err
}

// onShelf checks that the book is on the caller's shelf.
func onShelf(ctx context.Context, tx *repository.Reading, userID uuid.UUID, shelf domain.Shelf, bookID uuid.UUID) error {
	ok, err := tx.IsOnShelf(ctx, userID, shelf.ID, bookID)
	if err == nil && !ok {
		return notOnShelf()
	}
	return err
}

func notOnShelf() error { return domain.NotFound("This book is not on that shelf.") }

func forbidden(message string) error {
	return &domain.Error{Kind: domain.KindForbidden, Message: message}
}

// functionError maps errors raised by the PL/pgSQL functions and triggers to
// specific messages (docs/api/errors.md, "Trigger messages").
func functionError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "P0002":
		if strings.Contains(pgErr.Message, "shelf") {
			return domain.NotFound("Shelf not found.")
		}
		return domain.NotFound("Book not found.")
	case "22023":
		// Our own trigger messages ("current_page must be between 0 and 320").
		field := "status"
		if strings.Contains(pgErr.Message, "current_page") {
			field = "currentPage"
		}
		return &domain.Error{Kind: domain.KindValidation, Message: pgErr.Message, Details: map[string]any{"field": field}, Err: pgErr}
	}
	return err
}

func shelfNameConflict(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "shelves_user_name_unique" {
		return domain.Conflict("You already have a shelf with this name.", map[string]any{"field": "name", "constraint": pgErr.ConstraintName})
	}
	return err
}
