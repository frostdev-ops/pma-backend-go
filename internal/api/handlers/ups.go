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

// GetUPSStatus handles GET /api/v1/ups/status
func (h *Handlers) GetUPSStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	status, err := h.upsService.GetCurrentStatus(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get UPS status: "+err.Error())
		return
	}

	utils.SendSuccess(c, status)
}

// GetUPSHistory handles GET /api/v1/ups/history
func (h *Handlers) GetUPSHistory(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse limit parameter
	limitStr := c.Query("limit")
	limit := 100 // default
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	if limit > 1000 {
		limit = 1000 // max limit
	}

	history, err := h.upsService.GetStatusHistory(ctx, limit)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get UPS history: "+err.Error())
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"history": history,
		"count":   len(history),
		"limit":   limit,
	})
}

// GetUPSBatteryTrends handles GET /api/v1/ups/battery-trends
func (h *Handlers) GetUPSBatteryTrends(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse hours parameter
	hoursStr := c.Query("hours")
	hours := 24 // default 24 hours
	if hoursStr != "" {
		if parsedHours, err := strconv.Atoi(hoursStr); err == nil && parsedHours > 0 {
			hours = parsedHours
		}
	}

	if hours > 168 { // max 1 week
		hours = 168
	}

	trends, err := h.upsService.GetBatteryTrends(ctx, hours)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get battery trends: "+err.Error())
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"trends":     trends,
		"count":      len(trends),
		"hours":      hours,
		"time_range": time.Duration(hours) * time.Hour,
	})
}

// GetUPSInfo handles GET /api/v1/ups/info
func (h *Handlers) GetUPSInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	info, err := h.upsService.GetUPSInfo(ctx)
	if err != nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Failed to get UPS info: "+err.Error())
		return
	}

	utils.SendSuccess(c, info)
}

// GetUPSVariables handles GET /api/v1/ups/variables
func (h *Handlers) GetUPSVariables(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	variables, err := h.upsService.GetUPSVariables(ctx)
	if err != nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Failed to get UPS variables: "+err.Error())
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"variables": variables,
		"count":     len(variables),
	})
}

// StartUPSMonitoring handles POST /api/v1/ups/monitoring/start
func (h *Handlers) StartUPSMonitoring(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := h.upsService.StartMonitoring(ctx)
	if err != nil {
		utils.SendError(c, http.StatusConflict, "Failed to start UPS monitoring: "+err.Error())
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"message": "UPS monitoring started successfully",
		"status":  "monitoring",
	})
}

// StopUPSMonitoring handles POST /api/v1/ups/monitoring/stop
func (h *Handlers) StopUPSMonitoring(c *gin.Context) {
	err := h.upsService.StopMonitoring()
	if err != nil {
		utils.SendError(c, http.StatusConflict, "Failed to stop UPS monitoring: "+err.Error())
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"message": "UPS monitoring stopped successfully",
		"status":  "stopped",
	})
}

// TestUPSConnection handles POST /api/v1/ups/test-connection
func (h *Handlers) TestUPSConnection(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err := h.upsService.TestConnection(ctx)
	responseTime := time.Since(start)

	if err != nil {
		utils.SendError(c, http.StatusServiceUnavailable, "UPS connection test failed: "+err.Error())
		return
	}

	utils.SendSuccess(c, map[string]interface{}{
		"message":          "UPS connection test successful",
		"connection_time":  time.Now(),
		"response_time_ms": responseTime.Milliseconds(),
		"status":           "connected",
	})
}

// GetUPSConnectionInfo handles GET /api/v1/ups/connection
func (h *Handlers) GetUPSConnectionInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := h.upsService.GetConnectionInfo(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get connection info: "+err.Error())
		return
	}

	utils.SendSuccess(c, info)
}

// GetUPSMetrics handles GET /api/v1/ups/metrics
func (h *Handlers) GetUPSMetrics(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get current status
	status, err := h.upsService.GetCurrentStatus(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get UPS metrics: "+err.Error())
		return
	}

	// Get basic trend data (last 24 hours)
	trends, err := h.upsService.GetBatteryTrends(ctx, 24)
	if err != nil {
		h.log.WithError(err).Warn("Failed to get battery trends for metrics")
		trends = []*models.UPSStatus{} // Empty slice as fallback
	}

	// Calculate metrics
	metrics := map[string]interface{}{
		"current_status": map[string]interface{}{
			"battery_charge":    status.BatteryCharge,
			"battery_runtime":   status.BatteryRuntime,
			"input_voltage":     status.InputVoltage,
			"output_voltage":    status.OutputVoltage,
			"load":              status.Load,
			"temperature":       status.Temperature,
			"status":            status.Status,
			"connection_status": status.ConnectionStatus,
			"alert_level":       status.AlertLevel,
		},
		"alerts": map[string]interface{}{
			"active_alerts": status.Alerts,
			"alert_count":   len(status.Alerts),
			"has_critical":  status.AlertLevel == "critical",
			"has_warnings":  status.AlertLevel == "warning" || status.AlertLevel == "critical",
		},
		"trends": map[string]interface{}{
			"data_points":     len(trends),
			"time_span_hours": 24,
		},
		"system": map[string]interface{}{
			"monitoring_active": h.upsService.IsMonitoring(),
			"last_updated":      status.LastUpdated,
		},
	}

	// Add battery trend statistics if we have data
	if len(trends) > 0 {
		var minCharge, maxCharge, avgCharge float64
		minCharge = trends[0].BatteryCharge
		maxCharge = trends[0].BatteryCharge
		totalCharge := 0.0

		for _, trend := range trends {
			if trend.BatteryCharge < minCharge {
				minCharge = trend.BatteryCharge
			}
			if trend.BatteryCharge > maxCharge {
				maxCharge = trend.BatteryCharge
			}
			totalCharge += trend.BatteryCharge
		}
		avgCharge = totalCharge / float64(len(trends))

		metrics["trends"].(map[string]interface{})["battery_stats"] = map[string]interface{}{
			"min_charge": minCharge,
			"max_charge": maxCharge,
			"avg_charge": avgCharge,
		}
	}

	utils.SendSuccess(c, metrics)
}

// GetUPSConfiguration handles GET /api/v1/ups/config
func (h *Handlers) GetUPSConfiguration(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get connection info which includes configuration
	info, err := h.upsService.GetConnectionInfo(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to get UPS configuration: "+err.Error())
		return
	}

	// Add additional configuration details
	config := map[string]interface{}{
		"connection": info,
		"features": map[string]bool{
			"real_time_monitoring": true,
			"historical_data":      true,
			"battery_trends":       true,
			"alert_system":         true,
			"automatic_cleanup":    true,
			"websocket_events":     true,
		},
		"limits": map[string]interface{}{
			"max_history_limit":     1000,
			"max_trend_hours":       168, // 1 week
			"default_history_limit": 100,
			"default_trend_hours":   24,
		},
	}

	utils.SendSuccess(c, config)
}

// UpdateUPSAlertThresholds handles PUT /api/v1/ups/alerts/thresholds
func (h *Handlers) UpdateUPSAlertThresholds(c *gin.Context) {
	var request struct {
		LowBattery      *float64 `json:"low_battery,omitempty"`
		CriticalBattery *float64 `json:"critical_battery,omitempty"`
		HighTemperature *float64 `json:"high_temperature,omitempty"`
		HighLoad        *float64 `json:"high_load,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate thresholds
	if request.LowBattery != nil && (*request.LowBattery < 0 || *request.LowBattery > 100) {
		utils.SendError(c, http.StatusBadRequest, "Low battery threshold must be between 0 and 100")
		return
	}
	if request.CriticalBattery != nil && (*request.CriticalBattery < 0 || *request.CriticalBattery > 100) {
		utils.SendError(c, http.StatusBadRequest, "Critical battery threshold must be between 0 and 100")
		return
	}
	if request.HighTemperature != nil && (*request.HighTemperature < -20 || *request.HighTemperature > 100) {
		utils.SendError(c, http.StatusBadRequest, "High temperature threshold must be between -20 and 100")
		return
	}
	if request.HighLoad != nil && (*request.HighLoad < 0 || *request.HighLoad > 100) {
		utils.SendError(c, http.StatusBadRequest, "High load threshold must be between 0 and 100")
		return
	}

	// TODO: Implement threshold updates in UPS service
	// For now, return success with current config
	utils.SendSuccess(c, map[string]interface{}{
		"message":    "Alert thresholds updated successfully",
		"thresholds": request,
		"note":       "Threshold persistence will be implemented in a future update",
	})
}
