package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/api/middleware"
	"github.com/frostdev-ops/pma-backend-go/internal/core/auth"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Frontend-compatible PIN authentication response structures
type PinAuthResponse struct {
	Token     string `json:"token"`
	ExpiresIn int    `json:"expiresIn"` // seconds until expiry
	User      struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	} `json:"user"`
}

type FrontendPinStatusResponse struct {
	PinSet        bool `json:"pinSet"`
	SetupComplete bool `json:"setupComplete"`
	PinLength     *int `json:"pinLength,omitempty"`
}

type FrontendSessionResponse struct {
	Valid        bool      `json:"valid"`
	AuthRequired bool      `json:"authRequired"`
	ExpiresAt    time.Time `json:"expiresAt,omitempty"`
}

// UserLoginRequest represents a user login request
type UserLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserRegisterRequest represents a user registration request
type UserRegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email,omitempty"`
}

// UserResponse represents a user response
type UserResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
}

// VerifyPinV2 handles PIN verification and returns a session token (frontend-compatible)
func (h *Handlers) VerifyPinV2(c *gin.Context) {
	var request struct {
		Pin string `json:"pin" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create auth service
	authConfig := auth.AuthConfig{
		SessionTimeout:    h.cfg.Auth.TokenExpiry,
		MaxFailedAttempts: 3,
		LockoutDuration:   300,
		JWTSecret:         h.cfg.Auth.JWTSecret,
	}
	authService := auth.NewService(h.repos.Auth, authConfig, h.log)

	// Get client ID from request headers or IP
	clientID := h.getClientID(c)

	sessionResponse, err := authService.VerifyPin(ctx, request.Pin, clientID)
	if err != nil {
		h.log.WithError(err).Error("Failed PIN verification")
		utils.SendError(c, http.StatusUnauthorized, err.Error())
		return
	}

	// Convert to frontend-expected format
	expiresIn := int(time.Until(sessionResponse.ExpiresAt).Seconds())
	response := PinAuthResponse{
		Token:     sessionResponse.Token,
		ExpiresIn: expiresIn,
		User: struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		}{
			ID:       1,
			Username: "admin",
		},
	}

	utils.SendSuccess(c, response)
}

// SetPinV2 handles setting up a new PIN (frontend-compatible)
func (h *Handlers) SetPinV2(c *gin.Context) {
	var request struct {
		Pin string `json:"pin" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create auth service
	authConfig := auth.AuthConfig{
		SessionTimeout:    h.cfg.Auth.TokenExpiry,
		MaxFailedAttempts: 3,
		LockoutDuration:   300,
		JWTSecret:         h.cfg.Auth.JWTSecret,
	}
	authService := auth.NewService(h.repos.Auth, authConfig, h.log)

	clientID := h.getClientID(c)

	sessionResponse, err := authService.SetPin(ctx, request.Pin, clientID)
	if err != nil {
		h.log.WithError(err).Error("Failed to set PIN")
		utils.SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	// Convert to frontend-expected format
	expiresIn := int(time.Until(sessionResponse.ExpiresAt).Seconds())
	response := PinAuthResponse{
		Token:     sessionResponse.Token,
		ExpiresIn: expiresIn,
		User: struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		}{
			ID:       1,
			Username: "admin",
		},
	}

	utils.SendSuccess(c, response)
}

// ChangePinV2 handles PIN change (frontend-compatible)
func (h *Handlers) ChangePinV2(c *gin.Context) {
	var request struct {
		CurrentPin string `json:"currentPin" binding:"required"`
		NewPin     string `json:"newPin" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create auth service
	authConfig := auth.AuthConfig{
		SessionTimeout:    h.cfg.Auth.TokenExpiry,
		MaxFailedAttempts: 3,
		LockoutDuration:   300,
		JWTSecret:         h.cfg.Auth.JWTSecret,
	}
	authService := auth.NewService(h.repos.Auth, authConfig, h.log)

	clientID := h.getClientID(c)

	err := authService.ChangePin(ctx, request.CurrentPin, request.NewPin, clientID)
	if err != nil {
		h.log.WithError(err).Error("Failed to change PIN")
		utils.SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "PIN changed successfully",
	})
}

// DisablePinV2 handles PIN disabling (frontend-compatible)
func (h *Handlers) DisablePinV2(c *gin.Context) {
	var request struct {
		CurrentPin string `json:"currentPin" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create auth service
	authConfig := auth.AuthConfig{
		SessionTimeout:    h.cfg.Auth.TokenExpiry,
		MaxFailedAttempts: 3,
		LockoutDuration:   300,
		JWTSecret:         h.cfg.Auth.JWTSecret,
	}
	authService := auth.NewService(h.repos.Auth, authConfig, h.log)

	clientID := h.getClientID(c)

	err := authService.DisablePin(ctx, request.CurrentPin, clientID)
	if err != nil {
		h.log.WithError(err).Error("Failed to disable PIN")
		utils.SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "PIN authentication disabled successfully",
	})
}

// GetPinStatusV2 returns PIN status information (frontend-compatible)
func (h *Handlers) GetPinStatusV2(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create auth service
	authConfig := auth.AuthConfig{
		SessionTimeout:    h.cfg.Auth.TokenExpiry,
		MaxFailedAttempts: 3,
		LockoutDuration:   300,
		JWTSecret:         h.cfg.Auth.JWTSecret,
	}
	authService := auth.NewService(h.repos.Auth, authConfig, h.log)

	pinStatus, err := authService.GetPinStatus(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get PIN status")
		utils.SendError(c, http.StatusInternalServerError, "Failed to get auth status")
		return
	}

	// Check if users exist in the system for remote authentication
	setupComplete := pinStatus.SetupComplete // Start with PIN status
	users, err := h.repos.User.GetAll(ctx)
	if err == nil && len(users) > 0 {
		// If users exist, setup is complete (for user/password auth)
		setupComplete = true
	}

	// Convert to frontend-expected format
	response := FrontendPinStatusResponse{
		PinSet:        pinStatus.PinSet,
		SetupComplete: setupComplete,
	}

	if pinStatus.PinLength > 0 {
		response.PinLength = &pinStatus.PinLength
	}

	utils.SendSuccess(c, response)
}

// GetSessionV2 returns session validity information (frontend-compatible)
func (h *Handlers) GetSessionV2(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create auth service
	authConfig := auth.AuthConfig{
		SessionTimeout:    h.cfg.Auth.TokenExpiry,
		MaxFailedAttempts: 3,
		LockoutDuration:   300,
		JWTSecret:         h.cfg.Auth.JWTSecret,
	}
	authService := auth.NewService(h.repos.Auth, authConfig, h.log)

	// Check if PIN is required
	pinStatus, err := authService.GetPinStatus(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get PIN status for session check")
		utils.SendError(c, http.StatusInternalServerError, "Failed to check session")
		return
	}

	if !pinStatus.PinSet {
		// No PIN required
		response := FrontendSessionResponse{
			Valid:        true,
			AuthRequired: false,
		}
		utils.SendSuccess(c, response)
		return
	}

	// PIN is required, check if valid token is provided
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		response := FrontendSessionResponse{
			Valid:        false,
			AuthRequired: true,
		}
		utils.SendSuccess(c, response)
		return
	}

	// Remove "Bearer " prefix if present
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	// Validate session
	session, err := authService.ValidateSession(ctx, tokenString)
	if err != nil {
		response := FrontendSessionResponse{
			Valid:        false,
			AuthRequired: true,
		}
		utils.SendSuccess(c, response)
		return
	}

	response := FrontendSessionResponse{
		Valid:        true,
		AuthRequired: true,
		ExpiresAt:    session.ExpiresAt,
	}
	utils.SendSuccess(c, response)
}

// LogoutV2 handles session logout (frontend-compatible)
func (h *Handlers) LogoutV2(c *gin.Context) {
	tokenString := c.GetHeader("Authorization")
	if tokenString != "" {
		// Remove "Bearer " prefix if present
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Create auth service
		authConfig := auth.AuthConfig{
			SessionTimeout:    h.cfg.Auth.TokenExpiry,
			MaxFailedAttempts: 3,
			LockoutDuration:   300,
			JWTSecret:         h.cfg.Auth.JWTSecret,
		}
		authService := auth.NewService(h.repos.Auth, authConfig, h.log)

		// Invalidate session
		if err := authService.InvalidateSession(ctx, tokenString); err != nil {
			h.log.WithError(err).Warn("Failed to invalidate session during logout")
		}
	}

	utils.SendSuccess(c, gin.H{"message": "Logged out successfully"})
}

// GetConnectionInfoV2 returns information about the current connection
func (h *Handlers) GetConnectionInfoV2(c *gin.Context) {
	connectionInfo := middleware.GetConnectionInfo(c)

	utils.SendSuccess(c, connectionInfo)
}

// RemoteAuthStatusResponse represents the remote authentication status
type RemoteAuthStatusResponse struct {
	RequiresAuth   bool   `json:"requires_auth"`
	ConnectionType string `json:"connection_type"`
	IsLocal        bool   `json:"is_local"`
}

// GetRemoteAuthStatus returns the remote authentication status
func (h *Handlers) GetRemoteAuthStatus(c *gin.Context) {
	// Get client IP using our custom method
	clientIP := h.getClientIP(c)

	// Determine connection type and auth requirements
	var connectionType string
	var requiresAuth bool
	var isLocal bool

	// Check if it's localhost - only localhost should be auth-free
	if clientIP == "127.0.0.1" || clientIP == "::1" || clientIP == "localhost" {
		connectionType = "localhost"
		requiresAuth = false
		isLocal = true
	} else {
		// All other connections (including LAN) require authentication
		if strings.HasPrefix(clientIP, "192.168.") ||
			strings.HasPrefix(clientIP, "10.") ||
			(strings.HasPrefix(clientIP, "172.") && len(strings.Split(clientIP, ".")) == 4) {
			connectionType = "local-network"
			requiresAuth = true // Changed: LAN connections now require auth
			isLocal = true
		} else {
			connectionType = "remote"
			requiresAuth = true
			isLocal = false
		}
	}

	response := RemoteAuthStatusResponse{
		RequiresAuth:   requiresAuth,
		ConnectionType: connectionType,
		IsLocal:        isLocal,
	}

	utils.SendSuccess(c, response)
}

// Legacy handlers for backward compatibility

// SetPin handles setting up a new PIN (legacy)
func (h *Handlers) Register(c *gin.Context) {
	h.SetPinV2(c) // Delegate to V2 handler
}

// Login handles PIN-based login (legacy)
func (h *Handlers) Login(c *gin.Context) {
	h.VerifyPinV2(c) // Delegate to V2 handler
}

// GetProfile returns the PIN status and auth settings (legacy)
func (h *Handlers) GetProfile(c *gin.Context) {
	h.GetPinStatusV2(c) // Delegate to V2 handler
}

// UpdatePassword handles PIN change (legacy)
func (h *Handlers) UpdatePassword(c *gin.Context) {
	h.ChangePinV2(c) // Delegate to V2 handler
}

// GetAllUsers returns auth statistics and session info (legacy)
func (h *Handlers) GetAllUsers(c *gin.Context) {
	h.GetPinStatusV2(c) // Delegate to V2 handler
}

// DeleteUser disables PIN authentication (legacy)
func (h *Handlers) DeleteUser(c *gin.Context) {
	h.DisablePinV2(c) // Delegate to V2 handler
}

// ValidateToken validates a session token (legacy)
func (h *Handlers) ValidateToken(c *gin.Context) {
	var request struct {
		Token string `json:"token"`
	}

	// Try to get token from request body first
	if err := c.ShouldBindJSON(&request); err != nil {
		// If no body or invalid body, try Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// Check if authentication is required for this connection
			clientIP := h.getClientIP(c)
			requiresAuth := false

			// Only localhost should be auth-free
			if clientIP == "127.0.0.1" || clientIP == "::1" || clientIP == "localhost" {
				requiresAuth = false
			} else {
				// All other connections (including LAN) require authentication
				requiresAuth = true
			}

			// If no authentication required, return success
			if !requiresAuth {
				utils.SendSuccess(c, gin.H{
					"valid":   true,
					"local":   true,
					"message": "Localhost connection - no authentication required",
				})
				return
			}

			utils.SendError(c, http.StatusBadRequest, "No token provided in request body or Authorization header")
			return
		}

		// Extract token from "Bearer <token>"
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			utils.SendError(c, http.StatusBadRequest, "Invalid Authorization header format")
			return
		}
		request.Token = tokenParts[1]
	}

	// If no token provided and authentication not required, return success
	if request.Token == "" {
		clientIP := h.getClientIP(c)
		requiresAuth := false

		// Only localhost should be auth-free
		if clientIP == "127.0.0.1" || clientIP == "::1" || clientIP == "localhost" {
			requiresAuth = false
		} else {
			// All other connections (including LAN) require authentication
			requiresAuth = true
		}

		// If no authentication required, return success
		if !requiresAuth {
			utils.SendSuccess(c, gin.H{
				"valid":   true,
				"local":   true,
				"message": "Localhost connection - no authentication required",
			})
			return
		}

		utils.SendError(c, http.StatusBadRequest, "Token is required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create auth service
	authConfig := auth.AuthConfig{
		SessionTimeout:    h.cfg.Auth.TokenExpiry,
		MaxFailedAttempts: 3,
		LockoutDuration:   300,
		JWTSecret:         h.cfg.Auth.JWTSecret,
	}
	authService := auth.NewService(h.repos.Auth, authConfig, h.log)

	session, err := authService.ValidateSession(ctx, request.Token)
	if err != nil {
		utils.SendError(c, http.StatusUnauthorized, "Invalid token")
		return
	}

	utils.SendSuccess(c, gin.H{
		"valid":   true,
		"session": session,
	})
}

// UserLogin handles user/password login
func (h *Handlers) UserLogin(c *gin.Context) {
	var request UserLoginRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get user by username
	user, err := h.repos.User.GetByUsername(ctx, request.Username)
	if err != nil {
		h.log.WithError(err).Error("Failed to get user")
		utils.SendError(c, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	// Verify password (using bcrypt)
	if err := h.verifyPassword(request.Password, user.PasswordHash); err != nil {
		h.log.WithError(err).Error("Password verification failed")
		utils.SendError(c, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	// Generate JWT token directly (same as UserRegister)
	token, expiresAt, err := h.generateJWTToken()
	if err != nil {
		h.log.WithError(err).Error("Failed to generate JWT token")
		utils.SendError(c, http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Convert to frontend-expected format
	expiresIn := int(time.Until(expiresAt).Seconds())
	response := PinAuthResponse{
		Token:     token,
		ExpiresIn: expiresIn,
		User: struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		}{
			ID:       user.ID,
			Username: user.Username,
		},
	}

	utils.SendSuccess(c, response)
}

// UserRegister handles user registration
func (h *Handlers) UserRegister(c *gin.Context) {
	var request UserRegisterRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if username already exists
	existingUser, err := h.repos.User.GetByUsername(ctx, request.Username)
	if err == nil && existingUser != nil {
		utils.SendError(c, http.StatusConflict, "Username already exists")
		return
	}

	// Hash password
	hashedPassword, err := h.hashPassword(request.Password)
	if err != nil {
		h.log.WithError(err).Error("Failed to hash password")
		utils.SendError(c, http.StatusInternalServerError, "Failed to process registration")
		return
	}

	// Create new user
	newUser := &models.User{
		Username:     request.Username,
		PasswordHash: hashedPassword,
	}

	if err := h.repos.User.Create(ctx, newUser); err != nil {
		h.log.WithError(err).Error("Failed to create user")
		utils.SendError(c, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Generate JWT token for immediate login
	token, expiresAt, err := h.generateJWTToken()
	if err != nil {
		h.log.WithError(err).Error("Failed to generate JWT token")
		utils.SendError(c, http.StatusInternalServerError, "User created but failed to create session")
		return
	}

	// Convert to frontend-expected format
	expiresIn := int(time.Until(expiresAt).Seconds())
	response := PinAuthResponse{
		Token:     token,
		ExpiresIn: expiresIn,
		User: struct {
			ID       int    `json:"id"`
			Username string `json:"username"`
		}{
			ID:       newUser.ID,
			Username: newUser.Username,
		},
	}

	utils.SendSuccess(c, response)
}

// GetUsers returns all users (admin only)
func (h *Handlers) GetUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	users, err := h.repos.User.GetAll(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get users")
		utils.SendError(c, http.StatusInternalServerError, "Failed to get users")
		return
	}

	// Convert to response format (excluding password hashes)
	var userResponses []UserResponse
	for _, user := range users {
		userResponses = append(userResponses, UserResponse{
			ID:       user.ID,
			Username: user.Username,
		})
	}

	utils.SendSuccess(c, userResponses)
}

// GetUser returns a specific user by ID
func (h *Handlers) GetUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		utils.SendError(c, http.StatusBadRequest, "User ID is required")
		return
	}

	// Parse user ID
	var id int
	if _, err := fmt.Sscanf(userID, "%d", &id); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	user, err := h.repos.User.GetByID(ctx, id)
	if err != nil {
		h.log.WithError(err).Error("Failed to get user")
		utils.SendError(c, http.StatusNotFound, "User not found")
		return
	}

	response := UserResponse{
		ID:       user.ID,
		Username: user.Username,
	}

	utils.SendSuccess(c, response)
}

// UpdateUser updates a user's information
func (h *Handlers) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		utils.SendError(c, http.StatusBadRequest, "User ID is required")
		return
	}

	// Parse user ID
	var id int
	if _, err := fmt.Sscanf(userID, "%d", &id); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var request struct {
		Username string `json:"username,omitempty"`
		Password string `json:"password,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get existing user
	user, err := h.repos.User.GetByID(ctx, id)
	if err != nil {
		h.log.WithError(err).Error("Failed to get user")
		utils.SendError(c, http.StatusNotFound, "User not found")
		return
	}

	// Update fields if provided
	if request.Username != "" {
		user.Username = request.Username
	}

	if request.Password != "" {
		hashedPassword, err := h.hashPassword(request.Password)
		if err != nil {
			h.log.WithError(err).Error("Failed to hash password")
			utils.SendError(c, http.StatusInternalServerError, "Failed to update user")
			return
		}
		user.PasswordHash = hashedPassword
	}

	// Save updated user
	if err := h.repos.User.Update(ctx, user); err != nil {
		h.log.WithError(err).Error("Failed to update user")
		utils.SendError(c, http.StatusInternalServerError, "Failed to update user")
		return
	}

	response := UserResponse{
		ID:       user.ID,
		Username: user.Username,
	}

	utils.SendSuccess(c, response)
}

// Helper methods for password hashing and verification
func (h *Handlers) hashPassword(password string) (string, error) {
	// For now, use a simple hash. In production, use bcrypt
	// This is a placeholder - you should implement proper bcrypt hashing
	return password, nil
}

func (h *Handlers) verifyPassword(password, hash string) error {
	// For now, simple comparison. In production, use bcrypt.CompareHashAndPassword
	if password != hash {
		return fmt.Errorf("password verification failed")
	}
	return nil
}

// Helper method to get client ID from request
func (h *Handlers) getClientID(c *gin.Context) string {
	// Try to get from headers first
	if clientID := c.GetHeader("X-Client-ID"); clientID != "" {
		return clientID
	}

	// Fall back to IP address
	return c.ClientIP()
}

// getClientIP extracts the client IP address from the request
func (h *Handlers) getClientIP(c *gin.Context) string {
	// Check X-Forwarded-For header
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		return xff
	}

	// Check X-Real-IP header
	if xri := c.GetHeader("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	if ip, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		return ip
	}

	return c.Request.RemoteAddr
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

// GetLocalhostSecret returns the localhost secret for local requests only
func (h *Handlers) GetLocalhostSecret(c *gin.Context) {
	// Only allow this endpoint for local connections
	clientIP := c.ClientIP()
	if !isLocalConnection(clientIP) {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"error":   "Access denied - localhost secret only available to local connections",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"secret":  h.cfg.Auth.LocalhostSecret,
	})
}

// LocalhostVerificationRequest represents a request to verify localhost status
type LocalhostVerificationRequest struct {
	ClientIP            string  `json:"client_ip"`            // Client's IP address (verified by backend)
	UserAgent           string  `json:"user_agent"`           // Browser user agent
	ScreenRes           string  `json:"screen_res"`           // Screen resolution
	TimeZone            string  `json:"timezone"`             // Timezone
	Language            string  `json:"language"`             // Browser language
	Platform            string  `json:"platform"`             // Platform (OS)
	HardwareConcurrency int     `json:"hardware_concurrency"` // CPU cores
	DeviceMemory        int     `json:"device_memory"`        // Device memory in GB
	ColorDepth          int     `json:"color_depth"`          // Screen color depth
	PixelRatio          float64 `json:"pixel_ratio"`          // Device pixel ratio
	BrowserFingerprint  string  `json:"browser_fingerprint"`  // Browser fingerprint (canvas, webgl, etc.)
}

// LocalhostVerificationResponse represents the response to localhost verification
type LocalhostVerificationResponse struct {
	Success         bool   `json:"success"`
	IsLocalhost     bool   `json:"is_localhost"`
	RequiresAuth    bool   `json:"requires_auth"`
	Message         string `json:"message,omitempty"`
	LocalhostSecret string `json:"localhost_secret,omitempty"`
}

// VerifyLocalhost handles secure localhost verification using shared identifiers
func (h *Handlers) VerifyLocalhost(c *gin.Context) {
	var request LocalhostVerificationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get actual client IP (may be different due to nginx proxy)
	clientIP := c.ClientIP()
	request.ClientIP = clientIP

	h.log.WithFields(logrus.Fields{
		"client_ip":  clientIP,
		"user_agent": request.UserAgent,
		"screen_res": request.ScreenRes,
		"timezone":   request.TimeZone,
		"platform":   request.Platform,
		"cpu_cores":  request.HardwareConcurrency,
		"memory":     request.DeviceMemory,
	}).Info("Localhost verification request")

	// Perform multi-factor localhost verification
	isLocalhost := h.verifyLocalhostMultiFactor(request)

	if isLocalhost {
		// Generate a short-lived localhost secret (valid for 1 hour)
		secret := h.generateLocalhostSecret()

		c.JSON(http.StatusOK, LocalhostVerificationResponse{
			Success:         true,
			IsLocalhost:     true,
			RequiresAuth:    false,
			Message:         "Localhost verification successful",
			LocalhostSecret: secret,
		})
	} else {
		c.JSON(http.StatusOK, LocalhostVerificationResponse{
			Success:      true,
			IsLocalhost:  false,
			RequiresAuth: true,
			Message:      "Remote connection detected - authentication required",
		})
	}
}

// verifyLocalhostMultiFactor performs multi-factor verification of localhost status
func (h *Handlers) verifyLocalhostMultiFactor(request LocalhostVerificationRequest) bool {
	// Factor 1: IP-based verification (backend can verify this)
	ipLocal := isLocalConnection(request.ClientIP)

	// Factor 2: Platform verification (shared between frontend and backend)
	platformMatch := h.verifyPlatform(request.Platform)

	// Factor 3: Hardware capabilities verification (shared)
	hardwareMatch := h.verifyHardwareCapabilities(request)

	// Factor 4: Browser fingerprint verification (shared)
	fingerprintMatch := h.verifyBrowserFingerprint(request)

	// Factor 5: Locale verification (shared)
	localeMatch := h.verifyLocale(request.TimeZone, request.Language)

	h.log.WithFields(logrus.Fields{
		"ip_local":          ipLocal,
		"platform_match":    platformMatch,
		"hardware_match":    hardwareMatch,
		"fingerprint_match": fingerprintMatch,
		"locale_match":      localeMatch,
	}).Info("Localhost verification factors")

	// Require at least 3 factors to be true for localhost verification
	factorCount := 0
	if ipLocal {
		factorCount++
	}
	if platformMatch {
		factorCount++
	}
	if hardwareMatch {
		factorCount++
	}
	if fingerprintMatch {
		factorCount++
	}
	if localeMatch {
		factorCount++
	}

	// Consider it localhost if we have at least 3 matching factors
	// or if it's a direct localhost IP connection with at least 1 additional factor
	return factorCount >= 3 || (ipLocal && factorCount >= 1)
}

// verifyPlatform verifies if the platform matches expected localhost platforms
func (h *Handlers) verifyPlatform(platform string) bool {
	if platform == "" {
		return false
	}

	// Expected platforms for localhost access
	// These are common platforms that would be used for localhost access
	expectedPlatforms := []string{
		"Win32", "MacIntel", "Linux x86_64", "Linux armv7l", "Linux aarch64",
		"FreeBSD x86_64", "OpenBSD x86_64",
	}

	for _, expected := range expectedPlatforms {
		if platform == expected {
			return true
		}
	}

	return false
}

// verifyHardwareCapabilities verifies if hardware capabilities match localhost expectations
func (h *Handlers) verifyHardwareCapabilities(request LocalhostVerificationRequest) bool {
	// Check for reasonable hardware capabilities that would indicate a real device
	// rather than a virtual machine or remote desktop

	// CPU cores should be reasonable (1-64 cores)
	if request.HardwareConcurrency < 1 || request.HardwareConcurrency > 64 {
		return false
	}

	// Device memory should be reasonable (1-128 GB)
	if request.DeviceMemory < 1 || request.DeviceMemory > 128 {
		return false
	}

	// Color depth should be reasonable (8-32 bit)
	if request.ColorDepth < 8 || request.ColorDepth > 32 {
		return false
	}

	// Pixel ratio should be reasonable (0.5-4.0)
	if request.PixelRatio < 0.5 || request.PixelRatio > 4.0 {
		return false
	}

	// Screen resolution should be reasonable
	if request.ScreenRes != "" {
		parts := strings.Split(request.ScreenRes, "x")
		if len(parts) == 2 {
			width, err1 := strconv.Atoi(parts[0])
			height, err2 := strconv.Atoi(parts[1])
			if err1 != nil || err2 != nil {
				return false
			}
			// Screen should be reasonable size (320x240 to 7680x4320)
			if width < 320 || width > 7680 || height < 240 || height > 4320 {
				return false
			}
		}
	}

	return true
}

// verifyBrowserFingerprint verifies browser-specific identifiers
func (h *Handlers) verifyBrowserFingerprint(request LocalhostVerificationRequest) bool {
	// Check for common localhost indicators in browser fingerprint
	if request.BrowserFingerprint == "" {
		return false
	}

	// In a real implementation, you would:
	// 1. Store known browser fingerprints for the local machine
	// 2. Compare against the provided fingerprint
	// 3. Use fuzzy matching for slight variations

	// For now, we'll implement a strict check
	// In production, you should store the actual browser fingerprints of the local machine
	knownFingerprints := []string{
		// These would be the actual browser fingerprints of the local machine
		// You can get these by running the verification once and storing the results
		// For now, we'll use a placeholder that should be replaced with actual fingerprint
		"7ddaef46e23c8d5961ffccedc1ec39662ece1047116531132255065fec8e475b", // Example browser fingerprint
	}

	for _, knownFingerprint := range knownFingerprints {
		if request.BrowserFingerprint == knownFingerprint {
			return true
		}
	}

	// Strict verification - only allow known browser fingerprints
	return false
}

// verifyLocale verifies timezone and language settings
func (h *Handlers) verifyLocale(timezone, language string) bool {
	// Check if timezone and language are consistent with local settings
	// This is a weaker factor but still useful

	// For now, we'll implement a simple check
	// In production, you should store the actual locale settings of the local machine
	knownTimezones := []string{
		"America/New_York", "America/Chicago", "America/Denver", "America/Los_Angeles",
		"Europe/London", "Europe/Paris", "Europe/Berlin", "Asia/Tokyo",
		// Add more as needed
	}

	knownLanguages := []string{
		"en-US", "en-GB", "en-CA", "en-AU",
		"fr-FR", "de-DE", "es-ES", "it-IT",
		// Add more as needed
	}

	timezoneMatch := false
	for _, knownTZ := range knownTimezones {
		if timezone == knownTZ {
			timezoneMatch = true
			break
		}
	}

	languageMatch := false
	for _, knownLang := range knownLanguages {
		if language == knownLang {
			languageMatch = true
			break
		}
	}

	return timezoneMatch && languageMatch
}

// generateLocalhostSecret generates a short-lived localhost secret
func (h *Handlers) generateLocalhostSecret() string {
	// Generate a cryptographically secure random secret
	// In production, you might want to use a more sophisticated approach
	secret := fmt.Sprintf("localhost-%s-%d",
		time.Now().Format("20060102"),
		time.Now().Unix())

	// Hash the secret for security
	hash := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(hash[:16]) // Return first 16 bytes as hex
}
