// Package database opens the pgx pool and resolves database credentials.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Conservative limits for a small RDS instance (BACKEND_PLAN §6.1).
const (
	maxConns       = 10
	minConns       = 2
	connectTimeout = 5 * time.Second
)

// Open creates the pool. It does not fail when the database is down: the
// pool connects lazily and /readyz reports the state.
func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		// pgx error messages can echo the connection string; never wrap it.
		return nil, fmt.Errorf("parse database URL: invalid format")
	}
	poolConfig.MaxConns = maxConns
	poolConfig.MinConns = minConns
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.ConnConfig.ConnectTimeout = connectTimeout
	return pgxpool.NewWithConfig(ctx, poolConfig)
}
