package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"book-metaverse/backend/internal/auth"
	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/httpx"
)

// SessionCookie is the browser session cookie (BACKEND_PLAN §4.1).
const SessionCookie = "bm_session"

type userKey struct{}

// userHolder lets the outer logger see the user ID that the inner auth
// middleware resolves.
type userHolder struct{ userID string }

type userHolderKey struct{}

func withUserHolder(ctx context.Context, h *userHolder) context.Context {
	return context.WithValue(ctx, userHolderKey{}, h)
}

// UserID returns the authenticated user, or uuid.Nil on public routes.
func UserID(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(userKey{}).(uuid.UUID)
	return id, ok && id != uuid.Nil
}

// WithUserID is used by tests and by the auth middleware.
func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	if h, ok := ctx.Value(userHolderKey{}).(*userHolder); ok {
		h.userID = id.String()
	}
	return context.WithValue(ctx, userKey{}, id)
}

// token reads "Authorization: Bearer <t>" first (tests, CLI), then the cookie.
func token(r *http.Request) string {
	if header := r.Header.Get("Authorization"); header != "" {
		scheme, value, ok := strings.Cut(header, " ")
		if ok && strings.EqualFold(scheme, "Bearer") {
			return strings.TrimSpace(value)
		}
		return ""
	}
	if cookie, err := r.Cookie(SessionCookie); err == nil {
		return cookie.Value
	}
	return ""
}

// Authenticator validates sessions.
type Authenticator struct{ Tokens *auth.Tokens }

// Required rejects anonymous callers with 401.
func (a Authenticator) Required(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := token(r)
		if value == "" {
			httpx.Error(w, r, domain.Unauthenticated("Sign in to continue."))
			return
		}
		userID, _, err := a.Tokens.Parse(value)
		if err != nil {
			message := "Your session is invalid. Sign in again."
			if err == auth.ErrTokenExpired {
				message = "Your session has expired. Sign in again."
			}
			httpx.Error(w, r, domain.Unauthenticated(message))
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), userID)))
	})
}

// Optional attaches the user when a valid session is present and otherwise
// continues anonymously (public reads with a personal context).
func (a Authenticator) Optional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if value := token(r); value != "" {
			if userID, _, err := a.Tokens.Parse(value); err == nil {
				r = r.WithContext(WithUserID(r.Context(), userID))
			}
		}
		next.ServeHTTP(w, r)
	})
}
