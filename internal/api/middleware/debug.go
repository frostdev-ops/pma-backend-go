package middleware

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/pkg/debug"
	"github.com/gin-gonic/gin"
)

// DebugMiddleware provides comprehensive request/response logging
type DebugMiddleware struct {
	debugLogger *debug.DebugLogger
}

// NewDebugMiddleware creates a new debug middleware instance
func NewDebugMiddleware(debugLogger *debug.DebugLogger) *DebugMiddleware {
	return &DebugMiddleware{
		debugLogger: debugLogger,
	}
}

// DebugLoggingMiddleware logs detailed request and response information
func (dm *DebugMiddleware) DebugLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if dm.debugLogger == nil || !dm.debugLogger.IsComponentEnabled("handlers") {
			c.Next()
			return
		}

		start := time.Now()
		requestID := generateRequestID()

		// Add request ID to context
		ctx := context.WithValue(c.Request.Context(), "request_id", requestID)
		c.Request = c.Request.WithContext(ctx)

		// Log request details
		dm.logRequest(c, requestID, start)

		// Capture response body
		responseWriter := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           &bytes.Buffer{},
		}
		c.Writer = responseWriter

		// Process request
		c.Next()

		// Log response details
		dm.logResponse(c, requestID, start, responseWriter)
	}
}

// logRequest logs detailed request information
func (dm *DebugMiddleware) logRequest(c *gin.Context, requestID string, start time.Time) {
	// Read request body
	var requestBody []byte
	if c.Request.Body != nil {
		requestBody, _ = io.ReadAll(c.Request.Body)
		// Restore body for further processing
		c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
	}

	// Extract headers (excluding sensitive ones)
	headers := make(map[string]string)
	for k, v := range c.Request.Header {
		if !isSensitiveHeader(k) {
			headers[k] = strings.Join(v, ", ")
		}
	}

	// Extract query parameters
	queryParams := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		queryParams[k] = strings.Join(v, ", ")
	}

	// Extract path parameters
	pathParams := make(map[string]string)
	for _, param := range c.Params {
		pathParams[param.Key] = param.Value
	}

	fields := map[string]interface{}{
		"method":         c.Request.Method,
		"url":            c.Request.URL.String(),
		"path":           c.Request.URL.Path,
		"remote_addr":    c.ClientIP(),
		"user_agent":     c.Request.UserAgent(),
		"content_type":   c.GetHeader("Content-Type"),
		"content_length": len(requestBody),
		"headers":        headers,
		"query_params":   queryParams,
		"path_params":    pathParams,
		"timestamp":      start,
	}

	// Log request body if not too large and not binary
	if len(requestBody) > 0 && len(requestBody) < 10240 && !isBinaryContent(c.GetHeader("Content-Type")) {
		fields["request_body"] = string(requestBody)
	}

	dm.debugLogger.LogWithContext(c.Request.Context(), "debug", "handlers", "HTTP Request", fields)
}

// logResponse logs detailed response information
func (dm *DebugMiddleware) logResponse(c *gin.Context, requestID string, start time.Time, responseWriter *responseBodyWriter) {
	duration := time.Since(start)

	// Extract response headers
	headers := make(map[string]string)
	for k, v := range c.Writer.Header() {
		headers[k] = strings.Join(v, ", ")
	}

	fields := map[string]interface{}{
		"status_code":    c.Writer.Status(),
		"content_length": responseWriter.body.Len(),
		"duration":       duration.String(),
		"headers":        headers,
		"error":          c.Errors.String(),
	}

	// Log response body if not too large and not binary
	responseBody := responseWriter.body.String()
	if len(responseBody) > 0 && len(responseBody) < 10240 && !isBinaryContent(c.GetHeader("Content-Type")) {
		fields["response_body"] = responseBody
	}

	// Determine log level based on status code
	level := "debug"
	if c.Writer.Status() >= 400 {
		level = "error"
	} else if c.Writer.Status() >= 300 {
		level = "warn"
	}

	dm.debugLogger.LogWithContext(c.Request.Context(), level, "handlers", "HTTP Response", fields)
}

// responseBodyWriter captures the response body
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseBodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}

// isSensitiveHeader checks if a header contains sensitive information
func isSensitiveHeader(header string) bool {
	sensitiveHeaders := []string{
		"authorization",
		"cookie",
		"x-api-key",
		"x-auth-token",
		"x-jwt-token",
	}

	headerLower := strings.ToLower(header)
	for _, sensitive := range sensitiveHeaders {
		if strings.Contains(headerLower, sensitive) {
			return true
		}
	}
	return false
}

// isBinaryContent checks if content type indicates binary data
func isBinaryContent(contentType string) bool {
	binaryTypes := []string{
		"image/",
		"video/",
		"audio/",
		"application/octet-stream",
		"application/pdf",
		"application/zip",
	}

	contentTypeLower := strings.ToLower(contentType)
	for _, binaryType := range binaryTypes {
		if strings.HasPrefix(contentTypeLower, binaryType) {
			return true
		}
	}
	return false
}

// DebugDatabaseMiddleware logs database operations
func (dm *DebugMiddleware) DebugDatabaseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if dm.debugLogger == nil || !dm.debugLogger.IsComponentEnabled("database") {
			c.Next()
			return
		}

		// This would be integrated with database operations
		// For now, just pass through
		c.Next()
	}
}

// DebugWebSocketMiddleware logs WebSocket operations
func (dm *DebugMiddleware) DebugWebSocketMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if dm.debugLogger == nil || !dm.debugLogger.IsComponentEnabled("websocket") {
			c.Next()
			return
		}

		// Log WebSocket upgrade
		if strings.Contains(c.GetHeader("Upgrade"), "websocket") {
			fields := map[string]interface{}{
				"upgrade":                c.GetHeader("Upgrade"),
				"connection":             c.GetHeader("Connection"),
				"sec_websocket_key":      c.GetHeader("Sec-WebSocket-Key"),
				"sec_websocket_protocol": c.GetHeader("Sec-WebSocket-Protocol"),
			}

			dm.debugLogger.LogWithContext(c.Request.Context(), "debug", "websocket", "WebSocket Upgrade Request", fields)
		}

		c.Next()
	}
}
