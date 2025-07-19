package middleware

import (
	"net/http"
	"strings"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// AuthMiddleware validates JWT tokens (strict auth)
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		tokenString := tokenParts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			utils.SendError(c, http.StatusUnauthorized, "Invalid token")
			c.Abort()
			return
		}

		// Store claims in context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("user_id", claims["user_id"])
			c.Set("username", claims["username"])
		}

		c.Next()
	}
}

// OptionalAuthMiddleware provides optional authentication based on configuration
func OptionalAuthMiddleware(cfg *config.Config, configRepo repositories.ConfigRepository, logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Debug: Log that OptionalAuthMiddleware is being called
		logger.Infof("DEBUG: OptionalAuthMiddleware called for path %s, Auth.Enabled = %t", c.Request.URL.Path, cfg.Auth.Enabled)

		// TEMPORARY: Force auth to be disabled for debugging
		// If auth is completely disabled in config, allow through
		if !cfg.Auth.Enabled || true { // Always allow through for debugging
			logger.Infof("DEBUG: Auth disabled (or forced disabled), allowing request through for path %s", c.Request.URL.Path)
			c.Next()
			return
		}

		// Check if PIN authentication is configured
		_, err := configRepo.Get(c.Request.Context(), "auth.pin_hash")
		pinAuthEnabled := err == nil

		// If PIN auth is not configured, allow through (auth is optional)
		if !pinAuthEnabled {
			c.Next()
			return
		}

		// PIN auth is enabled, so check for token
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

		tokenString := tokenParts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.Auth.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			utils.SendError(c, http.StatusUnauthorized, "Invalid token")
			c.Abort()
			return
		}

		// Store claims in context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("user_id", claims["user_id"])
			c.Set("username", claims["username"])
		}

		c.Next()
	}
}
