package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// RingAuthRequest represents Ring authentication start request
type RingAuthRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Ring2FARequest represents Ring 2FA verification request
type Ring2FARequest struct {
	Code      string `json:"code" binding:"required"`
	SessionID string `json:"sessionId" binding:"required"`
}

// RingConfigRequest represents Ring configuration setup request
type RingConfigRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
	TokenExpiry  int64  `json:"tokenExpiry"`
}

// RingCameraResponse represents Ring camera information
type RingCameraResponse struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	DeviceType       string                 `json:"device_type"`
	BatteryLevel     *int                   `json:"battery_level,omitempty"`
	Online           bool                   `json:"online"`
	LastUpdate       time.Time              `json:"last_update"`
	HasLight         bool                   `json:"has_light"`
	LightOn          bool                   `json:"light_on"`
	HasSiren         bool                   `json:"has_siren"`
	SirenOn          bool                   `json:"siren_on"`
	MotionDetection  bool                   `json:"motion_detection"`
	RecordingEnabled bool                   `json:"recording_enabled"`
	Thumbnail        string                 `json:"thumbnail,omitempty"`
	Location         string                 `json:"location,omitempty"`
	Health           map[string]interface{} `json:"health,omitempty"`
}

// RingEventResponse represents Ring camera event
type RingEventResponse struct {
	ID         string     `json:"id"`
	CameraID   string     `json:"camera_id"`
	EventType  string     `json:"event_type"`
	CreatedAt  time.Time  `json:"created_at"`
	AnsweredAt *time.Time `json:"answered_at,omitempty"`
	Duration   int        `json:"duration"`
	VideoURL   string     `json:"video_url,omitempty"`
	Thumbnail  string     `json:"thumbnail,omitempty"`
	Motion     bool       `json:"motion"`
	Answered   bool       `json:"answered"`
}

// RingConfigStatus represents Ring configuration status
type RingConfigStatus struct {
	Configured    bool      `json:"configured"`
	Authenticated bool      `json:"authenticated"`
	LastAuth      time.Time `json:"last_auth,omitempty"`
	CameraCount   int       `json:"camera_count"`
	Status        string    `json:"status"`
}

// RingLightControlRequest represents light control request
type RingLightControlRequest struct {
	On       bool `json:"on"`
	Duration int  `json:"duration,omitempty"` // Duration in seconds
}

// RingSirenControlRequest represents siren control request
type RingSirenControlRequest struct {
	On       bool `json:"on"`
	Duration int  `json:"duration,omitempty"` // Duration in seconds
}

// GetRingConfigStatus returns Ring configuration status
func (h *Handlers) GetRingConfigStatus(c *gin.Context) {
	// Check if Ring is configured
	refreshTokenConfig, err := h.repos.Config.Get(c.Request.Context(), "ring.refresh_token")
	configured := err == nil && refreshTokenConfig.Value != ""

	status := RingConfigStatus{
		Configured:    configured,
		Authenticated: false,
		CameraCount:   0,
		Status:        "not_configured",
	}

	if configured {
		// Check if authentication is still valid
		expiryConfig, err := h.repos.Config.Get(c.Request.Context(), "ring.token_expiry")
		if err == nil {
			if expiry, err := strconv.ParseInt(expiryConfig.Value, 10, 64); err == nil {
				if time.Now().Unix() < expiry {
					status.Authenticated = true
					status.Status = "authenticated"

					// Get last auth time
					lastAuthConfig, err := h.repos.Config.Get(c.Request.Context(), "ring.last_auth")
					if err == nil {
						if lastAuth, err := strconv.ParseInt(lastAuthConfig.Value, 10, 64); err == nil {
							status.LastAuth = time.Unix(lastAuth, 0)
						}
					}
				} else {
					status.Status = "token_expired"
				}
			}
		}
	}

	utils.SendSuccess(c, status)
}

// SetupRingConfig sets up Ring configuration with refresh token
func (h *Handlers) SetupRingConfig(c *gin.Context) {
	var req RingConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Store refresh token
	err := h.repos.Config.Set(c.Request.Context(), &models.SystemConfig{
		Key:         "ring.refresh_token",
		Value:       req.RefreshToken,
		Encrypted:   true,
		Description: "Ring API refresh token",
	})
	if err != nil {
		h.log.WithError(err).Error("Failed to store Ring refresh token")
		utils.SendError(c, http.StatusInternalServerError, "Failed to save Ring configuration")
		return
	}

	// Store token expiry if provided
	if req.TokenExpiry > 0 {
		h.repos.Config.Set(c.Request.Context(), &models.SystemConfig{
			Key:         "ring.token_expiry",
			Value:       strconv.FormatInt(req.TokenExpiry, 10),
			Description: "Ring token expiry timestamp",
		})
	}

	// Store setup timestamp
	h.repos.Config.Set(c.Request.Context(), &models.SystemConfig{
		Key:         "ring.last_auth",
		Value:       strconv.FormatInt(time.Now().Unix(), 10),
		Description: "Ring last authentication timestamp",
	})

	h.log.Info("Ring configuration saved successfully")
	utils.SendSuccess(c, gin.H{"message": "Ring configuration saved successfully"})
}

// StartRingAuthentication initiates Ring authentication flow
func (h *Handlers) StartRingAuthentication(c *gin.Context) {
	var req RingAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// In a real implementation, this would call Ring API for authentication
	// For now, return a mock session ID
	sessionID := "ring_auth_" + strconv.FormatInt(time.Now().Unix(), 10)

	// Store temporary session
	h.repos.Config.Set(c.Request.Context(), &models.SystemConfig{
		Key:         "ring.temp_session." + sessionID,
		Value:       req.Email,
		Description: "Temporary Ring authentication session",
	})

	utils.SendSuccess(c, gin.H{
		"sessionId":   sessionID,
		"message":     "2FA code sent to your email/phone",
		"requires2FA": true,
	})
}

// Complete2FA completes Ring 2FA authentication
func (h *Handlers) Complete2FA(c *gin.Context) {
	var req Ring2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Verify session exists
	sessionConfig, err := h.repos.Config.Get(c.Request.Context(), "ring.temp_session."+req.SessionID)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid session ID")
		return
	}

	// In a real implementation, verify 2FA code with Ring API
	// For now, simulate successful authentication
	mockRefreshToken := "ring_refresh_token_" + strconv.FormatInt(time.Now().Unix(), 10)

	// Store refresh token
	err = h.repos.Config.Set(c.Request.Context(), &models.SystemConfig{
		Key:         "ring.refresh_token",
		Value:       mockRefreshToken,
		Encrypted:   true,
		Description: "Ring API refresh token",
	})
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to save authentication")
		return
	}

	// Store token expiry (24 hours from now)
	expiry := time.Now().Add(24 * time.Hour).Unix()
	h.repos.Config.Set(c.Request.Context(), &models.SystemConfig{
		Key:         "ring.token_expiry",
		Value:       strconv.FormatInt(expiry, 10),
		Description: "Ring token expiry timestamp",
	})

	// Clean up temp session
	h.repos.Config.Delete(c.Request.Context(), "ring.temp_session."+req.SessionID)

	h.log.Info("Ring 2FA authentication completed for: " + sessionConfig.Value)
	utils.SendSuccess(c, gin.H{
		"message":       "Ring authentication completed successfully",
		"authenticated": true,
	})
}

// GetRingCameras returns all Ring cameras
func (h *Handlers) GetRingCameras(c *gin.Context) {
	// Check authentication
	if !h.isRingAuthenticated(c.Request.Context()) {
		utils.SendError(c, http.StatusUnauthorized, "Ring not authenticated")
		return
	}

	// Mock camera data - in real implementation, fetch from Ring API
	cameras := []RingCameraResponse{
		{
			ID:               "camera_1",
			Name:             "Front Door",
			DeviceType:       "doorbell",
			BatteryLevel:     &[]int{85}[0],
			Online:           true,
			LastUpdate:       time.Now().Add(-5 * time.Minute),
			HasLight:         true,
			LightOn:          false,
			HasSiren:         false,
			MotionDetection:  true,
			RecordingEnabled: true,
			Location:         "Front Entrance",
		},
		{
			ID:               "camera_2",
			Name:             "Backyard",
			DeviceType:       "stickup_cam",
			Online:           true,
			LastUpdate:       time.Now().Add(-2 * time.Minute),
			HasLight:         true,
			LightOn:          false,
			HasSiren:         true,
			SirenOn:          false,
			MotionDetection:  true,
			RecordingEnabled: true,
			Location:         "Back Garden",
		},
	}

	utils.SendSuccess(c, gin.H{
		"cameras": cameras,
		"count":   len(cameras),
	})
}

// GetRingCamera returns a specific Ring camera
func (h *Handlers) GetRingCamera(c *gin.Context) {
	cameraID := c.Param("cameraId")
	if cameraID == "" {
		utils.SendError(c, http.StatusBadRequest, "Camera ID is required")
		return
	}

	// Check authentication
	if !h.isRingAuthenticated(c.Request.Context()) {
		utils.SendError(c, http.StatusUnauthorized, "Ring not authenticated")
		return
	}

	// Mock camera data - in real implementation, fetch from Ring API
	camera := RingCameraResponse{
		ID:               cameraID,
		Name:             "Front Door",
		DeviceType:       "doorbell",
		BatteryLevel:     &[]int{85}[0],
		Online:           true,
		LastUpdate:       time.Now().Add(-5 * time.Minute),
		HasLight:         true,
		LightOn:          false,
		HasSiren:         false,
		MotionDetection:  true,
		RecordingEnabled: true,
		Location:         "Front Entrance",
		Health: map[string]interface{}{
			"wifi_signal_strength": -45,
			"battery_percentage":   85,
			"last_update":          time.Now(),
		},
	}

	utils.SendSuccess(c, camera)
}

// GetRingCameraSnapshot returns a camera snapshot
func (h *Handlers) GetRingCameraSnapshot(c *gin.Context) {
	cameraID := c.Param("cameraId")
	if cameraID == "" {
		utils.SendError(c, http.StatusBadRequest, "Camera ID is required")
		return
	}

	// Check authentication
	if !h.isRingAuthenticated(c.Request.Context()) {
		utils.SendError(c, http.StatusUnauthorized, "Ring not authenticated")
		return
	}

	// In real implementation, fetch snapshot from Ring API
	// For now, return a mock response
	utils.SendSuccess(c, gin.H{
		"snapshot_url": "https://ring-snapshots.s3.amazonaws.com/mock/" + cameraID,
		"expires_at":   time.Now().Add(1 * time.Hour),
		"camera_id":    cameraID,
	})
}

// ControlRingLight controls camera light
func (h *Handlers) ControlRingLight(c *gin.Context) {
	cameraID := c.Param("cameraId")
	if cameraID == "" {
		utils.SendError(c, http.StatusBadRequest, "Camera ID is required")
		return
	}

	var req RingLightControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Check authentication
	if !h.isRingAuthenticated(c.Request.Context()) {
		utils.SendError(c, http.StatusUnauthorized, "Ring not authenticated")
		return
	}

	// In real implementation, send light control command to Ring API
	h.log.Infof("Ring light control: camera=%s, on=%v, duration=%d", cameraID, req.On, req.Duration)

	// Broadcast light change via WebSocket
	if h.wsHub != nil {
		data := map[string]interface{}{
			"camera_id": cameraID,
			"light_on":  req.On,
			"duration":  req.Duration,
		}
		go h.wsHub.BroadcastToAll("ring_light_changed", data)
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Light control command sent",
		"camera_id": cameraID,
		"light_on":  req.On,
	})
}

// ControlRingSiren controls camera siren
func (h *Handlers) ControlRingSiren(c *gin.Context) {
	cameraID := c.Param("cameraId")
	if cameraID == "" {
		utils.SendError(c, http.StatusBadRequest, "Camera ID is required")
		return
	}

	var req RingSirenControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Check authentication
	if !h.isRingAuthenticated(c.Request.Context()) {
		utils.SendError(c, http.StatusUnauthorized, "Ring not authenticated")
		return
	}

	// In real implementation, send siren control command to Ring API
	h.log.Infof("Ring siren control: camera=%s, on=%v, duration=%d", cameraID, req.On, req.Duration)

	// Broadcast siren change via WebSocket
	if h.wsHub != nil {
		data := map[string]interface{}{
			"camera_id": cameraID,
			"siren_on":  req.On,
			"duration":  req.Duration,
		}
		go h.wsHub.BroadcastToAll("ring_siren_changed", data)
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Siren control command sent",
		"camera_id": cameraID,
		"siren_on":  req.On,
	})
}

// GetRingCameraEvents returns camera events/history
func (h *Handlers) GetRingCameraEvents(c *gin.Context) {
	cameraID := c.Param("cameraId")
	if cameraID == "" {
		utils.SendError(c, http.StatusBadRequest, "Camera ID is required")
		return
	}

	// Check authentication
	if !h.isRingAuthenticated(c.Request.Context()) {
		utils.SendError(c, http.StatusUnauthorized, "Ring not authenticated")
		return
	}

	// Parse query parameters
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Mock events data - in real implementation, fetch from Ring API
	events := []RingEventResponse{
		{
			ID:        "event_1",
			CameraID:  cameraID,
			EventType: "motion",
			CreatedAt: time.Now().Add(-1 * time.Hour),
			Duration:  30,
			Motion:    true,
			Answered:  false,
		},
		{
			ID:        "event_2",
			CameraID:  cameraID,
			EventType: "ding",
			CreatedAt: time.Now().Add(-3 * time.Hour),
			Duration:  45,
			Motion:    false,
			Answered:  true,
		},
	}

	utils.SendSuccess(c, gin.H{
		"events":    events,
		"count":     len(events),
		"camera_id": cameraID,
		"limit":     limit,
	})
}

// GetRingStatus returns Ring service status
func (h *Handlers) GetRingStatus(c *gin.Context) {
	authenticated := h.isRingAuthenticated(c.Request.Context())

	status := map[string]interface{}{
		"service_name":  "Ring Integration",
		"version":       "1.0.0",
		"authenticated": authenticated,
		"cameras":       0,
		"last_update":   time.Now(),
		"status":        "disconnected",
	}

	if authenticated {
		status["status"] = "connected"
		status["cameras"] = 2 // Mock camera count
	}

	utils.SendSuccess(c, status)
}

// DeleteRingConfig removes Ring configuration
func (h *Handlers) DeleteRingConfig(c *gin.Context) {
	// Delete all Ring-related config
	configKeys := []string{
		"ring.refresh_token",
		"ring.token_expiry",
		"ring.last_auth",
	}

	for _, key := range configKeys {
		h.repos.Config.Delete(c.Request.Context(), key)
	}

	h.log.Info("Ring configuration deleted")
	utils.SendSuccess(c, gin.H{"message": "Ring configuration deleted successfully"})
}

// RestartRingService restarts Ring service (placeholder)
func (h *Handlers) RestartRingService(c *gin.Context) {
	h.log.Info("Ring service restart requested")

	// In real implementation, restart Ring service/connection
	utils.SendSuccess(c, gin.H{"message": "Ring service restart initiated"})
}

// TestRingConnection tests Ring API connection
func (h *Handlers) TestRingConnection(c *gin.Context) {
	authenticated := h.isRingAuthenticated(c.Request.Context())

	if !authenticated {
		utils.SendError(c, http.StatusUnauthorized, "Ring not authenticated")
		return
	}

	// In real implementation, test actual Ring API connection
	utils.SendSuccess(c, gin.H{
		"connected":     true,
		"response_time": "150ms",
		"api_version":   "11",
		"cameras":       2,
	})
}

// isRingAuthenticated checks if Ring is properly authenticated
func (h *Handlers) isRingAuthenticated(ctx context.Context) bool {
	// Check if refresh token exists
	refreshTokenConfig, err := h.repos.Config.Get(ctx, "ring.refresh_token")
	if err != nil || refreshTokenConfig.Value == "" {
		return false
	}

	// Check if token is still valid
	expiryConfig, err := h.repos.Config.Get(ctx, "ring.token_expiry")
	if err != nil {
		return false
	}

	expiry, err := strconv.ParseInt(expiryConfig.Value, 10, 64)
	if err != nil {
		return false
	}

	return time.Now().Unix() < expiry
}
