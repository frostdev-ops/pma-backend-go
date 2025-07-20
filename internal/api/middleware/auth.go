package middleware

import (
	"fmt"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthMiddleware validates JWT tokens and sets user context
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// DEBUG: Print to confirm our disabled auth middleware is running
		fmt.Printf("DEBUG AUTH MIDDLEWARE: DISABLED - Path: %s\n", c.Request.URL.Path)

		// DISABLED: All authentication completely removed
		c.Next()
		return

		// OLD CODE - ALL DISABLED:
		/*
			// Bypass auth for specific frontend compatibility endpoints
			if c.Request.URL.Path == "/api/settings/system" || c.Request.URL.Path == "/api/settings/theme" {
				// Add debug logging to confirm bypass is working
				// Note: Remove this in production
				fmt.Printf("DEBUG: Bypassing auth for path: %s\n", c.Request.URL.Path)
				c.Next()
				return
			}

			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				utils.SendError(c, http.StatusUnauthorized, "Authorization header required")
				c.Abort()
				return
			}

			// Extract token from "Bearer <token>"
			tokenParts := strings.Split(authHeader, " ")
			if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
				utils.SendError(c, http.StatusUnauthorized, "Invalid authorization header format")
				c.Abort()
				return
			}

			token := tokenParts[1]

			// Parse and validate token
			claims := &Claims{}
			jwtToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !jwtToken.Valid {
				utils.SendError(c, http.StatusUnauthorized, "Invalid or expired token")
				c.Abort()
				return
			}

			// Set user context
			c.Set("user_id", claims.UserID)
			c.Set("username", claims.Username)
			c.Next()
		*/
	}
}

// OptionalAuthMiddleware provides optional authentication based on configuration
func OptionalAuthMiddleware(cfg *config.Config, configRepo repositories.ConfigRepository, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// DEBUG: Print to confirm our disabled optional auth middleware is running
		fmt.Printf("DEBUG OPTIONAL AUTH MIDDLEWARE: DISABLED - Path: %s\n", c.Request.URL.Path)

		// DISABLED: All authentication completely removed
		c.Next()
		return

		// OLD CODE - ALL DISABLED:
		/*
			// Debug: Log that OptionalAuthMiddleware is being called
			logger.Infof("DEBUG: OptionalAuthMiddleware called for path %s, Auth.Enabled = %t", c.Request.URL.Path, cfg.Auth.Enabled)

			// Bypass auth for specific frontend compatibility endpoints
			if c.Request.URL.Path == "/api/settings/system" || c.Request.URL.Path == "/api/settings/theme" {
				logger.Infof("DEBUG: Bypassing OptionalAuth for path: %s", c.Request.URL.Path)
				c.Next()
				return
			}

			// TEMPORARY: Force auth to be disabled for debugging
			// If auth is completely disabled in config, allow through
			if !cfg.Auth.Enabled || true { // Always allow through for debugging
				logger.Infof("DEBUG: Auth disabled (or forced disabled), allowing request through for path %s", c.Request.URL.Path)
				c.Next()
				return
			}

			// The rest of the auth logic...
		*/
	}
}
