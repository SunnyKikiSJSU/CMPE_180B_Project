package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	Issuer      = "book-metaverse"
	TokenTTL    = 60 * time.Minute
	clockLeeway = 30 * time.Second
)

var (
	ErrTokenExpired = errors.New("token expired")
	ErrTokenInvalid = errors.New("token invalid")
)

// Claims are the session token contents (BACKEND_PLAN §4.2).
type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// Tokens issues and parses HS256 session tokens.
type Tokens struct {
	secret []byte
	now    func() time.Time
}

func NewTokens(secret []byte) *Tokens {
	return &Tokens{secret: secret, now: time.Now}
}

// Issue returns a signed token for the user and its expiry.
func (t *Tokens) Issue(userID uuid.UUID, email string) (string, time.Time, error) {
	now := t.now().UTC()
	expires := now.Add(TokenTTL)
	claims := Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			Issuer:    Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
	return signed, expires, err
}

// Parse validates the signature (HS256 only; "none" and every other
// algorithm are rejected), issuer, expiry, and subject.
func (t *Tokens) Parse(token string) (uuid.UUID, *Claims, error) {
	claims := &Claims{}
	parsed, err := jwt.ParseWithClaims(token, claims,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(clockLeeway),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return uuid.Nil, nil, ErrTokenExpired
		}
		return uuid.Nil, nil, ErrTokenInvalid
	}
	if !parsed.Valid {
		return uuid.Nil, nil, ErrTokenInvalid
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, nil, ErrTokenInvalid
	}
	return userID, claims, nil
}
