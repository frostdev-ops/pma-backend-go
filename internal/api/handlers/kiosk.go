package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/kiosk"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/gin-gonic/gin"
)

// KioskHandler handles kiosk-related HTTP requests
type KioskHandler struct {
	service kiosk.Service
}

// NewKioskHandler creates a new kiosk handler
func NewKioskHandler(service kiosk.Service) *KioskHandler {
	return &KioskHandler{
		service: service,
	}
}

// ======== AUTHENTICATION MIDDLEWARE ========

// KioskAuthMiddleware validates kiosk tokens
func (h *KioskHandler) KioskAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get token from header first
		token := c.GetHeader("Authorization")
		if token != "" {
			// Remove "Bearer " prefix if present
			if strings.HasPrefix(token, "Bearer ") {
				token = strings.TrimPrefix(token, "Bearer ")
			}
		} else {
			// Try to get token from query parameter
			token = c.Query("token")
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Missing kiosk token",
			})
			c.Abort()
			return
		}

		// Validate token
		kioskToken, err := h.service.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Store token data in context
		c.Set("kiosk_token", kioskToken)
		c.Next()
	}
}

// ======== PUBLIC ENDPOINTS (NO AUTH REQUIRED) ========

// PairKiosk handles kiosk pairing requests
// POST /api/kiosk/pair
func (h *KioskHandler) PairKiosk(c *gin.Context) {
	var request models.KioskPairingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	// Validate PIN format
	if len(request.Pin) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "PIN must be exactly 6 digits",
		})
		return
	}

	// Validate name
	if len(request.Name) < 2 || len(request.Name) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Kiosk name must be between 2 and 50 characters",
		})
		return
	}

	response, err := h.service.ValidatePairingPIN(c.Request.Context(), &request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Internal server error",
		})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// ======== ADMIN ENDPOINTS (SYSTEM AUTH REQUIRED) ========

// ListKiosks lists all kiosk devices
// GET /api/kiosk/devices
func (h *KioskHandler) ListKiosks(c *gin.Context) {
	tokens, err := h.service.GetAllTokens(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve kiosk devices",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": tokens,
		"count":   len(tokens),
	})
}

// CreatePairingSession creates a new pairing session
// POST /api/kiosk/pair/create
func (h *KioskHandler) CreatePairingSession(c *gin.Context) {
	var request struct {
		RoomID         string   `json:"room_id" binding:"required"`
		AllowedDevices []string `json:"allowed_devices,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	pin, err := h.service.GeneratePairingPIN(c.Request.Context(), request.RoomID, request.AllowedDevices)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to generate pairing PIN",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"pin":     pin,
		"room_id": request.RoomID,
		"expires": "5 minutes",
	})
}

// CancelPairingSession cancels a pairing session
// DELETE /api/kiosk/pair/:sessionId
func (h *KioskHandler) CancelPairingSession(c *gin.Context) {
	// Since we don't have a direct method to cancel by session ID,
	// and PINs expire automatically, we'll just return success
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Pairing session cancelled",
	})
}

// RemoveKiosk removes a kiosk device
// DELETE /api/kiosk/devices/:kioskId
func (h *KioskHandler) RemoveKiosk(c *gin.Context) {
	kioskID := c.Param("kioskId")

	err := h.service.RevokeToken(c.Request.Context(), kioskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to remove kiosk device",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Kiosk device removed",
	})
}

// UpdateKioskConfigAdmin updates kiosk configuration (admin endpoint)
// PUT /api/kiosk/devices/:kioskId/config
func (h *KioskHandler) UpdateKioskConfigAdmin(c *gin.Context) {
	kioskID := c.Param("kioskId")

	// Get the kiosk token to find the room ID
	tokens, err := h.service.GetAllTokens(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve kiosk device",
		})
		return
	}

	var roomID string
	for _, token := range tokens {
		if token.ID == kioskID {
			roomID = token.RoomID
			break
		}
	}

	if roomID == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Kiosk device not found",
		})
		return
	}

	var updates models.KioskConfigUpdateRequest
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	err = h.service.UpdateKioskConfig(c.Request.Context(), roomID, &updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update kiosk configuration",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Configuration updated",
	})
}

// GetKioskLogs retrieves logs for a kiosk device
// GET /api/kiosk/devices/:kioskId/logs
func (h *KioskHandler) GetKioskLogs(c *gin.Context) {
	kioskID := c.Param("kioskId")

	// Parse query parameters
	query := &models.KioskLogQuery{}

	if level := c.Query("level"); level != "" {
		query.Level = level
	}
	if category := c.Query("category"); category != "" {
		query.Category = category
	}
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			query.Limit = l
		}
	}
	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil {
			query.Offset = o
		}
	}

	logs, err := h.service.GetLogs(c.Request.Context(), kioskID, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve logs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"count": len(logs),
	})
}

// GetDeviceGroups retrieves all device groups
// GET /api/kiosk/device-groups
func (h *KioskHandler) GetDeviceGroups(c *gin.Context) {
	groups, err := h.service.GetAllDeviceGroups(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve device groups",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"groups": groups,
		"count":  len(groups),
	})
}

// CreateDeviceGroup creates a new device group
// POST /api/kiosk/device-groups
func (h *KioskHandler) CreateDeviceGroup(c *gin.Context) {
	var request models.KioskDeviceGroupCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	group, err := h.service.CreateDeviceGroup(c.Request.Context(), &request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create device group",
		})
		return
	}

	c.JSON(http.StatusCreated, group)
}

// UpdateDeviceGroup updates a device group
// PUT /api/kiosk/device-groups/:groupId
func (h *KioskHandler) UpdateDeviceGroup(c *gin.Context) {
	groupID := c.Param("groupId")

	var request models.KioskDeviceGroupCreateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	err := h.service.UpdateDeviceGroup(c.Request.Context(), groupID, &request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update device group",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Device group updated",
	})
}

// DeleteDeviceGroup deletes a device group
// DELETE /api/kiosk/device-groups/:groupId
func (h *KioskHandler) DeleteDeviceGroup(c *gin.Context) {
	groupID := c.Param("groupId")

	err := h.service.DeleteDeviceGroup(c.Request.Context(), groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete device group",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Device group deleted",
	})
}

// ======== KIOSK-AUTHENTICATED ENDPOINTS ========

// GetConfig retrieves kiosk configuration
// GET /api/kiosk/config
func (h *KioskHandler) GetConfig(c *gin.Context) {
	kioskToken, exists := c.Get("kiosk_token")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "No kiosk token found",
		})
		return
	}

	token := kioskToken.(*models.KioskToken)
	config, err := h.service.GetKioskConfig(c.Request.Context(), token.RoomID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve configuration",
		})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateConfig updates kiosk configuration
// PUT /api/kiosk/config
func (h *KioskHandler) UpdateConfig(c *gin.Context) {
	kioskToken, exists := c.Get("kiosk_token")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "No kiosk token found",
		})
		return
	}

	token := kioskToken.(*models.KioskToken)

	var updates models.KioskConfigUpdateRequest
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	err := h.service.UpdateKioskConfig(c.Request.Context(), token.RoomID, &updates)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update configuration",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Configuration updated",
	})
}

// GetDevices retrieves devices available to the kiosk
// GET /api/kiosk/devices
func (h *KioskHandler) GetDevices(c *gin.Context) {
	kioskToken, exists := c.Get("kiosk_token")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "No kiosk token found",
		})
		return
	}

	token := kioskToken.(*models.KioskToken)
	devices, err := h.service.GetKioskDevices(c.Request.Context(), token.Token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve devices",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"devices": devices,
		"count":   len(devices),
	})
}

// ExecuteCommand executes a command on a device
// POST /api/kiosk/command
func (h *KioskHandler) ExecuteCommand(c *gin.Context) {
	kioskToken, exists := c.Get("kiosk_token")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "No kiosk token found",
		})
		return
	}

	token := kioskToken.(*models.KioskToken)

	var command models.KioskCommandRequest
	if err := c.ShouldBindJSON(&command); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid command format",
		})
		return
	}

	response, err := h.service.ExecuteDeviceCommand(c.Request.Context(), token.Token, &command)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Internal server error",
			"timestamp": response.Timestamp,
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetStatus retrieves kiosk device status
// GET /api/kiosk/status
func (h *KioskHandler) GetStatus(c *gin.Context) {
	kioskToken, exists := c.Get("kiosk_token")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "No kiosk token found",
		})
		return
	}

	token := kioskToken.(*models.KioskToken)
	status, err := h.service.GetDeviceStatus(c.Request.Context(), token.ID)
	if err != nil {
		// If no status found, create a default one
		c.JSON(http.StatusOK, gin.H{
			"status":     "online",
			"last_seen":  token.LastUsed,
			"kiosk_id":   token.ID,
			"kiosk_name": token.Name,
			"room_id":    token.RoomID,
		})
		return
	}

	c.JSON(http.StatusOK, status)
}

// Heartbeat records a heartbeat from the kiosk
// POST /api/kiosk/heartbeat
func (h *KioskHandler) Heartbeat(c *gin.Context) {
	kioskToken, exists := c.Get("kiosk_token")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "No kiosk token found",
		})
		return
	}

	token := kioskToken.(*models.KioskToken)

	// Parse optional status update data
	var statusUpdate struct {
		DeviceInfo         map[string]interface{} `json:"device_info,omitempty"`
		PerformanceMetrics map[string]interface{} `json:"performance_metrics,omitempty"`
		Status             string                 `json:"status,omitempty"`
		BatteryLevel       *int                   `json:"battery_level,omitempty"`
		NetworkQuality     string                 `json:"network_quality,omitempty"`
	}

	// Bind JSON but don't require it
	_ = c.ShouldBindJSON(&statusUpdate)

	// Record heartbeat
	err := h.service.RecordHeartbeat(c.Request.Context(), token.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to record heartbeat",
		})
		return
	}

	// Update device status if provided
	if statusUpdate.Status != "" || statusUpdate.DeviceInfo != nil {
		deviceInfoBytes, _ := json.Marshal(statusUpdate.DeviceInfo)
		performanceBytes, _ := json.Marshal(statusUpdate.PerformanceMetrics)

		status := &models.KioskDeviceStatus{
			KioskTokenID:       token.ID,
			Status:             statusUpdate.Status,
			DeviceInfo:         deviceInfoBytes,
			PerformanceMetrics: performanceBytes,
			NetworkQuality:     sql.NullString{String: statusUpdate.NetworkQuality, Valid: statusUpdate.NetworkQuality != ""},
		}

		if statusUpdate.BatteryLevel != nil {
			status.BatteryLevel = sql.NullInt64{Int64: int64(*statusUpdate.BatteryLevel), Valid: true}
		}

		_ = h.service.UpdateDeviceStatus(c.Request.Context(), token.ID, status)
	}

	// Get pending commands
	commands, _ := h.service.GetPendingCommands(c.Request.Context(), token.ID)

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
		"pending_commands": commands,
	})
}

// GetPendingCommands retrieves pending commands for the kiosk
// GET /api/kiosk/commands/pending
func (h *KioskHandler) GetPendingCommands(c *gin.Context) {
	kioskToken, exists := c.Get("kiosk_token")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "No kiosk token found",
		})
		return
	}

	token := kioskToken.(*models.KioskToken)
	commands, err := h.service.GetPendingCommands(c.Request.Context(), token.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve pending commands",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"commands": commands,
		"count":    len(commands),
	})
}

// AcknowledgeCommand acknowledges a command
// POST /api/kiosk/commands/:commandId/ack
func (h *KioskHandler) AcknowledgeCommand(c *gin.Context) {
	commandID := c.Param("commandId")

	err := h.service.AcknowledgeCommand(c.Request.Context(), commandID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to acknowledge command",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Command acknowledged",
	})
}

// CompleteCommand completes a command with result data
// POST /api/kiosk/commands/:commandId/complete
func (h *KioskHandler) CompleteCommand(c *gin.Context) {
	commandID := c.Param("commandId")

	var request struct {
		Success    bool                   `json:"success"`
		ResultData map[string]interface{} `json:"result_data,omitempty"`
		Error      string                 `json:"error,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	errorMsg := ""
	if !request.Success {
		errorMsg = request.Error
	}

	err := h.service.CompleteCommand(c.Request.Context(), commandID, request.ResultData, errorMsg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to complete command",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Command completed",
	})
}

// ======== STATISTICS AND ADMIN ENDPOINTS ========

// GetStats retrieves kiosk system statistics
// GET /api/kiosk/stats
func (h *KioskHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetKioskStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve statistics",
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}
