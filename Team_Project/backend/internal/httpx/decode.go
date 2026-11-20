package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"book-metaverse/backend/internal/domain"
)

// MaxBodyBytes is the request body limit (BACKEND_PLAN §4.1).
const MaxBodyBytes = 1 << 20

// Decode reads one JSON object into dst. Unknown fields, trailing data, an
// empty body, and bodies over MaxBodyBytes are VALIDATION_ERRORs.
func Decode(r *http.Request, dst any) error {
	body := http.MaxBytesReader(nil, r.Body, MaxBodyBytes)
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return decodeError(err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return &domain.Error{Kind: domain.KindValidation, Message: "Request body must contain a single JSON object."}
	}
	return nil
}

func decodeError(err error) error {
	var syntax *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	var tooLarge *http.MaxBytesError
	switch {
	case errors.Is(err, io.EOF):
		return &domain.Error{Kind: domain.KindValidation, Message: "Request body is required."}
	case errors.As(err, &tooLarge):
		return &domain.Error{Kind: domain.KindValidation, Message: "Request body is too large (limit 1 MiB)."}
	case errors.As(err, &syntax), errors.Is(err, io.ErrUnexpectedEOF):
		return &domain.Error{Kind: domain.KindValidation, Message: "Request body is not valid JSON."}
	case errors.As(err, &typeErr):
		return domain.Validation(typeErr.Field, fmt.Sprintf("%s has the wrong type.", typeErr.Field))
	case strings.HasPrefix(err.Error(), "json: unknown field "):
		field := strings.Trim(strings.TrimPrefix(err.Error(), "json: unknown field "), `"`)
		return domain.Validation(field, fmt.Sprintf("Unknown field %q.", field))
	default:
		return &domain.Error{Kind: domain.KindValidation, Message: "Request body is not valid JSON."}
	}
}
