package metrics

import (
	"time"
)

// MetricsCollector defines the interface for collecting metrics
type MetricsCollector interface {
	RecordHTTPRequest(method, path string, status int, duration time.Duration)
	RecordWebSocketConnection(action string)
	RecordDatabaseQuery(operation string, duration time.Duration)
	RecordDeviceOperation(deviceType, operation string, success bool, duration time.Duration)
	RecordAutomationExecution(ruleID string, success bool, duration time.Duration)
	RecordLLMRequest(provider string, success bool, duration time.Duration, tokens int)
	RecordSystemResource(cpu, memory, disk float64)
	RecordAlert(severity, source, message string)
	IncrementCounter(name string, labels map[string]string)
	RecordHistogram(name string, value float64, labels map[string]string)
	SetGauge(name string, value float64, labels map[string]string)
}

// MetricsConfig contains configuration for metrics collection
type MetricsConfig struct {
	Enabled bool
	Prefix  string
}

// Labels is a convenience type for metric labels
type Labels map[string]string

// DefaultLabels creates a new Labels map with default values
func DefaultLabels() Labels {
	return make(Labels)
}

// Add adds a label to the Labels map
func (l Labels) Add(key, value string) Labels {
	l[key] = value
	return l
}

// With creates a new Labels map with the given key-value pair
func (l Labels) With(key, value string) Labels {
	newLabels := make(Labels)
	for k, v := range l {
		newLabels[k] = v
	}
	newLabels[key] = value
	return newLabels
}
