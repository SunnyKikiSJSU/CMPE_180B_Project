package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"book-metaverse/backend/internal/db"
	"book-metaverse/backend/internal/domain"
)

// Sizes of the two lists in the dashboard reading report.
const (
	readingReportCategories = 5
	readingReportActivity   = 10
)

// ReportStore is the read-only repository surface for reports.
type ReportStore interface {
	ReadingReport(ctx context.Context, userID uuid.UUID, categories, activity int) (domain.ReadingReport, error)
	UserStats(ctx context.Context, userID uuid.UUID) (domain.UserStatsReport, error)
	Popular(ctx context.Context, w domain.Window, limit int) ([]domain.PopularBook, error)
	TopRated(ctx context.Context, minRatings, limit int) ([]domain.TopRatedBook, error)
	Recommendations(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Recommendation, error)
	AuthorRankings(ctx context.Context, minRatings, limit int) ([]domain.AuthorRanking, error)
	CategoryRankings(ctx context.Context, minRatings, limit int) ([]domain.CategoryRanking, error)
	ReviewActivity(ctx context.Context, w domain.Window) ([]domain.ActivityPoint, error)
}

// Reports wraps the report queries. Parameters are validated by the handler
// (limit 1-100, days 1-365, minRatings 0-1000, endDate); results are never nil.
type Reports struct {
	ReportStore
	catalog []domain.QueryCatalogEntry
}

func NewReports(store ReportStore) *Reports {
	catalog, err := buildCatalog()
	if err != nil {
		panic(err) // a renamed query must fail at startup, not at request time
	}
	return &Reports{ReportStore: store, catalog: catalog}
}

func (s *Reports) ReadingReport(ctx context.Context, userID uuid.UUID) (domain.ReadingReport, error) {
	return s.ReportStore.ReadingReport(ctx, userID, readingReportCategories, readingReportActivity)
}

// Catalog is GET /api/reports/queries: fixed SQL text, never user input.
func (s *Reports) Catalog() []domain.QueryCatalogEntry { return s.catalog }

// catalogSpec maps the course evidence IDs (BACKEND_PLAN.md §5.4) to the
// endpoints that run them and the named queries in database/queries/.
var catalogSpec = []domain.QueryCatalogEntry{
	{ID: "q01", Name: "Library search", Description: "Keyword, title, author, and category search over books in PostgreSQL: full-text search with ts_rank relevance, plus ILIKE matching served by pg_trgm indexes.",
		Endpoints: []string{"GET /api/books"}, Queries: []string{"SearchBooks"}},
	{ID: "q02", Name: "Book detail", Description: "One book with its authors and categories (json_agg) and its rating summary.",
		Endpoints: []string{"GET /api/books/{bookId}"}, Queries: []string{"GetBookByID"}},
	{ID: "q03", Name: "Shelf contents", Description: "A user's shelf with book metadata, reading progress, latest history event, and a window-function total.",
		Endpoints: []string{"GET /api/shelves/{shelfId}/books"}, Queries: []string{"ListShelfBooks"}},
	{ID: "q04", Name: "Average ratings by book, category, and author", Description: "Average rating and rating count per book (with a minimum count), per category, and per author (GROUP BY ... HAVING).",
		Endpoints: []string{"GET /api/reports/top-rated"}, Queries: []string{"TopRatedBooks", "RatingSummaryByCategory", "RatingSummaryByAuthor"}},
	{ID: "q05", Name: "Popular books", Description: "Books most shelved and rated within a window of whole UTC days; activity is pre-aggregated per table to avoid join fan-out.",
		Endpoints: []string{"GET /api/reports/popular"}, Queries: []string{"PopularBooks"}},
	{ID: "q06", Name: "Recommendations", Description: "Books from categories the user completed or rated 4-5, weighted by how many liked books share the category, excluding books the user already tracks, shelved, rated, or reviewed.",
		Endpoints: []string{"GET /api/reports/recommendations"}, Queries: []string{"RecommendedBooks"}},
	{ID: "q07", Name: "Reading statistics", Description: "Per-user totals: books completed and in progress, pages read (page_count for completed books, current_page for reading/paused), ratings, reviews, favorite categories, and recent activity.",
		Endpoints: []string{"GET /api/reports/user-stats", "GET /api/reports/reading-history"}, Queries: []string{"UserReadingStats", "ReadingReportSummary", "FavoriteCategories", "RecentReadingActivity"}},
	{ID: "q08", Name: "Author and category rankings", Description: "Leaderboards ranked with RANK() OVER by average rating, then rating count.",
		Endpoints: []string{"GET /api/reports/authors", "GET /api/reports/categories"}, Queries: []string{"AuthorRankings", "CategoryRankings"}},
	{ID: "q09", Name: "Review activity per day", Description: "Reviews and ratings per UTC day; generate_series fills days with no activity.",
		Endpoints: []string{"GET /api/reports/review-activity"}, Queries: []string{"ReviewActivity"}},
	{ID: "q10", Name: "Rating distribution", Description: "Number of ratings for each value 1-5 for one book.",
		Endpoints: []string{"GET /api/books/{bookId}/ratings"}, Queries: []string{"GetRatingDistribution"}},
}

func buildCatalog() ([]domain.QueryCatalogEntry, error) {
	out := make([]domain.QueryCatalogEntry, 0, len(catalogSpec))
	for _, e := range catalogSpec {
		parts := make([]string, 0, len(e.Queries))
		for _, name := range e.Queries {
			sql, ok := db.QueryText(name)
			if !ok {
				return nil, fmt.Errorf("report catalog %s: unknown query %s", e.ID, name)
			}
			parts = append(parts, strings.TrimSpace(sql))
		}
		e.SQL = strings.Join(parts, "\n\n")
		out = append(out, e)
	}
	return out, nil
}
