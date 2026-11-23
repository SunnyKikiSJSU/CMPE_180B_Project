package handlers

import (
	"context"
	"net/http"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/httpx"
	"book-metaverse/backend/internal/services"
)

type ExternalService interface {
	Search(ctx context.Context, in services.ExternalSearchInput) ([]domain.ExternalBook, int64, error)
	Import(ctx context.Context, source, id string) (domain.Book, bool, error)
}

// External serves outside search and import (Open Library).
type External struct{ Service ExternalService }

type importRequest struct {
	Source     string `json:"source"`
	ExternalID string `json:"externalId"`
}

// Search is GET /api/external/books: a raw array plus pagination headers.
// X-Total-Count is Open Library's numFound.
func (h External) Search(w http.ResponseWriter, r *http.Request) {
	page, err := pageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	query := r.URL.Query()
	books, total, err := h.Service.Search(r.Context(), services.ExternalSearchInput{
		Q: query.Get("q"), Title: query.Get("title"), Author: query.Get("author"), Category: query.Get("category"),
		ISBN: query.Get("isbn"), Sort: query.Get("sort"), Page: page,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Paginate(w, total, page.Number, page.Size)
	httpx.JSON(w, http.StatusOK, books)
}

// Import is POST /api/books/import: 201 when the work was added, 200 when the
// library already had it.
func (h External) Import(w http.ResponseWriter, r *http.Request) {
	var req importRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	book, created, err := h.Service.Import(r.Context(), req.Source, req.ExternalID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	httpx.JSON(w, status, book)
}
