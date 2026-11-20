package services

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"book-metaverse/backend/internal/auth"
	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/repository"
)

// UserStore is the repository surface the auth and user services need.
type UserStore interface {
	InTx(ctx context.Context, fn func(*repository.Users) error) error
	CredentialsByEmail(ctx context.Context, email string) (domain.Credentials, error)
	CredentialsByID(ctx context.Context, id uuid.UUID) (domain.Credentials, error)
	Get(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetFull(ctx context.Context, id uuid.UUID) (domain.User, error)
	SetPassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type Auth struct{ users UserStore }

func NewAuth(users UserStore) *Auth { return &Auth{users: users} }

type RegisterInput struct {
	Email       string
	Password    string
	DisplayName string
}

// Register creates the account. The users_create_defaults trigger creates the
// profile and four default shelves in the same transaction.
func (a *Auth) Register(ctx context.Context, in RegisterInput) (domain.User, error) {
	email := NormalizeEmail(in.Email)
	if err := validateEmail(email); err != nil {
		return domain.User{}, err
	}
	if err := passwordRules("password", in.Password); err != nil {
		return domain.User{}, err
	}
	displayName, err := validateText("displayName", "Display name", in.DisplayName, 1, maxDisplayNameRunes, false)
	if err != nil {
		return domain.User{}, err
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return domain.User{}, err
	}

	var user domain.User
	err = a.users.InTx(ctx, func(tx *repository.Users) error {
		user, err = tx.Create(ctx, email, hash, displayName)
		return err
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_unique" {
		return domain.User{}, domain.Conflict("An account with this email already exists.", map[string]any{"field": "email", "constraint": pgErr.ConstraintName})
	}
	return user, err
}

// Login verifies the password. Missing accounts and wrong passwords get the
// same generic message and similar timing.
func (a *Auth) Login(ctx context.Context, email, password string) (domain.User, error) {
	email = NormalizeEmail(email)
	invalid := domain.Unauthenticated("Email or password is incorrect.")
	if email == "" || password == "" || len(password) > auth.MaxPasswordBytes {
		auth.CheckDummy(password)
		return domain.User{}, invalid
	}
	creds, err := a.users.CredentialsByEmail(ctx, email)
	if errors.Is(err, repository.ErrNoRows) {
		auth.CheckDummy(password)
		slog.InfoContext(ctx, "login failed", "reason", "unknown account")
		return domain.User{}, invalid
	}
	if err != nil {
		return domain.User{}, err
	}
	if !auth.CheckPassword(creds.PasswordHash, password) {
		slog.InfoContext(ctx, "login failed", "reason", "wrong password", "user_id", creds.UserID.String())
		return domain.User{}, invalid
	}
	return a.users.Get(ctx, creds.UserID)
}

// Me returns the session user. A valid token for a deleted account is a 401.
func (a *Auth) Me(ctx context.Context, id uuid.UUID) (domain.User, error) {
	user, err := a.users.Get(ctx, id)
	if errors.Is(err, repository.ErrNoRows) {
		return domain.User{}, domain.Unauthenticated("Your account no longer exists.")
	}
	return user, err
}

func passwordRules(field, password string) error {
	switch auth.ValidatePassword(password) {
	case auth.ErrPasswordLength:
		return domain.Validation(field, "Password must be 8 to 72 bytes long.")
	case auth.ErrPasswordControl:
		return domain.Validation(field, "Password must not contain control characters.")
	}
	return nil
}
