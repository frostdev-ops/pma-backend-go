package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrorHandlingMiddleware handles panics and errors with enhanced logging and recovery
func ErrorHandlingMiddleware(logger *logrus.Logger, recoveryManager *errors.RecoveryManager) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Create error context from request
		errorContext := &errors.ErrorContext{
			RequestID: getRequestID(c),
			UserID:    getUserID(c),
			SessionID: getSessionID(c),
			Operation: fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
			Component: "api_middleware",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"query":      c.Request.URL.RawQuery,
				"ip":         c.ClientIP(),
				"user_agent": c.GetHeader("User-Agent"),
			},
		}

		// Capture stack trace
		stack := debug.Stack()
		errorContext.StackTrace = string(stack)

		var enhancedErr *errors.EnhancedError

		// Convert panic to enhanced error
		switch err := recovered.(type) {
		case string:
			enhancedErr = errors.NewEnhanced(
				http.StatusInternalServerError,
				"Panic recovered",
				errors.CategoryInternal,
				errors.SeverityCritical,
			).WithContext(errorContext)
			enhancedErr.Details = err
			enhancedErr.UserFacing = false

		case error:
			// Check if it's already an enhanced error
			if existingEnhanced, ok := err.(*errors.EnhancedError); ok {
				enhancedErr = existingEnhanced.WithContext(errorContext)
			} else {
				enhancedErr = errors.Wrap(err, "Panic recovered", errors.CategoryInternal).
					WithContext(errorContext)
				enhancedErr.Code = http.StatusInternalServerError
				enhancedErr.Severity = errors.SeverityCritical
				enhancedErr.UserFacing = false
			}

		default:
			enhancedErr = errors.NewEnhanced(
				http.StatusInternalServerError,
				"Unknown panic recovered",
				errors.CategoryInternal,
				errors.SeverityCritical,
			).WithContext(errorContext)
			enhancedErr.Details = fmt.Sprintf("%+v", recovered)
			enhancedErr.UserFacing = false
		}

		// Enhanced logging with request context
		logEntry := logger.WithFields(logrus.Fields{
			"error_id":    enhancedErr.ErrorID,
			"category":    enhancedErr.Category,
			"severity":    enhancedErr.Severity,
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"query":       c.Request.URL.RawQuery,
			"ip":          c.ClientIP(),
			"user_agent":  c.GetHeader("User-Agent"),
			"request_id":  errorContext.RequestID,
			"user_id":     errorContext.UserID,
			"stack_trace": errorContext.StackTrace,
		})

		logEntry.Error("Panic recovered in API middleware")

		// Report error to recovery manager
		if recoveryManager != nil {
			recoveryManager.GetErrorReporter().ReportError(enhancedErr)
		}

		// Send appropriate response to client
		sendErrorResponse(c, enhancedErr)
		c.Abort()
	})
}

// ErrorResponseMiddleware converts errors to standardized responses
func ErrorResponseMiddleware(logger *logrus.Logger, recoveryManager *errors.RecoveryManager) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// Create error context
			errorContext := &errors.ErrorContext{
				RequestID: getRequestID(c),
				UserID:    getUserID(c),
				SessionID: getSessionID(c),
				Operation: fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
				Component: "api_response",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"method":        c.Request.Method,
					"path":          c.Request.URL.Path,
					"status_code":   c.Writer.Status(),
					"response_size": c.Writer.Size(),
				},
			}

			var enhancedErr *errors.EnhancedError

			// Convert to enhanced error if not already
			if existingEnhanced, ok := err.(*errors.EnhancedError); ok {
				enhancedErr = existingEnhanced.WithContext(errorContext)
			} else if appErr, ok := err.(*errors.AppError); ok {
				enhancedErr = errors.Wrap(appErr, appErr.Message, errors.CategoryInternal).
					WithContext(errorContext)
			} else {
				enhancedErr = errors.Wrap(err, "Request processing error", errors.CategoryInternal).
					WithContext(errorContext)
			}

			// Log the error
			logger.WithFields(logrus.Fields{
				"error_id":   enhancedErr.ErrorID,
				"category":   enhancedErr.Category,
				"severity":   enhancedErr.Severity,
				"method":     c.Request.Method,
				"path":       c.Request.URL.Path,
				"request_id": errorContext.RequestID,
			}).Error("API request error")

			// Report error to recovery manager
			if recoveryManager != nil {
				recoveryManager.GetErrorReporter().ReportError(enhancedErr)
			}

			// Send error response if not already sent
			if !c.Writer.Written() {
				sendErrorResponse(c, enhancedErr)
			}
		}
	})
}

// sendErrorResponse sends a standardized error response
func sendErrorResponse(c *gin.Context, err *errors.EnhancedError) {
	// Determine what information to include based on severity and user-facing flag
	response := gin.H{
		"success":   false,
		"error":     getPublicErrorMessage(err),
		"code":      err.Code,
		"timestamp": time.Now().Format(time.RFC3339),
		"path":      c.Request.URL.Path,
		"method":    c.Request.Method,
	}

	// Add error ID for tracking
	if err.ErrorID != "" {
		response["error_id"] = err.ErrorID
	}

	// Add context information if available
	if err.Context != nil && err.Context.RequestID != "" {
		response["request_id"] = err.Context.RequestID
	}

	// Add suggested actions for user-facing errors
	if err.UserFacing && len(err.SuggestedActions) > 0 {
		response["suggestions"] = err.SuggestedActions
	}

	// Add documentation URL if available
	if err.DocumentationURL != "" {
		response["documentation"] = err.DocumentationURL
	}

	// Add retry information for retryable errors
	if err.Retryable {
		response["retryable"] = true
		if err.RetryAfter != nil {
			response["retry_after"] = err.RetryAfter.Seconds()
		}
		if err.MaxRetries > 0 {
			response["max_retries"] = err.MaxRetries
		}
	}

	// Add development information in non-production environments
	if gin.Mode() == gin.DebugMode {
		response["category"] = err.Category
		response["severity"] = err.Severity
		if err.Details != "" {
			response["details"] = err.Details
		}
		if err.Context != nil && err.Context.StackTrace != "" {
			response["stack_trace"] = strings.Split(err.Context.StackTrace, "\n")[:10] // Limit stack trace
		}
	}

	c.JSON(err.Code, response)
}

// getPublicErrorMessage returns an appropriate error message for public consumption
func getPublicErrorMessage(err *errors.EnhancedError) string {
	if err.UserFacing {
		return err.Message
	}

	// Return generic messages for internal errors
	switch err.Category {
	case errors.CategoryDatabase:
		return "A database error occurred. Please try again later."
	case errors.CategoryNetwork:
		return "A network error occurred. Please check your connection and try again."
	case errors.CategoryTimeout:
		return "The request timed out. Please try again."
	case errors.CategoryUnavailable:
		return "The service is temporarily unavailable. Please try again later."
	case errors.CategoryRateLimit:
		return "Too many requests. Please wait before trying again."
	case errors.CategoryInternal:
		return "An internal error occurred. Please try again later."
	default:
		return "An error occurred. Please try again later."
	}
}

// Helper functions to extract context information

func getRequestID(c *gin.Context) string {
	if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
		return requestID
	}
	if requestID := c.GetString("request_id"); requestID != "" {
		return requestID
	}
	return ""
}

func getUserID(c *gin.Context) string {
	if userID := c.GetString("user_id"); userID != "" {
		return userID
	}
	if userID := c.GetHeader("X-User-ID"); userID != "" {
		return userID
	}
	return ""
}

func getSessionID(c *gin.Context) string {
	if sessionID := c.GetString("session_id"); sessionID != "" {
		return sessionID
	}
	if sessionID := c.GetHeader("X-Session-ID"); sessionID != "" {
		return sessionID
	}
	return ""
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate a simple request ID
			requestID = fmt.Sprintf("%d-%d", time.Now().UnixNano(),
				hashString(c.ClientIP()+c.Request.UserAgent()))
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	})
}

// Simple hash function for request ID generation
func hashString(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h *= 16777619
		h ^= uint32(s[i])
	}
	return h
}

// ValidationErrorMiddleware handles validation errors specifically
func ValidationErrorMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Next()

		// Look for validation errors in the context
		if validationErrors, exists := c.Get("validation_errors"); exists {
			if errList, ok := validationErrors.([]error); ok {
				var enhancedErrors []*errors.EnhancedError

				for _, err := range errList {
					enhancedErr := errors.NewValidationError("request", err.Error())
					enhancedErr.Context.RequestID = getRequestID(c)
					enhancedErr.Context.Operation = fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
					enhancedErrors = append(enhancedErrors, enhancedErr)
				}

				// Create a combined validation error
				if len(enhancedErrors) > 0 {
					mainErr := enhancedErrors[0]
					mainErr.Message = "Validation failed"
					mainErr.Details = fmt.Sprintf("%d validation errors occurred", len(enhancedErrors))

					// Add related errors
					for i := 1; i < len(enhancedErrors); i++ {
						mainErr = mainErr.AddRelatedError(enhancedErrors[i])
					}

					sendErrorResponse(c, mainErr)
					c.Abort()
				}
			}
		}
	})
}

// RateLimitErrorMiddleware handles rate limiting errors
func RateLimitErrorMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		c.Next()

		// Check if this is a rate limit error
		if rateLimited, exists := c.Get("rate_limited"); exists && rateLimited.(bool) {
			retryAfter := time.Second * 60 // Default retry after
			if ra, exists := c.Get("retry_after"); exists {
				if duration, ok := ra.(time.Duration); ok {
					retryAfter = duration
				}
			}

			err := errors.ErrRateLimit.WithContext(&errors.ErrorContext{
				RequestID: getRequestID(c),
				UserID:    getUserID(c),
				Operation: fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
				Component: "rate_limiter",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"client_ip": c.ClientIP(),
					"path":      c.Request.URL.Path,
				},
			})
			err.RetryAfter = &retryAfter

			c.Header("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()))
			sendErrorResponse(c, err)
			c.Abort()
		}
	})
}
