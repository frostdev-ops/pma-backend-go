package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ShellyDiscoveryResponse represents the response for device discovery
type ShellyDiscoveryResponse struct {
	Devices    []ShellyDeviceResponse `json:"devices"`
	Count      int                    `json:"count"`
	ScannedAt  time.Time              `json:"scanned_at"`
	ScanStatus string                 `json:"scan_status"`
}

// ShellyDeviceControlRequest represents a device control request
type ShellyDeviceControlRequest struct {
	Action     string                 `json:"action" binding:"required"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// ShellyConfigRequest represents a configuration update request
type ShellyConfigRequest struct {
	Key   string      `json:"key" binding:"required"`
	Value interface{} `json:"value"`
}

// DiscoverShellyDevices discovers Shelly devices on the network
func (h *Handlers) DiscoverShellyDevices(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	// Trigger discovery through the Shelly adapter
	adapter, err := h.adapterRegistry.GetAdapter(string(types.SourceShelly))
	if err != nil || adapter == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Shelly adapter not available")
		return
	}

	// Get entities from the Shelly adapter
	options := unified.GetAllOptions{
		AvailableOnly: false,
	}
	
	entitiesWithRooms, err := h.unifiedService.GetBySource(ctx, types.SourceShelly, options)
	if err != nil {
		h.log.WithError(err).Error("Failed to discover Shelly devices")
		utils.SendError(c, http.StatusInternalServerError, "Failed to discover Shelly devices")
		return
	}

	// Convert to response format
	devices := make([]ShellyDeviceResponse, 0, len(entitiesWithRooms))
	for _, entityWithRoom := range entitiesWithRooms {
		entity := entityWithRoom.Entity
		devices = append(devices, convertEntityToShellyDevice(entity))
	}

	response := ShellyDiscoveryResponse{
		Devices:    devices,
		Count:      len(devices),
		ScannedAt:  time.Now(),
		ScanStatus: "completed",
	}

	utils.SendSuccess(c, response)
}

// ListShellyDevices lists all Shelly devices
func (h *Handlers) ListShellyDevices(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get all Shelly entities through unified service
	options := unified.GetAllOptions{
		AvailableOnly: false,
	}

	entitiesWithRooms, err := h.unifiedService.GetBySource(ctx, types.SourceShelly, options)
	if err != nil {
		h.log.WithError(err).Error("Failed to get Shelly devices")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve Shelly devices")
		return
	}

	// Convert to response format
	devices := make([]ShellyDeviceResponse, 0, len(entitiesWithRooms))
	for _, entityWithRoom := range entitiesWithRooms {
		entity := entityWithRoom.Entity
		devices = append(devices, convertEntityToShellyDevice(entity))
	}

	utils.SendSuccess(c, gin.H{
		"devices": devices,
		"count":   len(devices),
	})
}

// GetShellyDevice retrieves a specific Shelly device
func (h *Handlers) GetShellyDevice(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		utils.SendError(c, http.StatusBadRequest, "Device ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get the entity through unified service
	options := unified.GetEntityOptions{
		IncludeRoom: false,
		IncludeArea: false,
	}

	entityWithRoom, err := h.unifiedService.GetByID(ctx, deviceID, options)
	if err != nil {
		if err.Error() == "entity not found" {
			utils.SendError(c, http.StatusNotFound, "Device not found")
			return
		}
		h.log.WithError(err).Error("Failed to get Shelly device")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve Shelly device")
		return
	}

	device := convertEntityToShellyDevice(entityWithRoom.Entity)
	utils.SendSuccess(c, device)
}

// ControlShellyDevice controls a Shelly device
func (h *Handlers) ControlShellyDeviceV2(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		utils.SendError(c, http.StatusBadRequest, "Device ID is required")
		return
	}

	var req ShellyDeviceControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Create PMA control action
	action := types.PMAControlAction{
		EntityID:   deviceID,
		Action:     req.Action,
		Parameters: req.Parameters,
		Context: &types.PMAContext{
			ID:          uuid.New().String(),
			Source:      "api",
			Timestamp:   time.Now(),
			Description: "Shelly device control via API",
		},
	}

	// Execute through unified service
	result, err := h.unifiedService.ExecuteAction(ctx, action)
	if err != nil {
		h.log.WithError(err).Error("Failed to control Shelly device")
		utils.SendError(c, http.StatusInternalServerError, "Failed to control Shelly device")
		return
	}

	if !result.Success {
		if result.Error != nil && result.Error.Code == "ENTITY_NOT_FOUND" {
			utils.SendError(c, http.StatusNotFound, "Device not found")
			return
		}
		utils.SendError(c, http.StatusBadRequest, result.Error.Message)
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "Device control command executed successfully",
		"result":  result,
	})
}

// UpdateShellyConfig updates the Shelly adapter configuration
func (h *Handlers) UpdateShellyConfig(c *gin.Context) {
	var configUpdate map[string]interface{}
	if err := c.ShouldBindJSON(&configUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid configuration format",
			"details": err.Error(),
		})
		return
	}

	// Get the Shelly adapter through the registry manager
	adapter, err := h.adapterRegistry.GetAdapter(string(types.SourceShelly))
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Shelly adapter not available",
			"details": err.Error(),
		})
		return
	}

	// In a real implementation, this would update the configuration
	// For now, we'll just acknowledge the request and log the config update
	h.log.WithField("config_update", configUpdate).Info("Shelly configuration update requested")
	
	// Get adapter metrics for response
	metrics := adapter.GetMetrics()

	c.JSON(http.StatusOK, gin.H{
		"message": "Shelly configuration update request received",
		"config":  configUpdate,
		"metrics": metrics,
	})
}

// GetShellyAdapterStatus returns the status of the Shelly adapter
func (h *Handlers) GetShellyAdapterStatus(c *gin.Context) {
	adapter, err := h.adapterRegistry.GetAdapter(string(types.SourceShelly))
	if err != nil || adapter == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Shelly adapter not available")
		return
	}

	health := adapter.GetHealth()

	utils.SendSuccess(c, gin.H{
		"status":  adapter.GetStatus(),
		"health":  health,
		"metrics": adapter.GetMetrics(),
	})
}

// Helper function to convert PMA entity to Shelly device response
func convertEntityToShellyDevice(entity types.PMAEntity) ShellyDeviceResponse {
	attributes := entity.GetAttributes()
	
	device := ShellyDeviceResponse{
		ID:         entity.GetID(),
		Name:       entity.GetFriendlyName(),
		DeviceType: string(entity.GetType()),
		Online:     entity.IsAvailable(),
		LastUpdate: entity.GetLastUpdated(),
		Status:     make(map[string]interface{}),
	}

	// Extract common attributes
	if ip, ok := attributes["ip_address"].(string); ok {
		device.IP = ip
	}
	if model, ok := attributes["model"].(string); ok {
		device.Model = model
	}
	if firmware, ok := attributes["firmware_version"].(string); ok {
		device.Firmware = firmware
	}
	if mac, ok := attributes["mac_address"].(string); ok {
		device.MAC = mac
	}

	// Copy all attributes to status
	for key, value := range attributes {
		device.Status[key] = value
	}

	// Set device type based on entity type
	switch entity.GetType() {
	case types.EntityTypeSwitch:
		device.DeviceType = "switch"
	case types.EntityTypeLight:
		device.DeviceType = "light"
	case types.EntityTypeSensor:
		device.DeviceType = "sensor"
	case types.EntityTypeCover:
		device.DeviceType = "cover"
	default:
		device.DeviceType = "unknown"
	}

	return device
}