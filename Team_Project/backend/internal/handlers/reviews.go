package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/httpx"
	"book-metaverse/backend/internal/services"
)

type ReviewService interface {
	Ratings(ctx context.Context, bookID uuid.UUID) (domain.RatingSummaryResponse, error)
	Rate(ctx context.Context, userID, bookID uuid.UUID, value *int) (domain.Rating, error)
	DeleteRating(ctx context.Context, userID, bookID uuid.UUID) error
	MyRatings(ctx context.Context, userID uuid.UUID, page domain.Page) ([]domain.Rating, int64, error)
	BookReviews(ctx context.Context, bookID uuid.UUID, sort string, page domain.Page) ([]domain.Review, int64, error)
	CreateReview(ctx context.Context, userID, bookID uuid.UUID, in services.ReviewInput) (domain.Review, error)
	SaveReview(ctx context.Context, userID, bookID uuid.UUID, rating *int, in services.ReviewInput) (domain.ReviewRatingResult, error)
	UpdateReview(ctx context.Context, userID, reviewID uuid.UUID, in services.ReviewInput) (domain.Review, error)
	DeleteReview(ctx context.Context, userID, reviewID uuid.UUID) error
	MyReviews(ctx context.Context, userID uuid.UUID, page domain.Page) ([]domain.Review, int64, error)
}

type Reviews struct{ Service ReviewService }

type ratingRequest struct {
	Value *int `json:"value"`
}

type reviewRequest struct {
	Title *string `json:"title"`
	Body  string  `json:"body"`
}

type reviewUpsertRequest struct {
	Rating *int    `json:"rating"`
	Title  *string `json:"title"`
	Body   string  `json:"body"`
}

func (h Reviews) Ratings(w http.ResponseWriter, r *http.Request) {
	bookID, err := pathID(r, "bookId", "Book")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	summary, err := h.Service.Ratings(r.Context(), bookID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, summary)
}

func (h Reviews) Rate(w http.ResponseWriter, r *http.Request) {
	bookID, err := pathID(r, "bookId", "Book")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	var req ratingRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	rating, err := h.Service.Rate(r.Context(), caller(r), bookID, req.Value)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, rating)
}

func (h Reviews) DeleteRating(w http.ResponseWriter, r *http.Request) {
	bookID, err := pathID(r, "bookId", "Book")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.Service.DeleteRating(r.Context(), caller(r), bookID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (h Reviews) BookReviews(w http.ResponseWriter, r *http.Request) {
	bookID, err := pathID(r, "bookId", "Book")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	page, err := pageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	reviews, total, err := h.Service.BookReviews(r.Context(), bookID, r.URL.Query().Get("sort"), page)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Paginate(w, total, page.Number, page.Size)
	httpx.JSON(w, http.StatusOK, reviews)
}

func (h Reviews) CreateReview(w http.ResponseWriter, r *http.Request) {
	bookID, err := pathID(r, "bookId", "Book")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	var req reviewRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	review, err := h.Service.CreateReview(r.Context(), caller(r), bookID, services.ReviewInput(req))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, review)
}

// SaveReview is PUT /api/books/{bookId}/review (review + rating in one transaction).
func (h Reviews) SaveReview(w http.ResponseWriter, r *http.Request) {
	bookID, err := pathID(r, "bookId", "Book")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	var req reviewUpsertRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	result, err := h.Service.SaveReview(r.Context(), caller(r), bookID, req.Rating, services.ReviewInput{Title: req.Title, Body: req.Body})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, result)
}

func (h Reviews) UpdateReview(w http.ResponseWriter, r *http.Request) {
	reviewID, err := pathID(r, "reviewId", "Review")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	var req reviewRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	review, err := h.Service.UpdateReview(r.Context(), caller(r), reviewID, services.ReviewInput(req))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, review)
}

func (h Reviews) DeleteReview(w http.ResponseWriter, r *http.Request) {
	reviewID, err := pathID(r, "reviewId", "Review")
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	if err := h.Service.DeleteReview(r.Context(), caller(r), reviewID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func (h Reviews) MyReviews(w http.ResponseWriter, r *http.Request) {
	page, err := pageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	reviews, total, err := h.Service.MyReviews(r.Context(), caller(r), page)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Paginate(w, total, page.Number, page.Size)
	httpx.JSON(w, http.StatusOK, reviews)
}

func (h Reviews) MyRatings(w http.ResponseWriter, r *http.Request) {
	page, err := pageParams(r)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	ratings, total, err := h.Service.MyRatings(r.Context(), caller(r), page)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.Paginate(w, total, page.Number, page.Size)
	httpx.JSON(w, http.StatusOK, ratings)
}
