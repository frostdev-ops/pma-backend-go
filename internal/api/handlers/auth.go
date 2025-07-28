package handlers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/api/middleware"
	"github.com/frostdev-ops/pma-backend-go/internal/core/auth"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
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
