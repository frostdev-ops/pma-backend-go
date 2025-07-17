package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/frostdev-ops/pma-backend-go/internal/core/analytics"
	"github.com/frostdev-ops/pma-backend-go/internal/core/metrics"
)

// MetricsMiddleware creates middleware for collecting HTTP metrics
func MetricsMiddleware(collector metrics.MetricsCollector, tracker *analytics.PerformanceTracker) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Get response details
		method := c.Request.Method
		statusCode := c.Writer.Status()
		responseSize := int64(c.Writer.Size())
		userAgent := c.Request.UserAgent()

		// Record metrics with Prometheus collector
		if collector != nil {
			collector.RecordHTTPRequest(method, path, statusCode, duration)
		}

		// Record performance data
		if tracker != nil {
			tracker.RecordRequest(method, path, statusCode, duration, userAgent, responseSize)
		}
	}
}

// WebSocketMetricsMiddleware tracks WebSocket connection metrics
func WebSocketMetricsMiddleware(collector metrics.MetricsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record WebSocket connection attempt
		if collector != nil {
			collector.RecordWebSocketConnection("connect")
		}

		c.Next()

		// Note: Disconnection would be tracked in the WebSocket handler itself
	}
}

// DatabaseMetricsMiddleware tracks database operation metrics
func DatabaseMetricsMiddleware(collector metrics.MetricsCollector) func(operation string, duration time.Duration) {
	return func(operation string, duration time.Duration) {
		if collector != nil {
			collector.RecordDatabaseQuery(operation, duration)
		}
	}
}

// ResponseSizeMiddleware tracks response sizes
func ResponseSizeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Response size is automatically tracked in MetricsMiddleware
		// This middleware can be used for additional response size logic if needed
	}
}

// ErrorRateMiddleware tracks error rates
func ErrorRateMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Error tracking is automatically handled in MetricsMiddleware
		// This middleware can be used for additional error handling logic if needed

		statusCode := c.Writer.Status()
		if statusCode >= 500 {
			// Log server errors for additional monitoring
			path := c.FullPath()
			if path == "" {
				path = c.Request.URL.Path
			}

			// Additional error logging can be added here
		}
	}
}

// CustomMetricsMiddleware allows for custom metric collection
func CustomMetricsMiddleware(collector metrics.MetricsCollector, metricName string, labels map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)

		if collector != nil {
			// Record custom histogram metric
			customLabels := make(map[string]string)
			for k, v := range labels {
				customLabels[k] = v
			}

			// Add request-specific labels
			customLabels["method"] = c.Request.Method
			customLabels["status"] = strconv.Itoa(c.Writer.Status())

			collector.RecordHistogram(metricName+"_duration", duration.Seconds(), customLabels)
			collector.IncrementCounter(metricName+"_total", customLabels)
		}
	}
}

// ThresholdAlertMiddleware triggers alerts based on request metrics
func ThresholdAlertMiddleware(collector metrics.MetricsCollector, slowRequestThreshold time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// Alert on slow requests
		if duration > slowRequestThreshold {
			if collector != nil {
				labels := map[string]string{
					"type":   "slow_request",
					"method": c.Request.Method,
					"path":   c.FullPath(),
					"status": strconv.Itoa(statusCode),
				}
				collector.IncrementCounter("slow_requests_total", labels)
			}
		}

		// Alert on error responses
		if statusCode >= 500 {
			if collector != nil {
				labels := map[string]string{
					"type":   "server_error",
					"method": c.Request.Method,
					"path":   c.FullPath(),
					"status": strconv.Itoa(statusCode),
				}
				collector.IncrementCounter("server_errors_total", labels)
			}
		}
	}
}
