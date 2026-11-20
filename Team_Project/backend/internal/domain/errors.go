// Package domain holds API-facing types and the errors services return.
package domain

import (
	"errors"
	"fmt"
)

// Kind is the API error category. Each kind maps to one HTTP status and one
// envelope code in httpx (docs/api/errors.md).
type Kind string

const (
	KindValidation      Kind = "VALIDATION_ERROR"
	KindUnauthenticated Kind = "UNAUTHENTICATED"
	KindForbidden       Kind = "FORBIDDEN"
	KindNotFound        Kind = "NOT_FOUND"
	KindConflict        Kind = "CONFLICT"
	KindStaleVersion    Kind = "STALE_VERSION"
	KindUpstream        Kind = "UPSTREAM_UNAVAILABLE"
	KindDatabase        Kind = "DATABASE_UNAVAILABLE"
	KindNotImplemented  Kind = "NOT_IMPLEMENTED"
	KindInternal        Kind = "INTERNAL"
)

// Error is a failure that is safe to describe to the client. Message must
// never contain SQL, stack traces, secrets, or another user's data.
type Error struct {
	Kind    Kind
	Message string
	Details map[string]any
	// Err is the underlying cause, logged but never returned to the client.
	Err error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

// Is lets errors.Is(err, domain.ErrNotFound) match on kind.
func (e *Error) Is(target error) bool {
	var t *Error
	if !errors.As(target, &t) {
		return false
	}
	return t.Kind == e.Kind && t.Message == ""
}

// Sentinel kinds for errors.Is checks.
var (
	ErrValidation      = &Error{Kind: KindValidation}
	ErrUnauthenticated = &Error{Kind: KindUnauthenticated}
	ErrForbidden       = &Error{Kind: KindForbidden}
	ErrNotFound        = &Error{Kind: KindNotFound}
	ErrConflict        = &Error{Kind: KindConflict}
)

// Validation reports invalid input for one field.
func Validation(field, message string) *Error {
	return &Error{Kind: KindValidation, Message: message, Details: map[string]any{"field": field}}
}

func Unauthenticated(message string) *Error {
	return &Error{Kind: KindUnauthenticated, Message: message}
}

func NotFound(message string) *Error {
	return &Error{Kind: KindNotFound, Message: message}
}

func Conflict(message string, details map[string]any) *Error {
	return &Error{Kind: KindConflict, Message: message, Details: details}
}
