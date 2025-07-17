//go:build ignore
// +build ignore

// This file demonstrates how to integrate the monitoring system with your application.
// To run this example: go run MONITORING_INTEGRATION_EXAMPLE.go

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/frostdev-ops/pma-backend-go/internal/api/handlers"
	"github.com/frostdev-ops/pma-backend-go/internal/api/middleware"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/metrics"
	"github.com/frostdev-ops/pma-backend-go/internal/core/monitor"
)

// Example of how to integrate the monitoring system with your application
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Initialize monitoring service
	monitoringService := setupMonitoring(cfg, logger)

	// Start monitoring service
	ctx := context.Background()
	if err := monitoringService.Start(ctx); err != nil {
		log.Fatal("Failed to start monitoring service:", err)
	}
	defer monitoringService.Stop()

	// Setup HTTP server with monitoring
	router := setupRouter(monitoringService)

	// Run server
	logger.Info("Starting server on :3001")
	log.Fatal(router.Run(":3001"))
}

// setupMonitoring initializes and configures the monitoring service
func setupMonitoring(cfg *config.Config, logger *logrus.Logger) *monitor.MonitoringService {
	// Create monitoring service configuration
	monitoringConfig := &monitor.MonitoringServiceConfig{
		Monitoring:         &cfg.Monitoring,
		MetricsRetention:   24 * time.Hour,
		SnapshotInterval:   30 * time.Second,
		AlertCheckInterval: 1 * time.Minute,
		CleanupInterval:    1 * time.Hour,
	}

	// Create monitoring service
	monitoringService := monitor.NewMonitoringService(monitoringConfig, logger)

	// Set up health checks
	setupHealthChecks(monitoringService, cfg, logger)

	// Set up custom alerts
	setupCustomAlerts(monitoringService)

	return monitoringService
}

// setupHealthChecks configures health checks for all system components
func setupHealthChecks(service *monitor.MonitoringService, cfg *config.Config, logger *logrus.Logger) {
	// Database health check
	service.SetDatabaseHealthCheck(func() metrics.HealthStatus {
		// In a real application, you would check database connectivity
		// For this example, we'll simulate a health check
		start := time.Now()

		// Simulate database ping
		time.Sleep(10 * time.Millisecond)

		status := "healthy"
		message := "Database is responsive"

		// Simulate occasional issues
		if time.Now().Unix()%100 == 0 {
			status = "degraded"
			message = "Database is slow to respond"
		}

		return metrics.NewHealthStatus(status, message).
			WithDetail("response_time", time.Since(start).String()).
			WithDetail("connection_pool", "active")
	})

	// Home Assistant health check
	service.SetHomeAssistantHealthCheck(func() metrics.HealthStatus {
		if cfg.HomeAssistant.URL == "" || cfg.HomeAssistant.Token == "" {
			return metrics.NewHealthStatus("unknown", "Home Assistant not configured")
		}

		// In a real application, you would make an HTTP request to Home Assistant
		// For this example, we'll simulate the check
		status := "healthy"
		message := "Home Assistant is reachable"

		return metrics.NewHealthStatus(status, message).
			WithDetail("url", cfg.HomeAssistant.URL).
			WithDetail("last_sync", time.Now().Add(-1*time.Minute).String())
	})

	// Device adapter health checks
	service.SetDeviceAdapterHealthCheck(func() map[string]metrics.HealthStatus {
		deviceStatuses := make(map[string]metrics.HealthStatus)

		// Ring devices
		if cfg.Devices.Ring.Enabled {
			deviceStatuses["ring"] = metrics.NewHealthStatus("healthy", "Ring integration active").
				WithDetail("devices_count", 3).
				WithDetail("last_poll", time.Now().Add(-30*time.Second).String())
		}

		// Shelly devices
		if cfg.Devices.Shelly.Enabled {
			deviceStatuses["shelly"] = metrics.NewHealthStatus("healthy", "Shelly devices online").
				WithDetail("devices_count", 5).
				WithDetail("discovery_active", true)
		}

		// UPS monitoring
		if cfg.Devices.UPS.Enabled {
			deviceStatuses["ups"] = metrics.NewHealthStatus("healthy", "UPS monitoring active").
				WithDetail("battery_charge", 95).
				WithDetail("load_percent", 23)
		}

		return deviceStatuses
	})

	// LLM provider health checks
	service.SetLLMProviderHealthCheck(func() map[string]metrics.HealthStatus {
		llmStatuses := make(map[string]metrics.HealthStatus)

		for _, provider := range cfg.AI.Providers {
			if !provider.Enabled {
				continue
			}

			switch provider.Type {
			case "ollama":
				llmStatuses["ollama"] = metrics.NewHealthStatus("healthy", "Ollama service running").
					WithDetail("model", provider.DefaultModel).
					WithDetail("url", provider.URL)

			case "openai":
				if provider.APIKey != "" {
					llmStatuses["openai"] = metrics.NewHealthStatus("healthy", "OpenAI API accessible").
						WithDetail("model", provider.DefaultModel)
				}

			case "gemini":
				if provider.APIKey != "" {
					llmStatuses["gemini"] = metrics.NewHealthStatus("healthy", "Gemini API accessible").
						WithDetail("model", provider.DefaultModel)
				}
			}
		}

		return llmStatuses
	})
}

// setupCustomAlerts configures custom alert rules and thresholds
func setupCustomAlerts(service *monitor.MonitoringService) {
	alertManager := service.GetAlertManager()

	// Add custom alert rules
	customRules := []monitor.AlertRule{
		{
			Name:      "High HTTP Error Rate",
			Metric:    "error_rate",
			Operator:  ">",
			Threshold: 0.05, // 5%
			Duration:  5 * time.Minute,
			Severity:  monitor.AlertSeverityWarning,
			Message:   "HTTP error rate is above 5%",
			Enabled:   true,
		},
		{
			Name:      "Slow Response Time",
			Metric:    "response_time",
			Operator:  ">",
			Threshold: 5.0, // 5 seconds
			Duration:  2 * time.Minute,
			Severity:  monitor.AlertSeverityWarning,
			Message:   "HTTP responses are taking longer than 5 seconds",
			Enabled:   true,
		},
		{
			Name:      "High Memory Usage",
			Metric:    "memory_usage",
			Operator:  ">",
			Threshold: 90.0, // 90%
			Duration:  10 * time.Minute,
			Severity:  monitor.AlertSeverityCritical,
			Message:   "System memory usage is critically high",
			Enabled:   true,
		},
	}

	for _, rule := range customRules {
		alertManager.AddRule(rule)
	}

	// Set up alert callbacks
	alertManager.OnAlertCreated(func(alert *monitor.Alert) {
		fmt.Printf("🚨 ALERT CREATED: [%s] %s - %s\n",
			alert.Severity, alert.Source, alert.Message)

		// In a real application, you would send notifications here
		// Examples:
		// - Send email
		// - Post to Slack
		// - Send push notification
		// - Create ticket in JIRA

		if alert.Severity == monitor.AlertSeverityCritical {
			fmt.Println("   ⚠️  This is a CRITICAL alert - immediate attention required!")
		}
	})

	alertManager.OnAlertResolved(func(alert *monitor.Alert) {
		fmt.Printf("✅ ALERT RESOLVED: [%s] %s - Duration: %s\n",
			alert.Severity, alert.Source, alert.Duration.String())
	})
}

// setupRouter configures the HTTP router with monitoring middleware and endpoints
func setupRouter(monitoringService *monitor.MonitoringService) *gin.Engine {
	router := gin.New()

	// Add basic middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Add monitoring middleware
	router.Use(middleware.MetricsMiddleware(
		monitoringService.GetMetricsCollector(),
		monitoringService.GetPerformanceTracker(),
	))

	// Add threshold alerting middleware
	router.Use(middleware.ThresholdAlertMiddleware(
		monitoringService.GetMetricsCollector(),
		5*time.Second, // Alert on requests slower than 5 seconds
	))

	// Create monitoring handler
	monitoringHandler := handlers.NewMonitoringHandler(
		monitoringService.GetHealthChecker(),
		monitoringService.GetMetricsCollector(),
		monitoringService.GetResourceMonitor(),
		monitoringService.GetAlertManager(),
		monitoringService.GetPerformanceTracker(),
	)

	// Register monitoring routes
	api := router.Group("/api")
	monitoringHandler.RegisterRoutes(api)

	// Add some example application routes
	setupApplicationRoutes(router, monitoringService)

	return router
}

// setupApplicationRoutes adds example application routes that generate metrics
func setupApplicationRoutes(router *gin.Engine, monitoringService *monitor.MonitoringService) {
	collector := monitoringService.GetMetricsCollector()

	// Example routes that demonstrate metrics collection
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "PMA Backend with Monitoring",
			"status":  "running",
			"time":    time.Now(),
		})
	})

	router.GET("/api/status", func(c *gin.Context) {
		// Record custom business metric
		collector.IncrementCounter("status_checks_total", map[string]string{
			"endpoint": "/api/status",
		})

		c.JSON(200, gin.H{
			"status":     "ok",
			"monitoring": monitoringService.GetStatus(),
		})
	})

	// Simulate a slow endpoint
	router.GET("/api/slow", func(c *gin.Context) {
		// Simulate slow processing
		time.Sleep(2 * time.Second)

		c.JSON(200, gin.H{
			"message":  "This endpoint is intentionally slow",
			"duration": "2s",
		})
	})

	// Simulate an endpoint that sometimes fails
	router.GET("/api/unreliable", func(c *gin.Context) {
		// Randomly fail 10% of the time
		if time.Now().UnixNano()%10 == 0 {
			c.JSON(500, gin.H{
				"error": "Random failure for testing",
			})
			return
		}

		c.JSON(200, gin.H{
			"message": "Success",
		})
	})

	// Endpoint to trigger a custom alert
	router.POST("/api/trigger-alert", func(c *gin.Context) {
		alertManager := monitoringService.GetAlertManager()

		alert := monitor.NewAlert(
			monitor.AlertSeverityWarning,
			"manual_trigger",
			"Alert manually triggered via API",
		).WithDetails(map[string]interface{}{
			"triggered_by": "user",
			"endpoint":     "/api/trigger-alert",
			"timestamp":    time.Now(),
		})

		err := alertManager.CreateAlert(alert)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"message":  "Alert created successfully",
			"alert_id": alert.ID,
		})
	})

	// Device simulation endpoints
	router.POST("/api/devices/:type/operation", func(c *gin.Context) {
		deviceType := c.Param("type")
		operation := c.DefaultQuery("operation", "status_check")

		// Simulate device operation
		start := time.Now()
		time.Sleep(100 * time.Millisecond) // Simulate operation time
		duration := time.Since(start)

		// Record device operation metric
		success := true
		if time.Now().UnixNano()%20 == 0 { // 5% failure rate
			success = false
		}

		collector.RecordDeviceOperation(deviceType, operation, success, duration)

		if !success {
			c.JSON(500, gin.H{
				"error":       "Device operation failed",
				"device_type": deviceType,
				"operation":   operation,
			})
			return
		}

		c.JSON(200, gin.H{
			"message":     "Device operation completed",
			"device_type": deviceType,
			"operation":   operation,
			"duration":    duration.String(),
		})
	})

	// LLM request simulation
	router.POST("/api/llm/query", func(c *gin.Context) {
		start := time.Now()

		// Simulate LLM processing
		time.Sleep(time.Duration(500+time.Now().UnixNano()%1500) * time.Millisecond)
		duration := time.Since(start)

		// Simulate token usage
		tokens := 50 + int(time.Now().UnixNano()%200) // 50-250 tokens

		// Record LLM metrics
		collector.RecordLLMRequest("ollama", true, duration, tokens)

		c.JSON(200, gin.H{
			"response":    "This is a simulated LLM response",
			"tokens_used": tokens,
			"duration":    duration.String(),
		})
	})
}
