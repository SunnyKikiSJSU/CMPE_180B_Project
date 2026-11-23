// Package router declares every route of docs/api/openapi.yaml, its auth
// group, and the middleware order (BACKEND_PLAN §6.2). Anything else is 404.
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
	Health   handlers.Health
	Auth     handlers.Auth
	Users    handlers.Users
	Catalog  handlers.Catalog
	Reading  handlers.Reading
	Reviews  handlers.Reviews
	Reports  handlers.Reports
	External handlers.External
	Authn    middleware.Authenticator
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

	r.Route("/api", func(api chi.Router) {
		// Public routes; a valid session adds the caller's context where a
		// later phase uses it (for example myProgress on a book).
		api.Group(func(pub chi.Router) {
			pub.Use(d.Authn.Optional)
			pub.Get("/books", d.Catalog.SearchBooks)
			pub.Get("/external/books", d.External.Search)
			pub.Get("/books/{bookId}", d.Catalog.GetBook)
			pub.Get("/books/{bookId}/ratings", d.Reviews.Ratings)
			pub.Get("/books/{bookId}/reviews", d.Reviews.BookReviews)
			pub.Get("/authors", d.Catalog.ListAuthors)
			pub.Get("/authors/{authorId}", d.Catalog.GetAuthor)
			pub.Get("/categories", d.Catalog.ListCategories)
			pub.Get("/categories/{categoryId}", d.Catalog.GetCategory)
			pub.Get("/reports/popular", d.Reports.Popular)
			pub.Get("/reports/top-rated", d.Reports.TopRated)
			pub.Get("/reports/authors", d.Reports.Authors)
			pub.Get("/reports/categories", d.Reports.Categories)
			pub.Get("/reports/review-activity", d.Reports.ReviewActivity)
			pub.Get("/reports/queries", d.Reports.Queries)
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

			priv.Get("/users/me/reviews", d.Reviews.MyReviews)
			priv.Get("/users/me/ratings", d.Reviews.MyRatings)
			priv.Post("/books", d.Catalog.CreateBook)
			priv.Post("/books/import", d.External.Import)
			priv.Patch("/books/{bookId}", d.Catalog.UpdateBook)
			priv.Post("/books/{bookId}/ratings", d.Reviews.Rate)
			priv.Delete("/books/{bookId}/ratings", d.Reviews.DeleteRating)
			priv.Post("/books/{bookId}/reviews", d.Reviews.CreateReview)
			priv.Put("/books/{bookId}/review", d.Reviews.SaveReview)
			priv.Patch("/reviews/{reviewId}", d.Reviews.UpdateReview)
			priv.Delete("/reviews/{reviewId}", d.Reviews.DeleteReview)
			priv.Get("/shelves", d.Reading.ListShelves)
			priv.Post("/shelves", d.Reading.CreateShelf)
			priv.Patch("/shelves/{shelfId}", d.Reading.RenameShelf)
			priv.Delete("/shelves/{shelfId}", d.Reading.DeleteShelf)
			priv.Get("/shelves/{shelfId}/books", d.Reading.ShelfBooks)
			priv.Post("/shelves/{shelfId}/books", d.Reading.AddBook)
			priv.Patch("/shelves/{shelfId}/books/{bookId}", d.Reading.MoveBook)
			priv.Delete("/shelves/{shelfId}/books/{bookId}", d.Reading.RemoveBook)
			priv.Get("/reading-progress", d.Reading.ListProgress)
			priv.Get("/reading-progress/{bookId}", d.Reading.GetProgress)
			priv.Patch("/reading-progress/{bookId}", d.Reading.UpdateProgress)
			priv.Delete("/reading-progress/{bookId}", d.Reading.DeleteProgress)
			priv.Get("/reading-history", d.Reading.History)
			priv.Get("/reports/reading-history", d.Reports.ReadingHistory)
			priv.Get("/reports/user-stats", d.Reports.UserStats)
			priv.Get("/reports/recommendations", d.Reports.Recommendations)
		})
	})
	return r
}
