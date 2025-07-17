package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// PinAuthRequest represents a PIN authentication request
type PinAuthRequest struct {
	Pin string `json:"pin" binding:"required"`
}

// SetPinRequest represents a set PIN request
type SetPinRequest struct {
	Pin string `json:"pin" binding:"required,min=4,max=10"`
}

// ChangePinRequest represents a change PIN request
type ChangePinRequest struct {
	CurrentPin string `json:"currentPin" binding:"required"`
	NewPin     string `json:"newPin" binding:"required,min=4,max=10"`
}

// PinStatusResponse represents PIN status response
type PinStatusResponse struct {
	HasPin    bool `json:"hasPin"`
	IsEnabled bool `json:"isEnabled"`
}

// SessionResponse represents session info response
type SessionResponse struct {
	Valid        bool      `json:"valid"`
	AuthRequired bool      `json:"authRequired"`
	ExpiresAt    time.Time `json:"expiresAt,omitempty"`
}

// AuthTokenResponse represents auth token response
type AuthTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// VerifyPin handles PIN verification and returns a JWT token
func (h *Handlers) VerifyPin(c *gin.Context) {
	var req PinAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Get stored PIN hash from config
	storedPinConfig, err := h.repos.Config.Get(c.Request.Context(), "auth.pin_hash")
	if err != nil {
		h.log.WithError(err).Debug("No PIN configured")
		utils.SendError(c, http.StatusUnauthorized, "PIN authentication not configured")
		return
	}

	// Hash the provided PIN
	hasher := sha256.New()
	hasher.Write([]byte(req.Pin))
	providedHash := hex.EncodeToString(hasher.Sum(nil))

	// Compare hashes
	if providedHash != storedPinConfig.Value {
		h.log.Warn("Invalid PIN attempt")
		utils.SendError(c, http.StatusUnauthorized, "Invalid PIN")
		return
	}

	// Generate JWT token
	token, expiresAt, err := h.generateJWTToken()
	if err != nil {
		h.log.WithError(err).Error("Failed to generate JWT token")
		utils.SendError(c, http.StatusInternalServerError, "Failed to generate authentication token")
		return
	}

	// Store session info (optional - could be stored in memory cache)
	sessionKey := "session." + token[:16] // Use first 16 chars as session key
	sessionData := map[string]interface{}{
		"authenticated": true,
		"created_at":    time.Now(),
		"expires_at":    expiresAt,
	}

	// Store session in config (simple approach - in production might use Redis/memory cache)
	sessionJSON, _ := json.Marshal(sessionData)
	h.repos.Config.Set(c.Request.Context(), &models.SystemConfig{
		Key:         sessionKey,
		Value:       string(sessionJSON),
		Description: "User session data",
	})

	utils.SendSuccess(c, AuthTokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
	})
}

// SetPin sets a new PIN (first time setup)
func (h *Handlers) SetPin(c *gin.Context) {
	var req SetPinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Check if PIN is already set
	_, err := h.repos.Config.Get(c.Request.Context(), "auth.pin_hash")
	if err == nil {
		utils.SendError(c, http.StatusConflict, "PIN is already configured")
		return
	}

	// Hash the PIN
	hasher := sha256.New()
	hasher.Write([]byte(req.Pin))
	pinHash := hex.EncodeToString(hasher.Sum(nil))

	// Store PIN hash
	err = h.repos.Config.Set(c.Request.Context(), &models.SystemConfig{
		Key:         "auth.pin_hash",
		Value:       pinHash,
		Encrypted:   false,
		Description: "Hashed PIN for authentication",
	})

	if err != nil {
		h.log.WithError(err).Error("Failed to store PIN hash")
		utils.SendError(c, http.StatusInternalServerError, "Failed to set PIN")
		return
	}

	h.log.Info("PIN authentication configured")
	utils.SendSuccess(c, gin.H{"message": "PIN set successfully"})
}

// ChangePin changes the current PIN (requires authentication)
func (h *Handlers) ChangePin(c *gin.Context) {
	var req ChangePinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Get stored PIN hash
	storedPinConfig, err := h.repos.Config.Get(c.Request.Context(), "auth.pin_hash")
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "PIN not configured")
		return
	}

	// Verify current PIN
	hasher := sha256.New()
	hasher.Write([]byte(req.CurrentPin))
	currentHash := hex.EncodeToString(hasher.Sum(nil))

	if currentHash != storedPinConfig.Value {
		utils.SendError(c, http.StatusUnauthorized, "Current PIN is incorrect")
		return
	}

	// Hash new PIN
	hasher = sha256.New()
	hasher.Write([]byte(req.NewPin))
	newHash := hex.EncodeToString(hasher.Sum(nil))

	// Update PIN hash
	err = h.repos.Config.Set(c.Request.Context(), &models.SystemConfig{
		Key:         "auth.pin_hash",
		Value:       newHash,
		Encrypted:   false,
		Description: "Hashed PIN for authentication",
	})

	if err != nil {
		h.log.WithError(err).Error("Failed to update PIN hash")
		utils.SendError(c, http.StatusInternalServerError, "Failed to change PIN")
		return
	}

	h.log.Info("PIN changed successfully")
	utils.SendSuccess(c, gin.H{"message": "PIN changed successfully"})
}

// DisablePin disables PIN authentication (requires authentication)
func (h *Handlers) DisablePin(c *gin.Context) {
	// Remove PIN hash from config
	err := h.repos.Config.Delete(c.Request.Context(), "auth.pin_hash")
	if err != nil {
		h.log.WithError(err).Error("Failed to disable PIN")
		utils.SendError(c, http.StatusInternalServerError, "Failed to disable PIN")
		return
	}

	h.log.Info("PIN authentication disabled")
	utils.SendSuccess(c, gin.H{"message": "PIN authentication disabled"})
}

// GetPinStatus returns whether PIN authentication is enabled
func (h *Handlers) GetPinStatus(c *gin.Context) {
	_, err := h.repos.Config.Get(c.Request.Context(), "auth.pin_hash")
	hasPin := err == nil

	utils.SendSuccess(c, PinStatusResponse{
		HasPin:    hasPin,
		IsEnabled: hasPin,
	})
}

// GetSession returns session information
func (h *Handlers) GetSession(c *gin.Context) {
	// Check if PIN is required
	_, err := h.repos.Config.Get(c.Request.Context(), "auth.pin_hash")
	authRequired := err == nil

	if !authRequired {
		utils.SendSuccess(c, SessionResponse{
			Valid:        true,
			AuthRequired: false,
		})
		return
	}

	// If auth is required, verify the token
	tokenString := c.GetHeader("Authorization")
	if tokenString == "" {
		utils.SendSuccess(c, SessionResponse{
			Valid:        false,
			AuthRequired: true,
		})
		return
	}

	// Remove "Bearer " prefix if present
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	// Verify JWT token
	valid, expiresAt := h.verifyJWTToken(tokenString)

	utils.SendSuccess(c, SessionResponse{
		Valid:        valid,
		AuthRequired: authRequired,
		ExpiresAt:    expiresAt,
	})
}

// Logout invalidates the current session
func (h *Handlers) Logout(c *gin.Context) {
	// For JWT tokens, we can't really "logout" without a blacklist
	// For now, just return success - frontend should delete the token
	utils.SendSuccess(c, gin.H{"message": "Logged out successfully"})
}

// generateJWTToken creates a new JWT token
func (h *Handlers) generateJWTToken() (string, time.Time, error) {
	expiresAt := time.Now().Add(24 * time.Hour) // 24 hour expiry

	claims := jwt.MapClaims{
		"authorized": true,
		"exp":        expiresAt.Unix(),
		"iat":        time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.cfg.Auth.JWTSecret))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// verifyJWTToken verifies a JWT token and returns validity and expiry
func (h *Handlers) verifyJWTToken(tokenString string) (bool, time.Time) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(h.cfg.Auth.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return false, time.Time{}
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if exp, ok := claims["exp"].(float64); ok {
			return true, time.Unix(int64(exp), 0)
		}
	}

	return false, time.Time{}
}
