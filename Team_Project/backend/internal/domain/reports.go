package domain

import (
	"time"

	"github.com/google/uuid"
)

// ReadingReport is GET /api/reports/reading-history (frontend ReadingReport).
type ReadingReport struct {
	BooksRead          int32            `json:"booksRead"`
	PagesRead          int32            `json:"pagesRead"`
	FavoriteCategories []Ref            `json:"favoriteCategories"`
	RecentActivity     []RecentActivity `json:"recentActivity"`
}

type RecentActivity struct {
	BookID      uuid.UUID `json:"bookId"`
	Status      string    `json:"status"`
	CurrentPage int32     `json:"currentPage"`
	TotalPages  int32     `json:"totalPages"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// UserStatsReport is GET /api/reports/user-stats.
type UserStatsReport struct {
	BooksCompleted  int32   `json:"booksCompleted"`
	BooksInProgress int32   `json:"booksInProgress"`
	PagesRead       int32   `json:"pagesRead"`
	RatingsGiven    int32   `json:"ratingsGiven"`
	AverageRating   float64 `json:"averageRating"`
	ReviewsWritten  int32   `json:"reviewsWritten"`
}

type PopularBook struct {
	BookID        uuid.UUID `json:"bookId"`
	Title         string    `json:"title"`
	CoverURL      *string   `json:"coverUrl"`
	PrimaryAuthor string    `json:"primaryAuthor"`
	ShelfAdds     int32     `json:"shelfAdds"`
	Ratings       int32     `json:"ratings"`
	AverageRating float64   `json:"averageRating"`
	Score         int32     `json:"score"`
}

type TopRatedBook struct {
	BookID        uuid.UUID `json:"bookId"`
	Title         string    `json:"title"`
	CoverURL      *string   `json:"coverUrl"`
	PrimaryAuthor string    `json:"primaryAuthor"`
	AverageRating float64   `json:"averageRating"`
	RatingCount   int32     `json:"ratingCount"`
}

type Recommendation struct {
	BookID        uuid.UUID `json:"bookId"`
	Title         string    `json:"title"`
	CoverURL      *string   `json:"coverUrl"`
	PrimaryAuthor string    `json:"primaryAuthor"`
	Score         int32     `json:"score"`
	AverageRating float64   `json:"averageRating"`
}

type AuthorRanking struct {
	AuthorID      uuid.UUID `json:"authorId"`
	Name          string    `json:"name"`
	BookCount     int32     `json:"bookCount"`
	RatingCount   int32     `json:"ratingCount"`
	AverageRating float64   `json:"averageRating"`
	Ranking       int32     `json:"ranking"`
}

type CategoryRanking struct {
	CategoryID    uuid.UUID `json:"categoryId"`
	Name          string    `json:"name"`
	BookCount     int32     `json:"bookCount"`
	RatingCount   int32     `json:"ratingCount"`
	AverageRating float64   `json:"averageRating"`
	Ranking       int32     `json:"ranking"`
}

// ActivityPoint is one UTC day; Date is YYYY-MM-DD.
type ActivityPoint struct {
	Date    string `json:"date"`
	Reviews int32  `json:"reviews"`
	Ratings int32  `json:"ratings"`
}

// Window is a report time window of Days whole UTC days ending on EndDate.
type Window struct {
	Days    int
	EndDate time.Time
}

// QueryCatalogEntry is one evidence query in GET /api/reports/queries.
type QueryCatalogEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Endpoints   []string `json:"endpoints"`
	Queries     []string `json:"queries"`
	SQL         string   `json:"sql"`
}
