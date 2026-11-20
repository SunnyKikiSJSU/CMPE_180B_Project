package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/httpx"
	"book-metaverse/backend/internal/middleware"
	"book-metaverse/backend/internal/services"
)

type AuthService interface {
	Register(ctx context.Context, in services.RegisterInput) (domain.User, error)
	Login(ctx context.Context, email, password string) (domain.User, error)
	Me(ctx context.Context, id uuid.UUID) (domain.User, error)
}

// TokenIssuer is satisfied by *auth.Tokens.
type TokenIssuer interface {
	Issue(userID uuid.UUID, email string) (string, time.Time, error)
}

type Auth struct {
	Service      AuthService
	Tokens       TokenIssuer
	CookieSecure bool
}

type authResponse struct {
	User        domain.User `json:"user"`
	AccessToken string      `json:"accessToken"`
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h Auth) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	user, err := h.Service.Register(r.Context(), services.RegisterInput(req))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	h.startSession(w, r, http.StatusCreated, user)
}

func (h Auth) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.Decode(r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	user, err := h.Service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	h.startSession(w, r, http.StatusOK, user)
}

// Logout clears the cookie. Tokens are stateless, so a copied Bearer token
// stays valid until it expires (60 minutes).
func (h Auth) Logout(w http.ResponseWriter, _ *http.Request) {
	h.clearCookie(w)
	httpx.NoContent(w)
}

func (h Auth) Me(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.UserID(r.Context())
	user, err := h.Service.Me(r.Context(), userID)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]domain.User{"user": user})
}

func (h Auth) startSession(w http.ResponseWriter, r *http.Request, status int, user domain.User) {
	token, expires, err := h.Tokens.Issue(user.ID, user.Email)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookie,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	httpx.JSON(w, status, authResponse{User: user, AccessToken: token})
}

func (h Auth) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookie,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}
