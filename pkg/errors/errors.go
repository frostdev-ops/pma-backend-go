package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application error with HTTP status code
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

// Common errors
var (
	ErrNotFound       = &AppError{Code: http.StatusNotFound, Message: "Resource not found"}
	ErrUnauthorized   = &AppError{Code: http.StatusUnauthorized, Message: "Unauthorized"}
	ErrForbidden      = &AppError{Code: http.StatusForbidden, Message: "Forbidden"}
	ErrBadRequest     = &AppError{Code: http.StatusBadRequest, Message: "Bad request"}
	ErrInternalServer = &AppError{Code: http.StatusInternalServerError, Message: "Internal server error"}
)

// New creates a new AppError
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// WithDetails adds details to an error
func WithDetails(err *AppError, details string) *AppError {
	return &AppError{
		Code:    err.Code,
		Message: err.Message,
		Details: details,
	}
}

// IsAppError checks if an error is an AppError
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetStatusCode returns the HTTP status code from an error
func GetStatusCode(err error) int {
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return http.StatusInternalServerError
}
