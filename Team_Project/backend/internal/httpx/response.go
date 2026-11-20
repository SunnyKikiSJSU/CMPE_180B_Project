// Package httpx holds the HTTP helpers every handler uses: bounded JSON
// decoding, JSON responses, pagination headers, and the error envelope.
package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
)

// JSON writes payload with the given status.
func JSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("write response", "error", err)
	}
}

// NoContent writes 204 with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// Paginate sets the list headers of the contract (raw array bodies).
func Paginate(w http.ResponseWriter, total int64, page, pageSize int) {
	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	w.Header().Set("X-Page", strconv.Itoa(page))
	w.Header().Set("X-Page-Size", strconv.Itoa(pageSize))
}
