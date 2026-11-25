package domain

import (
	"time"

	"github.com/google/uuid"
)

// RatingSummary is the trigger-maintained book_rating_summaries row.
type RatingSummary struct {
	AverageRating float64 `json:"averageRating"`
	RatingCount   int32   `json:"ratingCount"`
	ReviewCount   int32   `json:"reviewCount"`
}

// RatingSummaryResponse is GET /api/books/{bookId}/ratings.
type RatingSummaryResponse struct {
	AverageRating float64          `json:"averageRating"`
	RatingCount   int32            `json:"ratingCount"`
	Distribution  map[string]int32 `json:"distribution"`
}

type Rating struct {
	BookID    uuid.UUID      `json:"bookId"`
	UserID    uuid.UUID      `json:"userId"`
	Value     int16          `json:"value"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Summary   *RatingSummary `json:"summary,omitempty"`
	// BookTitle and BookCoverURL are set only by GET /api/users/me/ratings.
	BookTitle    string  `json:"bookTitle,omitempty"`
	BookCoverURL *string `json:"bookCoverUrl,omitempty"`
}

type ReviewUser struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl"`
}

// Review is docs/api/openapi.yaml#/components/schemas/Review. Rating is the
// reviewer's current rating of the book, if any.
type Review struct {
	ID        uuid.UUID   `json:"id"`
	BookID    uuid.UUID   `json:"bookId"`
	UserID    uuid.UUID   `json:"userId"`
	Title     *string     `json:"title"`
	Body      string      `json:"body"`
	Rating    *int16      `json:"rating"`
	User      *ReviewUser `json:"user,omitempty"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
	// BookTitle and BookCoverURL are set only by GET /api/users/me/reviews.
	BookTitle    string  `json:"bookTitle,omitempty"`
	BookCoverURL *string `json:"bookCoverUrl,omitempty"`
}

// ReviewRatingResult is PUT /api/books/{bookId}/review.
type ReviewRatingResult struct {
	Review  Review        `json:"review"`
	Rating  *Rating       `json:"rating"`
	Summary RatingSummary `json:"summary"`
}
