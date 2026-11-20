package handlers

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/httpx"
	"book-metaverse/backend/internal/middleware"
	"book-metaverse/backend/internal/services"
)

type UserService interface {
	Get(ctx context.Context, id uuid.UUID) (domain.User, error)
	Update(ctx context.Context, id uuid.UUID, in services.UpdateInput) (domain.User, error)
	ChangePassword(ctx context.Context, id uuid.UUID, current, next string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type Users struct {
	Service UserService
	// Auth clears the session cookie after the account is deleted.
	Auth Auth
}

// updateRequest: a missing or null field is left unchanged (the UpdateUser and
// UpsertUserProfile queries use coalesce, so a value cannot be cleared to null).
type updateRequest struct {
	DisplayName   *string `json:"displayName"`
	AvatarURL     *string `json:"avatarUrl"`
	Bio           *string `json:"bio"`
	FavoriteGenre *string `json:"favoriteGenre"`
}

type passwordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type deleteRequest struct {
	Confirm *bool `json:"confirm"`
}

func (h Users) Get(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserID(r.Context())
	user, err := h.Service.Get(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h Users) Update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	userID, _ := middleware.UserID(r.Context())
	user, err := h.Service.Update(r.Context(), userID, services.UpdateInput(req))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}

func (h Users) ChangePassword(w http.ResponseWriter, r *http.Request) {
	var req passwordRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.CurrentPassword == "" {
		httpx.Error(w, r, domain.Validation("currentPassword", "Current password is required."))
		return
	}
	userID, _ := middleware.UserID(r.Context())
	if err := h.Service.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

// Delete requires {"confirm": true} so an empty request cannot delete an account.
func (h Users) Delete(w http.ResponseWriter, r *http.Request) {
	var req deleteRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.Confirm == nil || !*req.Confirm {
		httpx.Error(w, r, domain.Validation("confirm", `Send {"confirm": true} to delete the account.`))
		return
	}
	userID, _ := middleware.UserID(r.Context())
	if err := h.Service.Delete(r.Context(), userID); err != nil {
		httpx.Error(w, r, err)
		return
	}
	h.Auth.clearCookie(w)
	httpx.NoContent(w)
}
