package api

import (
	"encoding/json"
	"net/http"
)

// ErrorCode is a machine-readable error identifier.
type ErrorCode string

const (
	ErrInvalidRequest ErrorCode = "INVALID_REQUEST"
	ErrInvalidClient  ErrorCode = "INVALID_CLIENT"
	ErrInvalidGrant   ErrorCode = "INVALID_GRANT"
	ErrUnauthorized   ErrorCode = "UNAUTHORIZED"
	ErrForbidden      ErrorCode = "FORBIDDEN"
	ErrNotFound       ErrorCode = "NOT_FOUND"
	ErrConflict       ErrorCode = "CONFLICT"
	ErrValidation     ErrorCode = "VALIDATION_ERROR"
	ErrInternal       ErrorCode = "INTERNAL_ERROR"
)

// ErrorResponse is the standard REST error envelope.
type ErrorResponse struct {
	Error   ErrorCode `json:"error"`
	Message string    `json:"message"`
}

// Error writes a JSON error response with the given status code.
func Error(w http.ResponseWriter, code ErrorCode, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: code, Message: message})
}

// Common error helpers.
func BadRequest(w http.ResponseWriter, message string) {
	Error(w, ErrInvalidRequest, message, http.StatusBadRequest)
}

func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, ErrUnauthorized, message, http.StatusUnauthorized)
}

func Forbidden(w http.ResponseWriter, message string) {
	Error(w, ErrForbidden, message, http.StatusForbidden)
}

func NotFound(w http.ResponseWriter, message string) {
	Error(w, ErrNotFound, message, http.StatusNotFound)
}

func Conflict(w http.ResponseWriter, message string) {
	Error(w, ErrConflict, message, http.StatusConflict)
}

func InternalError(w http.ResponseWriter) {
	Error(w, ErrInternal, "An internal error occurred", http.StatusInternalServerError)
}

func InvalidClient(w http.ResponseWriter, message string) {
	Error(w, ErrInvalidClient, message, http.StatusUnauthorized)
}

func InvalidGrant(w http.ResponseWriter, message string) {
	Error(w, ErrInvalidGrant, message, http.StatusBadRequest)
}
