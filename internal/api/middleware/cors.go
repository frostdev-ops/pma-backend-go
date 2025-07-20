package middleware

import (
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// CORSMiddleware returns a CORS middleware with proper configuration
func CORSMiddleware() gin.HandlerFunc {
	config := cors.Config{
		// Allow specific origins instead of wildcard when credentials are enabled
		AllowOrigins: []string{
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
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "X-Total-Count"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
		// Add a custom function to handle origins dynamically
		AllowOriginFunc: func(origin string) bool {
			// Allow any origin from the local network or localhost
			return origin == "http://192.168.100.1" ||
				origin == "http://localhost" ||
				origin == "http://127.0.0.1" ||
				origin == "http://192.168.100.1:80" ||
				origin == "http://192.168.100.1:3000" ||
				origin == "http://192.168.100.1:5173" ||
				origin == "http://localhost:80" ||
				origin == "http://localhost:3000" ||
				origin == "http://localhost:5173" ||
				origin == "http://127.0.0.1:80" ||
				origin == "http://127.0.0.1:3000" ||
				origin == "http://127.0.0.1:5173"
		},
	}

	return cors.New(config)
}

// CORSMiddlewareSSE returns a custom CORS handler for SSE endpoints
func CORSMiddlewareSSE() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Allow specific origins for SSE
		allowedOrigins := map[string]bool{
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

		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
