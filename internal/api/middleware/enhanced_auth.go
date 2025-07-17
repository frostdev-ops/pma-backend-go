package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AuthService interface for dependency injection
type AuthService interface {
	ValidateSession(ctx context.Context, token string) (*models.Session, error)
	HasPin(ctx context.Context) (bool, error)
}

// SessionInfo represents session information
type SessionInfo struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	ClientID  string    `json:"client_id,omitempty"`
}

// EnhancedAuthMiddleware provides PIN-based authentication middleware
type EnhancedAuthMiddleware struct {
	authService AuthService
	logger      *logrus.Logger
}

// NewEnhancedAuthMiddleware creates a new enhanced authentication middleware
func NewEnhancedAuthMiddleware(authService AuthService, logger *logrus.Logger) *EnhancedAuthMiddleware {
	return &EnhancedAuthMiddleware{
		authService: authService,
		logger:      logger,
	}
}

// RequiredAuth middleware that always requires authentication
func (m *EnhancedAuthMiddleware) RequiredAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token == "" {
			m.logger.WithField("path", c.Request.URL.Path).Warn("Missing authentication token")
			utils.SendError(c, http.StatusUnauthorized, "Authentication token required")
			c.Abort()
			return
		}

		// Validate session
		session, err := m.authService.ValidateSession(c.Request.Context(), token)
		if err != nil {
			m.logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"path":  c.Request.URL.Path,
				"ip":    m.getClientIP(c),
			}).Warn("Invalid authentication token")
			utils.SendError(c, http.StatusUnauthorized, "Invalid or expired authentication token")
			c.Abort()
			return
		}

		// Store session info in context
		c.Set("session", session)
		c.Set("authenticated", true)
		c.Set("auth_token", token)

		// Log successful authentication
		m.logger.WithFields(logrus.Fields{
			"path": c.Request.URL.Path,
			"ip":   m.getClientIP(c),
		}).Debug("Authentication successful")

		c.Next()
	}
}

// ConditionalAuth middleware that only requires authentication if PIN is set
func (m *EnhancedAuthMiddleware) ConditionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if PIN is set
		hasPin, err := m.authService.HasPin(c.Request.Context())
		if err != nil {
			m.logger.WithError(err).Error("Failed to check PIN status")
			utils.SendError(c, http.StatusInternalServerError, "Authentication service error")
			c.Abort()
			return
		}

		// If no PIN is set, skip authentication
		if !hasPin {
			m.logger.WithField("path", c.Request.URL.Path).Debug("No PIN set, skipping authentication")
			c.Set("authenticated", false)
			c.Set("pin_required", false)
			c.Next()
			return
		}

		// PIN is set, require authentication
		token := m.extractToken(c)
		if token == "" {
			m.logger.WithField("path", c.Request.URL.Path).Warn("Authentication required - PIN is set")
			utils.SendError(c, http.StatusUnauthorized, "Authentication required - PIN is set")
			c.Abort()
			return
		}

		// Validate session
		session, err := m.authService.ValidateSession(c.Request.Context(), token)
		if err != nil {
			m.logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"path":  c.Request.URL.Path,
				"ip":    m.getClientIP(c),
			}).Warn("Invalid authentication token")
			utils.SendError(c, http.StatusUnauthorized, "Invalid or expired authentication token")
			c.Abort()
			return
		}

		// Store session info in context
		c.Set("session", session)
		c.Set("authenticated", true)
		c.Set("pin_required", true)
		c.Set("auth_token", token)

		// Log successful authentication
		m.logger.WithFields(logrus.Fields{
			"path": c.Request.URL.Path,
			"ip":   m.getClientIP(c),
		}).Debug("Conditional authentication successful")

		c.Next()
	}
}

// OptionalAuth middleware that provides session info if token is present but doesn't require it
func (m *EnhancedAuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := m.extractToken(c)
		if token == "" {
			c.Set("authenticated", false)
			c.Next()
			return
		}

		// Validate session if token is provided
		session, err := m.authService.ValidateSession(c.Request.Context(), token)
		if err != nil {
			// Log but don't fail request
			m.logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"path":  c.Request.URL.Path,
				"ip":    m.getClientIP(c),
			}).Debug("Optional authentication failed")
			c.Set("authenticated", false)
			c.Next()
			return
		}

		// Store session info in context
		c.Set("session", session)
		c.Set("authenticated", true)
		c.Set("auth_token", token)

		m.logger.WithFields(logrus.Fields{
			"path": c.Request.URL.Path,
			"ip":   m.getClientIP(c),
		}).Debug("Optional authentication successful")

		c.Next()
	}
}

// extractToken extracts the authentication token from the request
func (m *EnhancedAuthMiddleware) extractToken(c *gin.Context) string {
	// Try Authorization header first (Bearer token)
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Try X-Auth-Token header
	if token := c.GetHeader("X-Auth-Token"); token != "" {
		return token
	}

	// Try query parameter (for WebSocket upgrades or special cases)
	if token := c.Query("token"); token != "" {
		return token
	}

	return ""
}

// getClientIP extracts the client IP address from the request
func (m *EnhancedAuthMiddleware) getClientIP(c *gin.Context) string {
	// Try X-Forwarded-For header first (for reverse proxies)
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if net.ParseIP(ip) != nil {
				return ip
			}
		}
	}

	// Try X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// Fall back to RemoteAddr
	if ip, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		return ip
	}

	return c.Request.RemoteAddr
}

// GetSession retrieves session information from the context
func GetSession(c *gin.Context) (*models.Session, bool) {
	if session, exists := c.Get("session"); exists {
		if s, ok := session.(*models.Session); ok {
			return s, true
		}
	}
	return nil, false
}

// IsAuthenticated checks if the request is authenticated
func IsAuthenticated(c *gin.Context) bool {
	if auth, exists := c.Get("authenticated"); exists {
		if authenticated, ok := auth.(bool); ok {
			return authenticated
		}
	}
	return false
}

// RequiresPIN checks if PIN authentication is required
func RequiresPIN(c *gin.Context) bool {
	if pinRequired, exists := c.Get("pin_required"); exists {
		if required, ok := pinRequired.(bool); ok {
			return required
		}
	}
	return false
}

// GetAuthToken retrieves the authentication token from the context
func GetAuthToken(c *gin.Context) (string, bool) {
	if token, exists := c.Get("auth_token"); exists {
		if t, ok := token.(string); ok {
			return t, true
		}
	}
	return "", false
}

// GetClientID retrieves a client identifier (currently IP address)
func GetClientID(c *gin.Context) string {
	// Use IP address as client ID
	if ip, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		return ip
	}
	return c.Request.RemoteAddr
}

// Helper function for handlers to check authentication status
func CheckAuthenticationStatus(c *gin.Context) (authenticated bool, hasPin bool, session *models.Session) {
	authenticated = IsAuthenticated(c)
	hasPin = RequiresPIN(c)
	session, _ = GetSession(c)
	return
}
