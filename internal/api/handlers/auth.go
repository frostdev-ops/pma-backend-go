package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/auth"
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

	// Convert to frontend-expected format
	response := FrontendPinStatusResponse{
		PinSet:        pinStatus.PinSet,
		SetupComplete: pinStatus.SetupComplete,
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
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
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

// Helper method to get client ID from request
func (h *Handlers) getClientID(c *gin.Context) string {
	// Try to get from headers first
	if clientID := c.GetHeader("X-Client-ID"); clientID != "" {
		return clientID
	}

	// Fall back to IP address
	return c.ClientIP()
}
