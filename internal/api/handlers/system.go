package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/system"
	"github.com/gin-gonic/gin"
)

// SystemHandler handles system management API endpoints
type SystemHandler struct {
	systemService *system.Service
}

// NewSystemHandler creates a new system handler
func NewSystemHandler(systemService *system.Service) *SystemHandler {
	return &SystemHandler{
		systemService: systemService,
	}
}

// GetSystemInfo returns basic system information
// GET /api/system/info
func (h *SystemHandler) GetSystemInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	deviceInfo, err := h.systemService.GetDeviceInfo(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get device info",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      deviceInfo,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetSystemStatus returns legacy system status format for frontend compatibility
// GET /api/system/status
func (h *SystemHandler) GetSystemStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	health, err := h.systemService.GetSystemHealth(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get system status",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Convert to legacy format expected by frontend
	status := map[string]interface{}{
		"server": map[string]interface{}{
			"name":      "PMA Backend (Go)",
			"status":    "connected",
			"lastCheck": time.Now().Format(time.RFC3339),
		},
		"homeAssistant": map[string]interface{}{
			"name":      "Home Assistant",
			"status":    getServiceStatusString(health.Services.HomeAssistant.Status),
			"lastCheck": health.Services.HomeAssistant.LastCheck.Format(time.RFC3339),
		},
		"database": map[string]interface{}{
			"name":      "SQLite Database",
			"status":    getServiceStatusString(health.Services.Database.Status),
			"lastCheck": health.Services.Database.LastCheck.Format(time.RFC3339),
		},
		"memory": map[string]interface{}{
			"used":       health.Memory.Used,
			"total":      health.Memory.Total,
			"percentage": health.Memory.UsedPercent,
		},
		"uptime": 3600, // Simplified - will be replaced with actual uptime from health data
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      status,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetBasicSystemHealth returns basic health check
// GET /api/system/health
func (h *SystemHandler) GetBasicSystemHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    3600, // Will be replaced with actual uptime
		"version":   "1.0.0",
	})
}

// GetSystemHealth returns detailed system health information
// GET /api/system/health/detailed
func (h *SystemHandler) GetSystemHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	health, err := h.systemService.GetSystemHealth(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get system health",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      health,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetDeviceInfo returns detailed device information
// GET /api/system/device-info
func (h *SystemHandler) GetDeviceInfo(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	deviceInfo, err := h.systemService.GetDeviceInfo(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get device info",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      deviceInfo,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetSystemLogs returns system logs
// GET /api/system/logs
func (h *SystemHandler) GetSystemLogs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	var req system.LogsRequest

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}

	if level := c.Query("level"); level != "" {
		req.Level = level
	}

	if service := c.Query("service"); service != "" {
		req.Service = service
	}

	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = startTime
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = endTime
		}
	}

	logs, err := h.systemService.GetSystemLogs(ctx, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to get system logs",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      logs,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// RebootSystem initiates a system reboot
// POST /api/system/reboot
func (h *SystemHandler) RebootSystem(c *gin.Context) {
	var action system.PowerAction
	if err := c.ShouldBindJSON(&action); err != nil {
		// Use defaults if no body provided
		action = system.PowerAction{
			Action:    "reboot",
			Reason:    "Manual reboot requested",
			RequestBy: c.GetString("user_id"), // From auth middleware
		}
	}
	action.Action = "reboot" // Ensure action is set correctly

	// Validate request
	if action.Delay < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid delay value",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Execute reboot in background
	go func() {
		ctx := context.Background()
		if err := h.systemService.RebootSystem(ctx, action); err != nil {
			// Log error - system may be rebooting
			h.systemService.AddLog("error", "system", "Failed to execute reboot", map[string]interface{}{
				"error": err.Error(),
			}, err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "System reboot initiated",
		"action":    action,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ShutdownSystem initiates a system shutdown
// POST /api/system/shutdown
func (h *SystemHandler) ShutdownSystem(c *gin.Context) {
	var action system.PowerAction
	if err := c.ShouldBindJSON(&action); err != nil {
		// Use defaults if no body provided
		action = system.PowerAction{
			Action:    "shutdown",
			Reason:    "Manual shutdown requested",
			RequestBy: c.GetString("user_id"), // From auth middleware
		}
	}
	action.Action = "shutdown" // Ensure action is set correctly

	// Validate request
	if action.Delay < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid delay value",
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Execute shutdown in background
	go func() {
		ctx := context.Background()
		if err := h.systemService.ShutdownSystem(ctx, action); err != nil {
			// Log error - system may be shutting down
			h.systemService.AddLog("error", "system", "Failed to execute shutdown", map[string]interface{}{
				"error": err.Error(),
			}, err)
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "System shutdown initiated",
		"action":    action,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetSystemConfig returns system configuration
// GET /api/system/config
func (h *SystemHandler) GetSystemConfig(c *gin.Context) {
	config := h.systemService.GetConfig()

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"data":      config,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// UpdateSystemConfig updates system configuration
// POST /api/system/config
func (h *SystemHandler) UpdateSystemConfig(c *gin.Context) {
	var config system.SystemConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid configuration data",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	if err := h.systemService.UpdateConfig(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"error":     "Failed to update system configuration",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "System configuration updated successfully",
		"data":      config,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ReportHealth accepts health reports from external sources
// POST /api/system/health-report
func (h *SystemHandler) ReportHealth(c *gin.Context) {
	var report map[string]interface{}
	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"error":     "Invalid health report data",
			"details":   err.Error(),
			"timestamp": time.Now().Format(time.RFC3339),
		})
		return
	}

	// Log the health report
	h.systemService.AddLog("info", "health_report", "External health report received", report, nil)

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Health report received",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Helper functions

// getServiceStatusString converts service health status to legacy string format
func getServiceStatusString(status string) string {
	switch status {
	case "healthy":
		return "connected"
	case "unhealthy":
		return "disconnected"
	default:
		return "unknown"
	}
}
