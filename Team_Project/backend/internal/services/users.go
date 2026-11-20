package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"book-metaverse/backend/internal/auth"
	"book-metaverse/backend/internal/domain"
	"book-metaverse/backend/internal/repository"
)

type Users struct{ users UserStore }

func NewUsers(users UserStore) *Users { return &Users{users: users} }

// UpdateInput holds the PATCH /api/users/me fields; nil means "leave as is".
type UpdateInput struct {
	DisplayName   *string
	AvatarURL     *string
	Bio           *string
	FavoriteGenre *string
}

func (s *Users) Get(ctx context.Context, id uuid.UUID) (domain.User, error) {
	user, err := s.users.GetFull(ctx, id)
	if errors.Is(err, repository.ErrNoRows) {
		return domain.User{}, domain.Unauthenticated("Your account no longer exists.")
	}
	return user, err
}

// Update validates and saves the profile in one transaction.
func (s *Users) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (domain.User, error) {
	var err error
	if in.DisplayName != nil {
		value, verr := validateText("displayName", "Display name", *in.DisplayName, 1, maxDisplayNameRunes, false)
		if verr != nil {
			return domain.User{}, verr
		}
		in.DisplayName = &value
	}
	if in.AvatarURL != nil {
		value, verr := validateAvatarURL(*in.AvatarURL)
		if verr != nil {
			return domain.User{}, verr
		}
		in.AvatarURL = &value
	}
	if in.Bio != nil {
		value, verr := validateText("bio", "Bio", *in.Bio, 0, maxBioRunes, true)
		if verr != nil {
			return domain.User{}, verr
		}
		in.Bio = &value
	}
	if in.FavoriteGenre != nil {
		value, verr := validateText("favoriteGenre", "Favorite genre", *in.FavoriteGenre, 0, maxGenreRunes, false)
		if verr != nil {
			return domain.User{}, verr
		}
		in.FavoriteGenre = &value
	}

	err = s.users.InTx(ctx, func(tx *repository.Users) error {
		if in.DisplayName != nil || in.AvatarURL != nil {
			if err := tx.Update(ctx, id, in.DisplayName, in.AvatarURL); err != nil {
				return err
			}
		}
		if in.Bio != nil || in.FavoriteGenre != nil {
			return tx.UpsertProfile(ctx, id, in.Bio, in.FavoriteGenre)
		}
		return nil
	})
	if errors.Is(err, repository.ErrNoRows) {
		return domain.User{}, domain.Unauthenticated("Your account no longer exists.")
	}
	if err != nil {
		return domain.User{}, err
	}
	return s.Get(ctx, id)
}

// ChangePassword verifies the current password. A wrong one is a 400 on the
// currentPassword field, never a 401, so the client keeps the session.
func (s *Users) ChangePassword(ctx context.Context, id uuid.UUID, current, next string) error {
	if err := passwordRules("newPassword", next); err != nil {
		return err
	}
	creds, err := s.users.CredentialsByID(ctx, id)
	if errors.Is(err, repository.ErrNoRows) {
		return domain.Unauthenticated("Your account no longer exists.")
	}
	if err != nil {
		return err
	}
	if !auth.CheckPassword(creds.PasswordHash, current) {
		return domain.Validation("currentPassword", "Current password is incorrect.")
	}
	hash, err := auth.HashPassword(next)
	if err != nil {
		return err
	}
	return s.users.SetPassword(ctx, id, hash)
}

// Delete removes the account and everything it owns.
func (s *Users) Delete(ctx context.Context, id uuid.UUID) error {
	return s.users.Delete(ctx, id)
}
