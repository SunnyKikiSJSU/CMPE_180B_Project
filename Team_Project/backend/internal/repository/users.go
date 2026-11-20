// Package repository maps sqlc rows to domain types. It owns SQL access;
// services own transactions (BACKEND_PLAN §3).
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"book-metaverse/backend/internal/db"
	"book-metaverse/backend/internal/domain"
)

// ErrNoRows is returned when a lookup finds nothing.
var ErrNoRows = errors.New("no rows")

type Users struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewUsers(pool *pgxpool.Pool) *Users {
	return &Users{pool: pool, q: db.New(pool)}
}

// InTx runs fn in one transaction (READ COMMITTED), committing only if fn
// returns nil.
func (u *Users) InTx(ctx context.Context, fn func(*Users) error) error {
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(&Users{pool: u.pool, q: u.q.WithTx(tx)}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (u *Users) Create(ctx context.Context, email, passwordHash, displayName string) (domain.User, error) {
	row, err := u.q.CreateUser(ctx, db.CreateUserParams{Email: email, PasswordHash: passwordHash, DisplayName: displayName})
	if err != nil {
		return domain.User{}, err
	}
	return domain.User{ID: row.UserID, Email: row.Email, DisplayName: row.DisplayName, AvatarURL: text(row.AvatarUrl), CreatedAt: timestamp(row.CreatedAt)}, nil
}

func (u *Users) CredentialsByEmail(ctx context.Context, email string) (domain.Credentials, error) {
	row, err := u.q.GetUserByEmail(ctx, email)
	if err != nil {
		return domain.Credentials{}, noRows(err)
	}
	return domain.Credentials{UserID: row.UserID, Email: row.Email, PasswordHash: row.PasswordHash}, nil
}

func (u *Users) CredentialsByID(ctx context.Context, id uuid.UUID) (domain.Credentials, error) {
	row, err := u.q.GetUserByID(ctx, id)
	if err != nil {
		return domain.Credentials{}, noRows(err)
	}
	return domain.Credentials{UserID: row.UserID, Email: row.Email, PasswordHash: row.PasswordHash}, nil
}

// Get returns the basic user (no profile or stats).
func (u *Users) Get(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := u.q.GetUserByID(ctx, id)
	if err != nil {
		return domain.User{}, noRows(err)
	}
	return domain.User{ID: row.UserID, Email: row.Email, DisplayName: row.DisplayName, AvatarURL: text(row.AvatarUrl), CreatedAt: timestamp(row.CreatedAt)}, nil
}

// GetFull returns the user with profile and reading counts.
func (u *Users) GetFull(ctx context.Context, id uuid.UUID) (domain.User, error) {
	row, err := u.q.GetUserWithProfile(ctx, id)
	if err != nil {
		return domain.User{}, noRows(err)
	}
	stats, err := u.q.GetUserStats(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	profile := &domain.UserProfile{Bio: text(row.Bio), FavoriteGenre: text(row.FavoriteGenre)}
	if row.ProfileUpdatedAt.Valid {
		updated := timestamp(row.ProfileUpdatedAt)
		profile.UpdatedAt = &updated
	}
	return domain.User{
		ID: row.UserID, Email: row.Email, DisplayName: row.DisplayName, AvatarURL: text(row.AvatarUrl),
		CreatedAt: timestamp(row.CreatedAt), Profile: profile,
		Stats: &domain.UserStats{ShelfCount: stats.ShelfCount, BooksCompleted: stats.BooksCompleted, ReviewsWritten: stats.ReviewsWritten, RatingsGiven: stats.RatingsGiven},
	}, nil
}

// Update changes the given fields; nil leaves a field as it is.
func (u *Users) Update(ctx context.Context, id uuid.UUID, displayName, avatarURL *string) error {
	_, err := u.q.UpdateUser(ctx, db.UpdateUserParams{UserID: id, DisplayName: pgText(displayName), AvatarUrl: pgText(avatarURL)})
	return noRows(err)
}

// UpsertProfile changes the given profile fields; nil leaves a field as it is.
func (u *Users) UpsertProfile(ctx context.Context, id uuid.UUID, bio, favoriteGenre *string) error {
	_, err := u.q.UpsertUserProfile(ctx, db.UpsertUserProfileParams{UserID: id, Bio: pgText(bio), FavoriteGenre: pgText(favoriteGenre)})
	return err
}

func (u *Users) SetPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return u.q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{UserID: id, PasswordHash: passwordHash})
}

// Delete removes the user; owned data cascades (migration 000005 lets the
// cascade remove reading history).
func (u *Users) Delete(ctx context.Context, id uuid.UUID) error {
	return u.q.DeleteUser(ctx, id)
}

func noRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoRows
	}
	return err
}

func text(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func pgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func timestamp(t pgtype.Timestamptz) time.Time {
	return t.Time.UTC()
}
