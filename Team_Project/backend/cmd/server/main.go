// Command server runs the Book Metaverse API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"book-metaverse/backend/internal/auth"
	"book-metaverse/backend/internal/config"
	"book-metaverse/backend/internal/database"
	"book-metaverse/backend/internal/handlers"
	"book-metaverse/backend/internal/middleware"
	"book-metaverse/backend/internal/openlibrary"
	"book-metaverse/backend/internal/repository"
	"book-metaverse/backend/internal/router"
	"book-metaverse/backend/internal/services"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("api failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	cfg.LogSummary()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	databaseURL := cfg.DatabaseURL
	if cfg.DatabaseSource == config.SourceSecret {
		secrets, err := database.NewAWSSecrets(ctx, cfg.AWSRegion)
		if err != nil {
			return err
		}
		if databaseURL, err = database.ResolveURL(ctx, cfg, secrets); err != nil {
			return err
		}
		slog.Info("database credentials read from Secrets Manager")
	}

	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	tokens := auth.NewTokens(cfg.JWTSecret)
	users := repository.NewUsers(pool)
	authHandler := handlers.Auth{Service: services.NewAuth(users), Tokens: tokens, CookieSecure: cfg.CookieSecure}
	clientIP := middleware.ClientIP(cfg.TrustProxy)

	server := &http.Server{
		Addr: ":" + strconv.Itoa(cfg.Port),
		Handler: router.New(router.Deps{
			Health:   handlers.Health{DB: pool},
			Auth:     authHandler,
			Users:    handlers.Users{Service: services.NewUsers(users), Auth: authHandler},
			Catalog:  handlers.Catalog{Service: services.NewCatalog(repository.NewCatalog(pool))},
			Reading:  handlers.Reading{Service: services.NewReading(repository.NewReading(pool))},
			Reviews:  handlers.Reviews{Service: services.NewReviews(repository.NewReviews(pool))},
			Reports:  handlers.Reports{Service: services.NewReports(repository.NewReports(pool))},
			External: handlers.External{Service: services.NewExternal(openlibrary.New(cfg.OpenLibraryURL, cfg.OpenLibraryUserAgent), repository.NewCatalog(pool))},
			Authn:    middleware.Authenticator{Tokens: tokens},
			// 10 tries at once, then one every 6 seconds (10 a minute).
			LoginLimit: middleware.NewRateLimiter(10, 6*time.Second, clientIP),
			// 5 sign-ups at once, then one a minute.
			RegisterLimit: middleware.NewRateLimiter(5, time.Minute, clientIP),
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		// Above the 10-second request timeout so handlers can still write their error.
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("api listening", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		slog.Info("shutting down", "reason", "signal")
	}

	// Drain in-flight requests, then close the pool (deferred above).
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	slog.Info("stopped")
	return nil
}
