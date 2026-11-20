//go:build integration

// Package integration runs API flows against the Compose PostgreSQL started by
// ./scripts/dev.sh local (migrations and seeds applied). Run with:
//
//	DATABASE_URL=postgres://book_app:...@localhost:5432/book_metaverse?sslmode=disable \
//	  go test -tags integration ./internal/integration/
//
// Each test creates its own user and deletes it afterwards.
package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"book-metaverse/backend/internal/auth"
	"book-metaverse/backend/internal/database"
	"book-metaverse/backend/internal/handlers"
	"book-metaverse/backend/internal/middleware"
	"book-metaverse/backend/internal/repository"
	"book-metaverse/backend/internal/router"
	"book-metaverse/backend/internal/services"
)

func setup(t *testing.T) (http.Handler, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := database.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	tokens := auth.NewTokens([]byte("integration-test-secret-0123456789"))
	users := repository.NewUsers(pool)
	authHandler := handlers.Auth{Service: services.NewAuth(users), Tokens: tokens}
	return router.New(router.Deps{
		Health: handlers.Health{DB: pool},
		Auth:   authHandler,
		Users:  handlers.Users{Service: services.NewUsers(users), Auth: authHandler},
		Authn:  middleware.Authenticator{Tokens: tokens},
	}), pool
}

func call(h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRegisterCreatesDefaultsAndDeleteCascades(t *testing.T) {
	h, pool := setup(t)
	ctx := context.Background()
	email := "it-" + uuid.NewString()[:8] + "@example.com"

	rec := call(h, "POST", "/api/auth/register", `{"email":"  `+strings.ToUpper(email)+` ","password":"integration-pass","displayName":"  IT User "}`, "")
	if rec.Code != 201 {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	var reg struct {
		User struct {
			ID          uuid.UUID
			Email       string
			DisplayName string
		}
		AccessToken string
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &reg)
	userID := reg.User.ID
	t.Cleanup(func() { _, _ = pool.Exec(ctx, "DELETE FROM users WHERE user_id = $1", userID) })
	if reg.User.Email != email || reg.User.DisplayName != "IT User" {
		t.Fatalf("normalization: %+v", reg.User)
	}

	// The users_create_defaults trigger made the profile and four status shelves.
	var shelves, profiles int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM shelves WHERE user_id = $1", userID).Scan(&shelves)
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM user_profiles WHERE user_id = $1", userID).Scan(&profiles)
	if shelves != 4 || profiles != 1 {
		t.Fatalf("defaults: %d shelves, %d profiles", shelves, profiles)
	}

	// The stored hash is bcrypt cost 12, never the plain password.
	var hash string
	_ = pool.QueryRow(ctx, "SELECT password_hash FROM users WHERE user_id = $1", userID).Scan(&hash)
	if !strings.HasPrefix(hash, "$2a$12$") {
		t.Fatalf("hash: %s", hash[:7])
	}

	if rec := call(h, "POST", "/api/auth/register", `{"email":"`+email+`","password":"integration-pass","displayName":"Again"}`, ""); rec.Code != 409 || !strings.Contains(rec.Body.String(), "users_email_unique") {
		t.Fatalf("duplicate: %d %s", rec.Code, rec.Body.String())
	}

	if rec := call(h, "GET", "/api/users/me", "", reg.AccessToken); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"shelfCount":4`) {
		t.Fatalf("users/me: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(h, "PATCH", "/api/users/me", `{"bio":"Reads at night.\nMostly fantasy.","favoriteGenre":"Fantasy","avatarUrl":"https://example.com/a.png"}`, reg.AccessToken); rec.Code != 200 ||
		!strings.Contains(rec.Body.String(), `"favoriteGenre":"Fantasy"`) || !strings.Contains(rec.Body.String(), `"avatarUrl":"https://example.com/a.png"`) {
		t.Fatalf("patch: %d %s", rec.Code, rec.Body.String())
	}

	// Password change, then log in with the new password only.
	if rec := call(h, "PUT", "/api/users/me/password", `{"currentPassword":"wrong-pass","newPassword":"new-integration-pass"}`, reg.AccessToken); rec.Code != 400 {
		t.Fatalf("wrong current: %d", rec.Code)
	}
	if rec := call(h, "PUT", "/api/users/me/password", `{"currentPassword":"integration-pass","newPassword":"new-integration-pass"}`, reg.AccessToken); rec.Code != 204 {
		t.Fatalf("change: %d %s", rec.Code, rec.Body.String())
	}
	if rec := call(h, "POST", "/api/auth/login", `{"email":"`+email+`","password":"integration-pass"}`, ""); rec.Code != 401 {
		t.Fatalf("old password still works: %d", rec.Code)
	}
	if rec := call(h, "POST", "/api/auth/login", `{"email":"`+email+`","password":"new-integration-pass"}`, ""); rec.Code != 200 {
		t.Fatalf("new password: %d", rec.Code)
	}

	// Delete cascades owned rows; the old token then gets 401.
	if rec := call(h, "DELETE", "/api/users/me", `{"confirm":true}`, reg.AccessToken); rec.Code != 204 {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM shelves WHERE user_id = $1", userID).Scan(&shelves)
	if shelves != 0 {
		t.Fatalf("shelves left after delete: %d", shelves)
	}
	if rec := call(h, "GET", "/api/auth/me", "", reg.AccessToken); rec.Code != 401 {
		t.Fatalf("me after delete: %d", rec.Code)
	}
}

// Seeded demo accounts were hashed by pgcrypto (crypt + gen_salt('bf', 12)).
func TestSeededDemoAccountLogsIn(t *testing.T) {
	h, _ := setup(t)
	if rec := call(h, "POST", "/api/auth/login", `{"email":"Reader1@Example.com","password":"Password123!"}`, ""); rec.Code != 200 {
		t.Fatalf("seeded login: %d %s (run the seeds first)", rec.Code, rec.Body.String())
	}
	if rec := call(h, "POST", "/api/auth/login", `{"email":"nobody-`+uuid.NewString()[:6]+`@example.com","password":"Password123!"}`, ""); rec.Code != 401 {
		t.Fatalf("unknown account: %d", rec.Code)
	}
}

func TestReady(t *testing.T) {
	h, _ := setup(t)
	if rec := call(h, "GET", "/readyz", "", ""); rec.Code != 200 {
		t.Fatalf("readyz: %d", rec.Code)
	}
}
