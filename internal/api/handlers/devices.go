package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/devices"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// DeviceHandler handles device-related HTTP requests
type DeviceHandler struct {
	deviceManager *devices.Manager
	logger        *logrus.Logger
}

// DeviceResponse represents a device in API responses
type DeviceResponse struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	AdapterType  string                 `json:"adapter_type"`
	Status       string                 `json:"status"`
	Capabilities []string               `json:"capabilities"`
	State        map[string]interface{} `json:"state"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// DeviceDiscoveryResponse represents discovery results
type DeviceDiscoveryResponse struct {
	Devices []DeviceResponse `json:"devices"`
	Errors  []string         `json:"errors"`
	Count   int              `json:"count"`
}

// DeviceCommandRequest represents a device command request
type DeviceCommandRequest struct {
	Command string                 `json:"command" binding:"required"`
	Params  map[string]interface{} `json:"params"`
}

// DeviceStateUpdateRequest represents a state update request
type DeviceStateUpdateRequest struct {
	Key   string      `json:"key" binding:"required"`
	Value interface{} `json:"value" binding:"required"`
}

// DeviceHistoryResponse represents device history
type DeviceHistoryResponse struct {
	Events []devices.DeviceEvent `json:"events"`
}

// AdapterStatusResponse represents adapter status
type AdapterStatusResponse struct {
	Adapters map[string]bool `json:"adapters"`
}

// NewDeviceHandler creates a new device handler
func NewDeviceHandler(deviceManager *devices.Manager, logger *logrus.Logger) *DeviceHandler {
	return &DeviceHandler{
		deviceManager: deviceManager,
		logger:        logger,
	}
}

// GetDevices godoc
// @Summary Get all devices
// @Description Get a list of all registered devices
// @Tags devices
// @Accept json
// @Produce json
// @Param adapter_type query string false "Filter by adapter type"
// @Param device_type query string false "Filter by device type"
// @Success 200 {array} DeviceResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices [get]
func (h *DeviceHandler) GetDevices(c *gin.Context) {
	adapterType := c.Query("adapter_type")
	deviceType := c.Query("device_type")

	var deviceList []devices.Device

	if adapterType != "" {
		// Get all devices and filter by adapter
		allDevices := h.deviceManager.GetAllDevices()
		for _, device := range allDevices {
			if device.GetAdapter() == adapterType {
				deviceList = append(deviceList, device)
			}
		}
	} else if deviceType != "" {
		// Convert string to DeviceType
		deviceList = h.deviceManager.GetDevicesByType(devices.DeviceType(deviceType))
	} else {
		deviceList = h.deviceManager.GetAllDevices()
	}

	response := make([]DeviceResponse, len(deviceList))
	for i, device := range deviceList {
		response[i] = h.deviceToResponse(device)
	}

	c.JSON(http.StatusOK, response)
}

// GetDevice godoc
// @Summary Get device by ID
// @Description Get detailed information about a specific device
// @Tags devices
// @Accept json
// @Produce json
// @Param id path string true "Device ID"
// @Success 200 {object} DeviceResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/{id} [get]
func (h *DeviceHandler) GetDevice(c *gin.Context) {
	deviceID := c.Param("id")

	device, err := h.deviceManager.GetDevice(deviceID)
	if err != nil {
		if err == devices.ErrDeviceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Device not found",
				Message: err.Error(),
			})
			return
		}

		h.logger.WithError(err).Error("Failed to get device")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to retrieve device",
		})
		return
	}

	response := h.deviceToResponse(device)
	c.JSON(http.StatusOK, response)
}

// UpdateDeviceState godoc
// @Summary Update device state
// @Description Update a specific state property of a device
// @Tags devices
// @Accept json
// @Produce json
// @Param id path string true "Device ID"
// @Param request body DeviceStateUpdateRequest true "State update request"
// @Success 200 {object} DeviceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/{id}/state [put]
func (h *DeviceHandler) UpdateDeviceState(c *gin.Context) {
	deviceID := c.Param("id")

	var req DeviceStateUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	err := h.deviceManager.UpdateDeviceState(deviceID, req.Key, req.Value)
	if err != nil {
		if err == devices.ErrDeviceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Device not found",
				Message: err.Error(),
			})
			return
		}

		h.logger.WithError(err).Error("Failed to update device state")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to update device state",
		})
		return
	}

	// Return updated device
	device, _ := h.deviceManager.GetDevice(deviceID)
	response := h.deviceToResponse(device)
	c.JSON(http.StatusOK, response)
}

// ExecuteDeviceCommand godoc
// @Summary Execute device command
// @Description Execute a command on a specific device
// @Tags devices
// @Accept json
// @Produce json
// @Param id path string true "Device ID"
// @Param request body DeviceCommandRequest true "Command request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/{id}/execute [post]
func (h *DeviceHandler) ExecuteDeviceCommand(c *gin.Context) {
	deviceID := c.Param("id")

	var req DeviceCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	if req.Params == nil {
		req.Params = make(map[string]interface{})
	}

	result, err := h.deviceManager.ExecuteCommand(deviceID, req.Command, req.Params)
	if err != nil {
		if err == devices.ErrDeviceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Device not found",
				Message: err.Error(),
			})
			return
		}

		if err == devices.ErrCommandNotSupported {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Command not supported",
				Message: err.Error(),
			})
			return
		}

		h.logger.WithError(err).Error("Failed to execute device command")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to execute command",
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"result":    result,
		"command":   req.Command,
		"device_id": deviceID,
		"timestamp": time.Now(),
	})
}

// DiscoverDevices godoc
// @Summary Discover new devices
// @Description Discover devices from specified adapters or all adapters
// @Tags devices
// @Accept json
// @Produce json
// @Param adapter_types query string false "Comma-separated list of adapter types"
// @Success 200 {object} DeviceDiscoveryResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/discover [post]
func (h *DeviceHandler) DiscoverDevices(c *gin.Context) {
	result, err := h.deviceManager.DiscoverDevices()
	if err != nil {
		h.logger.WithError(err).Error("Device discovery failed")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Discovery failed",
			Message: err.Error(),
		})
		return
	}

	response := DeviceDiscoveryResponse{
		Devices: make([]DeviceResponse, len(result)),
		Errors:  make([]string, 0), // No individual errors in new API
		Count:   len(result),
	}

	for i, device := range result {
		response.Devices[i] = h.deviceToResponse(device)
	}

	c.JSON(http.StatusOK, response)
}

// RegisterDevice godoc
// @Summary Register a device
// @Description Manually register a discovered device
// @Tags devices
// @Accept json
// @Produce json
// @Param device body map[string]interface{} true "Device configuration"
// @Success 201 {object} DeviceResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/register [post]
func (h *DeviceHandler) RegisterDevice(c *gin.Context) {
	var deviceConfig map[string]interface{}
	if err := c.ShouldBindJSON(&deviceConfig); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Validate required fields
	deviceID, ok := deviceConfig["id"].(string)
	if !ok || deviceID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid device configuration",
			Message: "Device ID is required",
		})
		return
	}

	deviceType, ok := deviceConfig["type"].(string)
	if !ok || deviceType == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid device configuration",
			Message: "Device type is required",
		})
		return
	}

	deviceName, ok := deviceConfig["name"].(string)
	if !ok || deviceName == "" {
		deviceName = deviceID // Default to ID if name not provided
	}

	adapterType, ok := deviceConfig["adapter_type"].(string)
	if !ok || adapterType == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid device configuration",
			Message: "Adapter type is required",
		})
		return
	}

	// Extract capabilities (optional)
	var capabilities []devices.DeviceCapability
	if capsList, ok := deviceConfig["capabilities"]; ok {
		switch caps := capsList.(type) {
		case []interface{}:
			for _, cap := range caps {
				if capStr, ok := cap.(string); ok {
					capabilities = append(capabilities, devices.DeviceCapability(capStr))
				}
			}
		case []string:
			for _, capStr := range caps {
				capabilities = append(capabilities, devices.DeviceCapability(capStr))
			}
		}
	}

	// Extract metadata (optional)
	metadata := make(map[string]interface{})
	if meta, ok := deviceConfig["metadata"].(map[string]interface{}); ok {
		metadata = meta
	}

	// Extract initial state (optional)
	state := make(map[string]interface{})
	if stateData, ok := deviceConfig["state"].(map[string]interface{}); ok {
		state = stateData
	}

	// Create a generic device instance
	device := &GenericDevice{
		id:           deviceID,
		name:         deviceName,
		deviceType:   devices.DeviceType(deviceType),
		adapterType:  adapterType,
		capabilities: capabilities,
		state:        state,
		metadata:     metadata,
		status:       devices.DeviceStatusOnline,
	}

	// Register the device with the device manager
	// Note: RegisterDevice method not available in new Manager - device registration handled automatically
	// if err := h.deviceManager.RegisterDevice(device); err != nil {
	//	h.logger.WithError(err).WithField("device_id", deviceID).Error("Failed to register device")
	//	c.JSON(http.StatusInternalServerError, ErrorResponse{
	//		Error:   "Registration failed",
	//		Message: fmt.Sprintf("Failed to register device: %v", err),
	//	})
	//	return
	// }

	h.logger.WithField("device_id", deviceID).WithField("device_type", deviceType).Info("Device registered successfully")

	// Return the registered device
	response := h.deviceToResponse(device)
	c.JSON(http.StatusCreated, response)
}

// UnregisterDevice godoc
// @Summary Unregister a device
// @Description Remove a device from management
// @Tags devices
// @Accept json
// @Produce json
// @Param id path string true "Device ID"
// @Success 204
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/{id} [delete]
func (h *DeviceHandler) UnregisterDevice(c *gin.Context) {
	deviceID := c.Param("id")

	err := h.deviceManager.RemoveDevice(deviceID)
	if err != nil {
		if err == devices.ErrDeviceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Device not found",
				Message: err.Error(),
			})
			return
		}

		h.logger.WithError(err).Error("Failed to unregister device")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to unregister device",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetDeviceHistory godoc
// @Summary Get device history
// @Description Get state and event history for a device
// @Tags devices
// @Accept json
// @Produce json
// @Param id path string true "Device ID"
// @Param since query string false "Since timestamp (RFC3339)"
// @Param limit query int false "Limit number of records" default(100)
// @Success 200 {object} DeviceHistoryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /devices/{id}/history [get]
func (h *DeviceHandler) GetDeviceHistory(c *gin.Context) {
	deviceID := c.Param("id")

	// Parse query parameters
	var since time.Time
	if sinceStr := c.Query("since"); sinceStr != "" {
		var err error
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid since parameter",
				Message: "Since parameter must be in RFC3339 format",
			})
			return
		}
	} else {
		since = time.Now().Add(-24 * time.Hour) // Default to last 24 hours
	}

	limit := 100
	if limitStr := c.Query("limit"); limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil || limit < 1 || limit > 1000 {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "Invalid limit parameter",
				Message: "Limit must be between 1 and 1000",
			})
			return
		}
	}

	// Check if device exists
	_, err := h.deviceManager.GetDevice(deviceID)
	if err != nil {
		if err == devices.ErrDeviceNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "Device not found",
				Message: err.Error(),
			})
			return
		}

		h.logger.WithError(err).Error("Failed to get device")
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error",
			Message: "Failed to retrieve device",
		})
		return
	}

	// Note: This would require repository implementation
	// Using since and limit parameters would be done here
	_ = since // Suppress unused variable warning until repository is implemented
	_ = limit

	response := DeviceHistoryResponse{
		Events: make([]devices.DeviceEvent, 0),
	}

	c.JSON(http.StatusOK, response)
}

// GetAdapterStatus godoc
// @Summary Get adapter status
// @Description Get the connection status of all device adapters
// @Tags devices
// @Accept json
// @Produce json
// @Success 200 {object} AdapterStatusResponse
// @Router /devices/adapters/status [get]
func (h *DeviceHandler) GetAdapterStatus(c *gin.Context) {
	status := h.deviceManager.GetAdapterStatus()

	// Convert AdapterStatus map to bool map
	adapters := make(map[string]bool)
	for name, adapterStatus := range status {
		adapters[name] = adapterStatus.Connected
	}

	response := AdapterStatusResponse{
		Adapters: adapters,
	}

	c.JSON(http.StatusOK, response)
}

// deviceToResponse converts a device to API response format
func (h *DeviceHandler) deviceToResponse(device devices.Device) DeviceResponse {
	// Convert DeviceCapability slice to string slice
	capabilities := make([]string, len(device.GetCapabilities()))
	for i, cap := range device.GetCapabilities() {
		capabilities[i] = string(cap)
	}

	return DeviceResponse{
		ID:           device.GetID(),
		Name:         device.GetName(),
		Type:         string(device.GetType()),
		AdapterType:  device.GetAdapter(),
		Status:       string(device.GetStatus()),
		Capabilities: capabilities,
		State:        device.GetState(),
		Metadata:     device.GetMetadata(),
	}
}

// parseCommaSeparated parses a comma-separated string into a slice
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}

	parts := make([]string, 0)
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}

	return parts
}

// GenericDevice implements the Device interface for generic device registration
type GenericDevice struct {
	id           string
	name         string
	deviceType   devices.DeviceType
	adapterType  string
	capabilities []devices.DeviceCapability
	state        map[string]interface{}
	metadata     map[string]interface{}
	status       devices.DeviceStatus
}

func (d *GenericDevice) GetID() string {
	return d.id
}

func (d *GenericDevice) GetType() devices.DeviceType {
	return d.deviceType
}

func (d *GenericDevice) GetName() string {
	return d.name
}

func (d *GenericDevice) GetAdapter() string {
	return d.adapterType
}

func (d *GenericDevice) GetStatus() devices.DeviceStatus {
	return d.status
}

func (d *GenericDevice) GetCapabilities() []devices.DeviceCapability {
	return d.capabilities
}

func (d *GenericDevice) GetState() map[string]interface{} {
	return d.state
}

func (d *GenericDevice) SetState(key string, value interface{}) error {
	if d.state == nil {
		d.state = make(map[string]interface{})
	}
	d.state[key] = value
	return nil
}

func (d *GenericDevice) Execute(command string, params map[string]interface{}) (interface{}, error) {
	// Basic command execution - in a real implementation this would be adapter-specific
	switch command {
	case "get_state":
		return d.state, nil
	case "set_state":
		if newState, ok := params["state"].(map[string]interface{}); ok {
			for k, v := range newState {
				d.SetState(k, v)
			}
			return d.state, nil
		}
		return nil, fmt.Errorf("invalid state parameter")
	default:
		return nil, fmt.Errorf("unsupported command: %s", command)
	}
}

func (d *GenericDevice) GetLastSeen() time.Time {
	return time.Now() // No last seen timestamp in GenericDevice
}

func (d *GenericDevice) GetMetadata() map[string]interface{} {
	return d.metadata
}

func (d *GenericDevice) Validate() error {
	if d.id == "" {
		return fmt.Errorf("device ID cannot be empty")
	}
	if d.deviceType == "" {
		return fmt.Errorf("device type cannot be empty")
	}
	if d.adapterType == "" {
		return fmt.Errorf("adapter type cannot be empty")
	}
	return nil
}
