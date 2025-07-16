package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/auth"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// Register handles user registration
func (h *Handlers) Register(c *gin.Context) {
	var request auth.RegisterRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	authService := auth.NewService(h.repos.User, h.cfg.Auth.JWTSecret, h.cfg.Auth.TokenExpiry, h.logger)

	userInfo, err := authService.Register(ctx, &request)
	if err != nil {
		h.logger.WithError(err).Errorf("Failed to register user: %s", request.Username)
		utils.SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "User registered successfully",
		"user":    userInfo,
	})
}

// Login handles user login
func (h *Handlers) Login(c *gin.Context) {
	var request auth.LoginRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	authService := auth.NewService(h.repos.User, h.cfg.Auth.JWTSecret, h.cfg.Auth.TokenExpiry, h.logger)

	loginResponse, err := authService.Login(ctx, &request)
	if err != nil {
		h.logger.WithError(err).Errorf("Failed login attempt: %s", request.Username)
		utils.SendError(c, http.StatusUnauthorized, err.Error())
		return
	}

	utils.SendSuccess(c, loginResponse)
}

// GetProfile returns the current user's profile
func (h *Handlers) GetProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.SendError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	userIDFloat, ok := userID.(float64)
	if !ok {
		utils.SendError(c, http.StatusInternalServerError, "Invalid user ID format")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	authService := auth.NewService(h.repos.User, h.cfg.Auth.JWTSecret, h.cfg.Auth.TokenExpiry, h.logger)

	userInfo, err := authService.GetUserByID(ctx, int(userIDFloat))
	if err != nil {
		h.logger.WithError(err).Errorf("Failed to get user profile: %f", userIDFloat)
		utils.SendError(c, http.StatusNotFound, "User not found")
		return
	}

	utils.SendSuccess(c, userInfo)
}

// UpdatePassword handles password update
func (h *Handlers) UpdatePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.SendError(c, http.StatusUnauthorized, "User not authenticated")
		return
	}

	userIDFloat, ok := userID.(float64)
	if !ok {
		utils.SendError(c, http.StatusInternalServerError, "Invalid user ID format")
		return
	}

	var request struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	authService := auth.NewService(h.repos.User, h.cfg.Auth.JWTSecret, h.cfg.Auth.TokenExpiry, h.logger)

	err := authService.UpdatePassword(ctx, int(userIDFloat), request.CurrentPassword, request.NewPassword)
	if err != nil {
		h.logger.WithError(err).Errorf("Failed to update password for user: %f", userIDFloat)
		utils.SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "Password updated successfully",
	})
}

// GetAllUsers returns all users (admin only)
func (h *Handlers) GetAllUsers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	authService := auth.NewService(h.repos.User, h.cfg.Auth.JWTSecret, h.cfg.Auth.TokenExpiry, h.logger)

	users, err := authService.GetAllUsers(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get all users")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve users")
		return
	}

	utils.SendSuccessWithMeta(c, users, gin.H{
		"count": len(users),
	})
}

// DeleteUser deletes a user (admin only)
func (h *Handlers) DeleteUser(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	authService := auth.NewService(h.repos.User, h.cfg.Auth.JWTSecret, h.cfg.Auth.TokenExpiry, h.logger)

	err = authService.DeleteUser(ctx, userID)
	if err != nil {
		h.logger.WithError(err).Errorf("Failed to delete user: %d", userID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "User deleted successfully",
		"user_id": userID,
	})
}

// ValidateToken validates a token and returns user info
func (h *Handlers) ValidateToken(c *gin.Context) {
	var request struct {
		Token string `json:"token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	authService := auth.NewService(h.repos.User, h.cfg.Auth.JWTSecret, h.cfg.Auth.TokenExpiry, h.logger)

	userInfo, err := authService.ValidateToken(request.Token)
	if err != nil {
		utils.SendError(c, http.StatusUnauthorized, "Invalid token")
		return
	}

	utils.SendSuccess(c, gin.H{
		"valid": true,
		"user":  userInfo,
	})
}
