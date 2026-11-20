// Package router declares every route of docs/api/openapi.yaml, its auth
// group, and the middleware order (BACKEND_PLAN §6.2). Routes that later
// phases implement answer 501 NOT_IMPLEMENTED; anything else is 404.
package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/handlers"
	"book-metaverse/backend/internal/httpx"
	"book-metaverse/backend/internal/middleware"
)

type Deps struct {
	Health handlers.Health
	Auth   handlers.Auth
	Users  handlers.Users
	Authn  middleware.Authenticator
}

func New(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recover, middleware.Logger, middleware.Timeout, middleware.BodyLimit)

	notFound := func(w http.ResponseWriter, req *http.Request) {
		httpx.Error(w, req, domain.NotFound("No such endpoint."))
	}
	r.NotFound(notFound)
	r.MethodNotAllowed(notFound)

	r.Get("/", d.Health.Root)
	r.Get("/healthz", d.Health.Live)
	r.Get("/readyz", d.Health.Ready)

	todo := httpx.NotImplemented

	r.Route("/api", func(api chi.Router) {
		// Public routes; a valid session adds the caller's context where a
		// later phase uses it (for example myProgress on a book).
		api.Group(func(pub chi.Router) {
			pub.Use(d.Authn.Optional)
			pub.Get("/books", todo)
			pub.Get("/external/books", todo)
			pub.Get("/books/{bookId}", todo)
			pub.Get("/books/{bookId}/ratings", todo)
			pub.Get("/books/{bookId}/reviews", todo)
			pub.Get("/authors", todo)
			pub.Get("/authors/{authorId}", todo)
			pub.Get("/categories", todo)
			pub.Get("/categories/{categoryId}", todo)
			pub.Get("/reports/popular", todo)
			pub.Get("/reports/top-rated", todo)
			pub.Get("/reports/authors", todo)
			pub.Get("/reports/categories", todo)
			pub.Get("/reports/review-activity", todo)
			pub.Get("/reports/queries", todo)
		})

		api.Group(func(open chi.Router) {
			open.Use(middleware.NoStore)
			open.Post("/auth/register", d.Auth.Register)
			open.Post("/auth/login", d.Auth.Login)
		})

		// Everything below needs a session; personal data is never cached.
		api.Group(func(priv chi.Router) {
			priv.Use(middleware.NoStore, d.Authn.Required)

			priv.Post("/auth/logout", d.Auth.Logout)
			priv.Get("/auth/me", d.Auth.Me)
			priv.Get("/users/me", d.Users.Get)
			priv.Patch("/users/me", d.Users.Update)
			priv.Delete("/users/me", d.Users.Delete)
			priv.Put("/users/me/password", d.Users.ChangePassword)

			priv.Get("/users/me/reviews", todo)
			priv.Get("/users/me/ratings", todo)
			priv.Post("/books", todo)
			priv.Post("/books/import", todo)
			priv.Patch("/books/{bookId}", todo)
			priv.Post("/books/{bookId}/ratings", todo)
			priv.Delete("/books/{bookId}/ratings", todo)
			priv.Post("/books/{bookId}/reviews", todo)
			priv.Put("/books/{bookId}/review", todo)
			priv.Patch("/reviews/{reviewId}", todo)
			priv.Delete("/reviews/{reviewId}", todo)
			priv.Get("/shelves", todo)
			priv.Post("/shelves", todo)
			priv.Patch("/shelves/{shelfId}", todo)
			priv.Delete("/shelves/{shelfId}", todo)
			priv.Get("/shelves/{shelfId}/books", todo)
			priv.Post("/shelves/{shelfId}/books", todo)
			priv.Patch("/shelves/{shelfId}/books/{bookId}", todo)
			priv.Delete("/shelves/{shelfId}/books/{bookId}", todo)
			priv.Get("/reading-progress", todo)
			priv.Get("/reading-progress/{bookId}", todo)
			priv.Patch("/reading-progress/{bookId}", todo)
			priv.Delete("/reading-progress/{bookId}", todo)
			priv.Get("/reading-history", todo)
			priv.Get("/reports/reading-history", todo)
			priv.Get("/reports/user-stats", todo)
			priv.Get("/reports/recommendations", todo)
		})
	})
	return r
}
