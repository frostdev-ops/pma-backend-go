package utils

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Response represents a standard API response
type Response struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp string      `json:"timestamp"`
	Meta      interface{} `json:"meta,omitempty"`
}

// ErrorResponse represents an enhanced error response with additional context
type ErrorResponse struct {
	Success   bool        `json:"success"`
	Error     string      `json:"error"`
	Code      int         `json:"code"`
	Timestamp string      `json:"timestamp"`
	Request   RequestInfo `json:"request"`
	Details   interface{} `json:"details,omitempty"`
}

// RequestInfo provides context about the failed request
type RequestInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Query  string `json:"query,omitempty"`
}

// SendSuccess sends a successful response
func SendSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// SendError sends an error response with enhanced context
func SendError(c *gin.Context, statusCode int, message string) {
	errorResponse := ErrorResponse{
		Success:   false,
		Error:     message,
		Code:      statusCode,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Request: RequestInfo{
			Method: c.Request.Method,
			Path:   c.Request.URL.Path,
			Query:  c.Request.URL.RawQuery,
		},
	}

	// Add helpful suggestions for common errors
	if statusCode == http.StatusNotFound {
		suggestions := generateNotFoundSuggestions(c.Request.URL.Path)
		if len(suggestions) > 0 {
			errorResponse.Details = map[string]interface{}{
				"suggestions": suggestions,
				"message":     "The requested endpoint does not exist. Check the suggestions below for similar endpoints.",
			}
		}
	} else if statusCode == http.StatusMethodNotAllowed {
		errorResponse.Details = map[string]interface{}{
			"message": "The HTTP method is not supported for this endpoint. Please check the API documentation for supported methods.",
		}
	}

	c.JSON(statusCode, errorResponse)
}

// SendSuccessWithMeta sends a successful response with metadata
func SendSuccessWithMeta(c *gin.Context, data interface{}, meta interface{}) {
	c.JSON(http.StatusOK, Response{
		Success:   true,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// generateNotFoundSuggestions provides helpful endpoint suggestions for 404 errors
func generateNotFoundSuggestions(path string) []string {
	commonEndpoints := []string{
		"/health",
		"/api/v1/entities",
		"/api/v1/rooms",
		"/api/v1/automation",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/config",
		"/api/v1/system/status",
		"/ws",
	}

	var suggestions []string
	pathLower := strings.ToLower(path)

	// Find similar endpoints
	for _, endpoint := range commonEndpoints {
		endpointLower := strings.ToLower(endpoint)

		// Check for partial matches or common patterns
		if strings.Contains(pathLower, "entity") || strings.Contains(pathLower, "entities") {
			if strings.Contains(endpointLower, "entities") {
				suggestions = append(suggestions, endpoint)
			}
		} else if strings.Contains(pathLower, "room") {
			if strings.Contains(endpointLower, "rooms") {
				suggestions = append(suggestions, endpoint)
			}
		} else if strings.Contains(pathLower, "auth") || strings.Contains(pathLower, "login") {
			if strings.Contains(endpointLower, "auth") {
				suggestions = append(suggestions, endpoint)
			}
		} else if strings.Contains(pathLower, "config") {
			if strings.Contains(endpointLower, "config") {
				suggestions = append(suggestions, endpoint)
			}
		} else if strings.Contains(pathLower, "automation") {
			if strings.Contains(endpointLower, "automation") {
				suggestions = append(suggestions, endpoint)
			}
		} else if strings.Contains(pathLower, "system") || strings.Contains(pathLower, "status") {
			if strings.Contains(endpointLower, "system") || strings.Contains(endpointLower, "health") {
				suggestions = append(suggestions, endpoint)
			}
		}
	}

	// Remove duplicates and limit suggestions
	seen := make(map[string]bool)
	var unique []string
	for _, suggestion := range suggestions {
		if !seen[suggestion] && len(unique) < 5 {
			seen[suggestion] = true
			unique = append(unique, suggestion)
		}
	}

	return unique
}
