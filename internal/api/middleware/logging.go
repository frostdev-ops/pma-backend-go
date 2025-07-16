package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoggingMiddleware returns a gin.HandlerFunc for logging requests
func LoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// Custom log format
		logger.WithFields(logrus.Fields{
			"client_ip":     param.ClientIP,
			"method":        param.Method,
			"path":          param.Path,
			"status_code":   param.StatusCode,
			"latency":       param.Latency,
			"user_agent":    param.Request.UserAgent(),
			"error_message": param.ErrorMessage,
		}).Info("HTTP Request")

		return ""
	})
}
