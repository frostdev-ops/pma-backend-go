package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/auth"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// SetPin handles setting up a new PIN
func (h *Handlers) Register(c *gin.Context) {
	var request struct {
		Pin      string `json:"pin" binding:"required"`
		ClientID string `json:"client_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create auth config from server config
	authConfig := auth.AuthConfig{
		SessionTimeout:    h.cfg.Auth.TokenExpiry,
		MaxFailedAttempts: 3,
		LockoutDuration:   300,
		JWTSecret:         h.cfg.Auth.JWTSecret,
	}

	authService := auth.NewService(h.repos.Auth, authConfig, h.log)

	sessionResponse, err := authService.SetPin(ctx, request.Pin, request.ClientID)
	if err != nil {
		h.log.WithError(err).Error("Failed to set PIN")
		utils.SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "PIN set successfully",
		"session": sessionResponse,
	})
}

// Login handles PIN-based login
func (h *Handlers) Login(c *gin.Context) {
	var request struct {
		Pin      string `json:"pin" binding:"required"`
		ClientID string `json:"client_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create auth config from server config
	authConfig := auth.AuthConfig{
		SessionTimeout:    h.cfg.Auth.TokenExpiry,
		MaxFailedAttempts: 3,
		LockoutDuration:   300,
		JWTSecret:         h.cfg.Auth.JWTSecret,
	}

	authService := auth.NewService(h.repos.Auth, authConfig, h.log)

	sessionResponse, err := authService.VerifyPin(ctx, request.Pin, request.ClientID)
	if err != nil {
		h.log.WithError(err).Error("Failed PIN verification")
		utils.SendError(c, http.StatusUnauthorized, err.Error())
		return
	}

	utils.SendSuccess(c, sessionResponse)
}

// GetProfile returns the PIN status and auth settings
func (h *Handlers) GetProfile(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create auth config from server config
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

	utils.SendSuccess(c, pinStatus)
}

// UpdatePassword handles PIN change
func (h *Handlers) UpdatePassword(c *gin.Context) {
	var request struct {
		CurrentPin string `json:"current_pin" binding:"required"`
		NewPin     string `json:"new_pin" binding:"required"`
		ClientID   string `json:"client_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create auth config from server config
	authConfig := auth.AuthConfig{
		SessionTimeout:    h.cfg.Auth.TokenExpiry,
		MaxFailedAttempts: 3,
		LockoutDuration:   300,
		JWTSecret:         h.cfg.Auth.JWTSecret,
	}

	authService := auth.NewService(h.repos.Auth, authConfig, h.log)

	err := authService.ChangePin(ctx, request.CurrentPin, request.NewPin, request.ClientID)
	if err != nil {
		h.log.WithError(err).Error("Failed to change PIN")
		utils.SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "PIN updated successfully",
	})
}

// GetAllUsers returns auth statistics and session info
func (h *Handlers) GetAllUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create auth config from server config
	authConfig := auth.AuthConfig{
		SessionTimeout:    h.cfg.Auth.TokenExpiry,
		MaxFailedAttempts: 3,
		LockoutDuration:   300,
		JWTSecret:         h.cfg.Auth.JWTSecret,
	}

	authService := auth.NewService(h.repos.Auth, authConfig, h.log)

	pinStatus, err := authService.GetPinStatus(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get auth info")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve auth info")
		return
	}

	utils.SendSuccess(c, gin.H{
		"pin_status": pinStatus,
		"message":    "PIN-based authentication system",
	})
}

// DeleteUser disables PIN authentication
func (h *Handlers) DeleteUser(c *gin.Context) {
	var request struct {
		CurrentPin string `json:"current_pin" binding:"required"`
		ClientID   string `json:"client_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create auth config from server config
	authConfig := auth.AuthConfig{
		SessionTimeout:    h.cfg.Auth.TokenExpiry,
		MaxFailedAttempts: 3,
		LockoutDuration:   300,
		JWTSecret:         h.cfg.Auth.JWTSecret,
	}

	authService := auth.NewService(h.repos.Auth, authConfig, h.log)

	err := authService.DisablePin(ctx, request.CurrentPin, request.ClientID)
	if err != nil {
		h.log.WithError(err).Error("Failed to disable PIN")
		utils.SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "PIN authentication disabled successfully",
	})
}

// ValidateToken validates a session token
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

	// Create auth config from server config
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
