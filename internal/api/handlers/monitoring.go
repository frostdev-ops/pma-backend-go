package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/frostdev-ops/pma-backend-go/internal/core/analytics"
	"github.com/frostdev-ops/pma-backend-go/internal/core/metrics"
	"github.com/frostdev-ops/pma-backend-go/internal/core/monitor"
)

// MonitoringHandler handles monitoring-related requests
type MonitoringHandler struct {
	healthChecker      metrics.HealthChecker
	metricsCollector   metrics.MetricsCollector
	resourceMonitor    *monitor.ResourceMonitor
	alertManager       *monitor.AlertManager
	performanceTracker *analytics.PerformanceTracker
}

// NewMonitoringHandler creates a new monitoring handler
func NewMonitoringHandler(
	healthChecker metrics.HealthChecker,
	metricsCollector metrics.MetricsCollector,
	resourceMonitor *monitor.ResourceMonitor,
	alertManager *monitor.AlertManager,
	performanceTracker *analytics.PerformanceTracker,
) *MonitoringHandler {
	return &MonitoringHandler{
		healthChecker:      healthChecker,
		metricsCollector:   metricsCollector,
		resourceMonitor:    resourceMonitor,
		alertManager:       alertManager,
		performanceTracker: performanceTracker,
	}
}

// RegisterRoutes registers monitoring routes
func (h *MonitoringHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check endpoints
	router.GET("/health", h.GetHealth)
	router.GET("/health/live", h.GetLiveness)
	router.GET("/health/ready", h.GetReadiness)

	// System monitoring endpoints
	router.GET("/monitor/system", h.GetSystemStats)
	router.GET("/monitor/services", h.GetServiceStatus)

	// Alert endpoints
	router.GET("/alerts", h.GetAlerts)
	router.POST("/alerts/:id/resolve", h.ResolveAlert)
	router.GET("/alerts/stats", h.GetAlertStats)

	// Performance analytics endpoints
	router.GET("/analytics/performance", h.GetPerformanceMetrics)
	router.GET("/analytics/usage", h.GetUsageStatistics)
	router.GET("/analytics/endpoints", h.GetEndpointMetrics)
	router.GET("/analytics/slow-requests", h.GetSlowRequests)
}

// GetHealth returns comprehensive health status
func (h *MonitoringHandler) GetHealth(c *gin.Context) {
	health := h.healthChecker.GetOverallHealth()

	status := http.StatusOK
	if health.Status == "unhealthy" {
		status = http.StatusServiceUnavailable
	} else if health.Status == "degraded" {
		status = http.StatusPartialContent
	}

	c.JSON(status, gin.H{
		"status":      health.Status,
		"message":     health.Message,
		"timestamp":   health.Timestamp,
		"duration":    health.Duration.String(),
		"components":  health.Components,
		"system_info": health.SystemInfo,
	})
}

// GetLiveness returns liveness probe status (for Kubernetes)
func (h *MonitoringHandler) GetLiveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now(),
	})
}

// GetReadiness returns readiness probe status (for Kubernetes)
func (h *MonitoringHandler) GetReadiness(c *gin.Context) {
	// Check if critical services are ready
	dbHealth := h.healthChecker.CheckDatabase()

	if !dbHealth.IsHealthy() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "not_ready",
			"reason":    "database_not_ready",
			"details":   dbHealth,
			"timestamp": time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now(),
	})
}

// GetSystemStats returns current system resource statistics
func (h *MonitoringHandler) GetSystemStats(c *gin.Context) {
	stats, err := h.resourceMonitor.GetResourceStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get system stats",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetServiceStatus returns the status of all monitored services
func (h *MonitoringHandler) GetServiceStatus(c *gin.Context) {
	health := h.healthChecker.GetOverallHealth()

	services := make(map[string]interface{})
	for name, component := range health.Components {
		services[name] = gin.H{
			"status":     component.Status,
			"message":    component.Message,
			"last_check": component.Timestamp,
			"duration":   component.Duration.String(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"overall_status": health.Status,
		"services":       services,
		"timestamp":      time.Now(),
	})
}

// GetAlerts returns alerts with optional filtering
func (h *MonitoringHandler) GetAlerts(c *gin.Context) {
	// Parse query parameters
	activeOnly := c.Query("active") == "true"
	severity := c.Query("severity")
	source := c.Query("source")
	limitStr := c.DefaultQuery("limit", "100")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	var alerts []monitor.Alert

	if activeOnly {
		alerts = h.alertManager.GetActiveAlerts()
	} else if severity != "" {
		alerts = h.alertManager.GetAlertsBySeverity(monitor.AlertSeverity(severity))
	} else if source != "" {
		alerts = h.alertManager.GetAlertsBySource(source)
	} else {
		alerts = h.alertManager.GetAllAlerts()
	}

	// Apply limit
	if limit > 0 && len(alerts) > limit {
		alerts = alerts[:limit]
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts":    alerts,
		"count":     len(alerts),
		"timestamp": time.Now(),
	})
}

// ResolveAlert resolves a specific alert
func (h *MonitoringHandler) ResolveAlert(c *gin.Context) {
	alertID := c.Param("id")
	if alertID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Alert ID is required",
		})
		return
	}

	resolvedBy := c.DefaultQuery("resolved_by", "user")

	err := h.alertManager.ResolveAlertBy(alertID, resolvedBy)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Failed to resolve alert",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Alert resolved successfully",
		"alert_id":    alertID,
		"resolved_by": resolvedBy,
		"timestamp":   time.Now(),
	})
}

// GetAlertStats returns alert statistics
func (h *MonitoringHandler) GetAlertStats(c *gin.Context) {
	stats := h.alertManager.GetAlertStats()

	c.JSON(http.StatusOK, gin.H{
		"stats":     stats,
		"timestamp": time.Now(),
	})
}

// GetPerformanceMetrics returns performance metrics for a specified period
func (h *MonitoringHandler) GetPerformanceMetrics(c *gin.Context) {
	periodStr := c.DefaultQuery("period", "1h")
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid period format",
			"details": err.Error(),
		})
		return
	}

	metrics := h.performanceTracker.GetPerformanceMetrics(period)

	// Add current resource metrics
	if h.resourceMonitor != nil {
		cpu, memory, disk, err := h.resourceMonitor.GetUsagePercentages(c.Request.Context())
		if err == nil {
			metrics.ResourceMetrics = &analytics.ResourcePerformanceMetrics{
				CPUUsage:    cpu,
				MemoryUsage: memory,
				DiskUsage:   disk,
			}
		}
	}

	c.JSON(http.StatusOK, metrics)
}

// GetUsageStatistics returns usage statistics
func (h *MonitoringHandler) GetUsageStatistics(c *gin.Context) {
	periodStr := c.DefaultQuery("period", "24h")
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid period format",
			"details": err.Error(),
		})
		return
	}

	// Get performance metrics for the period
	metrics := h.performanceTracker.GetPerformanceMetrics(period)

	// Get error rates by endpoint
	errorRates := h.performanceTracker.GetErrorRateByEndpoint(period)

	// Get throughput time series
	bucketSize := period / 24 // 24 data points
	if bucketSize < time.Minute {
		bucketSize = time.Minute
	}
	throughputData := h.performanceTracker.GetThroughputTimeSeries(period, bucketSize)

	c.JSON(http.StatusOK, gin.H{
		"period":                 period.String(),
		"total_requests":         metrics.RequestCount,
		"error_rate":             metrics.ErrorRate,
		"average_latency":        metrics.AverageLatency.String(),
		"throughput":             metrics.Throughput,
		"endpoint_error_rates":   errorRates,
		"throughput_time_series": throughputData,
		"timestamp":              time.Now(),
	})
}

// GetEndpointMetrics returns detailed metrics for each endpoint
func (h *MonitoringHandler) GetEndpointMetrics(c *gin.Context) {
	periodStr := c.DefaultQuery("period", "1h")
	period, err := time.ParseDuration(periodStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid period format",
			"details": err.Error(),
		})
		return
	}

	metrics := h.performanceTracker.GetPerformanceMetrics(period)

	c.JSON(http.StatusOK, gin.H{
		"period":    period.String(),
		"endpoints": metrics.EndpointMetrics,
		"timestamp": time.Now(),
	})
}

// GetSlowRequests returns requests that exceed a specified latency threshold
func (h *MonitoringHandler) GetSlowRequests(c *gin.Context) {
	thresholdStr := c.DefaultQuery("threshold", "5s")
	threshold, err := time.ParseDuration(thresholdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid threshold format",
			"details": err.Error(),
		})
		return
	}

	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50
	}

	slowRequests := h.performanceTracker.GetSlowRequests(threshold, limit)

	c.JSON(http.StatusOK, gin.H{
		"threshold":     threshold.String(),
		"slow_requests": slowRequests,
		"count":         len(slowRequests),
		"timestamp":     time.Now(),
	})
}

// HealthCheckResponse represents a simplified health check response
type HealthCheckResponse struct {
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// GetCustomHealth returns custom health check with specific components
func (h *MonitoringHandler) GetCustomHealth(c *gin.Context) {
	components := c.QueryArray("component")
	if len(components) == 0 {
		// Return full health check
		h.GetHealth(c)
		return
	}

	result := make(map[string]interface{})
	overallHealthy := true

	for _, component := range components {
		switch component {
		case "database":
			health := h.healthChecker.CheckDatabase()
			result["database"] = health
			if !health.IsHealthy() {
				overallHealthy = false
			}

		case "home_assistant":
			health := h.healthChecker.CheckHomeAssistant()
			result["home_assistant"] = health
			if !health.IsHealthy() {
				overallHealthy = false
			}

		case "system_resources":
			health := h.healthChecker.CheckSystemResources()
			result["system_resources"] = health
			if !health.IsHealthy() {
				overallHealthy = false
			}

		case "devices":
			devices := h.healthChecker.CheckDeviceAdapters()
			result["devices"] = devices
			for _, device := range devices {
				if !device.IsHealthy() {
					overallHealthy = false
					break
				}
			}

		case "llm":
			llmProviders := h.healthChecker.CheckLLMProviders()
			result["llm_providers"] = llmProviders
			for _, provider := range llmProviders {
				if !provider.IsHealthy() {
					overallHealthy = false
					break
				}
			}
		}
	}

	status := "healthy"
	if !overallHealthy {
		status = "unhealthy"
	}

	httpStatus := http.StatusOK
	if status == "unhealthy" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":     status,
		"components": result,
		"timestamp":  time.Now(),
	})
}

// GetMetricsSnapshot returns a snapshot of key metrics
func (h *MonitoringHandler) GetMetricsSnapshot(c *gin.Context) {
	// Get system stats
	systemStats, _ := h.resourceMonitor.GetResourceStats(c.Request.Context())

	// Get performance metrics for the last hour
	perfMetrics := h.performanceTracker.GetPerformanceMetrics(time.Hour)

	// Get alert stats
	alertStats := h.alertManager.GetAlertStats()

	snapshot := gin.H{
		"timestamp": time.Now(),
		"system": gin.H{
			"cpu_usage":    systemStats.CPU.TotalPercent,
			"memory_usage": systemStats.Memory.UsedPercent,
			"disk_usage":   systemStats.Disk.UsedPercent,
			"goroutines":   systemStats.Runtime.Goroutines,
		},
		"performance": gin.H{
			"request_count":   perfMetrics.RequestCount,
			"average_latency": perfMetrics.AverageLatency.String(),
			"error_rate":      perfMetrics.ErrorRate,
			"throughput":      perfMetrics.Throughput,
		},
		"alerts": alertStats,
	}

	c.JSON(http.StatusOK, snapshot)
}
