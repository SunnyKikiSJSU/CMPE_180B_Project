package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"book-metaverse/backend/internal/db"
	"book-metaverse/backend/internal/domain"
)

// Reports runs the read-only report queries (database/queries/reports.sql).
type Reports struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewReports(pool *pgxpool.Pool) *Reports { return &Reports{pool: pool, q: db.New(pool)} }

func windowEnd(w domain.Window) pgtype.Date {
	return pgtype.Date{Time: w.EndDate, Valid: true}
}

// ReadingReport runs its three queries in one read-only REPEATABLE READ
// transaction, so the totals and both lists come from the same snapshot even
// if the user updates progress while the report is being built.
func (r *Reports) ReadingReport(ctx context.Context, userID uuid.UUID, categories, activity int) (domain.ReadingReport, error) {
	var out domain.ReadingReport
	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		sum, err := q.ReadingReportSummary(ctx, userID)
		if err != nil {
			return err
		}
		out.BooksRead, out.PagesRead = sum.BooksRead, sum.PagesRead
		cats, err := q.FavoriteCategories(ctx, db.FavoriteCategoriesParams{UserID: userID, Limit: int32(categories)})
		if err != nil {
			return err
		}
		out.FavoriteCategories = make([]domain.Ref, 0, len(cats))
		for _, c := range cats {
			out.FavoriteCategories = append(out.FavoriteCategories, domain.Ref{ID: c.CategoryID, Name: c.Name})
		}
		recent, err := q.RecentReadingActivity(ctx, db.RecentReadingActivityParams{UserID: userID, Limit: int32(activity)})
		if err != nil {
			return err
		}
		out.RecentActivity = make([]domain.RecentActivity, 0, len(recent))
		for _, a := range recent {
			out.RecentActivity = append(out.RecentActivity, domain.RecentActivity{
				BookID: a.BookID, Status: a.Status, CurrentPage: a.CurrentPage, TotalPages: a.TotalPages, UpdatedAt: a.UpdatedAt.Time,
			})
		}
		return nil
	})
	return out, err
}

func (r *Reports) UserStats(ctx context.Context, userID uuid.UUID) (domain.UserStatsReport, error) {
	s, err := r.q.UserReadingStats(ctx, userID)
	if err != nil {
		return domain.UserStatsReport{}, err
	}
	return domain.UserStatsReport{
		BooksCompleted: s.BooksCompleted, BooksInProgress: s.BooksInProgress, PagesRead: s.PagesRead,
		RatingsGiven: s.RatingsGiven, AverageRating: s.AverageRating, ReviewsWritten: s.ReviewsWritten,
	}, nil
}

func (r *Reports) Popular(ctx context.Context, w domain.Window, limit int) ([]domain.PopularBook, error) {
	rows, err := r.q.PopularBooks(ctx, db.PopularBooksParams{EndDate: windowEnd(w), Days: int32(w.Days), Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.PopularBook, 0, len(rows))
	for _, b := range rows {
		out = append(out, domain.PopularBook{
			BookID: b.BookID, Title: b.Title, CoverURL: text(b.CoverUrl), PrimaryAuthor: b.PrimaryAuthor,
			ShelfAdds: b.ShelfAdds, Ratings: b.Ratings, AverageRating: b.AverageRating, Score: b.Score,
		})
	}
	return out, nil
}

func (r *Reports) TopRated(ctx context.Context, minRatings, limit int) ([]domain.TopRatedBook, error) {
	rows, err := r.q.TopRatedBooks(ctx, db.TopRatedBooksParams{MinRatings: int32(minRatings), Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.TopRatedBook, 0, len(rows))
	for _, b := range rows {
		out = append(out, domain.TopRatedBook{
			BookID: b.BookID, Title: b.Title, CoverURL: text(b.CoverUrl), PrimaryAuthor: b.PrimaryAuthor,
			AverageRating: b.AverageRating, RatingCount: b.RatingCount,
		})
	}
	return out, nil
}

func (r *Reports) Recommendations(ctx context.Context, userID uuid.UUID, limit int) ([]domain.Recommendation, error) {
	rows, err := r.q.RecommendedBooks(ctx, db.RecommendedBooksParams{UserID: userID, Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Recommendation, 0, len(rows))
	for _, b := range rows {
		out = append(out, domain.Recommendation{
			BookID: b.BookID, Title: b.Title, CoverURL: text(b.CoverUrl), PrimaryAuthor: b.PrimaryAuthor,
			Score: b.Score, AverageRating: b.AverageRating,
		})
	}
	return out, nil
}

func (r *Reports) AuthorRankings(ctx context.Context, minRatings, limit int) ([]domain.AuthorRanking, error) {
	rows, err := r.q.AuthorRankings(ctx, db.AuthorRankingsParams{MinRatings: int32(minRatings), Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.AuthorRanking, 0, len(rows))
	for _, a := range rows {
		out = append(out, domain.AuthorRanking{
			AuthorID: a.AuthorID, Name: a.Name, BookCount: a.BookCount, RatingCount: a.RatingCount,
			AverageRating: a.AverageRating, Ranking: a.Ranking,
		})
	}
	return out, nil
}

func (r *Reports) CategoryRankings(ctx context.Context, minRatings, limit int) ([]domain.CategoryRanking, error) {
	rows, err := r.q.CategoryRankings(ctx, db.CategoryRankingsParams{MinRatings: int32(minRatings), Limit: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.CategoryRanking, 0, len(rows))
	for _, c := range rows {
		out = append(out, domain.CategoryRanking{
			CategoryID: c.CategoryID, Name: c.Name, BookCount: c.BookCount, RatingCount: c.RatingCount,
			AverageRating: c.AverageRating, Ranking: c.Ranking,
		})
	}
	return out, nil
}

func (r *Reports) ReviewActivity(ctx context.Context, w domain.Window) ([]domain.ActivityPoint, error) {
	rows, err := r.q.ReviewActivity(ctx, db.ReviewActivityParams{EndDate: windowEnd(w), Days: int32(w.Days)})
	if err != nil {
		return nil, err
	}
	out := make([]domain.ActivityPoint, 0, len(rows))
	for _, p := range rows {
		out = append(out, domain.ActivityPoint{Date: p.Day.Time.Format("2006-01-02"), Reviews: p.Reviews, Ratings: p.Ratings})
	}
	return out, nil
}
