package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// RemoteAuthMiddleware provides authentication that varies based on connection type
// - Localhost connections: No authentication required
// - Local network connections: No authentication required
// - Remote connections: User/password authentication required
// - Localhost secret header: Secondary bypass for nginx-proxied local requests
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

		// Check for localhost secret header first (for nginx-proxied local requests)
		localhostSecret := c.GetHeader("X-Localhost-Secret")
		if localhostSecret != "" && localhostSecret == cfg.Auth.LocalhostSecret {
			c.Set("user_id", "1")
			c.Set("username", "localhost_secret")
			c.Set("auth_type", "localhost_secret_bypass")
			c.Set("localhost_secret_bypass", true)
			c.Next()
			return
		}

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

		tokenString := tokenParts[1]

		// Validate JWT token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.Auth.JWTSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid JWT token",
				"code":    "INVALID_JWT",
			})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid or expired JWT token",
				"code":    "INVALID_JWT",
			})
			c.Abort()
			return
		}

		// Extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// Check if token is expired
			if exp, ok := claims["exp"].(float64); ok {
				if time.Now().Unix() > int64(exp) {
					c.JSON(http.StatusUnauthorized, gin.H{
						"success": false,
						"error":   "JWT token has expired",
						"code":    "EXPIRED_JWT",
					})
					c.Abort()
					return
				}
			}

			// Set user context from JWT claims
			userID := "1"      // Default user ID
			username := "user" // Default username

			if userIDClaim, ok := claims["user_id"].(string); ok {
				userID = userIDClaim
			}
			if usernameClaim, ok := claims["username"].(string); ok {
				username = usernameClaim
			}

			c.Set("user_id", userID)
			c.Set("username", username)
			c.Set("auth_type", "jwt")
			c.Set("jwt_claims", claims)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid JWT claims",
				"code":    "INVALID_JWT_CLAIMS",
			})
			c.Abort()
			return
		}

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

		// Validate JWT token
		parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.Auth.JWTSecret), nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid JWT token",
				"code":    "INVALID_JWT",
			})
			c.Abort()
			return
		}

		if !parsedToken.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid or expired JWT token",
				"code":    "INVALID_JWT",
			})
			c.Abort()
			return
		}

		// Extract claims
		if claims, ok := parsedToken.Claims.(jwt.MapClaims); ok {
			// Check if token is expired
			if exp, ok := claims["exp"].(float64); ok {
				if time.Now().Unix() > int64(exp) {
					c.JSON(http.StatusUnauthorized, gin.H{
						"success": false,
						"error":   "JWT token has expired",
						"code":    "EXPIRED_JWT",
					})
					c.Abort()
					return
				}
			}

			// Set user context from JWT claims
			userID := "1"      // Default user ID
			username := "user" // Default username

			if userIDClaim, ok := claims["user_id"].(string); ok {
				userID = userIDClaim
			}
			if usernameClaim, ok := claims["username"].(string); ok {
				username = usernameClaim
			}

			c.Set("user_id", userID)
			c.Set("username", username)
			c.Set("auth_type", "jwt")
			c.Set("jwt_claims", claims)
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid JWT claims",
				"code":    "INVALID_JWT_CLAIMS",
			})
			c.Abort()
			return
		}

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
