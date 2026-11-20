package router

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"book-metaverse/backend/internal/auth"
	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/handlers"
	"book-metaverse/backend/internal/middleware"
	"book-metaverse/backend/internal/services"
)

var testUser = domain.User{ID: uuid.MustParse("11111111-1111-4111-8111-111111111111"), Email: "reader@example.com", DisplayName: "Reader", CreatedAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}

type fakeAuth struct{ registered services.RegisterInput }

func (f *fakeAuth) Register(_ context.Context, in services.RegisterInput) (domain.User, error) {
	f.registered = in
	if in.Email == "taken@example.com" {
		return domain.User{}, domain.Conflict("An account with this email already exists.", map[string]any{"field": "email"})
	}
	return testUser, nil
}

func (f *fakeAuth) Login(_ context.Context, email, password string) (domain.User, error) {
	if email == testUser.Email && password == "right-password" {
		return testUser, nil
	}
	return domain.User{}, domain.Unauthenticated("Email or password is incorrect.")
}

func (f *fakeAuth) Me(_ context.Context, id uuid.UUID) (domain.User, error) {
	if id != testUser.ID {
		return domain.User{}, domain.Unauthenticated("Your account no longer exists.")
	}
	return testUser, nil
}

type fakeUsers struct{ deleted bool }

func (f *fakeUsers) Get(context.Context, uuid.UUID) (domain.User, error) {
	u := testUser
	u.Stats = &domain.UserStats{ShelfCount: 4}
	return u, nil
}

func (f *fakeUsers) Update(_ context.Context, _ uuid.UUID, in services.UpdateInput) (domain.User, error) {
	u := testUser
	if in.DisplayName != nil {
		u.DisplayName = *in.DisplayName
	}
	return u, nil
}

func (f *fakeUsers) ChangePassword(_ context.Context, _ uuid.UUID, current, _ string) error {
	if current != "right-password" {
		return domain.Validation("currentPassword", "Current password is incorrect.")
	}
	return nil
}

func (f *fakeUsers) Delete(context.Context, uuid.UUID) error { f.deleted = true; return nil }

type fakeCatalog struct {
	lastSearch services.SearchInput
	lastUser   *uuid.UUID
}

var testBook = domain.Book{ID: uuid.MustParse("22222222-2222-4222-8222-222222222222"), Title: "A Book", Author: "Unknown author", Authors: []domain.Ref{}, Categories: []domain.Ref{}, Source: "user"}

func (f *fakeCatalog) Search(_ context.Context, in services.SearchInput) ([]domain.Book, int64, error) {
	f.lastSearch = in
	return []domain.Book{testBook}, 41, nil
}

func (f *fakeCatalog) Get(_ context.Context, id uuid.UUID, userID *uuid.UUID) (domain.Book, error) {
	f.lastUser = userID
	if id != testBook.ID {
		return domain.Book{}, domain.NotFound("Book not found.")
	}
	return testBook, nil
}

func (f *fakeCatalog) Create(_ context.Context, in services.BookFields) (domain.Book, error) {
	return testBook, nil
}

func (f *fakeCatalog) Update(_ context.Context, id uuid.UUID, _ services.BookFields) (domain.Book, error) {
	return testBook, nil
}

func (f *fakeCatalog) ListAuthors(context.Context, string, domain.Page) ([]domain.Author, int64, error) {
	return []domain.Author{}, 0, nil
}

func (f *fakeCatalog) Author(_ context.Context, id uuid.UUID, _ domain.Page) (domain.Author, int64, error) {
	return domain.Author{ID: id, Name: "A"}, 0, nil
}

func (f *fakeCatalog) ListCategories(context.Context, string, domain.Page) ([]domain.Category, int64, error) {
	return []domain.Category{}, 0, nil
}

func (f *fakeCatalog) Category(_ context.Context, id uuid.UUID, _ domain.Page) (domain.Category, int64, error) {
	return domain.Category{ID: id, Name: "C"}, 0, nil
}

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

type harness struct {
	handler http.Handler
	tokens  *auth.Tokens
	users   *fakeUsers
	auth    *fakeAuth
	catalog *fakeCatalog
}

func newHarness(dbErr error) harness {
	tokens := auth.NewTokens([]byte("router-test-secret-0123456789abcdef"))
	fa, fu, fc := &fakeAuth{}, &fakeUsers{}, &fakeCatalog{}
	authHandler := handlers.Auth{Service: fa, Tokens: tokens}
	return harness{
		handler: New(Deps{
			Health:  handlers.Health{DB: fakePinger{err: dbErr}},
			Auth:    authHandler,
			Users:   handlers.Users{Service: fu, Auth: authHandler},
			Catalog: handlers.Catalog{Service: fc},
			Authn:   middleware.Authenticator{Tokens: tokens},
		}),
		tokens: tokens, users: fu, auth: fa, catalog: fc,
	}
}

func (h harness) do(method, path, body string, header http.Header) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range header {
		req.Header[k] = v
	}
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

func (h harness) bearer() http.Header {
	token, _, _ := h.tokens.Issue(testUser.ID, testUser.Email)
	return http.Header{"Authorization": {"Bearer " + token}}
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var env struct {
		Error struct {
			Code      string
			RequestID string
			Details   map[string]any
		}
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("not an error envelope: %s", rec.Body.String())
	}
	if env.Error.RequestID == "" || env.Error.RequestID != rec.Header().Get("X-Request-ID") {
		t.Fatalf("request ID missing or different: %s vs %s", env.Error.RequestID, rec.Header().Get("X-Request-ID"))
	}
	return env.Error.Code
}

func TestHealth(t *testing.T) {
	h := newHarness(nil)
	for _, path := range []string{"/", "/healthz", "/readyz"} {
		if rec := h.do("GET", path, "", nil); rec.Code != 200 {
			t.Errorf("%s: %d", path, rec.Code)
		}
	}
	down := newHarness(errors.New("connection refused"))
	if rec := down.do("GET", "/readyz", "", nil); rec.Code != 503 || !strings.Contains(rec.Body.String(), "not_ready") {
		t.Fatalf("readyz with database down: %d %s", rec.Code, rec.Body.String())
	}
	if rec := down.do("GET", "/healthz", "", nil); rec.Code != 200 {
		t.Fatal("liveness must not depend on the database")
	}
}

func TestRegisterSetsSessionCookie(t *testing.T) {
	h := newHarness(nil)
	rec := h.do("POST", "/api/auth/register", `{"email":"reader@example.com","password":"long-enough","displayName":"Reader"}`, nil)
	if rec.Code != 201 {
		t.Fatalf("status %d %s", rec.Code, rec.Body.String())
	}
	cookie := rec.Result().Cookies()[0]
	if cookie.Name != "bm_session" || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || cookie.Secure {
		t.Fatalf("cookie flags: %+v", cookie)
	}
	var body struct {
		User        domain.User
		AccessToken string
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.AccessToken != cookie.Value || body.User.ID != testUser.ID {
		t.Fatalf("body: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Fatal("response mentions a password")
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("auth responses must not be cached")
	}
	if rec := h.do("POST", "/api/auth/register", `{"email":"taken@example.com","password":"long-enough","displayName":"x"}`, nil); rec.Code != 409 || errorCode(t, rec) != "CONFLICT" {
		t.Fatalf("duplicate: %d", rec.Code)
	}
	if rec := h.do("POST", "/api/auth/register", `{"email":"a@b.co","password":"x","displayName":"x","role":"admin"}`, nil); rec.Code != 400 || errorCode(t, rec) != "VALIDATION_ERROR" {
		t.Fatalf("unknown field: %d", rec.Code)
	}
}

func TestLogin(t *testing.T) {
	h := newHarness(nil)
	if rec := h.do("POST", "/api/auth/login", `{"email":"reader@example.com","password":"right-password"}`, nil); rec.Code != 200 || len(rec.Result().Cookies()) != 1 {
		t.Fatalf("login: %d", rec.Code)
	}
	rec := h.do("POST", "/api/auth/login", `{"email":"reader@example.com","password":"nope"}`, nil)
	if rec.Code != 401 || errorCode(t, rec) != "UNAUTHENTICATED" || len(rec.Result().Cookies()) != 0 {
		t.Fatalf("bad login: %d", rec.Code)
	}
}

func TestProtectedRoutesRejectAnonymous(t *testing.T) {
	h := newHarness(nil)
	for _, route := range [][2]string{
		{"POST", "/api/auth/logout"}, {"GET", "/api/auth/me"}, {"GET", "/api/users/me"}, {"PATCH", "/api/users/me"},
		{"DELETE", "/api/users/me"}, {"PUT", "/api/users/me/password"}, {"GET", "/api/shelves"}, {"POST", "/api/books/import"},
		{"GET", "/api/reports/recommendations"},
	} {
		rec := h.do(route[0], route[1], "{}", nil)
		if rec.Code != 401 || errorCode(t, rec) != "UNAUTHENTICATED" {
			t.Errorf("%s %s: %d", route[0], route[1], rec.Code)
		}
	}
	// Invalid and expired tokens are also rejected.
	bad := http.Header{"Authorization": {"Bearer not-a-token"}}
	if rec := h.do("GET", "/api/auth/me", "", bad); rec.Code != 401 {
		t.Fatalf("bad token: %d", rec.Code)
	}
}

func TestSessionByCookieAndBearer(t *testing.T) {
	h := newHarness(nil)
	token, _, _ := h.tokens.Issue(testUser.ID, testUser.Email)
	cookie := http.Header{"Cookie": {"bm_session=" + token}}
	for _, header := range []http.Header{cookie, h.bearer()} {
		rec := h.do("GET", "/api/auth/me", "", header)
		if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"user":{`) {
			t.Fatalf("me: %d %s", rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("personal responses must not be cached")
		}
	}
	rec := h.do("POST", "/api/auth/logout", "", h.bearer())
	if rec.Code != 204 || rec.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("logout must clear the cookie: %d", rec.Code)
	}
}

func TestUsersMe(t *testing.T) {
	h := newHarness(nil)
	if rec := h.do("GET", "/api/users/me", "", h.bearer()); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"shelfCount":4`) {
		t.Fatalf("get: %d %s", rec.Code, rec.Body.String())
	}
	if rec := h.do("PATCH", "/api/users/me", `{"displayName":"New"}`, h.bearer()); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"displayName":"New"`) {
		t.Fatalf("patch: %d", rec.Code)
	}
	rec := h.do("PUT", "/api/users/me/password", `{"currentPassword":"typo","newPassword":"new-password"}`, h.bearer())
	if rec.Code != 400 || errorCode(t, rec) != "VALIDATION_ERROR" || !strings.Contains(rec.Body.String(), `"field":"currentPassword"`) {
		t.Fatalf("wrong current password must be 400, not 401: %d %s", rec.Code, rec.Body.String())
	}
	if rec := h.do("PUT", "/api/users/me/password", `{"currentPassword":"right-password","newPassword":"new-password"}`, h.bearer()); rec.Code != 204 {
		t.Fatalf("change: %d", rec.Code)
	}
	for _, body := range []string{"", `{}`, `{"confirm":false}`} {
		if rec := h.do("DELETE", "/api/users/me", body, h.bearer()); rec.Code != 400 || h.users.deleted {
			t.Fatalf("delete without confirm (%q): %d", body, rec.Code)
		}
	}
	if rec := h.do("DELETE", "/api/users/me", `{"confirm":true}`, h.bearer()); rec.Code != 204 || !h.users.deleted || rec.Result().Cookies()[0].MaxAge >= 0 {
		t.Fatalf("delete: %d", rec.Code)
	}
}

func TestSkeletonAndUnknownRoutes(t *testing.T) {
	h := newHarness(nil)
	if rec := h.do("GET", "/api/external/books", "", nil); rec.Code != 501 || errorCode(t, rec) != "NOT_IMPLEMENTED" {
		t.Fatalf("public stub: %d", rec.Code)
	}
	if rec := h.do("GET", "/api/shelves", "", h.bearer()); rec.Code != 501 {
		t.Fatalf("protected stub: %d", rec.Code)
	}
	for _, route := range [][2]string{{"GET", "/api/admin"}, {"GET", "/debug/pprof"}, {"DELETE", "/api/books"}} {
		if rec := h.do(route[0], route[1], "", nil); rec.Code != 404 || errorCode(t, rec) != "NOT_FOUND" {
			t.Errorf("%s %s: %d", route[0], route[1], rec.Code)
		}
	}
}

func TestRequestIDPropagation(t *testing.T) {
	h := newHarness(nil)
	rec := h.do("GET", "/healthz", "", http.Header{"X-Request-Id": {"client-req-0001"}})
	if rec.Header().Get("X-Request-ID") != "client-req-0001" {
		t.Fatalf("kept: %s", rec.Header().Get("X-Request-ID"))
	}
	rec = h.do("GET", "/healthz", "", http.Header{"X-Request-Id": {"bad id\n<script>"}})
	if id := rec.Header().Get("X-Request-ID"); id == "" || strings.ContainsAny(id, " <\n") {
		t.Fatalf("unsafe ID not replaced: %q", id)
	}
}

func TestCatalogRoutes(t *testing.T) {
	h := newHarness(nil)

	rec := h.do("GET", "/api/books?q=%20night%20&sort=title&page=3&pageSize=10&authorId=11111111-1111-4111-8111-111111111111", "", nil)
	if rec.Code != 200 || rec.Header().Get("X-Total-Count") != "41" || rec.Header().Get("X-Page") != "3" || rec.Header().Get("X-Page-Size") != "10" {
		t.Fatalf("search: %d %v", rec.Code, rec.Header())
	}
	if !strings.HasPrefix(rec.Body.String(), "[") {
		t.Fatalf("list must be a raw array: %s", rec.Body.String())
	}
	if in := h.catalog.lastSearch; in.Q != " night " || in.Sort != "title" || in.Page.Number != 3 || in.Page.Size != 10 || in.AuthorID == nil {
		t.Fatalf("search input: %+v", in)
	}
	for _, query := range []string{"page=0", "page=x", "pageSize=101", "pageSize=0", "authorId=nope", "categoryId=1"} {
		if rec := h.do("GET", "/api/books?"+query, "", nil); rec.Code != 400 || errorCode(t, rec) != "VALIDATION_ERROR" {
			t.Errorf("%s: %d", query, rec.Code)
		}
	}

	// Detail: anonymous gets no personal context; a session adds it and no-store.
	if rec := h.do("GET", "/api/books/"+testBook.ID.String(), "", nil); rec.Code != 200 || h.catalog.lastUser != nil || rec.Header().Get("Cache-Control") == "no-store" {
		t.Fatalf("anonymous detail: %d %v", rec.Code, h.catalog.lastUser)
	}
	if rec := h.do("GET", "/api/books/"+testBook.ID.String(), "", h.bearer()); rec.Code != 200 || h.catalog.lastUser == nil || rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("signed-in detail: %d", rec.Code)
	}
	if rec := h.do("GET", "/api/books/not-a-uuid", "", nil); rec.Code != 404 || errorCode(t, rec) != "NOT_FOUND" {
		t.Fatalf("malformed id: %d", rec.Code)
	}
	if rec := h.do("GET", "/api/books/"+uuid.NewString(), "", nil); rec.Code != 404 {
		t.Fatalf("unknown id: %d", rec.Code)
	}

	// Writes need a session; PATCH accepts only BookUpdate fields.
	if rec := h.do("POST", "/api/books", `{"title":"x"}`, nil); rec.Code != 401 {
		t.Fatalf("anonymous create: %d", rec.Code)
	}
	if rec := h.do("POST", "/api/books", `{"title":"x","authors":["A"]}`, h.bearer()); rec.Code != 201 {
		t.Fatalf("create: %d %s", rec.Code, rec.Body.String())
	}
	if rec := h.do("PATCH", "/api/books/"+testBook.ID.String(), `{"authors":["A"]}`, h.bearer()); rec.Code != 400 || !strings.Contains(rec.Body.String(), `"field":"authors"`) {
		t.Fatalf("PATCH authors: %d %s", rec.Code, rec.Body.String())
	}

	// Detail responses always carry "books" as an array.
	for _, path := range []string{"/api/authors/" + uuid.NewString(), "/api/categories/" + uuid.NewString()} {
		if rec := h.do("GET", path, "", nil); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"books":[]`) {
			t.Errorf("%s: %d %s", path, rec.Code, rec.Body.String())
		}
	}
	if rec := h.do("GET", "/api/authors?pageSize=500", "", nil); rec.Code != 400 {
		t.Fatalf("authors pageSize: %d", rec.Code)
	}
}
