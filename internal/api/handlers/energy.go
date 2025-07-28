package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/energy"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// GetEnergySettings retrieves energy settings
func (h *Handlers) GetEnergySettings(c *gin.Context) {
	if h.energyService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Energy service not available")
		return
	}

	settings := h.energyService.GetSettings()
	if settings == nil {
		h.log.Error("Failed to get energy settings")
		utils.SendError(c, http.StatusInternalServerError, "Failed to get energy settings")
		return
	}

	// Get current energy data if available
	energyData, err := h.energyService.GetCurrentEnergyData()
	if err != nil {
		h.log.WithError(err).Warn("Failed to get current energy data")
	}

	// Create comprehensive settings response
	response := map[string]interface{}{
		"enabled":                  settings.TrackingEnabled,
		"cost_per_kwh":             settings.EnergyRate,
		"currency":                 settings.Currency,
		"update_interval":          settings.UpdateInterval,
		"retention_days":           settings.HistoricalPeriod,
		"track_individual_devices": true, // Always enabled in our implementation

		// Current energy status
		"current_data": map[string]interface{}{
			"total_power_consumption": 0.0,
			"total_energy_usage":      0.0,
			"total_cost":              0.0,
			"device_count":            0,
		},

		// Service status
		"service_status": map[string]interface{}{
			"initialized": h.energyService != nil,
			"tracking":    settings.TrackingEnabled,
			"last_update": settings.UpdatedAt,
		},
	}

	// Add current energy data if available
	if energyData != nil {
		response["current_data"] = map[string]interface{}{
			"total_power_consumption": energyData.TotalPowerConsumption,
			"total_energy_usage":      energyData.TotalEnergyUsage,
			"total_cost":              energyData.TotalCost,
			"device_count":            len(energyData.DeviceBreakdown),
			"last_updated":            energyData.Timestamp,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}

// UpdateEnergySettings updates energy settings
func (h *Handlers) UpdateEnergySettings(c *gin.Context) {
	if h.energyService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Energy service not available")
		return
	}

	var request map[string]interface{}
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request data")
		return
	}

	// Get current settings
	currentSettings := h.energyService.GetSettings()
	if currentSettings == nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get current energy settings")
		return
	}

	// Create updated settings from request
	updatedSettings := &energy.EnergySettingsRequest{
		EnergyRate:       &currentSettings.EnergyRate,
		Currency:         &currentSettings.Currency,
		TrackingEnabled:  &currentSettings.TrackingEnabled,
		UpdateInterval:   &currentSettings.UpdateInterval,
		HistoricalPeriod: &currentSettings.HistoricalPeriod,
	}

	// Update fields from request
	if enabled, ok := request["enabled"].(bool); ok {
		updatedSettings.TrackingEnabled = &enabled
	}
	if rate, ok := request["cost_per_kwh"].(float64); ok && rate > 0 {
		updatedSettings.EnergyRate = &rate
	}
	if currency, ok := request["currency"].(string); ok && currency != "" {
		updatedSettings.Currency = &currency
	}
	if interval, ok := request["update_interval"].(float64); ok && interval > 0 {
		intervalInt := int(interval)
		updatedSettings.UpdateInterval = &intervalInt
	}
	if retention, ok := request["retention_days"].(float64); ok && retention > 0 {
		retentionInt := int(retention)
		updatedSettings.HistoricalPeriod = &retentionInt
	}

	// Update settings through service
	if err := h.energyService.UpdateSettings(updatedSettings); err != nil {
		h.log.WithError(err).Error("Failed to update energy settings")
		utils.SendError(c, http.StatusInternalServerError, "Failed to update energy settings")
		return
	}

	h.log.Info("Energy settings updated successfully")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Energy settings updated successfully",
		"data":    updatedSettings,
	})
}

// GetEnergyData retrieves current energy data
func (h *Handlers) GetEnergyData(c *gin.Context) {
	if h.energyService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Energy service not available"})
		return
	}

	energyData, err := h.energyService.GetCurrentEnergyData()
	if err != nil {
		h.log.WithError(err).Error("Failed to get energy data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get energy data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    energyData,
	})
}

// GetEnergyHistory retrieves energy consumption history
func (h *Handlers) GetEnergyHistory(c *gin.Context) {
	if h.energyService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Energy service not available"})
		return
	}

	// Parse query parameters
	filter := &energy.EnergyHistoryFilter{}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartDate = &startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndDate = &endTime
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	history, err := h.energyService.GetEnergyHistory(filter)
	if err != nil {
		h.log.WithError(err).Error("Failed to get energy history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get energy history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    history,
	})
}

// GetDeviceEnergyHistory retrieves energy history for a specific device
func (h *Handlers) GetDeviceEnergyHistory(c *gin.Context) {
	if h.energyService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Energy service not available"})
		return
	}

	entityID := c.Param("entityId")
	if entityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entity ID is required"})
		return
	}

	// Parse query parameters
	filter := &energy.DeviceEnergyFilter{
		EntityID: &entityID,
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartDate = &startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndDate = &endTime
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	history, err := h.energyService.GetDeviceEnergyHistory(filter)
	if err != nil {
		h.log.WithError(err).Error("Failed to get device energy history")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get device energy history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    history,
	})
}

// GetEnergyStatistics retrieves energy statistics
func (h *Handlers) GetEnergyStatistics(c *gin.Context) {
	if h.energyService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Energy service not available"})
		return
	}

	// Parse query parameters
	var startTime, endTime *time.Time

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	stats, err := h.energyService.GetEnergyStatistics(startTime, endTime)
	if err != nil {
		h.log.WithError(err).Error("Failed to get energy statistics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get energy statistics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetEnergyDeviceBreakdown retrieves energy breakdown by device
func (h *Handlers) GetEnergyDeviceBreakdown(c *gin.Context) {
	if h.energyService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Energy service not available"})
		return
	}

	breakdown, err := h.energyService.GetEnergyDeviceBreakdown()
	if err != nil {
		h.log.WithError(err).Error("Failed to get energy device breakdown")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get energy device breakdown"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    breakdown,
	})
}

// GetDeviceEnergyData retrieves comprehensive energy data for a specific device
func (h *Handlers) GetDeviceEnergyData(c *gin.Context) {
	if h.energyService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Energy service not available"})
		return
	}

	entityID := c.Param("entityId")
	if entityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Entity ID is required"})
		return
	}

	deviceData, err := h.energyService.GetDeviceEnergyData(entityID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get device energy data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get device energy data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    deviceData,
	})
}

// StartEnergyTracking starts energy monitoring
func (h *Handlers) StartEnergyTracking(c *gin.Context) {
	if h.energyService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Energy service not available"})
		return
	}

	if err := h.energyService.StartTracking(); err != nil {
		h.log.WithError(err).Error("Failed to start energy tracking")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start energy tracking"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Energy tracking started successfully",
	})
}

// StopEnergyTracking stops energy monitoring
func (h *Handlers) StopEnergyTracking(c *gin.Context) {
	if h.energyService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Energy service not available"})
		return
	}

	if err := h.energyService.StopTracking(); err != nil {
		h.log.WithError(err).Error("Failed to stop energy tracking")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop energy tracking"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Energy tracking stopped successfully",
	})
}

// GetEnergyMetrics retrieves energy monitoring metrics
func (h *Handlers) GetEnergyMetrics(c *gin.Context) {
	if h.energyService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Energy service not available"})
		return
	}

	metrics, err := h.energyService.GetEnergyMetrics()
	if err != nil {
		h.log.WithError(err).Error("Failed to get energy metrics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get energy metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    metrics,
	})
}

// CleanupOldEnergyData removes old energy data
func (h *Handlers) CleanupOldEnergyData(c *gin.Context) {
	if h.energyService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Energy service not available"})
		return
	}

	if err := h.energyService.CleanupOldData(); err != nil {
		h.log.WithError(err).Error("Failed to cleanup old energy data")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cleanup old energy data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Old energy data cleaned up successfully",
	})
}

// GetEnergyServiceStatus retrieves energy service status
func (h *Handlers) GetEnergyServiceStatus(c *gin.Context) {
	if h.energyService == nil {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"available":     false,
				"initialized":   false,
				"tracking":      false,
				"last_update":   nil,
				"total_devices": 0,
			},
		})
		return
	}

	status, err := h.energyService.GetServiceStatus()
	if err != nil {
		h.log.WithError(err).Error("Failed to get energy service status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get energy service status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}
