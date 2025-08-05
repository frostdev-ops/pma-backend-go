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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid JWT token"})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired JWT token"})
			c.Abort()
			return
		}

		// Extract claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// Check if token is expired
			if exp, ok := claims["exp"].(float64); ok {
				if time.Now().Unix() > int64(exp) {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "JWT token has expired"})
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid JWT claims"})
			c.Abort()
			return
		}

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
			c.Set("user_id", "1")
			c.Set("username", "localhost")
			c.Set("auth_type", "localhost_bypass")
			c.Next()
			return
		}

		// Check for API secret header
		apiSecret := c.GetHeader("X-API-Secret")
		if apiSecret != "" && apiSecret == cfg.Auth.APISecret {
			c.Set("user_id", "1")
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
				tokenString := tokenParts[1]

				// Validate JWT token
				token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, jwt.ErrSignatureInvalid
					}
					return []byte(cfg.Auth.JWTSecret), nil
				})

				if err == nil && token.Valid {
					if claims, ok := token.Claims.(jwt.MapClaims); ok {
						// Check expiration
						if exp, ok := claims["exp"].(float64); ok {
							if time.Now().Unix() <= int64(exp) {
								userID := "1"
								username := "user"

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
							}
						}
					}
				}
			}
		}

		// Continue regardless of auth status for optional auth
		c.Next()
	}
}
