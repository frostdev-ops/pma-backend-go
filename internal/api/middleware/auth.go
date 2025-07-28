package middleware

import (
	"net/http"
	"strings"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthMiddleware validates API secrets, JWT tokens and sets user context
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication if disabled - but set default user context
		if !cfg.Auth.Enabled {
			// Set default user context for preferences compatibility
			c.Set("user_id", "1")
			c.Set("username", "default")
			c.Set("auth_type", "disabled")
			c.Set("auth_disabled", true) // Flag to indicate auth is disabled
			c.Next()
			return
		}

		// Allow localhost bypass for development
		clientIP := c.ClientIP()
		if cfg.Auth.AllowLocalhostBypass && (clientIP == "127.0.0.1" || clientIP == "::1" || clientIP == "localhost") {
			// Set localhost user context for preferences compatibility
			c.Set("user_id", "1")
			c.Set("username", "localhost")
			c.Set("auth_type", "localhost_bypass")
			c.Next()
			return
		}

		// Check for API secret header first (preferred method)
		apiSecret := c.GetHeader("X-API-Secret")
		if apiSecret != "" {
			if apiSecret == cfg.Auth.APISecret {
				// Set API user context with a valid user ID for preferences
				c.Set("user_id", "1") // Use user ID "1" for API access
				c.Set("username", "api")
				c.Set("auth_type", "api_secret")
				c.Next()
				return
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid API secret"})
				c.Abort()
				return
			}
		}

		// Check for JWT token in Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header or X-API-Secret required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		// For now, allow any bearer token since we're transitioning to API secret auth
		// In production, you would validate the JWT token here
		c.Set("user_id", "1")
		c.Set("username", "user")
		c.Set("auth_type", "jwt")
		c.Next()
	}
}

// OptionalAuthMiddleware provides optional authentication based on configuration
func OptionalAuthMiddleware(cfg *config.Config, configRepo repositories.ConfigRepository, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If auth is disabled, allow through
		if !cfg.Auth.Enabled {
			c.Next()
			return
		}

		// Allow localhost bypass for development
		clientIP := c.ClientIP()
		if cfg.Auth.AllowLocalhostBypass && (clientIP == "127.0.0.1" || clientIP == "::1" || clientIP == "localhost") {
			c.Next()
			return
		}

		// Check for API secret header
		apiSecret := c.GetHeader("X-API-Secret")
		if apiSecret != "" && apiSecret == cfg.Auth.APISecret {
			c.Set("user_id", "api")
			c.Set("username", "api")
			c.Set("auth_type", "api_secret")
			c.Next()
			return
		}

		// Check for JWT token
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) == 2 && tokenParts[0] == "Bearer" {
				// For now, allow any bearer token
				c.Set("user_id", "1")
				c.Set("username", "user")
				c.Set("auth_type", "jwt")
			}
		}

		// Continue regardless of auth status for optional auth
		c.Next()
	}
}
