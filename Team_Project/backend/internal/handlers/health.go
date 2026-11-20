// Package handlers is the HTTP layer: decode, call a service, write JSON.
// Handlers never contain SQL.
package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"book-metaverse/backend/internal/httpx"
)

const serviceName = "book-metaverse-api"

// Pinger is satisfied by *pgxpool.Pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

type Health struct{ DB Pinger }

func (h Health) Root(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"service": serviceName, "status": "running"})
}

// Live is the liveness probe; it never touches the database.
func (h Health) Live(w http.ResponseWriter, _ *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"service": serviceName, "status": "ok"})
}

// Ready pings PostgreSQL: 200 when reachable, 503 when not.
func (h Health) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	if err := h.DB.Ping(ctx); err != nil {
		slog.WarnContext(r.Context(), "database readiness check failed", "request_id", httpx.RequestID(r.Context()), "error", err)
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"database": "unavailable", "status": "not_ready"})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"database": "connected", "status": "ready"})
}
