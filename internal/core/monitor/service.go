package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/analytics"
	"github.com/frostdev-ops/pma-backend-go/internal/core/metrics"
)

// MonitoringService coordinates all monitoring components
type MonitoringService struct {
	config *config.MonitoringConfig
	logger *logrus.Logger

	// Core components
	metricsCollector   metrics.MetricsCollector
	healthChecker      metrics.HealthChecker
	resourceMonitor    *ResourceMonitor
	alertManager       *AlertManager
	performanceTracker *analytics.PerformanceTracker

	// Background workers
	stopChan chan struct{}
	wg       sync.WaitGroup
	running  bool
	mu       sync.RWMutex

	// Health check functions
	databaseHealthCheck       func() metrics.HealthStatus
	homeAssistantHealthCheck  func() metrics.HealthStatus
	deviceAdapterHealthCheck  func() map[string]metrics.HealthStatus
	llmProviderHealthCheck    func() map[string]metrics.HealthStatus
	systemResourceHealthCheck func() metrics.HealthStatus
}

// MonitoringServiceConfig contains configuration for the monitoring service
type MonitoringServiceConfig struct {
	Monitoring         *config.MonitoringConfig
	MetricsRetention   time.Duration
	SnapshotInterval   time.Duration
	AlertCheckInterval time.Duration
	CleanupInterval    time.Duration
}

// NewMonitoringService creates a new monitoring service
func NewMonitoringService(cfg *MonitoringServiceConfig, logger *logrus.Logger) *MonitoringService {
	if cfg == nil {
		cfg = &MonitoringServiceConfig{
			MetricsRetention:   24 * time.Hour,
			SnapshotInterval:   30 * time.Second,
			AlertCheckInterval: 1 * time.Minute,
			CleanupInterval:    1 * time.Hour,
		}
	}

	if cfg.Monitoring == nil {
		cfg.Monitoring = &config.MonitoringConfig{
			Enabled:          true,
			MetricsRetention: "24h",
			SnapshotInterval: "30s",
		}
	}

	// Create metrics collector
	metricsConfig := &metrics.MetricsConfig{
		Enabled: cfg.Monitoring.Enabled,
		Prefix:  "pma",
	}
	metricsCollector := metrics.NewPrometheusCollector(metricsConfig)

	// Create health checker
	healthChecker := metrics.NewDefaultHealthChecker()

	// Create resource monitor
	resourceMonitor := NewResourceMonitor(logger)

	// Create alert manager
	alertConfig := &AlertManagerConfig{
		Enabled:             cfg.Monitoring.Alerts.Enabled,
		MaxAlerts:           1000,
		RetentionPeriod:     cfg.MetricsRetention,
		NotificationEnabled: true,
	}
	alertManager := NewAlertManager(alertConfig, logger)

	// Create performance tracker
	performanceTracker := analytics.NewPerformanceTracker(10000, cfg.MetricsRetention)
	performanceTracker.SetOptions(
		cfg.Monitoring.Performance.TrackUserAgents,
		cfg.Monitoring.Performance.CalculateP99,
		cfg.Monitoring.Performance.CalculateP95,
		cfg.Monitoring.Performance.CalculateP50,
	)

	service := &MonitoringService{
		config:             cfg.Monitoring,
		logger:             logger,
		metricsCollector:   metricsCollector,
		healthChecker:      healthChecker,
		resourceMonitor:    resourceMonitor,
		alertManager:       alertManager,
		performanceTracker: performanceTracker,
		stopChan:           make(chan struct{}),
	}

	// Set up system resource health check
	service.setupSystemResourceHealthCheck()

	// Set up alert callbacks
	service.setupAlertCallbacks()

	return service
}

// Start starts the monitoring service
func (s *MonitoringService) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("monitoring service is already running")
	}

	if !s.config.Enabled {
		s.logger.Info("Monitoring service is disabled")
		return nil
	}

	s.logger.Info("Starting monitoring service")

	// Parse durations from config
	snapshotInterval, err := time.ParseDuration(s.config.SnapshotInterval)
	if err != nil {
		snapshotInterval = 30 * time.Second
	}

	metricsRetention, err := time.ParseDuration(s.config.MetricsRetention)
	if err != nil {
		metricsRetention = 24 * time.Hour
	}

	// Start background workers
	s.wg.Add(4)

	// System metrics collection
	go s.systemMetricsWorker(ctx, snapshotInterval)

	// Alert threshold checking
	go s.alertThresholdWorker(ctx, time.Minute)

	// Data cleanup
	go s.cleanupWorker(ctx, time.Hour, metricsRetention)

	// Health monitoring
	go s.healthMonitorWorker(ctx, 5*time.Minute)

	s.running = true
	s.logger.Info("Monitoring service started successfully")

	return nil
}

// Stop stops the monitoring service
func (s *MonitoringService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("Stopping monitoring service")

	close(s.stopChan)
	s.wg.Wait()

	s.running = false
	s.logger.Info("Monitoring service stopped")

	return nil
}

// GetMetricsCollector returns the metrics collector
func (s *MonitoringService) GetMetricsCollector() metrics.MetricsCollector {
	return s.metricsCollector
}

// GetHealthChecker returns the health checker
func (s *MonitoringService) GetHealthChecker() metrics.HealthChecker {
	return s.healthChecker
}

// GetResourceMonitor returns the resource monitor
func (s *MonitoringService) GetResourceMonitor() *ResourceMonitor {
	return s.resourceMonitor
}

// GetAlertManager returns the alert manager
func (s *MonitoringService) GetAlertManager() *AlertManager {
	return s.alertManager
}

// GetPerformanceTracker returns the performance tracker
func (s *MonitoringService) GetPerformanceTracker() *analytics.PerformanceTracker {
	return s.performanceTracker
}

// SetDatabaseHealthCheck sets the database health check function
func (s *MonitoringService) SetDatabaseHealthCheck(check func() metrics.HealthStatus) {
	s.databaseHealthCheck = check
	s.healthChecker.(*metrics.DefaultHealthChecker).SetDatabaseChecker(check)
}

// SetHomeAssistantHealthCheck sets the Home Assistant health check function
func (s *MonitoringService) SetHomeAssistantHealthCheck(check func() metrics.HealthStatus) {
	s.homeAssistantHealthCheck = check
	s.healthChecker.(*metrics.DefaultHealthChecker).SetHomeAssistantChecker(check)
}

// SetDeviceAdapterHealthCheck sets the device adapter health check function
func (s *MonitoringService) SetDeviceAdapterHealthCheck(check func() map[string]metrics.HealthStatus) {
	s.deviceAdapterHealthCheck = check
	s.healthChecker.(*metrics.DefaultHealthChecker).SetDeviceAdapterChecker(check)
}

// SetLLMProviderHealthCheck sets the LLM provider health check function
func (s *MonitoringService) SetLLMProviderHealthCheck(check func() map[string]metrics.HealthStatus) {
	s.llmProviderHealthCheck = check
	s.healthChecker.(*metrics.DefaultHealthChecker).SetLLMProviderChecker(check)
}

// systemMetricsWorker collects system metrics periodically
func (s *MonitoringService) systemMetricsWorker(ctx context.Context, interval time.Duration) {
	defer s.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.Debug("System metrics worker started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.collectSystemMetrics(ctx)
		}
	}
}

// alertThresholdWorker checks alert thresholds periodically
func (s *MonitoringService) alertThresholdWorker(ctx context.Context, interval time.Duration) {
	defer s.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.Debug("Alert threshold worker started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkAlertThresholds(ctx)
		}
	}
}

// cleanupWorker cleans up old data periodically
func (s *MonitoringService) cleanupWorker(ctx context.Context, interval, retention time.Duration) {
	defer s.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.Debug("Cleanup worker started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.performanceTracker.ClearOldData(retention)
		}
	}
}

// healthMonitorWorker monitors health status periodically
func (s *MonitoringService) healthMonitorWorker(ctx context.Context, interval time.Duration) {
	defer s.wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.Debug("Health monitor worker started")

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.checkOverallHealth(ctx)
		}
	}
}

// collectSystemMetrics collects and records system metrics
func (s *MonitoringService) collectSystemMetrics(ctx context.Context) {
	stats, err := s.resourceMonitor.GetResourceStats(ctx)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to collect system metrics")
		return
	}

	// Record system resource metrics
	s.metricsCollector.RecordSystemResource(
		stats.CPU.TotalPercent,
		stats.Memory.UsedPercent,
		stats.Disk.UsedPercent,
	)

	s.logger.WithFields(logrus.Fields{
		"cpu":    stats.CPU.TotalPercent,
		"memory": stats.Memory.UsedPercent,
		"disk":   stats.Disk.UsedPercent,
	}).Debug("System metrics collected")
}

// checkAlertThresholds checks if any alert thresholds are exceeded
func (s *MonitoringService) checkAlertThresholds(ctx context.Context) {
	if !s.config.Alerts.Enabled {
		return
	}

	// Get current system metrics
	cpu, memory, disk, err := s.resourceMonitor.GetUsagePercentages(ctx)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get system usage for alert checking")
		return
	}

	// Check thresholds
	metrics := map[string]float64{
		"cpu_usage":    cpu,
		"memory_usage": memory,
		"disk_usage":   disk,
	}

	s.alertManager.CheckThresholds(metrics)
}

// checkOverallHealth performs overall health check and creates alerts if needed
func (s *MonitoringService) checkOverallHealth(ctx context.Context) {
	health := s.healthChecker.GetOverallHealth()

	if health.Status == "unhealthy" {
		alert := NewAlert(AlertSeverityCritical, "health_monitor", "System health check failed")
		alert = alert.WithDetails(map[string]interface{}{
			"components": health.Components,
			"message":    health.Message,
		})

		s.alertManager.CreateAlert(alert)
	}
}

// setupSystemResourceHealthCheck sets up the system resource health check
func (s *MonitoringService) setupSystemResourceHealthCheck() {
	healthCheck := func() metrics.HealthStatus {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cpu, memory, disk, err := s.resourceMonitor.GetUsagePercentages(ctx)
		if err != nil {
			return metrics.NewHealthStatus("unhealthy", "Failed to get system resource usage").
				WithDetail("error", err.Error())
		}

		status := "healthy"
		message := "System resources are within normal limits"

		if cpu > s.config.Alerts.Thresholds.CPUPercent ||
			memory > s.config.Alerts.Thresholds.MemoryPercent ||
			disk > s.config.Alerts.Thresholds.DiskPercent {
			status = "degraded"
			message = "System resources are under pressure"
		}

		if cpu > 95 || memory > 98 || disk > 98 {
			status = "unhealthy"
			message = "System resources are critically low"
		}

		return metrics.NewHealthStatus(status, message).
			WithDetails(map[string]interface{}{
				"cpu_usage":    cpu,
				"memory_usage": memory,
				"disk_usage":   disk,
			})
	}

	s.systemResourceHealthCheck = healthCheck
	s.healthChecker.(*metrics.DefaultHealthChecker).SetSystemResourceChecker(healthCheck)
}

// setupAlertCallbacks sets up callbacks for alert events
func (s *MonitoringService) setupAlertCallbacks() {
	// Log alert creation
	s.alertManager.OnAlertCreated(func(alert *Alert) {
		s.logger.WithFields(logrus.Fields{
			"alert_id": alert.ID,
			"severity": alert.Severity,
			"source":   alert.Source,
		}).Warn("Alert created")

		// Record alert metric
		s.metricsCollector.RecordAlert(string(alert.Severity), alert.Source, alert.Message)
	})

	// Log alert resolution
	s.alertManager.OnAlertResolved(func(alert *Alert) {
		s.logger.WithFields(logrus.Fields{
			"alert_id":    alert.ID,
			"duration":    alert.Duration,
			"resolved_by": alert.ResolvedBy,
		}).Info("Alert resolved")
	})
}

// GetStatus returns the current status of the monitoring service
func (s *MonitoringService) GetStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"running": s.running,
		"enabled": s.config.Enabled,
		"components": map[string]interface{}{
			"metrics_collector":   s.metricsCollector != nil,
			"health_checker":      s.healthChecker != nil,
			"resource_monitor":    s.resourceMonitor != nil,
			"alert_manager":       s.alertManager != nil,
			"performance_tracker": s.performanceTracker != nil,
		},
	}
}
