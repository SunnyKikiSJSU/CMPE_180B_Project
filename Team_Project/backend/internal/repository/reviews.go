package repository

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"book-metaverse/backend/internal/db"
	"book-metaverse/backend/internal/domain"
)

// Reviews covers ratings, reviews, and the trigger-maintained summaries.
// Every write is scoped by user_id in SQL.
type Reviews struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewReviews(pool *pgxpool.Pool) *Reviews { return &Reviews{pool: pool, q: db.New(pool)} }

// InTx runs fn in one READ COMMITTED transaction; the summary triggers run
// inside it, so a rollback also undoes the summary change.
func (r *Reviews) InTx(ctx context.Context, fn func(*Reviews) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(&Reviews{pool: r.pool, q: r.q.WithTx(tx)}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Summary returns the book's summary; ErrNoRows means the book does not exist
// (every book gets a summary row when it is inserted).
func (r *Reviews) Summary(ctx context.Context, bookID uuid.UUID) (domain.RatingSummary, error) {
	s, err := r.q.GetBookRatingSummary(ctx, bookID)
	if err != nil {
		return domain.RatingSummary{}, noRows(err)
	}
	return domain.RatingSummary{AverageRating: s.AverageRating, RatingCount: s.RatingCount, ReviewCount: s.ReviewCount}, nil
}

// Distribution counts ratings per value, with every value 1-5 present.
func (r *Reviews) Distribution(ctx context.Context, bookID uuid.UUID) (map[string]int32, error) {
	rows, err := r.q.GetRatingDistribution(ctx, bookID)
	if err != nil {
		return nil, err
	}
	out := map[string]int32{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0}
	for _, row := range rows {
		out[strconv.Itoa(int(row.RatingValue))] = row.RatingCount
	}
	return out, nil
}

func (r *Reviews) MyRating(ctx context.Context, userID, bookID uuid.UUID) (domain.Rating, error) {
	row, err := r.q.GetMyRating(ctx, db.GetMyRatingParams{UserID: userID, BookID: bookID})
	if err != nil {
		return domain.Rating{}, noRows(err)
	}
	return rating(row), nil
}

func (r *Reviews) UpsertRating(ctx context.Context, userID, bookID uuid.UUID, value int16) (domain.Rating, error) {
	row, err := r.q.UpsertRating(ctx, db.UpsertRatingParams{UserID: userID, BookID: bookID, RatingValue: value})
	if err != nil {
		return domain.Rating{}, err
	}
	return rating(row), nil
}

func (r *Reviews) DeleteRating(ctx context.Context, userID, bookID uuid.UUID) (int64, error) {
	return r.q.DeleteMyRating(ctx, db.DeleteMyRatingParams{UserID: userID, BookID: bookID})
}

func (r *Reviews) MyRatings(ctx context.Context, userID uuid.UUID, page domain.Page) ([]domain.Rating, int64, error) {
	rows, err := r.q.ListMyRatings(ctx, db.ListMyRatingsParams{UserID: userID, Limit: int32(page.Size), Offset: page.Offset()})
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Rating, 0, len(rows))
	var total int64
	for _, row := range rows {
		total = int64(row.TotalCount)
		out = append(out, domain.Rating{BookID: row.BookID, UserID: row.UserID, Value: row.RatingValue,
			CreatedAt: timestamp(row.CreatedAt), UpdatedAt: timestamp(row.UpdatedAt)})
	}
	return out, total, nil
}

func (r *Reviews) BookReviews(ctx context.Context, bookID uuid.UUID, sort string, page domain.Page) ([]domain.Review, int64, error) {
	rows, err := r.q.ListBookReviews(ctx, db.ListBookReviewsParams{
		BookID: bookID, Sort: pgtype.Text{String: sort, Valid: true}, Limit: int32(page.Size), Offset: page.Offset(),
	})
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Review, 0, len(rows))
	var total int64
	for _, row := range rows {
		total = int64(row.TotalCount)
		out = append(out, domain.Review{
			ID: row.ReviewID, BookID: row.BookID, UserID: row.UserID, Title: text(row.Title), Body: row.ReviewText,
			Rating: int2(row.RatingValue), CreatedAt: timestamp(row.CreatedAt), UpdatedAt: timestamp(row.UpdatedAt),
			User: &domain.ReviewUser{ID: row.UserID, DisplayName: row.DisplayName, AvatarURL: text(row.AvatarUrl)},
		})
	}
	return out, total, nil
}

// Review returns one review with its author and the author's rating.
func (r *Reviews) Review(ctx context.Context, reviewID uuid.UUID) (domain.Review, error) {
	row, err := r.q.GetReviewByID(ctx, reviewID)
	if err != nil {
		return domain.Review{}, noRows(err)
	}
	return domain.Review{
		ID: row.ReviewID, BookID: row.BookID, UserID: row.UserID, Title: text(row.Title), Body: row.ReviewText,
		Rating: int2(row.RatingValue), CreatedAt: timestamp(row.CreatedAt), UpdatedAt: timestamp(row.UpdatedAt),
		User: &domain.ReviewUser{ID: row.UserID, DisplayName: row.DisplayName, AvatarURL: text(row.AvatarUrl)},
	}, nil
}

func (r *Reviews) CreateReview(ctx context.Context, userID, bookID uuid.UUID, title *string, body string) (uuid.UUID, error) {
	row, err := r.q.CreateReview(ctx, db.CreateReviewParams{UserID: userID, BookID: bookID, Title: pgText(title), ReviewText: body})
	return row.ReviewID, err
}

func (r *Reviews) UpsertReview(ctx context.Context, userID, bookID uuid.UUID, title *string, body string) (uuid.UUID, error) {
	row, err := r.q.UpsertReview(ctx, db.UpsertReviewParams{UserID: userID, BookID: bookID, Title: pgText(title), ReviewText: body})
	return row.ReviewID, err
}

// UpdateReview edits the caller's own review; ErrNoRows if missing or not theirs.
// UpdateReview changes the title only when setTitle is true; a nil title then clears it.
func (r *Reviews) UpdateReview(ctx context.Context, userID, reviewID uuid.UUID, setTitle bool, title *string, body string) error {
	_, err := r.q.UpdateReview(ctx, db.UpdateReviewParams{ReviewID: reviewID, UserID: userID, SetTitle: setTitle, Title: pgText(title), ReviewText: pgtype.Text{String: body, Valid: true}})
	return noRows(err)
}

func (r *Reviews) DeleteReview(ctx context.Context, userID, reviewID uuid.UUID) (int64, error) {
	return r.q.DeleteReview(ctx, db.DeleteReviewParams{ReviewID: reviewID, UserID: userID})
}

func (r *Reviews) MyReviews(ctx context.Context, userID uuid.UUID, page domain.Page) ([]domain.Review, int64, error) {
	rows, err := r.q.ListMyReviews(ctx, db.ListMyReviewsParams{UserID: userID, Limit: int32(page.Size), Offset: page.Offset()})
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Review, 0, len(rows))
	var total int64
	for _, row := range rows {
		total = int64(row.TotalCount)
		out = append(out, domain.Review{
			ID: row.ReviewID, BookID: row.BookID, UserID: row.UserID, Title: text(row.Title), Body: row.ReviewText,
			Rating: int2(row.RatingValue), CreatedAt: timestamp(row.CreatedAt), UpdatedAt: timestamp(row.UpdatedAt),
		})
	}
	return out, total, nil
}

func rating(row db.Rating) domain.Rating {
	return domain.Rating{BookID: row.BookID, UserID: row.UserID, Value: row.RatingValue, CreatedAt: timestamp(row.CreatedAt), UpdatedAt: timestamp(row.UpdatedAt)}
}

func int2(v pgtype.Int2) *int16 {
	if !v.Valid {
		return nil
	}
	return &v.Int16
}
