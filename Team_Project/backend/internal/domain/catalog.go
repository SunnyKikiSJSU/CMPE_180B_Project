package domain

import (
	"time"

	"github.com/google/uuid"
)

// Ref is an author or category inside a Book.
type Ref struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// Book is docs/api/openapi.yaml#/components/schemas/Book.
type Book struct {
	ID                     uuid.UUID        `json:"id"`
	Title                  string           `json:"title"`
	Author                 string           `json:"author"`
	Authors                []Ref            `json:"authors"`
	Categories             []Ref            `json:"categories"`
	Source                 string           `json:"source"`
	InfoLink               *string          `json:"infoLink"`
	Description            *string          `json:"description"`
	PublishedDate          *string          `json:"publishedDate"`
	PublishedDatePrecision *string          `json:"publishedDatePrecision"`
	Publisher              *string          `json:"publisher"`
	PageCount              *int32           `json:"pageCount"`
	ISBN13                 *string          `json:"isbn13"`
	CoverURL               *string          `json:"coverUrl"`
	AverageRating          float64          `json:"averageRating"`
	RatingCount            int32            `json:"ratingCount"`
	ReviewCount            int32            `json:"reviewCount"`
	MyProgress             *ReadingProgress `json:"myProgress"`
	MyRating               *int16           `json:"myRating"`

	// GoogleBooksID is internal (deduplication); never serialized.
	GoogleBooksID *string `json:"-"`
}

// ReadingProgress is the caller's progress on a book (myProgress).
type ReadingProgress struct {
	BookID          uuid.UUID    `json:"bookId"`
	Status          string       `json:"status"`
	CurrentPage     int32        `json:"currentPage"`
	TotalPages      int32        `json:"totalPages"`
	PercentComplete float64      `json:"percentComplete"`
	Version         int32        `json:"version"`
	UpdatedAt       time.Time    `json:"updatedAt"`
	Book            *BookSummary `json:"book,omitempty"`
}

// Author and Category are list rows; detail responses add Books.
type Author struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	BookCount *int32    `json:"bookCount,omitempty"`
	Books     []Book    `json:"books,omitempty"`
}

type Category struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	BookCount *int32    `json:"bookCount,omitempty"`
	Books     []Book    `json:"books,omitempty"`
}

// Page is a requested page; Total is filled by the service.
type Page struct {
	Number int
	Size   int
}

func (p Page) Offset() int32 { return int32((p.Number - 1) * p.Size) }

// ExternalBook is docs/api/openapi.yaml#/components/schemas/ExternalBook: one
// normalized outside search result (an Open Library work). Not stored.
type ExternalBook struct {
	Source        string     `json:"source"`
	ExternalID    string     `json:"externalId"`
	BookID        *uuid.UUID `json:"bookId"`
	Title         string     `json:"title"`
	Subtitle      *string    `json:"subtitle"`
	Authors       []string   `json:"authors"`
	Categories    []string   `json:"categories"`
	Description   *string    `json:"description"`
	PublishedDate *string    `json:"publishedDate"`
	Publisher     *string    `json:"publisher"`
	PageCount     *int32     `json:"pageCount"`
	ISBN13        *string    `json:"isbn13"`
	CoverURL      *string    `json:"coverUrl"`
	InfoLink      string     `json:"infoLink"`

	// ISBN13s lists every ISBN-13 of the work's editions; it finds a local
	// book with the same ISBN (deduplication) and is never serialized.
	ISBN13s []string `json:"-"`
}
