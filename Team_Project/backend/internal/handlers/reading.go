package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/httpx"
	"book-metaverse/backend/internal/middleware"
	"book-metaverse/backend/internal/services"
)

type ReadingService interface {
	Shelves(ctx context.Context, userID uuid.UUID) ([]domain.Shelf, error)
	CreateShelf(ctx context.Context, userID uuid.UUID, name, shelfType string) (domain.Shelf, error)
	RenameShelf(ctx context.Context, userID, shelfID uuid.UUID, name string) (domain.Shelf, error)
	DeleteShelf(ctx context.Context, userID, shelfID uuid.UUID) error
	ShelfBooks(ctx context.Context, userID, shelfID uuid.UUID, sort string, page domain.Page) ([]domain.Book, int64, error)
	AddBook(ctx context.Context, userID, shelfID, bookID uuid.UUID) (domain.ShelfBookResult, error)
	MoveBook(ctx context.Context, userID, fromShelfID, bookID, toShelfID uuid.UUID) (domain.ShelfBookResult, error)
	RemoveBook(ctx context.Context, userID, shelfID, bookID uuid.UUID) error
	Progress(ctx context.Context, userID, bookID uuid.UUID) (domain.ReadingProgress, error)
	ListProgress(ctx context.Context, userID uuid.UUID, status string, page domain.Page) ([]domain.ReadingProgress, int64, error)
	UpdateProgress(ctx context.Context, userID, bookID uuid.UUID, in services.ProgressUpdate) (domain.ReadingProgress, error)
	DeleteProgress(ctx context.Context, userID, bookID uuid.UUID) error
	History(ctx context.Context, userID uuid.UUID, bookID *uuid.UUID, limit int) ([]domain.HistoryEvent, error)
}

// Reading serves /api/shelves*, /api/reading-progress*, and /api/reading-history.
// Every route requires a session; all data is scoped to the caller.
type Reading struct{ Service ReadingService }

type shelfRequest struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type renameRequest struct {
	Name string `json:"name"`
}

type addBookRequest struct {
	BookID *uuid.UUID `json:"bookId"`
}

type moveBookRequest struct {
	TargetShelfID *uuid.UUID `json:"targetShelfId"`
}

type progressRequest struct {
	CurrentPage     *int32   `json:"currentPage"`
	PercentComplete *float64 `json:"percentComplete"`
	Status          *string  `json:"status"`
	Version         *int32   `json:"version"`
}

func caller(r *http.Request) uuid.UUID {
	id, _ := middleware.UserID(r.Context())
	return id
}

// ListShelves returns all shelves that belong to the caller, with status 200.
func (h Reading) ListShelves(w http.ResponseWriter, r *http.Request) {
	shelves, err := h.Service.Shelves(r.Context(), caller(r))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, shelves)
}

// CreateShelf creates a shelf for the caller from the JSON body {name, type}.
// Responds 201 with the new shelf.
func (h Reading) CreateShelf(w http.ResponseWriter, r *http.Request) {
	var req shelfRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	shelf, err := h.Service.CreateShelf(r.Context(), caller(r), req.Name, req.Type)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, shelf)
}

// RenameShelf renames the shelf in {shelfId} using the JSON body {name}.
// Responds 200 with the updated shelf.
func (h Reading) RenameShelf(w http.ResponseWriter, r *http.Request) {
	shelfID, err := pathID(r, "shelfId", "Shelf")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	var req renameRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	shelf, err := h.Service.RenameShelf(r.Context(), caller(r), shelfID, req.Name)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, shelf)
}

// DeleteShelf deletes the caller's shelf in {shelfId}. Responds 204 with no body.
func (h Reading) DeleteShelf(w http.ResponseWriter, r *http.Request) {
	shelfID, err := pathID(r, "shelfId", "Shelf")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.Service.DeleteShelf(r.Context(), caller(r), shelfID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ShelfBooks lists the books on the shelf in {shelfId}. Accepts page, pageSize,
// and sort query parameters; pagination is returned in headers. Responds 200.
func (h Reading) ShelfBooks(w http.ResponseWriter, r *http.Request) {
	shelfID, err := pathID(r, "shelfId", "Shelf")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	page, err := pageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	books, total, err := h.Service.ShelfBooks(r.Context(), caller(r), shelfID, r.URL.Query().Get("sort"), page)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Paginate(w, total, page.Number, page.Size)
	httpx.JSON(w, http.StatusOK, books)
}

// AddBook adds a book to the shelf in {shelfId}. The JSON body must contain
// bookId. Responds 200 with the ShelfBookResult.
func (h Reading) AddBook(w http.ResponseWriter, r *http.Request) {
	shelfID, err := pathID(r, "shelfId", "Shelf")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	var req addBookRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.BookID == nil {
		httpx.Error(w, r, domain.Validation("bookId", "bookId is required."))
		return
	}
	result, err := h.Service.AddBook(r.Context(), caller(r), shelfID, *req.BookID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

// MoveBook moves the book in {bookId} from the shelf in {shelfId} to the shelf
// in the JSON body's targetShelfId, which is required. Responds 200 with the
// ShelfBookResult.
func (h Reading) MoveBook(w http.ResponseWriter, r *http.Request) {
	shelfID, err := pathID(r, "shelfId", "Shelf")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	bookID, err := pathID(r, "bookId", "Book")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	var req moveBookRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.TargetShelfID == nil {
		httpx.Error(w, r, domain.Validation("targetShelfId", "targetShelfId is required."))
		return
	}
	result, err := h.Service.MoveBook(r.Context(), caller(r), shelfID, bookID, *req.TargetShelfID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

// RemoveBook removes the book in {bookId} from the shelf in {shelfId}.
// Responds 204 with no body.
func (h Reading) RemoveBook(w http.ResponseWriter, r *http.Request) {
	shelfID, err := pathID(r, "shelfId", "Shelf")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	bookID, err := pathID(r, "bookId", "Book")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.Service.RemoveBook(r.Context(), caller(r), shelfID, bookID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// ListProgress lists the caller's reading progress records, paginated, with an
// optional status filter. Responds 200.
func (h Reading) ListProgress(w http.ResponseWriter, r *http.Request) {
	page, err := pageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	items, total, err := h.Service.ListProgress(r.Context(), caller(r), r.URL.Query().Get("status"), page)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Paginate(w, total, page.Number, page.Size)
	httpx.JSON(w, http.StatusOK, items)
}

// GetProgress returns the caller's reading progress for the book in {bookId}.
// Responds 200.
func (h Reading) GetProgress(w http.ResponseWriter, r *http.Request) {
	bookID, err := pathID(r, "bookId", "Book")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	progress, err := h.Service.Progress(r.Context(), caller(r), bookID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, progress)
}

// UpdateProgress updates the caller's reading progress for the book in {bookId}
// from the JSON body (currentPage, percentComplete, status, version). Responds 200
// with the updated progress.
func (h Reading) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	bookID, err := pathID(r, "bookId", "Book")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	var req progressRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	progress, err := h.Service.UpdateProgress(r.Context(), caller(r), bookID, services.ProgressUpdate(req))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, progress)
}

// DeleteProgress deletes the caller's reading progress for the book in {bookId}.
// Responds 204 with no body.
func (h Reading) DeleteProgress(w http.ResponseWriter, r *http.Request) {
	bookID, err := pathID(r, "bookId", "Book")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.Service.DeleteProgress(r.Context(), caller(r), bookID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// History returns the caller's reading history, newest first. Optional query
// parameters: limit (default 20) and bookId. Responds 200.
func (h Reading) History(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			httpx.Error(w, r, domain.Validation("limit", "limit must be between 1 and 100."))
			return
		}
		limit = n
	}
	bookID, err := queryID(r, "bookId")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	events, err := h.Service.History(r.Context(), caller(r), bookID, limit)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, events)
}
