package services

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/repository"
)

const (
	maxReviewTitleRunes = 200
	maxReviewBodyRunes  = 5000
)

// ReviewStore is the repository surface for ratings and reviews.
type ReviewStore interface {
	InTx(ctx context.Context, fn func(*repository.Reviews) error) error
	Summary(ctx context.Context, bookID uuid.UUID) (domain.RatingSummary, error)
	Distribution(ctx context.Context, bookID uuid.UUID) (map[string]int32, error)
	MyRatings(ctx context.Context, userID uuid.UUID, page domain.Page) ([]domain.Rating, int64, error)
	BookReviews(ctx context.Context, bookID uuid.UUID, sort string, page domain.Page) ([]domain.Review, int64, error)
	Review(ctx context.Context, reviewID uuid.UUID) (domain.Review, error)
	MyReviews(ctx context.Context, userID uuid.UUID, page domain.Page) ([]domain.Review, int64, error)
}

type Reviews struct{ store ReviewStore }

func NewReviews(store ReviewStore) *Reviews { return &Reviews{store: store} }

// ReviewInput is a validated review body and optional title.
type ReviewInput struct {
	Title *string
	Body  string
}

// inTx retries once on a serialization failure or deadlock.
func (s *Reviews) inTx(ctx context.Context, fn func(*repository.Reviews) error) error {
	var err error
	for attempt := 0; attempt < 2; attempt++ {
		err = s.store.InTx(ctx, fn)
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || (pgErr.Code != "40001" && pgErr.Code != "40P01") {
			return err
		}
	}
	return err
}

// Ratings is GET /api/books/{bookId}/ratings.
func (s *Reviews) Ratings(ctx context.Context, bookID uuid.UUID) (domain.RatingSummaryResponse, error) {
	summary, err := s.summary(ctx, bookID)
	if err != nil {
		return domain.RatingSummaryResponse{}, err
	}
	distribution, err := s.store.Distribution(ctx, bookID)
	if err != nil {
		return domain.RatingSummaryResponse{}, err
	}
	return domain.RatingSummaryResponse{AverageRating: summary.AverageRating, RatingCount: summary.RatingCount, Distribution: distribution}, nil
}

// Rate upserts the caller's 1-5 rating; the summary trigger runs in the same
// transaction and the response carries the new summary.
func (s *Reviews) Rate(ctx context.Context, userID, bookID uuid.UUID, value *int) (domain.Rating, error) {
	v, err := ratingValue("value", value, true)
	if err != nil {
		return domain.Rating{}, err
	}
	var result domain.Rating
	err = s.inTx(ctx, func(tx *repository.Reviews) error {
		result, err = tx.UpsertRating(ctx, userID, bookID, *v)
		if err != nil {
			return err
		}
		summary, err := tx.Summary(ctx, bookID)
		result.Summary = &summary
		return err
	})
	return result, bookMissing(err)
}

func (s *Reviews) DeleteRating(ctx context.Context, userID, bookID uuid.UUID) error {
	return s.inTx(ctx, func(tx *repository.Reviews) error {
		n, err := tx.DeleteRating(ctx, userID, bookID)
		if err == nil && n == 0 {
			return domain.NotFound("You have not rated this book.")
		}
		return err
	})
}

func (s *Reviews) MyRatings(ctx context.Context, userID uuid.UUID, page domain.Page) ([]domain.Rating, int64, error) {
	items, total, err := s.store.MyRatings(ctx, userID, page)
	if err == nil && len(items) == 0 && page.Number > 1 {
		_, total, err = s.store.MyRatings(ctx, userID, domain.Page{Number: 1, Size: 1})
	}
	return items, total, err
}

func (s *Reviews) BookReviews(ctx context.Context, bookID uuid.UUID, sort string, page domain.Page) ([]domain.Review, int64, error) {
	if sort == "" {
		sort = "newest"
	}
	if sort != "newest" && sort != "rating" {
		return nil, 0, domain.Validation("sort", "sort must be newest or rating.")
	}
	if _, err := s.summary(ctx, bookID); err != nil {
		return nil, 0, err
	}
	items, total, err := s.store.BookReviews(ctx, bookID, sort, page)
	if err == nil && len(items) == 0 && page.Number > 1 {
		_, total, err = s.store.BookReviews(ctx, bookID, sort, domain.Page{Number: 1, Size: 1})
	}
	return items, total, err
}

// CreateReview adds the caller's one review of the book; a second is 409.
func (s *Reviews) CreateReview(ctx context.Context, userID, bookID uuid.UUID, in ReviewInput) (domain.Review, error) {
	in, err := validateReview(in)
	if err != nil {
		return domain.Review{}, err
	}
	var id uuid.UUID
	err = s.inTx(ctx, func(tx *repository.Reviews) error {
		id, err = tx.CreateReview(ctx, userID, bookID, in.Title, in.Body)
		return err
	})
	if err := reviewErrors(bookMissing(err)); err != nil {
		return domain.Review{}, err
	}
	return s.store.Review(ctx, id)
}

// SaveReview is the combined PUT: rating upsert (when given), review upsert,
// and summary read in one transaction. A failure in any step leaves the
// rating, review, and summary unchanged.
func (s *Reviews) SaveReview(ctx context.Context, userID, bookID uuid.UUID, rating *int, in ReviewInput) (domain.ReviewRatingResult, error) {
	value, err := ratingValue("rating", rating, false)
	if err != nil {
		return domain.ReviewRatingResult{}, err
	}
	in, err = validateReview(in)
	if err != nil {
		return domain.ReviewRatingResult{}, err
	}
	var result domain.ReviewRatingResult
	var reviewID uuid.UUID
	err = s.inTx(ctx, func(tx *repository.Reviews) error {
		result = domain.ReviewRatingResult{}
		if value != nil {
			r, err := tx.UpsertRating(ctx, userID, bookID, *value)
			if err != nil {
				return err
			}
			result.Rating = &r
		} else if r, err := tx.MyRating(ctx, userID, bookID); err == nil {
			result.Rating = &r
		} else if !errors.Is(err, repository.ErrNoRows) {
			return err
		}
		if reviewID, err = tx.UpsertReview(ctx, userID, bookID, in.Title, in.Body); err != nil {
			return err
		}
		if result.Review, err = tx.Review(ctx, reviewID); err != nil {
			return err
		}
		result.Summary, err = tx.Summary(ctx, bookID)
		return err
	})
	if err := reviewErrors(bookMissing(err)); err != nil {
		return domain.ReviewRatingResult{}, err
	}
	if result.Rating != nil {
		result.Rating.Summary = &result.Summary
	}
	return result, nil
}

// UpdateReview edits the caller's own review; anyone else's is a 404. An
// omitted title keeps the current one; a blank title clears it.
func (s *Reviews) UpdateReview(ctx context.Context, userID, reviewID uuid.UUID, in ReviewInput) (domain.Review, error) {
	setTitle := in.Title != nil
	in, err := validateReview(in)
	if err != nil {
		return domain.Review{}, err
	}
	err = s.inTx(ctx, func(tx *repository.Reviews) error {
		return tx.UpdateReview(ctx, userID, reviewID, setTitle, in.Title, in.Body)
	})
	if errors.Is(err, repository.ErrNoRows) {
		return domain.Review{}, domain.NotFound("Review not found.")
	}
	if err != nil {
		return domain.Review{}, err
	}
	return s.store.Review(ctx, reviewID)
}

func (s *Reviews) DeleteReview(ctx context.Context, userID, reviewID uuid.UUID) error {
	return s.inTx(ctx, func(tx *repository.Reviews) error {
		n, err := tx.DeleteReview(ctx, userID, reviewID)
		if err == nil && n == 0 {
			return domain.NotFound("Review not found.")
		}
		return err
	})
}

func (s *Reviews) MyReviews(ctx context.Context, userID uuid.UUID, page domain.Page) ([]domain.Review, int64, error) {
	items, total, err := s.store.MyReviews(ctx, userID, page)
	if err == nil && len(items) == 0 && page.Number > 1 {
		_, total, err = s.store.MyReviews(ctx, userID, domain.Page{Number: 1, Size: 1})
	}
	return items, total, err
}

func (s *Reviews) summary(ctx context.Context, bookID uuid.UUID) (domain.RatingSummary, error) {
	summary, err := s.store.Summary(ctx, bookID)
	if errors.Is(err, repository.ErrNoRows) {
		return domain.RatingSummary{}, domain.NotFound("Book not found.")
	}
	return summary, err
}

// ratingValue checks a 1-5 rating; field is the JSON name in the request body.
func ratingValue(field string, value *int, required bool) (*int16, error) {
	if value == nil {
		if required {
			return nil, domain.Validation(field, field+" is required (1 to 5).")
		}
		return nil, nil
	}
	if *value < 1 || *value > 5 {
		return nil, domain.Validation(field, "Rating must be a whole number from 1 to 5.")
	}
	v := int16(*value)
	return &v, nil
}

func validateReview(in ReviewInput) (ReviewInput, error) {
	body, err := validateText("body", "Review", in.Body, 1, maxReviewBodyRunes, true)
	if err != nil {
		return in, err
	}
	in.Body = body
	if in.Title != nil {
		title, err := validateText("title", "Review title", *in.Title, 0, maxReviewTitleRunes, false)
		if err != nil {
			return in, err
		}
		if title == "" {
			in.Title = nil
		} else {
			in.Title = &title
		}
	}
	return in, nil
}

// bookMissing: a rating or review for an unknown book violates the book
// foreign key (23503).
func bookMissing(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" && strings.Contains(pgErr.ConstraintName, "book_fk") {
		return domain.NotFound("Book not found.")
	}
	return err
}

func reviewErrors(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "reviews_user_book_unique" {
		return domain.Conflict("You already reviewed this book. Edit your review instead.", map[string]any{"constraint": pgErr.ConstraintName})
	}
	return err
}
