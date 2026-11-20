// Package middleware holds the router middleware, in the order of
// BACKEND_PLAN §6.2: request ID, recovery, logging, timeout, body limit, auth.
package middleware

import (
	"net/http"
	"regexp"

	"github.com/google/uuid"

	"book-metaverse/backend/internal/httpx"
)

const RequestIDHeader = "X-Request-ID"

// A client-supplied ID is kept only if it is short and plain, so it is safe
// to put in logs.
var safeRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if !safeRequestID.MatchString(id) {
			id = uuid.NewString()
		}
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(httpx.WithRequestID(r.Context(), id)))
	})
}
