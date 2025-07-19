package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ShellyDeviceRequest represents a Shelly device configuration
type ShellyDeviceRequest struct {
	IP         string     `json:"ip" binding:"required"`
	Name       string     `json:"name" binding:"required"`
	DeviceType string     `json:"device_type"`
	Auth       ShellyAuth `json:"auth,omitempty"`
}

// ShellyAuth represents Shelly device authentication
type ShellyAuth struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// ShellyDeviceResponse represents a Shelly device
type ShellyDeviceResponse struct {
	ID         string                 `json:"id"`
	IP         string                 `json:"ip"`
	Name       string                 `json:"name"`
	DeviceType string                 `json:"device_type"`
	Model      string                 `json:"model"`
	Online     bool                   `json:"online"`
	LastUpdate time.Time              `json:"last_update"`
	Firmware   string                 `json:"firmware,omitempty"`
	MAC        string                 `json:"mac,omitempty"`
	Components []ShellyComponent      `json:"components,omitempty"`
	Power      *ShellyPowerInfo       `json:"power,omitempty"`
	Status     map[string]interface{} `json:"status,omitempty"`
}

// ShellyComponent represents a Shelly device component
type ShellyComponent struct {
	ID     int                    `json:"id"`
	Type   string                 `json:"type"`
	Name   string                 `json:"name"`
	State  string                 `json:"state"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// ShellyPowerInfo represents power consumption data
type ShellyPowerInfo struct {
	Current      float64   `json:"current"`      // Current power consumption in W
	Total        float64   `json:"total"`        // Total energy consumed in Wh
	Voltage      float64   `json:"voltage"`      // Voltage in V
	PowerFactor  float64   `json:"power_factor"` // Power factor
	LastMeasured time.Time `json:"last_measured"`
}

// ShellyControlRequest represents device control request
type ShellyControlRequest struct {
	Component int                    `json:"component"`
	Action    string                 `json:"action" binding:"required"`
	Params    map[string]interface{} `json:"params,omitempty"`
}

// ShellyDiscoveredDevice represents a discovered Shelly device
type ShellyDiscoveredDevice struct {
	IP         string    `json:"ip"`
	MAC        string    `json:"mac"`
	Model      string    `json:"model"`
	Name       string    `json:"name"`
	DeviceType string    `json:"device_type"`
	Firmware   string    `json:"firmware"`
	Online     bool      `json:"online"`
	Discovered time.Time `json:"discovered"`
}

// AddShellyDevice adds a new Shelly device
func (h *Handlers) AddShellyDevice(c *gin.Context) {
	var req ShellyDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Generate device ID
	deviceID := fmt.Sprintf("shelly_%s_%d", req.IP, time.Now().Unix())

	// Store device configuration
	deviceConfig := map[string]interface{}{
		"id":          deviceID,
		"ip":          req.IP,
		"name":        req.Name,
		"device_type": req.DeviceType,
		"auth":        req.Auth,
		"added_at":    time.Now(),
		"online":      false,
	}

	configJSON, _ := json.Marshal(deviceConfig)
	err := h.repos.Config.Set(c.Request.Context(), &models.SystemConfig{
		Key:         "shelly.device." + deviceID,
		Value:       string(configJSON),
		Description: "Shelly device configuration",
	})

	if err != nil {
		h.log.WithError(err).Error("Failed to store Shelly device configuration")
		utils.SendError(c, http.StatusInternalServerError, "Failed to add Shelly device")
		return
	}

	h.log.Infof("Added Shelly device: %s (%s)", req.Name, req.IP)

	// Broadcast device addition via WebSocket
	if h.wsHub != nil {
		data := map[string]interface{}{
			"device_id": deviceID,
			"name":      req.Name,
			"ip":        req.IP,
		}
		go h.wsHub.BroadcastToAll("shelly_device_added", data)
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Shelly device added successfully",
		"device_id": deviceID,
	})
}

// RemoveShellyDevice removes a Shelly device
func (h *Handlers) RemoveShellyDevice(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		utils.SendError(c, http.StatusBadRequest, "Device ID is required")
		return
	}

	// Remove device configuration
	err := h.repos.Config.Delete(c.Request.Context(), "shelly.device."+deviceID)
	if err != nil {
		h.log.WithError(err).Error("Failed to remove Shelly device")
		utils.SendError(c, http.StatusInternalServerError, "Failed to remove Shelly device")
		return
	}

	h.log.Infof("Removed Shelly device: %s", deviceID)

	// Broadcast device removal via WebSocket
	if h.wsHub != nil {
		data := map[string]interface{}{
			"device_id": deviceID,
		}
		go h.wsHub.BroadcastToAll("shelly_device_removed", data)
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Shelly device removed successfully",
		"device_id": deviceID,
	})
}

// GetShellyDevices returns all Shelly devices using the unified PMA service
func (h *Handlers) GetShellyDevices(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get Shelly device entities through unified service
	options := unified.GetAllOptions{
		AvailableOnly: false, // Include offline devices too
	}

	entitiesWithRooms, err := h.unifiedService.GetBySource(ctx, types.SourceShelly, options)
	if err != nil {
		h.log.WithError(err).Error("Failed to get Shelly devices from unified service")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve Shelly devices")
		return
	}

	// Convert PMA entities to Shelly device format for backward compatibility
	devices := make([]ShellyDeviceResponse, 0, len(entitiesWithRooms))
	for _, entityWithRoom := range entitiesWithRooms {
		entity := entityWithRoom.Entity
		attributes := entity.GetAttributes()

		device := ShellyDeviceResponse{
			ID:         entity.GetID(),
			Name:       entity.GetFriendlyName(),
			Online:     entity.IsAvailable(),
			LastUpdate: entity.GetLastUpdated(),
		}

		// Extract device-specific attributes if available
		if ip, ok := attributes["ip"].(string); ok {
			device.IP = ip
		}
		if deviceType, ok := attributes["device_type"].(string); ok {
			device.DeviceType = deviceType
		}
		if model, ok := attributes["model"].(string); ok {
			device.Model = model
		}
		if firmware, ok := attributes["firmware"].(string); ok {
			device.Firmware = firmware
		}
		if mac, ok := attributes["mac"].(string); ok {
			device.MAC = mac
		}

		// Extract components if available
		if componentsData, ok := attributes["components"].([]interface{}); ok {
			components := make([]ShellyComponent, 0, len(componentsData))
			for _, compData := range componentsData {
				if comp, ok := compData.(map[string]interface{}); ok {
					component := ShellyComponent{
						Name:  getString(comp, "name"),
						Type:  getString(comp, "type"),
						State: getString(comp, "state"),
					}
					if id, ok := comp["id"].(float64); ok {
						component.ID = int(id)
					}
					if config, ok := comp["config"].(map[string]interface{}); ok {
						component.Config = config
					}
					components = append(components, component)
				}
			}
			device.Components = components
		}

		// Extract power information if available
		if powerData, ok := attributes["power"].(map[string]interface{}); ok {
			power := &ShellyPowerInfo{}
			if current, ok := powerData["current"].(float64); ok {
				power.Current = current
			}
			if total, ok := powerData["total"].(float64); ok {
				power.Total = total
			}
			if voltage, ok := powerData["voltage"].(float64); ok {
				power.Voltage = voltage
			}
			if powerFactor, ok := powerData["power_factor"].(float64); ok {
				power.PowerFactor = powerFactor
			}
			if lastMeasured, ok := powerData["last_measured"].(string); ok {
				if parsedTime, err := time.Parse(time.RFC3339, lastMeasured); err == nil {
					power.LastMeasured = parsedTime
				}
			}
			device.Power = power
		}

		// Extract status if available
		if status, ok := attributes["status"].(map[string]interface{}); ok {
			device.Status = status
		}

		devices = append(devices, device)
	}

	utils.SendSuccess(c, gin.H{
		"devices": devices,
		"count":   len(devices),
	})
}

// GetShellyDeviceStatus returns device status
func (h *Handlers) GetShellyDeviceStatus(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		utils.SendError(c, http.StatusBadRequest, "Device ID is required")
		return
	}

	// Get device configuration
	config, err := h.repos.Config.Get(c.Request.Context(), "shelly.device."+deviceID)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "Device not found")
		return
	}

	var deviceData map[string]interface{}
	if err := json.Unmarshal([]byte(config.Value), &deviceData); err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Invalid device configuration")
		return
	}

	// Mock status data - in real implementation, query actual device
	status := map[string]interface{}{
		"online":      true,
		"temperature": 45.2,
		"uptime":      3600,
		"ram_free":    48000,
		"ram_total":   52000,
		"wifi": map[string]interface{}{
			"connected": true,
			"ssid":      "HomeNetwork",
			"rssi":      -67,
		},
		"relays": []map[string]interface{}{
			{
				"ison":           false,
				"has_timer":      false,
				"timer_duration": 0,
				"overpower":      false,
				"source":         "input",
			},
		},
		"meters": []map[string]interface{}{
			{
				"power":     12.5,
				"total":     156789,
				"timestamp": time.Now().Unix(),
			},
		},
	}

	utils.SendSuccess(c, status)
}

// ControlShellyDevice controls a Shelly device using the unified PMA service
func (h *Handlers) ControlShellyDevice(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		utils.SendError(c, http.StatusBadRequest, "Device ID is required")
		return
	}

	var req ShellyControlRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Create PMA control action for Shelly device control
	action := types.PMAControlAction{
		EntityID: deviceID,
		Action:   req.Action,
		Parameters: map[string]interface{}{
			"component": req.Component,
		},
		Context: &types.PMAContext{
			ID:          uuid.New().String(),
			Source:      "api",
			Timestamp:   time.Now(),
			Description: "Shelly device control via API",
		},
	}

	// Add any additional parameters from the request
	if req.Params != nil {
		for key, value := range req.Params {
			action.Parameters[key] = value
		}
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

	// Broadcast device state change via WebSocket
	if h.wsHub != nil {
		data := map[string]interface{}{
			"device_id": deviceID,
			"component": req.Component,
			"action":    req.Action,
			"result":    result,
		}
		go h.wsHub.BroadcastToAll("shelly_device_controlled", data)
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Device control command sent",
		"device_id": deviceID,
		"action":    req.Action,
		"component": req.Component,
		"result":    result,
	})
}

// GetShellyDeviceEnergy returns energy consumption data
func (h *Handlers) GetShellyDeviceEnergy(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		utils.SendError(c, http.StatusBadRequest, "Device ID is required")
		return
	}

	// Parse hours parameter
	hours := 24
	if hoursStr := c.Query("hours"); hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 {
			hours = h
		}
	}

	// Mock energy history data
	now := time.Now()
	var history []map[string]interface{}

	// Generate mock hourly data
	for i := hours; i > 0; i-- {
		timestamp := now.Add(-time.Duration(i) * time.Hour)
		power := 10.0 + float64(i%5)*2.5 // Vary between 10-20W
		energy := power * 1.0            // 1 hour consumption

		history = append(history, map[string]interface{}{
			"timestamp": timestamp,
			"power":     power,
			"energy":    energy,
		})
	}

	// Current power usage
	currentPower := 12.5

	utils.SendSuccess(c, gin.H{
		"device_id":     deviceID,
		"current_power": currentPower,
		"history":       history,
		"hours":         hours,
		"total_energy":  156.7,
		"last_updated":  time.Now(),
	})
}

// GetDiscoveredShellyDevices returns discovered Shelly devices
func (h *Handlers) GetDiscoveredShellyDevices(c *gin.Context) {
	// Mock discovered devices - in real implementation, perform network scan
	discovered := []ShellyDiscoveredDevice{}

	// Use mock devices from configuration if enabled
	if h.cfg.Devices.Shelly.MockDevices.Enabled {
		for _, mockDevice := range h.cfg.Devices.Shelly.MockDevices.Devices {
			discovered = append(discovered, ShellyDiscoveredDevice{
				IP:         mockDevice.IP,
				MAC:        mockDevice.MAC,
				Model:      mockDevice.Model,
				Name:       mockDevice.Name,
				DeviceType: mockDevice.Type,
				Firmware:   "20230913-114340/v1.14.0-gcb84623",
				Online:     true,
				Discovered: time.Now().Add(-5 * time.Minute),
			})
		}
	} else {
		// Default mock devices for demo (when config mock is disabled)
		discovered = []ShellyDiscoveredDevice{
			{
				IP:         "192.168.100.50",
				MAC:        "AA:BB:CC:DD:EE:01",
				Model:      "SHSW-1PM",
				Name:       "shelly1pm-DDEE01",
				DeviceType: "switch_pm",
				Firmware:   "20230913-114340/v1.14.0-gcb84623",
				Online:     true,
				Discovered: time.Now().Add(-5 * time.Minute),
			},
			{
				IP:         "192.168.100.51",
				MAC:        "AA:BB:CC:DD:EE:02",
				Model:      "SHSW-25",
				Name:       "shelly25-DDEE02",
				DeviceType: "roller",
				Firmware:   "20230913-114340/v1.14.0-gcb84623",
				Online:     true,
				Discovered: time.Now().Add(-10 * time.Minute),
			},
		}
	}

	utils.SendSuccess(c, gin.H{
		"devices":    discovered,
		"count":      len(discovered),
		"scanned_at": time.Now(),
	})
}

// StartShellyDiscovery starts device discovery
func (h *Handlers) StartShellyDiscovery(c *gin.Context) {
	h.log.Info("Starting Shelly device discovery")

	// In real implementation, start mDNS/network discovery process
	// For now, just return success

	// Store discovery status
	h.repos.Config.Set(c.Request.Context(), &models.SystemConfig{
		Key:         "shelly.discovery.status",
		Value:       "running",
		Description: "Shelly discovery status",
	})

	h.repos.Config.Set(c.Request.Context(), &models.SystemConfig{
		Key:         "shelly.discovery.started_at",
		Value:       strconv.FormatInt(time.Now().Unix(), 10),
		Description: "Shelly discovery start time",
	})

	utils.SendSuccess(c, gin.H{
		"message": "Shelly discovery started",
		"status":  "running",
	})
}

// StopShellyDiscovery stops device discovery
func (h *Handlers) StopShellyDiscovery(c *gin.Context) {
	h.log.Info("Stopping Shelly device discovery")

	// Update discovery status
	h.repos.Config.Set(c.Request.Context(), &models.SystemConfig{
		Key:         "shelly.discovery.status",
		Value:       "stopped",
		Description: "Shelly discovery status",
	})

	utils.SendSuccess(c, gin.H{
		"message": "Shelly discovery stopped",
		"status":  "stopped",
	})
}

// Helper functions
func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getBool(data map[string]interface{}, key string) bool {
	if val, ok := data[key].(bool); ok {
		return val
	}
	return false
}
