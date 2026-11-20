package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"book-metaverse/backend/internal/httpx"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	if s.status == 0 {
		s.status = status
	}
	s.ResponseWriter.WriteHeader(status)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if s.status == 0 {
		s.status = http.StatusOK
	}
	return s.ResponseWriter.Write(b)
}

// Logger records request ID, method, route template, status, duration, and
// user ID. It never logs bodies, cookies, tokens, or query strings.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		holder := &userHolder{}
		next.ServeHTTP(recorder, r.WithContext(withUserHolder(r.Context(), holder)))

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		attrs := []any{
			"request_id", httpx.RequestID(r.Context()),
			"method", r.Method,
			"route", route,
			"status", recorder.status,
			"duration_ms", time.Since(start).Milliseconds(),
		}
		if holder.userID != "" {
			attrs = append(attrs, "user_id", holder.userID)
		}
		slog.InfoContext(r.Context(), "request", attrs...)
	})
}
