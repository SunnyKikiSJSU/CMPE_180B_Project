package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"book-metaverse/backend/internal/httpx"
)

// Recover turns a panic into a 500 envelope and logs the stack with the request ID.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				slog.ErrorContext(r.Context(), "panic", "request_id", httpx.RequestID(r.Context()), "panic", v, "stack", string(debug.Stack()))
				httpx.Error(w, r, nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
