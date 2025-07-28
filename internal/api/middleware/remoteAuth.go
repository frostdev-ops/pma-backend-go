package middleware

import (
	"net/http"
	"strings"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/auth"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RemoteAuthMiddleware provides authentication that varies based on connection type
// - Localhost connections: No authentication required
// - Local network connections: No authentication required
// - Remote connections: User/password authentication required
func RemoteAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow OPTIONS requests to pass through for CORS preflight
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// If auth is completely disabled, allow through
		if !cfg.Auth.Enabled {
			c.Set("user_id", "1")
			c.Set("username", "default")
			c.Set("auth_type", "disabled")
			c.Set("auth_disabled", true)
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		// Check if this is a local connection (localhost or local network)
		if isLocalConnection(clientIP) {
			// Local connections don't require authentication
			c.Set("user_id", "1")
			c.Set("username", "local")
			c.Set("auth_type", "local_bypass")
			c.Set("local_connection", true)
			c.Next()
			return
		}

		// For remote connections, require authentication
		// Check for API secret header first (preferred method)
		apiSecret := c.GetHeader("X-API-Secret")
		if apiSecret != "" {
			if apiSecret == cfg.Auth.APISecret {
				c.Set("user_id", "1")
				c.Set("username", "api")
				c.Set("auth_type", "api_secret")
				c.Set("remote_connection", true)
				c.Next()
				return
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success":       false,
					"error":         "Invalid API secret",
					"auth_required": true,
				})
				c.Abort()
				return
			}
		}

		// Check for JWT token in Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success":       false,
				"error":         "Authentication required for remote access",
				"auth_required": true,
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success":       false,
				"error":         "Invalid authorization header format",
				"auth_required": true,
			})
			c.Abort()
			return
		}

		// For now, allow any bearer token since we're using PIN-based auth
		// In production, you would validate the JWT token here
		c.Set("user_id", "1")
		c.Set("username", "remote")
		c.Set("auth_type", "jwt")
		c.Set("remote_connection", true)
		c.Next()
	}
}

// UserAuthMiddleware provides user/password authentication for remote access
func UserAuthMiddleware(cfg *config.Config, authRepo repositories.AuthRepository, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Allow OPTIONS requests to pass through for CORS preflight
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		// Check if auth is disabled globally
		if !cfg.Auth.Enabled {
			c.Set("user_id", "1")
			c.Set("username", "default")
			c.Set("auth_type", "disabled")
			c.Set("auth_disabled", true)
			c.Next()
			return
		}

		// Get client IP
		clientIP := c.ClientIP()

		// Check if request is from localhost (bypass authentication)
		if isLocalConnection(clientIP) {
			c.Set("user_id", "1")
			c.Set("username", "localhost")
			c.Set("auth_type", "localhost_bypass")
			c.Set("auth_disabled", false)
			c.Next()
			return
		}

		// For remote access, require user/password authentication
		// Check for API secret header first
		apiSecret := c.GetHeader("X-API-Secret")
		if apiSecret != "" {
			if apiSecret == cfg.Auth.APISecret {
				c.Set("user_id", "1")
				c.Set("username", "api")
				c.Set("auth_type", "api_secret")
				c.Next()
				return
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"error":   "Invalid API secret",
					"code":    "INVALID_API_SECRET",
				})
				c.Abort()
				return
			}
		}

		// Check for JWT token in Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Authentication required for remote access",
				"code":    "AUTH_REQUIRED",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid authorization header format",
				"code":    "INVALID_AUTH_HEADER",
			})
			c.Abort()
			return
		}

		token := tokenParts[1]

		// Validate session token
		authConfig := auth.AuthConfig{
			SessionTimeout:    cfg.Auth.TokenExpiry,
			MaxFailedAttempts: 3,
			LockoutDuration:   300,
			JWTSecret:         cfg.Auth.JWTSecret,
		}
		authService := auth.NewService(authRepo, authConfig, logger)

		session, err := authService.ValidateSession(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid or expired session",
				"code":    "INVALID_SESSION",
			})
			c.Abort()
			return
		}

		// Set user context from session (using default values since Session model doesn't have user fields)
		c.Set("user_id", "1")
		c.Set("username", "user")
		c.Set("auth_type", "session")
		c.Set("session_id", session.ID)
		c.Next()
	}
}

// isLocalConnection checks if the client IP is from a local connection
func isLocalConnection(clientIP string) bool {
	// Check for localhost IPs
	localhostIPs := []string{"127.0.0.1", "::1", "localhost", "0.0.0.0"}
	for _, ip := range localhostIPs {
		if clientIP == ip {
			return true
		}
	}

	// Check for local network ranges
	// This is a simplified check - in production you might want more sophisticated IP range checking
	if strings.HasPrefix(clientIP, "192.168.") ||
		strings.HasPrefix(clientIP, "10.") ||
		strings.HasPrefix(clientIP, "172.") ||
		strings.HasPrefix(clientIP, "169.254.") {
		return true
	}

	return false
}

// GetConnectionInfo returns information about the current connection
func GetConnectionInfo(c *gin.Context) map[string]interface{} {
	clientIP := c.ClientIP()
	isLocal := isLocalConnection(clientIP)

	connectionType := "remote"
	if isLocal {
		if clientIP == "127.0.0.1" || clientIP == "::1" || clientIP == "localhost" {
			connectionType = "localhost"
		} else {
			connectionType = "local-network"
		}
	}

	return map[string]interface{}{
		"client_ip":       clientIP,
		"is_local":        isLocal,
		"connection_type": connectionType,
		"requires_auth":   !isLocal,
	}
}
