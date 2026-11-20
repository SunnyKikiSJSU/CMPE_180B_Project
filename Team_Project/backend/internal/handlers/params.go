package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"book-metaverse/backend/internal/domain"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
	maxPage         = 10000
)

// pageParams reads ?page and ?pageSize (BACKEND_PLAN §4.1: default 20, max 100).
func pageParams(r *http.Request) (domain.Page, error) {
	page := domain.Page{Number: 1, Size: defaultPageSize}
	if raw := r.URL.Query().Get("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxPage {
			return page, domain.Validation("page", "page must be a whole number from 1 to 10000.")
		}
		page.Number = n
	}
	if raw := r.URL.Query().Get("pageSize"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > maxPageSize {
			return page, domain.Validation("pageSize", "pageSize must be a whole number from 1 to 100.")
		}
		page.Size = n
	}
	return page, nil
}

// pathID parses a UUID path parameter; a malformed ID is a 404 (it cannot exist).
func pathID(r *http.Request, name, label string) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		return uuid.Nil, domain.NotFound(label + " not found.")
	}
	return id, nil
}

// queryID parses an optional UUID query parameter.
func queryID(r *http.Request, name string) (*uuid.UUID, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, domain.Validation(name, name+" must be a UUID.")
	}
	return &id, nil
}
