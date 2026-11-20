package domain

import (
	"time"

	"github.com/google/uuid"
)

// Shelf is docs/api/openapi.yaml#/components/schemas/Shelf.
type Shelf struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Count     int32     `json:"count"`
	CreatedAt time.Time `json:"createdAt"`
}

// BookSummary is the small book card inside a progress row.
type BookSummary struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	Author   string    `json:"author"`
	CoverURL *string   `json:"coverUrl"`
}

// ShelfBookResult is the response of adding or moving a book.
type ShelfBookResult struct {
	ShelfID      uuid.UUID       `json:"shelfId"`
	BookID       uuid.UUID       `json:"bookId"`
	Progress     ReadingProgress `json:"progress"`
	HistoryEvent string          `json:"historyEvent"`
}

// HistoryEvent is one append-only reading_history row.
type HistoryEvent struct {
	ID         uuid.UUID `json:"id"`
	BookID     uuid.UUID `json:"bookId"`
	BookTitle  string    `json:"bookTitle"`
	EventType  string    `json:"eventType"`
	FromStatus *string   `json:"fromStatus"`
	ToStatus   *string   `json:"toStatus"`
	CreatedAt  time.Time `json:"createdAt"`
}
