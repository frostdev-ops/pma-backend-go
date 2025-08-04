package middleware

import (
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware returns a CORS middleware with dynamic configuration
func CORSMiddleware(cfg *config.Config) gin.HandlerFunc {
	// Default allowed origins (fallback)
	defaultOrigins := []string{
		"http://192.168.100.1",
		"http://192.168.100.1:80",
		"http://192.168.100.1:3000",
		"http://192.168.100.1:5173",
		"http://localhost",
		"http://localhost:80",
		"http://localhost:3000",
		"http://localhost:5173",
		"http://127.0.0.1",
		"http://127.0.0.1:80",
		"http://127.0.0.1:3000",
		"http://127.0.0.1:5173",
	}

	// Use configured origins if available, otherwise use defaults
	var allowedOrigins []string
	if cfg.Security.EnableCORS && len(cfg.Security.AllowedOrigins) > 0 {
		// Check if wildcard is specified
		if len(cfg.Security.AllowedOrigins) == 1 && cfg.Security.AllowedOrigins[0] == "*" {
			// Allow all origins when wildcard is specified
			allowedOrigins = []string{"*"}
		} else {
			allowedOrigins = cfg.Security.AllowedOrigins
		}
	} else {
		allowedOrigins = defaultOrigins
	}

	config := cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With", "X-API-Secret"},
		ExposeHeaders:    []string{"Content-Length", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
		// Add a custom function to handle origins dynamically
		AllowOriginFunc: func(origin string) bool {
			// If wildcard is allowed, accept all origins
			if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
				return true
			}

			// Check against configured origins
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					return true
				}
			}

			// Fallback to default origins for backward compatibility
			defaultOrigins := []string{
				"http://192.168.100.1",
				"http://192.168.100.1:80",
				"http://192.168.100.1:3000",
				"http://192.168.100.1:5173",
				"http://localhost",
				"http://localhost:80",
				"http://localhost:3000",
				"http://localhost:5173",
				"http://127.0.0.1",
				"http://127.0.0.1:80",
				"http://127.0.0.1:3000",
				"http://127.0.0.1:5173",
			}

			for _, defaultOrigin := range defaultOrigins {
				if origin == defaultOrigin {
					return true
				}
			}

			return false
		},
	}

	return cors.New(config)
}

// CORSMiddlewareSSE returns a custom CORS handler for SSE endpoints
func CORSMiddlewareSSE(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Get allowed origins from config
		var allowedOrigins map[string]bool
		if cfg.Security.EnableCORS && len(cfg.Security.AllowedOrigins) > 0 {
			allowedOrigins = make(map[string]bool)
			for _, allowedOrigin := range cfg.Security.AllowedOrigins {
				allowedOrigins[allowedOrigin] = true
			}
		} else {
			// Fallback to default origins
			allowedOrigins = map[string]bool{
				"http://192.168.100.1":      true,
				"http://192.168.100.1:80":   true,
				"http://192.168.100.1:3000": true,
				"http://192.168.100.1:5173": true,
				"http://localhost":          true,
				"http://localhost:80":       true,
				"http://localhost:3000":     true,
				"http://localhost:5173":     true,
				"http://127.0.0.1":          true,
				"http://127.0.0.1:80":       true,
				"http://127.0.0.1:3000":     true,
				"http://127.0.0.1:5173":     true,
			}
		}

		// Check if wildcard is allowed
		if len(cfg.Security.AllowedOrigins) == 1 && cfg.Security.AllowedOrigins[0] == "*" {
			c.Header("Access-Control-Allow-Origin", origin)
		} else if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-API-Secret")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
