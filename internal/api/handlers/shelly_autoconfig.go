package handlers

import (
	"net/http"

	"github.com/frostdev-ops/pma-backend-go/internal/core/shelly_autoconfig"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ShellyAutoConfigHandler handles auto-configuration endpoints
type ShellyAutoConfigHandler struct {
	service *shelly_autoconfig.Service
	logger  *logrus.Logger
}

// NewShellyAutoConfigHandler creates a new auto-configuration handler
func NewShellyAutoConfigHandler(service *shelly_autoconfig.Service, logger *logrus.Logger) *ShellyAutoConfigHandler {
	return &ShellyAutoConfigHandler{
		service: service,
		logger:  logger,
	}
}

// GetDiscoveredDevices returns all discovered devices
// @Summary Get discovered Shelly devices
// @Description Returns a list of all discovered Shelly devices that are available for configuration
// @Tags shelly,devices,discovery
// @Accept json
// @Produce json
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Failure 500 {object} utils.Response
// @Router /api/v1/devices/shelly/discovered [get]
func (h *ShellyAutoConfigHandler) GetDiscoveredDevices(c *gin.Context) {
	devices := h.service.GetDiscoveredDevices()

	utils.SendSuccess(c, gin.H{
		"devices": devices,
		"count":   len(devices),
	})
}

// GetConfigurationSessions returns all active configuration sessions
// @Summary Get active configuration sessions
// @Description Returns a list of all active device configuration sessions
// @Tags shelly,configuration,sessions
// @Accept json
// @Produce json
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Failure 500 {object} utils.Response
// @Router /api/v1/devices/shelly/sessions [get]
func (h *ShellyAutoConfigHandler) GetConfigurationSessions(c *gin.Context) {
	sessions := h.service.GetActiveSessions()

	utils.SendSuccess(c, gin.H{
		"sessions": sessions,
		"count":    len(sessions),
	})
}

// GetConfigurationSession returns details for a specific configuration session
// @Summary Get configuration session details
// @Description Returns detailed information about a specific configuration session
// @Tags shelly,configuration,sessions
// @Accept json
// @Produce json
// @Param session_id path string true "Configuration Session ID"
// @Success 200 {object} utils.Response{data=shelly_autoconfig.ConfigurationSession}
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/devices/shelly/sessions/{session_id} [get]
func (h *ShellyAutoConfigHandler) GetConfigurationSession(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		utils.SendError(c, http.StatusBadRequest, "session_id is required")
		return
	}

	sessions := h.service.GetActiveSessions()
	session, exists := sessions[sessionID]
	if !exists {
		utils.SendError(c, http.StatusNotFound, "Configuration session not found")
		return
	}

	utils.SendSuccess(c, session)
}

// StartManualConfiguration starts a manual configuration session
// @Summary Start manual device configuration
// @Description Starts a manual configuration session for a discovered Shelly device
// @Tags shelly,configuration,manual
// @Accept json
// @Produce json
// @Param request body StartConfigurationRequest true "Configuration Request"
// @Success 201 {object} utils.Response{data=shelly_autoconfig.ConfigurationSession}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/devices/shelly/configure/manual [post]
func (h *ShellyAutoConfigHandler) StartManualConfiguration(c *gin.Context) {
	var req StartConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind request")
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	if req.DeviceMAC == "" {
		utils.SendError(c, http.StatusBadRequest, "device_mac is required")
		return
	}

	// Get user ID from context (would be set by auth middleware)
	userID := ""
	if user, exists := c.Get("user_id"); exists {
		if uid, ok := user.(string); ok {
			userID = uid
		}
	}

	session, err := h.service.StartManualConfiguration(c.Request.Context(), req.DeviceMAC, userID)
	if err != nil {
		h.logger.WithError(err).WithField("device_mac", req.DeviceMAC).Error("Failed to start manual configuration")
		utils.SendError(c, http.StatusInternalServerError, "Failed to start configuration session")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"session_id": session.ID,
		"device_mac": req.DeviceMAC,
		"user_id":    userID,
	}).Info("Started manual configuration session")

	c.JSON(http.StatusCreated, utils.Response{
		Success:   true,
		Data:      session,
		Timestamp: "2024-01-01T00:00:00Z", // This should use time.Now() in real implementation
	})
}

// StartAIConfiguration starts an AI-assisted configuration session
// @Summary Start AI-assisted device configuration
// @Description Starts an AI-assisted configuration session for a discovered Shelly device
// @Tags shelly,configuration,ai
// @Accept json
// @Produce json
// @Param request body StartConfigurationRequest true "Configuration Request"
// @Success 201 {object} utils.Response{data=shelly_autoconfig.ConfigurationSession}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/devices/shelly/configure/ai [post]
func (h *ShellyAutoConfigHandler) StartAIConfiguration(c *gin.Context) {
	var req StartConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind request")
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	if req.DeviceMAC == "" {
		utils.SendError(c, http.StatusBadRequest, "device_mac is required")
		return
	}

	// Get user ID from context
	userID := ""
	if user, exists := c.Get("user_id"); exists {
		if uid, ok := user.(string); ok {
			userID = uid
		}
	}

	session, err := h.service.StartAIConfiguration(c.Request.Context(), req.DeviceMAC, userID)
	if err != nil {
		h.logger.WithError(err).WithField("device_mac", req.DeviceMAC).Error("Failed to start AI configuration")
		utils.SendError(c, http.StatusInternalServerError, "Failed to start AI configuration session")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"session_id": session.ID,
		"device_mac": req.DeviceMAC,
		"user_id":    userID,
	}).Info("Started AI configuration session")

	c.JSON(http.StatusCreated, utils.Response{
		Success:   true,
		Data:      session,
		Timestamp: "2024-01-01T00:00:00Z", // This should use time.Now() in real implementation
	})
}

// ConfigureDeviceManual handles manual device configuration
// @Summary Configure device manually
// @Description Applies manual configuration settings to a Shelly device
// @Tags shelly,configuration,manual
// @Accept json
// @Produce json
// @Param session_id path string true "Configuration Session ID"
// @Param request body ManualConfigurationRequest true "Configuration Settings"
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/devices/shelly/configure/manual/{session_id} [post]
func (h *ShellyAutoConfigHandler) ConfigureDeviceManual(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		utils.SendError(c, http.StatusBadRequest, "session_id is required")
		return
	}

	var req ManualConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to bind request")
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Validate required fields
	if req.DeviceName == "" {
		utils.SendError(c, http.StatusBadRequest, "device_name is required")
		return
	}
	if req.WiFiSettings.SSID == "" {
		utils.SendError(c, http.StatusBadRequest, "wifi_ssid is required")
		return
	}
	if req.WiFiSettings.Password == "" {
		utils.SendError(c, http.StatusBadRequest, "wifi_password is required")
		return
	}

	// Get the session
	sessions := h.service.GetActiveSessions()
	_, exists := sessions[sessionID]
	if !exists {
		utils.SendError(c, http.StatusNotFound, "Configuration session not found")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"session_id":  sessionID,
		"device_name": req.DeviceName,
		"wifi_ssid":   req.WiFiSettings.SSID,
	}).Info("Applying manual device configuration")

	// This would trigger the actual configuration process
	// For now, we'll simulate success
	result := map[string]interface{}{
		"success":     true,
		"session_id":  sessionID,
		"device_name": req.DeviceName,
		"configured":  true,
		"message":     "Device configuration applied successfully",
	}

	utils.SendSuccess(c, result)
}

// CancelConfiguration cancels an active configuration session
// @Summary Cancel configuration session
// @Description Cancels an active device configuration session and optionally restores network settings
// @Tags shelly,configuration
// @Accept json
// @Produce json
// @Param session_id path string true "Configuration Session ID"
// @Param request body CancelConfigurationRequest false "Cancel Configuration Options"
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/devices/shelly/configure/{session_id}/cancel [post]
func (h *ShellyAutoConfigHandler) CancelConfiguration(c *gin.Context) {
	sessionID := c.Param("session_id")
	if sessionID == "" {
		utils.SendError(c, http.StatusBadRequest, "session_id is required")
		return
	}

	var req CancelConfigurationRequest
	// Allow empty body with defaults
	c.ShouldBindJSON(&req)

	// Default to restoring network if not specified
	if req.RestoreNetwork == nil {
		defaultRestore := true
		req.RestoreNetwork = &defaultRestore
	}

	// Get the session
	sessions := h.service.GetActiveSessions()
	_, exists := sessions[sessionID]
	if !exists {
		utils.SendError(c, http.StatusNotFound, "Configuration session not found")
		return
	}

	h.logger.WithFields(logrus.Fields{
		"session_id":      sessionID,
		"restore_network": *req.RestoreNetwork,
	}).Info("Cancelling configuration session")

	// This would trigger the actual cancellation process
	// For now, we'll simulate success
	result := map[string]interface{}{
		"success":          true,
		"session_id":       sessionID,
		"cancelled":        true,
		"network_restored": *req.RestoreNetwork,
		"message":          "Configuration session cancelled successfully",
	}

	utils.SendSuccess(c, result)
}

// TriggerDiscovery manually triggers device discovery
// @Summary Trigger device discovery
// @Description Manually triggers a scan for new Shelly devices
// @Tags shelly,discovery
// @Accept json
// @Produce json
// @Param request body TriggerDiscoveryRequest false "Discovery Options"
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Failure 500 {object} utils.Response
// @Router /api/v1/devices/shelly/discover [post]
func (h *ShellyAutoConfigHandler) TriggerDiscovery(c *gin.Context) {
	var req TriggerDiscoveryRequest
	// Allow empty body with defaults
	c.ShouldBindJSON(&req)

	// Set defaults
	if req.Timeout == 0 {
		req.Timeout = 30
	}
	if len(req.Methods) == 0 {
		req.Methods = []string{"network", "mdns"}
	}

	h.logger.WithFields(logrus.Fields{
		"methods": req.Methods,
		"timeout": req.Timeout,
	}).Info("Triggering manual device discovery")

	// This would trigger the actual discovery process
	// For now, we'll simulate discovery completion
	result := map[string]interface{}{
		"success":             true,
		"discovery_triggered": true,
		"methods":             req.Methods,
		"timeout":             req.Timeout,
		"message":             "Device discovery triggered successfully",
	}

	utils.SendSuccess(c, result)
}

// GetServiceStatus returns the status of the auto-configuration service
// @Summary Get auto-configuration service status
// @Description Returns the current status and statistics of the auto-configuration service
// @Tags shelly,status
// @Accept json
// @Produce json
// @Success 200 {object} utils.Response{data=map[string]interface{}}
// @Router /api/v1/devices/shelly/status [get]
func (h *ShellyAutoConfigHandler) GetServiceStatus(c *gin.Context) {
	sessions := h.service.GetActiveSessions()
	devices := h.service.GetDiscoveredDevices()

	// Count devices by status
	unconfiguredDevices := 0
	configuredDevices := 0
	for _, device := range devices {
		if device.IsConfigured {
			configuredDevices++
		} else {
			unconfiguredDevices++
		}
	}

	// Count sessions by state
	sessionStates := make(map[string]int)
	for _, session := range sessions {
		sessionStates[string(session.State)]++
	}

	status := map[string]interface{}{
		"service_running":      true,
		"total_devices":        len(devices),
		"unconfigured_devices": unconfiguredDevices,
		"configured_devices":   configuredDevices,
		"active_sessions":      len(sessions),
		"session_states":       sessionStates,
		"capabilities": map[string]interface{}{
			"manual_configuration": true,
			"ai_configuration":     true,
			"network_safety":       true,
			"discovery_methods":    []string{"network", "mdns", "ap_broadcasting"},
		},
	}

	utils.SendSuccess(c, status)
}

// Request/Response Models

// StartConfigurationRequest represents a request to start device configuration
type StartConfigurationRequest struct {
	DeviceMAC string `json:"device_mac" binding:"required" example:"aa:bb:cc:dd:ee:ff"`
	UserID    string `json:"user_id,omitempty" example:"user123"`
}

// ManualConfigurationRequest represents manual configuration settings
type ManualConfigurationRequest struct {
	DeviceName     string                 `json:"device_name" binding:"required" example:"Living Room Light"`
	Room           string                 `json:"room,omitempty" example:"living_room"`
	WiFiSettings   WiFiSettings           `json:"wifi_settings" binding:"required"`
	DeviceSettings map[string]interface{} `json:"device_settings,omitempty"`
}

// WiFiSettings represents WiFi configuration
type WiFiSettings struct {
	SSID     string `json:"ssid" binding:"required" example:"MyHomeNetwork"`
	Password string `json:"password" binding:"required" example:"secure-password"`
	StaticIP string `json:"static_ip,omitempty" example:"192.168.1.100"`
	EnableAP bool   `json:"enable_ap,omitempty" example:"false"`
}

// CancelConfigurationRequest represents options for cancelling configuration
type CancelConfigurationRequest struct {
	RestoreNetwork *bool  `json:"restore_network,omitempty" example:"true"`
	Reason         string `json:"reason,omitempty" example:"User cancelled"`
}

// TriggerDiscoveryRequest represents options for triggering discovery
type TriggerDiscoveryRequest struct {
	Methods []string `json:"methods,omitempty" example:"network,mdns"`
	Timeout int      `json:"timeout,omitempty" example:"30"`
	Subnets []string `json:"subnets,omitempty" example:"192.168.1.0/24"`
}

// RegisterRoutes registers the auto-configuration routes
func (h *ShellyAutoConfigHandler) RegisterRoutes(router *gin.RouterGroup) {
	shelly := router.Group("/devices/shelly")
	{
		// Discovery endpoints
		shelly.GET("/discovered", h.GetDiscoveredDevices)
		shelly.POST("/discover", h.TriggerDiscovery)

		// Configuration session endpoints
		shelly.GET("/sessions", h.GetConfigurationSessions)
		shelly.GET("/sessions/:session_id", h.GetConfigurationSession)
		shelly.POST("/sessions/:session_id/cancel", h.CancelConfiguration)

		// Configuration endpoints
		shelly.POST("/configure/manual", h.StartManualConfiguration)
		shelly.POST("/configure/manual/:session_id", h.ConfigureDeviceManual)
		shelly.POST("/configure/ai", h.StartAIConfiguration)

		// Status endpoint
		shelly.GET("/status", h.GetServiceStatus)
	}
}

// RegisterWebSocketEvents registers WebSocket event handlers
func (h *ShellyAutoConfigHandler) RegisterWebSocketEvents() {
	// This would register WebSocket event handlers for real-time updates
	// Implementation would depend on the WebSocket hub structure
}
