package middleware

// Enhanced Logging Middleware with Batching for Performance
//
// This middleware integrates with the BatchLogger from the logger package to provide:
//
// BATCHING BEHAVIOR:
// - 200 status codes are batched together for efficiency
// - Non-200 status codes are logged immediately for quick error detection
// - Batch summaries are sent every 100 requests (configurable in BatchLogger)
// - Provides aggregated metrics: count, min/max/avg latency per endpoint
//
// PERFORMANCE BENEFITS:
// - Reduces log volume by ~80-90% for healthy API endpoints
// - Provides better performance insights through aggregated metrics
// - Immediate visibility into errors and slow requests
// - Cleaner log output with batch summaries for successful requests
//
// EXAMPLE OUTPUT:
// Immediate logging for errors:
//   {"level":"error","msg":"POST /api/entities - Status: 500, Latency: 45ms"}
//
// Batched summary for successful requests:
//   {"level":"info","msg":"Request batch summary (200 status codes)",
//    "batch_summary":true,"total_requests":100,
//    "endpoints":{"GET /api/entities":{"count":80,"avg_latency":"12ms"}}}

import (
	"time"

	"github.com/frostdev-ops/pma-backend-go/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoggingMiddleware returns a gin.HandlerFunc for logging requests using BatchLogger
func LoggingMiddleware(batchLogger *logger.BatchLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Gather request information
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := c.Writer.Status()

		// Prepare additional fields for non-200 status codes
		fields := logrus.Fields{
			"client_ip":     c.ClientIP(),
			"method":        method,
			"path":          path,
			"status_code":   statusCode,
			"latency":       latency,
			"user_agent":    c.Request.UserAgent(),
			"response_size": c.Writer.Size(),
		}

		// Add query parameters if present
		if c.Request.URL.RawQuery != "" {
			fields["query"] = c.Request.URL.RawQuery
		}

		// Add error message if present
		if len(c.Errors) > 0 {
			fields["error_message"] = c.Errors.String()
		}

		// Add request ID if present in headers
		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			fields["request_id"] = requestID
		}

		// Use BatchLogger's LogRequest method which handles batching for 200s
		batchLogger.LogRequest(method, path, statusCode, latency, fields)
	}
}

// EnhancedLoggingMiddleware provides additional logging context and batching
func EnhancedLoggingMiddleware(batchLogger *logger.BatchLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Gather comprehensive request information
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := c.Writer.Status()

		// Enhanced fields with more context
		fields := logrus.Fields{
			"client_ip":     c.ClientIP(),
			"method":        method,
			"path":          path,
			"status_code":   statusCode,
			"latency":       latency,
			"latency_ms":    latency.Milliseconds(),
			"user_agent":    c.Request.UserAgent(),
			"response_size": c.Writer.Size(),
			"request_size":  c.Request.ContentLength,
			"protocol":      c.Request.Proto,
		}

		// Add query parameters if present
		if c.Request.URL.RawQuery != "" {
			fields["query"] = c.Request.URL.RawQuery
		}

		// Add error message if present
		if len(c.Errors) > 0 {
			fields["error_message"] = c.Errors.String()
		}

		// Add request ID if present in headers
		if requestID := c.GetHeader("X-Request-ID"); requestID != "" {
			fields["request_id"] = requestID
		}

		// Add authentication status if available
		if authenticated, exists := c.Get("authenticated"); exists {
			fields["authenticated"] = authenticated
		}

		// Add session info if available
		if session, exists := c.Get("session"); exists && session != nil {
			fields["has_session"] = true
		}

		// Add forwarded headers if present
		if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
			fields["x_forwarded_for"] = xff
		}

		if xri := c.GetHeader("X-Real-IP"); xri != "" {
			fields["x_real_ip"] = xri
		}

		// Use BatchLogger's LogRequest method which handles batching for 200s
		batchLogger.LogRequest(method, path, statusCode, latency, fields)
	}
}

// DebugLoggingMiddleware provides detailed logging for debugging (does not use batching)
func DebugLoggingMiddleware(batchLogger *logger.BatchLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Log request start for debugging
		batchLogger.WithFields(logrus.Fields{
			"method":    c.Request.Method,
			"path":      c.Request.URL.Path,
			"client_ip": c.ClientIP(),
		}).Debug("Request started")

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Detailed debug fields
		fields := logrus.Fields{
			"client_ip":     c.ClientIP(),
			"method":        c.Request.Method,
			"path":          c.Request.URL.Path,
			"status_code":   c.Writer.Status(),
			"latency":       latency,
			"latency_ns":    latency.Nanoseconds(),
			"user_agent":    c.Request.UserAgent(),
			"response_size": c.Writer.Size(),
			"request_size":  c.Request.ContentLength,
			"protocol":      c.Request.Proto,
			"host":          c.Request.Host,
			"referer":       c.Request.Referer(),
		}

		// Add all headers for debugging
		headers := make(map[string]string)
		for k, v := range c.Request.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
		fields["headers"] = headers

		// Always log debug requests immediately (don't batch)
		batchLogger.WithFields(fields).Debug("Request completed (debug mode)")
	}
}
