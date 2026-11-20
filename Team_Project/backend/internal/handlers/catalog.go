package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/httpx"
	"book-metaverse/backend/internal/middleware"
	"book-metaverse/backend/internal/services"
)

type CatalogService interface {
	Search(ctx context.Context, in services.SearchInput) ([]domain.Book, int64, error)
	Get(ctx context.Context, id uuid.UUID, userID *uuid.UUID) (domain.Book, error)
	Create(ctx context.Context, in services.BookFields) (domain.Book, error)
	Update(ctx context.Context, id uuid.UUID, in services.BookFields) (domain.Book, error)
	ListAuthors(ctx context.Context, q string, page domain.Page) ([]domain.Author, int64, error)
	Author(ctx context.Context, id uuid.UUID, page domain.Page) (domain.Author, int64, error)
	ListCategories(ctx context.Context, q string, page domain.Page) ([]domain.Category, int64, error)
	Category(ctx context.Context, id uuid.UUID, page domain.Page) (domain.Category, int64, error)
}

type Catalog struct{ Service CatalogService }

// createBookRequest is BookInput; updateBookRequest is BookUpdate (no
// authors or categories, so sending them is a 400 unknown field).
type createBookRequest struct {
	Title         *string  `json:"title"`
	Authors       []string `json:"authors"`
	Categories    []string `json:"categories"`
	Description   *string  `json:"description"`
	PublishedDate *string  `json:"publishedDate"`
	Publisher     *string  `json:"publisher"`
	PageCount     *int32   `json:"pageCount"`
	ISBN13        *string  `json:"isbn13"`
	CoverURL      *string  `json:"coverUrl"`
}

type updateBookRequest struct {
	Title         *string `json:"title"`
	Description   *string `json:"description"`
	PublishedDate *string `json:"publishedDate"`
	Publisher     *string `json:"publisher"`
	PageCount     *int32  `json:"pageCount"`
	ISBN13        *string `json:"isbn13"`
	CoverURL      *string `json:"coverUrl"`
}

// SearchBooks is GET /api/books: a raw array plus pagination headers.
func (h Catalog) SearchBooks(w http.ResponseWriter, r *http.Request) {
	page, err := pageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	authorID, err := queryID(r, "authorId")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	categoryID, err := queryID(r, "categoryId")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	query := r.URL.Query()
	books, total, err := h.Service.Search(r.Context(), services.SearchInput{
		Q: query.Get("q"), Title: query.Get("title"), Sort: query.Get("sort"),
		AuthorID: authorID, CategoryID: categoryID, Page: page,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Paginate(w, total, page.Number, page.Size)
	httpx.JSON(w, http.StatusOK, books)
}

// GetBook adds myProgress and myRating for a signed-in caller; personal data
// is never cached.
func (h Catalog) GetBook(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "bookId", "Book")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	var userID *uuid.UUID
	if uid, ok := middleware.UserID(r.Context()); ok {
		userID = &uid
		w.Header().Set("Cache-Control", "no-store")
	}
	book, err := h.Service.Get(r.Context(), id, userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, book)
}

func (h Catalog) CreateBook(w http.ResponseWriter, r *http.Request) {
	var req createBookRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	book, err := h.Service.Create(r.Context(), services.BookFields{
		Title: req.Title, Description: req.Description, PublishedDate: req.PublishedDate, Publisher: req.Publisher,
		ISBN13: req.ISBN13, CoverURL: req.CoverURL, PageCount: req.PageCount, Authors: req.Authors, Categories: req.Categories,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, book)
}

func (h Catalog) UpdateBook(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "bookId", "Book")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	var req updateBookRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	book, err := h.Service.Update(r.Context(), id, services.BookFields{
		Title: req.Title, Description: req.Description, PublishedDate: req.PublishedDate, Publisher: req.Publisher,
		ISBN13: req.ISBN13, CoverURL: req.CoverURL, PageCount: req.PageCount,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, book)
}

func (h Catalog) ListAuthors(w http.ResponseWriter, r *http.Request) {
	page, err := pageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	authors, total, err := h.Service.ListAuthors(r.Context(), r.URL.Query().Get("q"), page)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Paginate(w, total, page.Number, page.Size)
	httpx.JSON(w, http.StatusOK, authors)
}

// GetAuthor returns the author with one page of books; the headers page the books.
func (h Catalog) GetAuthor(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "authorId", "Author")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	page, err := pageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	author, total, err := h.Service.Author(r.Context(), id, page)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Paginate(w, total, page.Number, page.Size)
	httpx.JSON(w, http.StatusOK, authorDetail{Author: author, Books: nonNil(author.Books)})
}

func (h Catalog) ListCategories(w http.ResponseWriter, r *http.Request) {
	page, err := pageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	categories, total, err := h.Service.ListCategories(r.Context(), r.URL.Query().Get("q"), page)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Paginate(w, total, page.Number, page.Size)
	httpx.JSON(w, http.StatusOK, categories)
}

func (h Catalog) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "categoryId", "Category")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	page, err := pageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	category, total, err := h.Service.Category(r.Context(), id, page)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Paginate(w, total, page.Number, page.Size)
	httpx.JSON(w, http.StatusOK, categoryDetail{Category: category, Books: nonNil(category.Books)})
}

// Detail responses always include "books" (an empty array, not omitted).
type authorDetail struct {
	domain.Author
	Books []domain.Book `json:"books"`
}

type categoryDetail struct {
	domain.Category
	Books []domain.Book `json:"books"`
}

func nonNil(books []domain.Book) []domain.Book {
	if books == nil {
		return []domain.Book{}
	}
	return books
}
