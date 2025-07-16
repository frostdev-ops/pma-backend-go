package homeassistant

import (
	"fmt"
	"net/http"
)

// HAError represents a Home Assistant-specific error
type HAError struct {
	Code    int                    `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *HAError) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("HA Error %d: %s (details: %v)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("HA Error %d: %s", e.Code, e.Message)
}

// Predefined error types
var (
	ErrUnauthorized = &HAError{
		Code:    http.StatusUnauthorized,
		Message: "Unauthorized access to Home Assistant",
	}
	ErrEntityNotFound = &HAError{
		Code:    http.StatusNotFound,
		Message: "Entity not found",
	}
	ErrConnectionFailed = &HAError{
		Code:    0,
		Message: "Connection to Home Assistant failed",
	}
	ErrInvalidResponse = &HAError{
		Code:    0,
		Message: "Invalid response from Home Assistant",
	}
	ErrTimeout = &HAError{
		Code:    0,
		Message: "Request timeout",
	}
	ErrWebSocketNotConnected = &HAError{
		Code:    0,
		Message: "WebSocket connection not established",
	}
	ErrInvalidURL = &HAError{
		Code:    0,
		Message: "Invalid Home Assistant URL",
	}
	ErrMissingToken = &HAError{
		Code:    0,
		Message: "Home Assistant access token not configured",
	}
)

// NewHAError creates a new HAError with custom details
func NewHAError(code int, message string, details map[string]interface{}) *HAError {
	return &HAError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// IsConnectionError checks if the error is a connection-related error
func IsConnectionError(err error) bool {
	if haErr, ok := err.(*HAError); ok {
		return haErr.Code == 0 || haErr == ErrConnectionFailed || haErr == ErrTimeout
	}
	return false
}

// IsAuthError checks if the error is an authentication error
func IsAuthError(err error) bool {
	if haErr, ok := err.(*HAError); ok {
		return haErr.Code == http.StatusUnauthorized || haErr == ErrUnauthorized
	}
	return false
}
