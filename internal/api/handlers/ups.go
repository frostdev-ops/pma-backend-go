package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// GetUPSStatus returns the current UPS status
func (h *Handlers) GetUPSStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	status, err := h.upsService.GetCurrentStatus(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get UPS status")
		return
	}

	utils.SendSuccess(c, status)
}

// GetUPSHistory returns UPS status history
func (h *Handlers) GetUPSHistory(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")
	limitStr := c.Query("limit")

	var startTime, endTime time.Time
	var limit int = 100

	// Parse start time
	if startTimeStr != "" {
		if parsed, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = parsed
		}
	} else {
		startTime = time.Now().Add(-24 * time.Hour) // Default to last 24 hours
	}

	// Parse end time
	if endTimeStr != "" {
		if parsed, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = parsed
		}
	} else {
		endTime = time.Now()
	}

	// Parse limit
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := h.upsService.GetStatusHistory(ctx, startTime, endTime, limit)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get UPS history")
		return
	}

	utils.SendSuccess(c, history)
}

// GetUPSBatteryTrends returns battery level trends
func (h *Handlers) GetUPSBatteryTrends(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	period := c.Query("period")
	if period == "" {
		period = "24h"
	}

	trends, err := h.upsService.GetBatteryTrends(ctx, period)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get battery trends")
		return
	}

	utils.SendSuccess(c, trends)
}

// GetUPSMetrics returns UPS metrics
func (h *Handlers) GetUPSMetrics(c *gin.Context) {
	metrics := h.upsService.GetMetrics()
	utils.SendSuccess(c, metrics)
}

// GetUPSConfiguration returns UPS configuration
func (h *Handlers) GetUPSConfiguration(c *gin.Context) {
	config := h.upsService.GetConfiguration()
	utils.SendSuccess(c, config)
}

// GetUPSInfo returns UPS information
func (h *Handlers) GetUPSInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	info, err := h.upsService.GetUPSInfo(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get UPS info")
		return
	}

	utils.SendSuccess(c, info)
}

// GetUPSVariables returns UPS variables
func (h *Handlers) GetUPSVariables(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	variables, err := h.upsService.GetUPSVariables(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get UPS variables")
		return
	}

	utils.SendSuccess(c, variables)
}

// GetUPSConnectionInfo returns UPS connection information
func (h *Handlers) GetUPSConnectionInfo(c *gin.Context) {
	info, err := h.upsService.GetConnectionInfo()
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get connection info")
		return
	}

	utils.SendSuccess(c, info)
}

// TestUPSConnection tests the UPS connection
func (h *Handlers) TestUPSConnection(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	result, err := h.upsService.TestConnection(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to test UPS connection")
		return
	}

	utils.SendSuccess(c, result)
}

// StartUPSMonitoring starts UPS monitoring
func (h *Handlers) StartUPSMonitoring(c *gin.Context) {
	err := h.upsService.StartMonitoring()
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to start UPS monitoring")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "UPS monitoring started successfully",
		"status":  "started",
	})
}

// StopUPSMonitoring stops UPS monitoring
func (h *Handlers) StopUPSMonitoring(c *gin.Context) {
	err := h.upsService.StopMonitoring()
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to stop UPS monitoring")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "UPS monitoring stopped successfully",
		"status":  "stopped",
	})
}

// UpdateUPSAlertThresholds updates UPS alert thresholds
func (h *Handlers) UpdateUPSAlertThresholds(c *gin.Context) {
	var thresholds map[string]interface{}
	if err := c.ShouldBindJSON(&thresholds); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := h.upsService.UpdateAlertThresholds(thresholds)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to update alert thresholds")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "Alert thresholds updated successfully",
	})
}

// GetUPSHealth returns UPS health status
func (h *Handlers) GetUPSHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	status, err := h.upsService.GetCurrentStatus(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get UPS health")
		return
	}

	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now(),
		"details":   status,
	}

	utils.SendSuccess(c, health)
}
