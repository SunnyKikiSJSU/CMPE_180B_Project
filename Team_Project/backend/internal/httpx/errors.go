package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"book-metaverse/backend/internal/domain"
)

type requestIDKey struct{}

// WithRequestID stores the request ID for error bodies and logs.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID returns the request ID set by the middleware, or "".
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

var statusByKind = map[domain.Kind]int{
	domain.KindValidation:      http.StatusBadRequest,
	domain.KindUnauthenticated: http.StatusUnauthorized,
	domain.KindForbidden:       http.StatusForbidden,
	domain.KindNotFound:        http.StatusNotFound,
	domain.KindConflict:        http.StatusConflict,
	domain.KindStaleVersion:    http.StatusConflict,
	domain.KindRateLimited:     http.StatusTooManyRequests,
	domain.KindUpstream:        http.StatusBadGateway,
	domain.KindDatabase:        http.StatusServiceUnavailable,
	domain.KindInternal:        http.StatusInternalServerError,
}

type envelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code      domain.Kind    `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"requestId,omitempty"`
}

// Error writes the envelope for err. Unknown errors become INTERNAL with a
// generic message; their cause is logged with the request ID only.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := Classify(err)
	status := statusByKind[apiErr.Kind]
	if status == 0 {
		status = http.StatusInternalServerError
	}
	requestID := RequestID(r.Context())
	if status >= 500 {
		slog.ErrorContext(r.Context(), "request failed", "request_id", requestID, "code", apiErr.Kind, "error", err)
	}
	JSON(w, status, envelope{Error: errorBody{
		Code:      apiErr.Kind,
		Message:   apiErr.Message,
		Details:   apiErr.Details,
		RequestID: requestID,
	}})
}

// Classify turns any error into a client-safe domain.Error, including the
// PostgreSQL SQLSTATE mapping of docs/api/errors.md.
func Classify(err error) *domain.Error {
	var apiErr *domain.Error
	if errors.As(err, &apiErr) && apiErr.Message != "" {
		return apiErr
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return fromPostgres(pgErr)
	}

	var connectErr *pgconn.ConnectError
	if errors.As(err, &connectErr) || errors.Is(err, context.DeadlineExceeded) {
		return &domain.Error{Kind: domain.KindDatabase, Message: "The database is unavailable. Try again shortly.", Err: err}
	}

	return &domain.Error{Kind: domain.KindInternal, Message: "Something went wrong. Try again later.", Err: err}
}

func fromPostgres(pgErr *pgconn.PgError) *domain.Error {
	details := map[string]any{}
	if pgErr.ConstraintName != "" {
		details["constraint"] = pgErr.ConstraintName
	}
	switch pgErr.Code {
	case "23505":
		return &domain.Error{Kind: domain.KindConflict, Message: "This already exists.", Details: details, Err: pgErr}
	case "23503", "23514", "22023":
		return &domain.Error{Kind: domain.KindValidation, Message: "The request breaks a data rule.", Details: details, Err: pgErr}
	case "P0002":
		return &domain.Error{Kind: domain.KindNotFound, Message: "Not found.", Err: pgErr}
	case "BM001":
		return &domain.Error{Kind: domain.KindStaleVersion, Message: "This changed since you loaded it. Reload and try again.", Err: pgErr}
	case "40001", "40P01":
		return &domain.Error{Kind: domain.KindConflict, Message: "Another change happened at the same time. Try again.", Err: pgErr}
	case "55000":
		return &domain.Error{Kind: domain.KindConflict, Message: "Reading history cannot be changed.", Err: pgErr}
	}
	if strings.HasPrefix(pgErr.Code, "08") || strings.HasPrefix(pgErr.Code, "57") {
		// Class 08 (connection exception) and 57 (operator intervention, e.g. shutdown).
		return &domain.Error{Kind: domain.KindDatabase, Message: "The database is unavailable. Try again shortly.", Err: pgErr}
	}
	return &domain.Error{Kind: domain.KindInternal, Message: "Something went wrong. Try again later.", Err: pgErr}
}
