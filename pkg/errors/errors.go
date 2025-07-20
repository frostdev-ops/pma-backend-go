package errors

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	SeverityCritical ErrorSeverity = "critical"
	SeverityHigh     ErrorSeverity = "high"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityLow      ErrorSeverity = "low"
	SeverityInfo     ErrorSeverity = "info"
)

// ErrorCategory represents the category/domain of an error
type ErrorCategory string

const (
	CategoryValidation     ErrorCategory = "validation"
	CategoryAuthentication ErrorCategory = "authentication"
	CategoryAuthorization  ErrorCategory = "authorization"
	CategoryNotFound       ErrorCategory = "not_found"
	CategoryConflict       ErrorCategory = "conflict"
	CategoryRateLimit      ErrorCategory = "rate_limit"
	CategoryNetwork        ErrorCategory = "network"
	CategoryDatabase       ErrorCategory = "database"
	CategoryService        ErrorCategory = "service"
	CategoryAdapter        ErrorCategory = "adapter"
	CategoryInternal       ErrorCategory = "internal"
	CategoryExternal       ErrorCategory = "external"
	CategoryTimeout        ErrorCategory = "timeout"
	CategoryUnavailable    ErrorCategory = "unavailable"
)

// RecoveryStrategy defines how an error should be handled for recovery
type RecoveryStrategy string

const (
	RecoveryRetry     RecoveryStrategy = "retry"
	RecoveryFallback  RecoveryStrategy = "fallback"
	RecoveryIgnore    RecoveryStrategy = "ignore"
	RecoveryPropagate RecoveryStrategy = "propagate"
	RecoveryCircuit   RecoveryStrategy = "circuit_breaker"
)

// ErrorContext provides additional context about where and when an error occurred
type ErrorContext struct {
	RequestID  string                 `json:"request_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	SessionID  string                 `json:"session_id,omitempty"`
	Operation  string                 `json:"operation,omitempty"`
	Component  string                 `json:"component,omitempty"`
	EntityID   string                 `json:"entity_id,omitempty"`
	EntityType string                 `json:"entity_type,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	File       string                 `json:"file,omitempty"`
	Line       int                    `json:"line,omitempty"`
	Function   string                 `json:"function,omitempty"`
}

// EnhancedError represents a comprehensive application error with full context
type EnhancedError struct {
	Code             int              `json:"code"`
	Message          string           `json:"message"`
	Details          string           `json:"details,omitempty"`
	Category         ErrorCategory    `json:"category"`
	Severity         ErrorSeverity    `json:"severity"`
	Retryable        bool             `json:"retryable"`
	RetryAfter       *time.Duration   `json:"retry_after,omitempty"`
	RetryStrategy    RecoveryStrategy `json:"retry_strategy"`
	MaxRetries       int              `json:"max_retries"`
	Permanent        bool             `json:"permanent"`
	UserFacing       bool             `json:"user_facing"`
	Context          *ErrorContext    `json:"context,omitempty"`
	Underlying       error            `json:"-"`
	RelatedErrors    []*EnhancedError `json:"related_errors,omitempty"`
	SuggestedActions []string         `json:"suggested_actions,omitempty"`
	DocumentationURL string           `json:"documentation_url,omitempty"`
	ErrorID          string           `json:"error_id,omitempty"`
}

func (e *EnhancedError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("[%s] %s: %s (underlying: %v)",
			e.Category, e.Message, e.Details, e.Underlying)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Category, e.Message, e.Details)
}

func (e *EnhancedError) Unwrap() error {
	return e.Underlying
}

// Is implements error interface for error comparison
func (e *EnhancedError) Is(target error) bool {
	t, ok := target.(*EnhancedError)
	if !ok {
		return false
	}
	return e.Code == t.Code && e.Category == t.Category
}

// WithContext adds context to an error
func (e *EnhancedError) WithContext(ctx *ErrorContext) *EnhancedError {
	newErr := *e
	if newErr.Context == nil {
		newErr.Context = &ErrorContext{}
	}

	// Merge contexts
	if ctx.RequestID != "" {
		newErr.Context.RequestID = ctx.RequestID
	}
	if ctx.UserID != "" {
		newErr.Context.UserID = ctx.UserID
	}
	if ctx.Operation != "" {
		newErr.Context.Operation = ctx.Operation
	}
	if ctx.Component != "" {
		newErr.Context.Component = ctx.Component
	}
	if ctx.EntityID != "" {
		newErr.Context.EntityID = ctx.EntityID
	}
	if ctx.EntityType != "" {
		newErr.Context.EntityType = ctx.EntityType
	}
	if ctx.Metadata != nil {
		if newErr.Context.Metadata == nil {
			newErr.Context.Metadata = make(map[string]interface{})
		}
		for k, v := range ctx.Metadata {
			newErr.Context.Metadata[k] = v
		}
	}

	return &newErr
}

// WithStackTrace adds stack trace information to the error
func (e *EnhancedError) WithStackTrace() *EnhancedError {
	newErr := *e
	if newErr.Context == nil {
		newErr.Context = &ErrorContext{}
	}

	// Capture stack trace
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	newErr.Context.StackTrace = string(buf[:n])

	// Capture caller information
	if pc, file, line, ok := runtime.Caller(1); ok {
		newErr.Context.File = file
		newErr.Context.Line = line
		if fn := runtime.FuncForPC(pc); fn != nil {
			newErr.Context.Function = fn.Name()
		}
	}

	return &newErr
}

// AddRelatedError adds a related error that provides additional context
func (e *EnhancedError) AddRelatedError(relatedErr *EnhancedError) *EnhancedError {
	newErr := *e
	newErr.RelatedErrors = append(newErr.RelatedErrors, relatedErr)
	return &newErr
}

// AppError represents a simple application error (backward compatibility)
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("code=%d, message=%s", e.Code, e.Message)
}

// Common enhanced errors with proper categorization
var (
	// Validation errors
	ErrValidationFailed = &EnhancedError{
		Code:          http.StatusBadRequest,
		Message:       "Validation failed",
		Category:      CategoryValidation,
		Severity:      SeverityMedium,
		Retryable:     false,
		RetryStrategy: RecoveryPropagate,
		UserFacing:    true,
	}

	// Authentication errors
	ErrUnauthorized = &EnhancedError{
		Code:          http.StatusUnauthorized,
		Message:       "Authentication required",
		Category:      CategoryAuthentication,
		Severity:      SeverityMedium,
		Retryable:     false,
		RetryStrategy: RecoveryPropagate,
		UserFacing:    true,
		SuggestedActions: []string{
			"Check authentication credentials",
			"Verify token expiration",
			"Re-authenticate if necessary",
		},
	}

	ErrInvalidCredentials = &EnhancedError{
		Code:          http.StatusUnauthorized,
		Message:       "Invalid credentials",
		Category:      CategoryAuthentication,
		Severity:      SeverityMedium,
		Retryable:     false,
		RetryStrategy: RecoveryPropagate,
		UserFacing:    true,
	}

	// Authorization errors
	ErrForbidden = &EnhancedError{
		Code:          http.StatusForbidden,
		Message:       "Access forbidden",
		Category:      CategoryAuthorization,
		Severity:      SeverityMedium,
		Retryable:     false,
		RetryStrategy: RecoveryPropagate,
		UserFacing:    true,
	}

	// Not found errors
	ErrNotFound = &EnhancedError{
		Code:          http.StatusNotFound,
		Message:       "Resource not found",
		Category:      CategoryNotFound,
		Severity:      SeverityLow,
		Retryable:     false,
		RetryStrategy: RecoveryPropagate,
		UserFacing:    true,
	}

	// Conflict errors
	ErrConflict = &EnhancedError{
		Code:          http.StatusConflict,
		Message:       "Resource conflict",
		Category:      CategoryConflict,
		Severity:      SeverityMedium,
		Retryable:     true,
		RetryStrategy: RecoveryRetry,
		MaxRetries:    3,
		UserFacing:    true,
	}

	// Rate limiting errors
	ErrRateLimit = &EnhancedError{
		Code:          http.StatusTooManyRequests,
		Message:       "Rate limit exceeded",
		Category:      CategoryRateLimit,
		Severity:      SeverityMedium,
		Retryable:     true,
		RetryStrategy: RecoveryRetry,
		MaxRetries:    5,
		UserFacing:    true,
		SuggestedActions: []string{
			"Wait before retrying",
			"Reduce request frequency",
			"Check rate limit headers",
		},
	}

	// Network errors
	ErrNetworkUnavailable = &EnhancedError{
		Code:          http.StatusServiceUnavailable,
		Message:       "Network unavailable",
		Category:      CategoryNetwork,
		Severity:      SeverityHigh,
		Retryable:     true,
		RetryStrategy: RecoveryCircuit,
		MaxRetries:    3,
		UserFacing:    false,
	}

	ErrTimeout = &EnhancedError{
		Code:          http.StatusGatewayTimeout,
		Message:       "Operation timeout",
		Category:      CategoryTimeout,
		Severity:      SeverityHigh,
		Retryable:     true,
		RetryStrategy: RecoveryRetry,
		MaxRetries:    3,
		UserFacing:    false,
	}

	// Database errors
	ErrDatabaseUnavailable = &EnhancedError{
		Code:          http.StatusServiceUnavailable,
		Message:       "Database unavailable",
		Category:      CategoryDatabase,
		Severity:      SeverityCritical,
		Retryable:     true,
		RetryStrategy: RecoveryCircuit,
		MaxRetries:    5,
		UserFacing:    false,
	}

	ErrDatabaseConstraint = &EnhancedError{
		Code:          http.StatusConflict,
		Message:       "Database constraint violation",
		Category:      CategoryDatabase,
		Severity:      SeverityMedium,
		Retryable:     false,
		RetryStrategy: RecoveryPropagate,
		UserFacing:    true,
	}

	// Service errors
	ErrServiceUnavailable = &EnhancedError{
		Code:          http.StatusServiceUnavailable,
		Message:       "Service unavailable",
		Category:      CategoryService,
		Severity:      SeverityHigh,
		Retryable:     true,
		RetryStrategy: RecoveryCircuit,
		MaxRetries:    3,
		UserFacing:    false,
	}

	// Internal errors
	ErrInternalServer = &EnhancedError{
		Code:          http.StatusInternalServerError,
		Message:       "Internal server error",
		Category:      CategoryInternal,
		Severity:      SeverityCritical,
		Retryable:     false,
		RetryStrategy: RecoveryPropagate,
		UserFacing:    false,
	}

	// Legacy errors for backward compatibility
	ErrBadRequest = &AppError{Code: http.StatusBadRequest, Message: "Bad request"}
)

// Error creation functions

// NewEnhanced creates a new enhanced error with full context
func NewEnhanced(code int, message string, category ErrorCategory, severity ErrorSeverity) *EnhancedError {
	return &EnhancedError{
		Code:       code,
		Message:    message,
		Category:   category,
		Severity:   severity,
		Context:    &ErrorContext{Timestamp: time.Now()},
		UserFacing: true,
	}
}

// NewValidationError creates a validation error
func NewValidationError(field, message string) *EnhancedError {
	return NewEnhanced(http.StatusBadRequest,
		fmt.Sprintf("Validation failed for field '%s'", field),
		CategoryValidation, SeverityMedium).
		WithContext(&ErrorContext{
			Operation: "validation",
			Metadata: map[string]interface{}{
				"field":              field,
				"validation_message": message,
			},
		})
}

// NewNotFoundError creates a not found error
func NewNotFoundError(resourceType, resourceID string) *EnhancedError {
	return NewEnhanced(http.StatusNotFound,
		fmt.Sprintf("%s not found", resourceType),
		CategoryNotFound, SeverityLow).
		WithContext(&ErrorContext{
			Operation:  "lookup",
			EntityType: resourceType,
			EntityID:   resourceID,
		})
}

// NewDatabaseError creates a database error
func NewDatabaseError(operation string, underlying error) *EnhancedError {
	err := NewEnhanced(http.StatusInternalServerError,
		"Database operation failed",
		CategoryDatabase, SeverityHigh)
	err.Underlying = underlying
	err.Retryable = isRetryableDatabaseError(underlying)
	err.UserFacing = false
	err.Context = &ErrorContext{
		Operation: operation,
		Component: "database",
		Timestamp: time.Now(),
	}
	return err
}

// NewNetworkError creates a network error
func NewNetworkError(operation string, underlying error) *EnhancedError {
	err := NewEnhanced(http.StatusServiceUnavailable,
		"Network operation failed",
		CategoryNetwork, SeverityHigh)
	err.Underlying = underlying
	err.Retryable = true
	err.RetryStrategy = RecoveryRetry
	err.MaxRetries = 3
	err.UserFacing = false
	err.Context = &ErrorContext{
		Operation: operation,
		Component: "network",
		Timestamp: time.Now(),
	}
	return err
}

// NewServiceError creates a service error
func NewServiceError(service, operation string, underlying error) *EnhancedError {
	err := NewEnhanced(http.StatusServiceUnavailable,
		fmt.Sprintf("Service '%s' error", service),
		CategoryService, SeverityHigh)
	err.Underlying = underlying
	err.Retryable = true
	err.RetryStrategy = RecoveryCircuit
	err.MaxRetries = 3
	err.UserFacing = false
	err.Context = &ErrorContext{
		Operation: operation,
		Component: service,
		Timestamp: time.Now(),
	}
	return err
}

// Wrap wraps an existing error with enhanced context
func Wrap(err error, message string, category ErrorCategory) *EnhancedError {
	if err == nil {
		return nil
	}

	// If it's already an enhanced error, wrap it
	if enhancedErr, ok := err.(*EnhancedError); ok {
		wrapped := *enhancedErr
		wrapped.Message = message
		wrapped.Underlying = err
		return &wrapped
	}

	// If it's an AppError, convert it
	if appErr, ok := err.(*AppError); ok {
		return &EnhancedError{
			Code:       appErr.Code,
			Message:    message,
			Details:    appErr.Details,
			Category:   category,
			Severity:   SeverityMedium,
			Underlying: err,
			Context:    &ErrorContext{Timestamp: time.Now()},
		}
	}

	// Wrap regular error
	return &EnhancedError{
		Code:       http.StatusInternalServerError,
		Message:    message,
		Category:   category,
		Severity:   SeverityHigh,
		Underlying: err,
		Context:    &ErrorContext{Timestamp: time.Now()},
	}
}

// Legacy functions for backward compatibility

// New creates a new AppError (legacy)
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// WithDetails adds details to an error (legacy)
func WithDetails(err *AppError, details string) *AppError {
	return &AppError{
		Code:    err.Code,
		Message: err.Message,
		Details: details,
	}
}

// IsAppError checks if an error is an AppError (legacy)
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// IsEnhancedError checks if an error is an EnhancedError
func IsEnhancedError(err error) bool {
	_, ok := err.(*EnhancedError)
	return ok
}

// GetStatusCode returns the HTTP status code from an error
func GetStatusCode(err error) int {
	if enhancedErr, ok := err.(*EnhancedError); ok {
		return enhancedErr.Code
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr.Code
	}
	return http.StatusInternalServerError
}

// GetCategory returns the error category
func GetCategory(err error) ErrorCategory {
	if enhancedErr, ok := err.(*EnhancedError); ok {
		return enhancedErr.Category
	}
	return CategoryInternal
}

// IsRetryable determines if an error is retryable
func IsRetryable(err error) bool {
	if enhancedErr, ok := err.(*EnhancedError); ok {
		return enhancedErr.Retryable
	}
	return false
}

// GetRetryStrategy returns the recovery strategy for an error
func GetRetryStrategy(err error) RecoveryStrategy {
	if enhancedErr, ok := err.(*EnhancedError); ok {
		return enhancedErr.RetryStrategy
	}
	return RecoveryPropagate
}

// Helper functions

func isRetryableDatabaseError(err error) bool {
	if err == nil {
		return false
	}

	// Common retryable database errors
	errorStr := err.Error()
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"too many connections",
		"deadlock",
		"lock wait timeout",
	}

	for _, pattern := range retryablePatterns {
		if contains(errorStr, pattern) {
			return true
		}
	}

	return false
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) &&
		(str == substr ||
			(len(str) > len(substr) &&
				(str[:len(substr)] == substr ||
					str[len(str)-len(substr):] == substr ||
					containsSubstring(str, substr))))
}

func containsSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// FromContext extracts error context from a Go context
func FromContext(ctx context.Context) *ErrorContext {
	if ctx == nil {
		return &ErrorContext{Timestamp: time.Now()}
	}

	errorCtx := &ErrorContext{Timestamp: time.Now()}

	// Extract common context values
	if requestID := ctx.Value("request_id"); requestID != nil {
		if reqID, ok := requestID.(string); ok {
			errorCtx.RequestID = reqID
		}
	}

	if userID := ctx.Value("user_id"); userID != nil {
		if uID, ok := userID.(string); ok {
			errorCtx.UserID = uID
		}
	}

	if sessionID := ctx.Value("session_id"); sessionID != nil {
		if sID, ok := sessionID.(string); ok {
			errorCtx.SessionID = sID
		}
	}

	return errorCtx
}
