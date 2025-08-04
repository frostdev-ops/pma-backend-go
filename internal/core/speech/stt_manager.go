package speech

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/sirupsen/logrus"
)

// STTServiceManager manages STT service lifecycle and health
type STTServiceManager struct {
	config        *config.STTConfig
	logger        *logrus.Logger
	service       *Service
	healthChecker *HealthChecker
	
	// Service state
	isRunning        bool
	lastHealthCheck  time.Time
	healthStatus     *STTHealthStatus
	errorCount       int
	maxErrors        int
	restartCount     int
	maxRestarts      int
	
	// Concurrency control
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	
	// Metrics
	totalRequests    int64
	successfulRequests int64
	failedRequests   int64
	averageLatency   time.Duration
	
	// Event handlers
	onHealthChange   func(status *STTHealthStatus)
	onError          func(error)
	onRestart        func()
}

// STTManagerConfig contains configuration for the STT manager
type STTManagerConfig struct {
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	MaxErrors          int           `json:"max_errors"`
	MaxRestarts        int           `json:"max_restarts"`
	RestartDelay       time.Duration `json:"restart_delay"`
	EnableAutoRestart  bool          `json:"enable_auto_restart"`
	EnableMetrics      bool          `json:"enable_metrics"`
}

// NewSTTServiceManager creates a new STT service manager
func NewSTTServiceManager(config *config.STTConfig, service *Service, logger *logrus.Logger) *STTServiceManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	manager := &STTServiceManager{
		config:        config,
		logger:        logger,
		service:       service,
		healthChecker: NewHealthChecker(service),
		maxErrors:     10,
		maxRestarts:   5,
		ctx:           ctx,
		cancel:        cancel,
	}

	return manager
}

// Start starts the STT service manager
func (m *STTServiceManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("STT service manager is already running")
	}

	m.logger.Info("Starting STT service manager")

	// Initial health check
	m.performHealthCheck()

	// Start monitoring goroutine
	go m.monitorService()

	m.isRunning = true
	return nil
}

// Stop stops the STT service manager
func (m *STTServiceManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return fmt.Errorf("STT service manager is not running")
	}

	m.logger.Info("Stopping STT service manager")

	m.cancel()
	m.isRunning = false
	return nil
}

// GetStatus returns the current status of the STT service
func (m *STTServiceManager) GetStatus() *STTManagerStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uptime := time.Duration(0)
	if m.isRunning {
		uptime = time.Since(m.lastHealthCheck)
	}

	return &STTManagerStatus{
		IsRunning:          m.isRunning,
		HealthStatus:       m.healthStatus,
		ErrorCount:         m.errorCount,
		RestartCount:       m.restartCount,
		Uptime:             uptime,
		TotalRequests:      m.totalRequests,
		SuccessfulRequests: m.successfulRequests,
		FailedRequests:     m.failedRequests,
		AverageLatency:     m.averageLatency,
		LastHealthCheck:    m.lastHealthCheck,
	}
}

// GetMetrics returns performance metrics
func (m *STTServiceManager) GetMetrics() *STTMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	successRate := float64(0)
	if m.totalRequests > 0 {
		successRate = float64(m.successfulRequests) / float64(m.totalRequests) * 100
	}

	return &STTMetrics{
		TotalRequests:      m.totalRequests,
		SuccessfulRequests: m.successfulRequests,
		FailedRequests:     m.failedRequests,
		SuccessRate:        successRate,
		AverageLatency:     m.averageLatency,
		ErrorCount:         m.errorCount,
		RestartCount:       m.restartCount,
		UptimeSeconds:      time.Since(time.Now().Add(-time.Duration(m.restartCount)*time.Hour)).Seconds(),
	}
}

// RecordRequest records a request for metrics
func (m *STTServiceManager) RecordRequest(success bool, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++
	
	if success {
		m.successfulRequests++
		m.errorCount = 0 // Reset error count on success
	} else {
		m.failedRequests++
		m.errorCount++
	}

	// Update average latency (simple moving average)
	if m.totalRequests == 1 {
		m.averageLatency = latency
	} else {
		m.averageLatency = (m.averageLatency + latency) / 2
	}

	// Check if we need to restart due to too many errors
	if m.errorCount >= m.maxErrors {
		m.logger.WithField("error_count", m.errorCount).Warn("STT service has too many errors, triggering restart")
		go m.triggerRestart()
	}
}

// RecordError records an error
func (m *STTServiceManager) RecordError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.errorCount++
	m.failedRequests++

	m.logger.WithError(err).WithField("error_count", m.errorCount).Error("STT service error recorded")

	if m.onError != nil {
		go m.onError(err)
	}
}

// SetOnHealthChange sets a callback for health status changes
func (m *STTServiceManager) SetOnHealthChange(callback func(status *STTHealthStatus)) {
	m.onHealthChange = callback
}

// SetOnError sets a callback for errors
func (m *STTServiceManager) SetOnError(callback func(error)) {
	m.onError = callback
}

// SetOnRestart sets a callback for restarts
func (m *STTServiceManager) SetOnRestart(callback func()) {
	m.onRestart = callback
}

// monitorService monitors the STT service health
func (m *STTServiceManager) monitorService() {
	ticker := time.NewTicker(30 * time.Second) // Health check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performHealthCheck()
		}
	}
}

// performHealthCheck performs a health check
func (m *STTServiceManager) performHealthCheck() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debug("Performing STT service health check")

	oldStatus := m.healthStatus
	newStatus := m.healthChecker.CheckHealth()
	
	m.healthStatus = newStatus
	m.lastHealthCheck = time.Now()

	// Check if health status changed
	if oldStatus == nil || oldStatus.Overall != newStatus.Overall {
		m.logger.WithFields(logrus.Fields{
			"old_status": oldStatus,
			"new_status": newStatus.Overall,
		}).Info("STT service health status changed")

		if m.onHealthChange != nil {
			go m.onHealthChange(newStatus)
		}
	}

	// Log unhealthy status
	if newStatus.Overall == "unhealthy" {
		m.logger.WithField("components", newStatus.Components).Error("STT service is unhealthy")
		
		// Consider restart if too many health check failures
		if m.errorCount >= m.maxErrors/2 { // Restart at half the error threshold for health issues
			go m.triggerRestart()
		}
	}
}

// triggerRestart attempts to restart the STT service
func (m *STTServiceManager) triggerRestart() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.restartCount >= m.maxRestarts {
		m.logger.WithField("restart_count", m.restartCount).Error("Maximum restart attempts reached, STT service disabled")
		return
	}

	m.restartCount++
	m.errorCount = 0 // Reset error count

	m.logger.WithField("restart_count", m.restartCount).Info("Restarting STT service")

	if m.onRestart != nil {
		go m.onRestart()
	}

	// Perform restart logic here
	// This could involve:
	// - Clearing caches
	// - Reinitializing Python environment
	// - Restarting dependent services
	// - Clearing temporary files

	// Simulate restart delay
	time.Sleep(5 * time.Second)

	// Perform immediate health check after restart
	m.performHealthCheck()
}

// IsHealthy returns true if the service is healthy
func (m *STTServiceManager) IsHealthy() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.healthStatus != nil && m.healthStatus.Overall == "healthy"
}

// GetLastError returns the last recorded error
func (m *STTServiceManager) GetLastError() *STTError {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.healthStatus != nil {
		return m.healthStatus.LastError
	}
	return nil
}

// STTManagerStatus represents the status of the STT service manager
type STTManagerStatus struct {
	IsRunning          bool              `json:"is_running"`
	HealthStatus       *STTHealthStatus  `json:"health_status"`
	ErrorCount         int               `json:"error_count"`
	RestartCount       int               `json:"restart_count"`
	Uptime             time.Duration     `json:"uptime"`
	TotalRequests      int64             `json:"total_requests"`
	SuccessfulRequests int64             `json:"successful_requests"`
	FailedRequests     int64             `json:"failed_requests"`
	AverageLatency     time.Duration     `json:"average_latency"`
	LastHealthCheck    time.Time         `json:"last_health_check"`
}

// STTMetrics represents performance metrics for the STT service
type STTMetrics struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	SuccessRate        float64       `json:"success_rate"`
	AverageLatency     time.Duration `json:"average_latency"`
	ErrorCount         int           `json:"error_count"`
	RestartCount       int           `json:"restart_count"`
	UptimeSeconds      float64       `json:"uptime_seconds"`
}

// Enhanced STT service with manager integration
func (s *Service) WithManager(logger *logrus.Logger) *Service {
	manager := NewSTTServiceManager(&s.config.STT, s, logger)
	
	// Start manager
	if err := manager.Start(); err != nil {
		logger.WithError(err).Error("Failed to start STT service manager")
	}
	
	// Set up event handlers
	manager.SetOnHealthChange(func(status *STTHealthStatus) {
		logger.WithField("status", status.Overall).Info("STT health status changed")
	})
	
	manager.SetOnError(func(err error) {
		logger.WithError(err).Error("STT service error reported")
	})
	
	manager.SetOnRestart(func() {
		logger.Info("STT service restarted")
	})
	
	return s
}