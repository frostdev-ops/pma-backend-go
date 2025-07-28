package handlers

import (
	"net/http"
	"regexp"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/bluetooth"
	"github.com/gin-gonic/gin"
)

// validateBluetoothAddress validates Bluetooth address format
func validateBluetoothAddress(address string) bool {
	regex := regexp.MustCompile(`^[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}$`)
	return regex.MatchString(address)
}

// GetBluetoothStatus gets Bluetooth availability and adapter status
func (h *Handlers) GetBluetoothStatus(c *gin.Context) {
	availability, err := h.bluetoothService.CheckAvailability(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to check Bluetooth availability")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to check Bluetooth availability",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if !availability.Available {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"available": false,
				"error":     availability.Error,
				"adapter":   nil,
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	adapter, err := h.bluetoothService.GetAdapterInfo(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get adapter info")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get adapter information",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"available": true,
			"adapter":   adapter,
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetBluetoothCapabilities gets Bluetooth service capabilities
func (h *Handlers) GetBluetoothCapabilities(c *gin.Context) {
	capabilities, err := h.bluetoothService.GetCapabilities(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get Bluetooth capabilities")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get Bluetooth capabilities",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      capabilities,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// SetBluetoothPower enables or disables Bluetooth adapter
func (h *Handlers) SetBluetoothPower(c *gin.Context) {
	var request struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid request format",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if err := h.bluetoothService.SetPower(c.Request.Context(), request.Enabled); err != nil {
		h.log.WithError(err).Error("Failed to set Bluetooth power state")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to set Bluetooth power state",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Bluetooth power state updated successfully",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// SetBluetoothDiscoverable makes adapter discoverable/non-discoverable
func (h *Handlers) SetBluetoothDiscoverable(c *gin.Context) {
	var request struct {
		Discoverable bool `json:"discoverable"`
		Timeout      int  `json:"timeout,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid request format",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if err := h.bluetoothService.SetDiscoverable(c.Request.Context(), request.Discoverable, request.Timeout); err != nil {
		h.log.WithError(err).Error("Failed to set Bluetooth discoverable state")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to set Bluetooth discoverable state",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Bluetooth discoverable state updated successfully",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ScanForDevices scans for nearby Bluetooth devices
func (h *Handlers) ScanForDevices(c *gin.Context) {
	var request bluetooth.ScanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid request format",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	// Validate duration
	if request.Duration < 1 || request.Duration > 60 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Duration must be between 1 and 60 seconds",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	h.log.Infof("Starting Bluetooth device scan for %d seconds", request.Duration)

	devices, err := h.bluetoothService.ScanForDevices(c.Request.Context(), request.Duration)
	if err != nil {
		h.log.WithError(err).Error("Failed to scan for Bluetooth devices")

		// Determine appropriate status code based on error
		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to scan for devices"

		if err.Error() == "device scan already in progress" {
			statusCode = http.StatusConflict
			errorMessage = err.Error()
		}

		c.JSON(statusCode, gin.H{
			"success":   false,
			"error":     errorMessage,
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      devices,
		"message":   "Device scan completed successfully",
		"count":     len(devices),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetPairedDevices gets all paired Bluetooth devices
func (h *Handlers) GetPairedDevices(c *gin.Context) {
	devices, err := h.bluetoothService.GetPairedDevices(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get paired devices")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get paired devices",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      devices,
		"count":     len(devices),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetConnectedDevices gets all connected Bluetooth devices
func (h *Handlers) GetConnectedDevices(c *gin.Context) {
	devices, err := h.bluetoothService.GetConnectedDevices(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get connected devices")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get connected devices",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      devices,
		"count":     len(devices),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetAllBluetoothDevices gets all Bluetooth devices from database
func (h *Handlers) GetAllBluetoothDevices(c *gin.Context) {
	devices, err := h.bluetoothService.GetAllDevices(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get all devices")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get all devices",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      devices,
		"count":     len(devices),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetBluetoothDevice gets information about a specific device
func (h *Handlers) GetBluetoothDevice(c *gin.Context) {
	address := c.Param("address")

	if !validateBluetoothAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid Bluetooth address format. Expected format: XX:XX:XX:XX:XX:XX",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	device, err := h.bluetoothService.GetDeviceInfo(c.Request.Context(), address)
	if err != nil {
		h.log.WithError(err).Error("Failed to get device info")
		c.JSON(http.StatusNotFound, gin.H{
			"success":   false,
			"error":     "Device not found or not accessible",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      device,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// PairBluetoothDevice pairs with a Bluetooth device
func (h *Handlers) PairBluetoothDevice(c *gin.Context) {
	address := c.Param("address")

	if !validateBluetoothAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid Bluetooth address format",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	var requestBody struct {
		PIN     string                  `json:"pin,omitempty"`
		Method  bluetooth.PairingMethod `json:"method,omitempty"`
		Timeout int                     `json:"timeout,omitempty"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		// If no body provided, use defaults
		requestBody = struct {
			PIN     string                  `json:"pin,omitempty"`
			Method  bluetooth.PairingMethod `json:"method,omitempty"`
			Timeout int                     `json:"timeout,omitempty"`
		}{}
	}

	request := &bluetooth.PairRequest{
		Address: address,
		Method:  requestBody.Method,
		PIN:     requestBody.PIN,
		Timeout: requestBody.Timeout,
	}

	response, err := h.bluetoothService.PairDevice(c.Request.Context(), request)
	if err != nil {
		h.log.WithError(err).Error("Failed to pair device")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to pair device",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if !response.Success {
		statusCode := http.StatusBadRequest
		if response.Message == "Another pairing operation is already in progress" {
			statusCode = http.StatusConflict
		}

		c.JSON(statusCode, gin.H{
			"success":   false,
			"error":     response.Message,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      response,
		"message":   response.Message,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ConnectBluetoothDevice connects to a paired device
func (h *Handlers) ConnectBluetoothDevice(c *gin.Context) {
	address := c.Param("address")

	if !validateBluetoothAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid Bluetooth address format",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	var requestBody struct {
		AutoConnect bool `json:"auto_connect"`
		Trust       bool `json:"trust"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		// If no body provided, use defaults
		requestBody = struct {
			AutoConnect bool `json:"auto_connect"`
			Trust       bool `json:"trust"`
		}{
			AutoConnect: false,
			Trust:       false,
		}
	}

	request := &bluetooth.ConnectRequest{
		Address:     address,
		AutoConnect: requestBody.AutoConnect,
		Trust:       requestBody.Trust,
	}

	if err := h.bluetoothService.ConnectDevice(c.Request.Context(), request); err != nil {
		h.log.WithError(err).Error("Failed to connect to device")

		statusCode := http.StatusInternalServerError
		errorMessage := "Failed to connect to device"

		if err.Error() == "device must be paired before connecting" {
			statusCode = http.StatusBadRequest
			errorMessage = err.Error()
		}

		c.JSON(statusCode, gin.H{
			"success":   false,
			"error":     errorMessage,
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Successfully connected to device",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// DisconnectBluetoothDevice disconnects from a connected device
func (h *Handlers) DisconnectBluetoothDevice(c *gin.Context) {
	address := c.Param("address")

	if !validateBluetoothAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid Bluetooth address format",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if err := h.bluetoothService.DisconnectDevice(c.Request.Context(), address); err != nil {
		h.log.WithError(err).Error("Failed to disconnect from device")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to disconnect from device",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Successfully disconnected from device",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// RemoveBluetoothDevice removes/unpairs a device
func (h *Handlers) RemoveBluetoothDevice(c *gin.Context) {
	address := c.Param("address")

	if !validateBluetoothAddress(address) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid Bluetooth address format",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	if err := h.bluetoothService.RemoveDevice(c.Request.Context(), address); err != nil {
		h.log.WithError(err).Error("Failed to remove device")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to remove device",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Successfully removed device",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetBluetoothStats gets Bluetooth device statistics
func (h *Handlers) GetBluetoothStats(c *gin.Context) {
	stats, err := h.bluetoothService.GetDeviceStats(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get Bluetooth stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get Bluetooth statistics",
			"details":   err.Error(),
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      stats,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
