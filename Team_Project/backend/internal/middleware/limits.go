package middleware

import (
	"context"
	"net/http"
	"time"

	"book-metaverse/backend/internal/httpx"
)

// RequestTimeout bounds every request's work (database calls use the context).
const RequestTimeout = 10 * time.Second

func Timeout(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), RequestTimeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// BodyLimit caps request bodies at 1 MiB; httpx.Decode reports the overflow.
func BodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, httpx.MaxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// NoStore marks auth and personal responses as uncacheable.
func NoStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
