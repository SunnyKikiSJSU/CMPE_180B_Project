package domain

import (
	"time"

	"github.com/google/uuid"
)

// User is the API's user object (docs/api/openapi.yaml#/components/schemas/User).
// Profile and Stats are present only on GET/PATCH /api/users/me.
type User struct {
	ID          uuid.UUID    `json:"id"`
	Email       string       `json:"email"`
	DisplayName string       `json:"displayName"`
	AvatarURL   *string      `json:"avatarUrl"`
	CreatedAt   time.Time    `json:"createdAt"`
	Profile     *UserProfile `json:"profile,omitempty"`
	Stats       *UserStats   `json:"stats,omitempty"`
}

type UserProfile struct {
	Bio           *string    `json:"bio"`
	FavoriteGenre *string    `json:"favoriteGenre"`
	UpdatedAt     *time.Time `json:"updatedAt,omitempty"`
}

type UserStats struct {
	ShelfCount     int32 `json:"shelfCount"`
	BooksCompleted int32 `json:"booksCompleted"`
	ReviewsWritten int32 `json:"reviewsWritten"`
	RatingsGiven   int32 `json:"ratingsGiven"`
}

// Credentials is the stored login data; it never leaves the service layer.
type Credentials struct {
	UserID       uuid.UUID
	Email        string
	PasswordHash string
}
