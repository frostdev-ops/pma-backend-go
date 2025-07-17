package metrics

import (
	"context"
	"fmt"
	"time"
)

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status    string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
}

// HealthReport represents the overall health report
type HealthReport struct {
	Status     string                  `json:"status"`
	Message    string                  `json:"message"`
	Timestamp  time.Time               `json:"timestamp"`
	Duration   time.Duration           `json:"duration"`
	Components map[string]HealthStatus `json:"components"`
	SystemInfo map[string]interface{}  `json:"system_info"`
}

// HealthChecker defines the interface for health checking
type HealthChecker interface {
	CheckDatabase() HealthStatus
	CheckHomeAssistant() HealthStatus
	CheckDeviceAdapters() map[string]HealthStatus
	CheckLLMProviders() map[string]HealthStatus
	CheckSystemResources() HealthStatus
	GetOverallHealth() HealthReport
	RegisterCustomCheck(name string, check func() HealthStatus)
}

// CustomHealthCheck represents a custom health check function
type CustomHealthCheck func() HealthStatus

// DefaultHealthChecker implements HealthChecker
type DefaultHealthChecker struct {
	databaseChecker       func() HealthStatus
	homeAssistantChecker  func() HealthStatus
	deviceAdapterChecker  func() map[string]HealthStatus
	llmProviderChecker    func() map[string]HealthStatus
	systemResourceChecker func() HealthStatus
	customChecks          map[string]CustomHealthCheck
}

// NewDefaultHealthChecker creates a new health checker
func NewDefaultHealthChecker() *DefaultHealthChecker {
	return &DefaultHealthChecker{
		customChecks: make(map[string]CustomHealthCheck),
	}
}

// SetDatabaseChecker sets the database health check function
func (h *DefaultHealthChecker) SetDatabaseChecker(checker func() HealthStatus) {
	h.databaseChecker = checker
}

// SetHomeAssistantChecker sets the Home Assistant health check function
func (h *DefaultHealthChecker) SetHomeAssistantChecker(checker func() HealthStatus) {
	h.homeAssistantChecker = checker
}

// SetDeviceAdapterChecker sets the device adapter health check function
func (h *DefaultHealthChecker) SetDeviceAdapterChecker(checker func() map[string]HealthStatus) {
	h.deviceAdapterChecker = checker
}

// SetLLMProviderChecker sets the LLM provider health check function
func (h *DefaultHealthChecker) SetLLMProviderChecker(checker func() map[string]HealthStatus) {
	h.llmProviderChecker = checker
}

// SetSystemResourceChecker sets the system resource health check function
func (h *DefaultHealthChecker) SetSystemResourceChecker(checker func() HealthStatus) {
	h.systemResourceChecker = checker
}

// CheckDatabase performs database health check
func (h *DefaultHealthChecker) CheckDatabase() HealthStatus {
	start := time.Now()

	if h.databaseChecker == nil {
		return HealthStatus{
			Status:    "unknown",
			Message:   "Database health checker not configured",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}

	result := h.databaseChecker()
	result.Duration = time.Since(start)
	return result
}

// CheckHomeAssistant performs Home Assistant health check
func (h *DefaultHealthChecker) CheckHomeAssistant() HealthStatus {
	start := time.Now()

	if h.homeAssistantChecker == nil {
		return HealthStatus{
			Status:    "unknown",
			Message:   "Home Assistant health checker not configured",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}

	result := h.homeAssistantChecker()
	result.Duration = time.Since(start)
	return result
}

// CheckDeviceAdapters performs device adapter health checks
func (h *DefaultHealthChecker) CheckDeviceAdapters() map[string]HealthStatus {
	if h.deviceAdapterChecker == nil {
		return map[string]HealthStatus{
			"unknown": {
				Status:    "unknown",
				Message:   "Device adapter health checker not configured",
				Timestamp: time.Now(),
			},
		}
	}

	return h.deviceAdapterChecker()
}

// CheckLLMProviders performs LLM provider health checks
func (h *DefaultHealthChecker) CheckLLMProviders() map[string]HealthStatus {
	if h.llmProviderChecker == nil {
		return map[string]HealthStatus{
			"unknown": {
				Status:    "unknown",
				Message:   "LLM provider health checker not configured",
				Timestamp: time.Now(),
			},
		}
	}

	return h.llmProviderChecker()
}

// CheckSystemResources performs system resource health check
func (h *DefaultHealthChecker) CheckSystemResources() HealthStatus {
	start := time.Now()

	if h.systemResourceChecker == nil {
		return HealthStatus{
			Status:    "unknown",
			Message:   "System resource health checker not configured",
			Timestamp: time.Now(),
			Duration:  time.Since(start),
		}
	}

	result := h.systemResourceChecker()
	result.Duration = time.Since(start)
	return result
}

// RegisterCustomCheck registers a custom health check
func (h *DefaultHealthChecker) RegisterCustomCheck(name string, check func() HealthStatus) {
	h.customChecks[name] = check
}

// GetOverallHealth returns the overall system health
func (h *DefaultHealthChecker) GetOverallHealth() HealthReport {
	start := time.Now()

	components := make(map[string]HealthStatus)

	// Check database
	components["database"] = h.CheckDatabase()

	// Check Home Assistant
	components["home_assistant"] = h.CheckHomeAssistant()

	// Check device adapters
	deviceStatuses := h.CheckDeviceAdapters()
	for name, status := range deviceStatuses {
		components["device_"+name] = status
	}

	// Check LLM providers
	llmStatuses := h.CheckLLMProviders()
	for name, status := range llmStatuses {
		components["llm_"+name] = status
	}

	// Check system resources
	components["system_resources"] = h.CheckSystemResources()

	// Check custom health checks
	for name, check := range h.customChecks {
		components["custom_"+name] = check()
	}

	// Determine overall status
	overallStatus, message := h.calculateOverallStatus(components)

	// Gather system info
	systemInfo := h.gatherSystemInfo()

	return HealthReport{
		Status:     overallStatus,
		Message:    message,
		Timestamp:  time.Now(),
		Duration:   time.Since(start),
		Components: components,
		SystemInfo: systemInfo,
	}
}

// calculateOverallStatus determines the overall health status based on component statuses
func (h *DefaultHealthChecker) calculateOverallStatus(components map[string]HealthStatus) (string, string) {
	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0
	unknownCount := 0
	totalCount := len(components)

	for _, status := range components {
		switch status.Status {
		case "healthy":
			healthyCount++
		case "degraded":
			degradedCount++
		case "unhealthy":
			unhealthyCount++
		default:
			unknownCount++
		}
	}

	// Determine overall status
	if unhealthyCount > 0 {
		return "unhealthy", fmt.Sprintf("%d/%d components unhealthy", unhealthyCount, totalCount)
	}

	if degradedCount > 0 {
		return "degraded", fmt.Sprintf("%d/%d components degraded", degradedCount, totalCount)
	}

	if unknownCount > 0 {
		return "unknown", fmt.Sprintf("%d/%d components unknown", unknownCount, totalCount)
	}

	return "healthy", fmt.Sprintf("All %d components healthy", totalCount)
}

// gatherSystemInfo collects system information
func (h *DefaultHealthChecker) gatherSystemInfo() map[string]interface{} {
	info := make(map[string]interface{})

	info["timestamp"] = time.Now().UTC().Format(time.RFC3339)
	info["uptime"] = time.Since(startTime).String()

	return info
}

// startTime tracks when the application started
var startTime = time.Now()

// NewHealthStatus creates a new health status
func NewHealthStatus(status, message string) HealthStatus {
	return HealthStatus{
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}
}

// WithDetails adds details to a health status
func (h HealthStatus) WithDetails(details map[string]interface{}) HealthStatus {
	if h.Details == nil {
		h.Details = make(map[string]interface{})
	}

	for k, v := range details {
		h.Details[k] = v
	}

	return h
}

// WithDetail adds a single detail to a health status
func (h HealthStatus) WithDetail(key string, value interface{}) HealthStatus {
	if h.Details == nil {
		h.Details = make(map[string]interface{})
	}

	h.Details[key] = value
	return h
}

// IsHealthy returns true if the status is healthy
func (h HealthStatus) IsHealthy() bool {
	return h.Status == "healthy"
}

// IsDegraded returns true if the status is degraded
func (h HealthStatus) IsDegraded() bool {
	return h.Status == "degraded"
}

// IsUnhealthy returns true if the status is unhealthy
func (h HealthStatus) IsUnhealthy() bool {
	return h.Status == "unhealthy"
}

// HealthCheckWithTimeout performs a health check with timeout
func HealthCheckWithTimeout(ctx context.Context, timeout time.Duration, check func() HealthStatus) HealthStatus {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resultChan := make(chan HealthStatus, 1)

	go func() {
		resultChan <- check()
	}()

	select {
	case result := <-resultChan:
		return result
	case <-ctx.Done():
		return NewHealthStatus("unhealthy", "Health check timed out").
			WithDetail("timeout", timeout.String())
	}
}
