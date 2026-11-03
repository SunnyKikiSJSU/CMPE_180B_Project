package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type server struct {
	db *sql.DB
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	port := envOrDefault("PORT", "8080")

	// sql.Open creates a connection pool handle. The pgx driver is registered
	// by the underscore import above, and the first real connection is made by
	// the readiness check.
	db, err := sql.Open("pgx", databaseURL())
	if err != nil {
		slog.Error("configure database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Keep the pool bounded so the small API container cannot exhaust RDS.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	api := &server{db: db}
	mux := http.NewServeMux()

	// "{$}" matches only the root path, so unknown routes return 404 instead
	// of incorrectly receiving the root response.
	mux.HandleFunc("GET /{$}", api.handleRoot)
	mux.HandleFunc("GET /healthz", api.handleHealth)
	mux.HandleFunc("GET /readyz", api.handleReady)

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Convert container stop signals into a context used for graceful shutdown.
	shutdownSignal, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	go func() {
		slog.Info("api listening", "port", port)
		if err := httpServer.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			slog.Error("api stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownSignal.Done()
	slog.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}

func (s *server) handleRoot(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"service": "book-metaverse-api",
		"status":  "running",
	})
}

func (s *server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"service": "book-metaverse-api",
		"status":  "ok",
	})
}

func (s *server) handleReady(w http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 5*time.Second)
	defer cancel()

	// PingContext performs a real database round trip for Kubernetes, Compose,
	// or a load balancer that needs to know whether this instance is ready.
	if err := s.db.PingContext(ctx); err != nil {
		slog.Warn("database readiness check failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"database": "unavailable",
			"status":   "not_ready",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"database": "connected",
		"status":   "ready",
	})
}

func databaseURL() string {
	// DATABASE_URL is useful when one complete connection string is provided.
	if value := strings.TrimSpace(os.Getenv("DATABASE_URL")); value != "" {
		return value
	}

	host := envOrDefault("DB_HOST", "localhost")
	port := envOrDefault("DB_PORT", "5432")
	name := envOrDefault("DB_NAME", "book_metaverse")
	user := envOrDefault("DB_USER", "book_app")
	sslmode := envOrDefault("DB_SSLMODE", "prefer")

	// url.UserPassword escapes reserved characters in the username/password.
	connection := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, os.Getenv("DB_PASSWORD")),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + name,
	}

	query := connection.Query()
	query.Set("sslmode", sslmode)
	connection.RawQuery = query.Encode()

	return connection.String()
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("write response", "error", err)
	}
}
