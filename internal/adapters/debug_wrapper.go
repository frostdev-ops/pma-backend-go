package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/pkg/debug"
)

// DebugAdapterWrapper wraps any adapter with debug logging
type DebugAdapterWrapper struct {
	adapter     interface{}
	debugLogger *debug.DebugLogger
	adapterName string
}

// NewDebugAdapterWrapper creates a new debug wrapper for an adapter
func NewDebugAdapterWrapper(adapter interface{}, debugLogger *debug.DebugLogger, adapterName string) *DebugAdapterWrapper {
	return &DebugAdapterWrapper{
		adapter:     adapter,
		debugLogger: debugLogger,
		adapterName: adapterName,
	}
}

// LogMethodCall logs a method call with parameters and timing
func (daw *DebugAdapterWrapper) LogMethodCall(ctx context.Context, methodName string, args ...interface{}) (func(), map[string]interface{}) {
	if daw.debugLogger == nil || !daw.debugLogger.IsComponentEnabled("adapters") {
		return func() {}, nil
	}

	start := time.Now()
	fields := map[string]interface{}{
		"adapter":   daw.adapterName,
		"method":    methodName,
		"args":      daw.sanitizeArgs(args),
		"timestamp": start,
	}

	// Log method call
	daw.debugLogger.LogWithContext(ctx, "debug", "adapters", "Method Call", fields)

	// Return cleanup function
	return func() {
		duration := time.Since(start)
		fields["duration"] = duration.String()
		daw.debugLogger.LogWithContext(ctx, "debug", "adapters", "Method Completed", fields)
	}, fields
}

// LogMethodError logs a method error
func (daw *DebugAdapterWrapper) LogMethodError(ctx context.Context, methodName string, err error, args ...interface{}) {
	if daw.debugLogger == nil || !daw.debugLogger.IsComponentEnabled("adapters") {
		return
	}

	fields := map[string]interface{}{
		"adapter":   daw.adapterName,
		"method":    methodName,
		"args":      daw.sanitizeArgs(args),
		"timestamp": time.Now(),
	}

	daw.debugLogger.LogError("adapters", fmt.Sprintf("Method Error: %s.%s", daw.adapterName, methodName), err, fields)
}

// LogDataExchange logs data exchange operations (read/write)
func (daw *DebugAdapterWrapper) LogDataExchange(ctx context.Context, operation string, data interface{}, metadata map[string]interface{}) {
	if daw.debugLogger == nil || !daw.debugLogger.IsComponentEnabled("adapters") {
		return
	}

	fields := map[string]interface{}{
		"adapter":   daw.adapterName,
		"operation": operation,
		"data_type": reflect.TypeOf(data).String(),
		"data_size": daw.getDataSize(data),
		"timestamp": time.Now(),
	}

	// Add metadata
	for k, v := range metadata {
		fields[k] = v
	}

	// Add data sample if not too large
	if dataSample := daw.getDataSample(data); dataSample != nil {
		fields["data_sample"] = dataSample
	}

	daw.debugLogger.LogWithContext(ctx, "debug", "adapters", "Data Exchange", fields)
}

// LogConnection logs connection events
func (daw *DebugAdapterWrapper) LogConnection(ctx context.Context, event string, details map[string]interface{}) {
	if daw.debugLogger == nil || !daw.debugLogger.IsComponentEnabled("adapters") {
		return
	}

	fields := map[string]interface{}{
		"adapter":   daw.adapterName,
		"event":     event,
		"timestamp": time.Now(),
	}

	// Add connection details
	for k, v := range details {
		fields[k] = v
	}

	daw.debugLogger.LogWithContext(ctx, "debug", "adapters", "Connection Event", fields)
}

// LogHealthCheck logs health check results
func (daw *DebugAdapterWrapper) LogHealthCheck(ctx context.Context, status string, details map[string]interface{}) {
	if daw.debugLogger == nil || !daw.debugLogger.IsComponentEnabled("adapters") {
		return
	}

	fields := map[string]interface{}{
		"adapter":   daw.adapterName,
		"status":    status,
		"timestamp": time.Now(),
	}

	// Add health check details
	for k, v := range details {
		fields[k] = v
	}

	level := "debug"
	if status == "error" || status == "failed" {
		level = "error"
	} else if status == "warning" {
		level = "warn"
	}

	daw.debugLogger.LogWithContext(ctx, level, "adapters", "Health Check", fields)
}

// LogConfiguration logs configuration changes
func (daw *DebugAdapterWrapper) LogConfiguration(ctx context.Context, action string, config interface{}) {
	if daw.debugLogger == nil || !daw.debugLogger.IsComponentEnabled("adapters") {
		return
	}

	fields := map[string]interface{}{
		"adapter":   daw.adapterName,
		"action":    action,
		"timestamp": time.Now(),
	}

	// Add configuration data (sanitized)
	if configSample := daw.getConfigSample(config); configSample != nil {
		fields["config_sample"] = configSample
	}

	daw.debugLogger.LogWithContext(ctx, "debug", "adapters", "Configuration Change", fields)
}

// LogDiscovery logs discovery operations
func (daw *DebugAdapterWrapper) LogDiscovery(ctx context.Context, operation string, results interface{}, metadata map[string]interface{}) {
	if daw.debugLogger == nil || !daw.debugLogger.IsComponentEnabled("adapters") {
		return
	}

	fields := map[string]interface{}{
		"adapter":   daw.adapterName,
		"operation": operation,
		"timestamp": time.Now(),
	}

	// Add metadata
	for k, v := range metadata {
		fields[k] = v
	}

	// Add results summary
	if resultsSummary := daw.getResultsSummary(results); resultsSummary != nil {
		fields["results_summary"] = resultsSummary
	}

	daw.debugLogger.LogWithContext(ctx, "debug", "adapters", "Discovery Operation", fields)
}

// sanitizeArgs sanitizes method arguments for logging
func (daw *DebugAdapterWrapper) sanitizeArgs(args []interface{}) []interface{} {
	sanitized := make([]interface{}, len(args))
	for i, arg := range args {
		sanitized[i] = daw.sanitizeValue(arg)
	}
	return sanitized
}

// sanitizeValue sanitizes a value for logging (removes sensitive data)
func (daw *DebugAdapterWrapper) sanitizeValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	// Convert to string and check for sensitive patterns
	str := fmt.Sprintf("%v", value)
	lowerStr := strings.ToLower(str)

	// Check for sensitive patterns
	sensitivePatterns := []string{
		"password",
		"token",
		"secret",
		"key",
		"auth",
		"credential",
	}

	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerStr, pattern) {
			return "[REDACTED]"
		}
	}

	// Limit string length
	if len(str) > 1000 {
		return str[:1000] + "..."
	}

	return value
}

// getDataSize gets the size of data for logging
func (daw *DebugAdapterWrapper) getDataSize(data interface{}) interface{} {
	if data == nil {
		return 0
	}

	switch v := data.(type) {
	case string:
		return len(v)
	case []byte:
		return len(v)
	case map[string]interface{}:
		return len(v)
	case []interface{}:
		return len(v)
	default:
		// Try to get size via reflection
		val := reflect.ValueOf(data)
		switch val.Kind() {
		case reflect.Slice, reflect.Array, reflect.Map:
			return val.Len()
		default:
			return "unknown"
		}
	}
}

// getDataSample gets a sample of data for logging
func (daw *DebugAdapterWrapper) getDataSample(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	// Limit data size for logging
	switch v := data.(type) {
	case string:
		if len(v) > 500 {
			return v[:500] + "..."
		}
		return v
	case []byte:
		if len(v) > 500 {
			return fmt.Sprintf("%x...", v[:250])
		}
		return fmt.Sprintf("%x", v)
	case map[string]interface{}:
		// Limit map entries
		if len(v) > 10 {
			sample := make(map[string]interface{})
			count := 0
			for k, val := range v {
				if count >= 10 {
					break
				}
				sample[k] = daw.sanitizeValue(val)
				count++
			}
			return sample
		}
		return daw.sanitizeValue(v)
	case []interface{}:
		// Limit slice entries
		if len(v) > 10 {
			return v[:10]
		}
		return v
	default:
		// Try to convert to JSON
		if jsonData, err := json.Marshal(data); err == nil {
			if len(jsonData) > 500 {
				return string(jsonData[:500]) + "..."
			}
			return string(jsonData)
		}
		return fmt.Sprintf("%v", data)
	}
}

// getConfigSample gets a sample of configuration for logging
func (daw *DebugAdapterWrapper) getConfigSample(config interface{}) interface{} {
	return daw.getDataSample(config)
}

// getResultsSummary gets a summary of discovery results
func (daw *DebugAdapterWrapper) getResultsSummary(results interface{}) interface{} {
	if results == nil {
		return nil
	}

	val := reflect.ValueOf(results)
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		return map[string]interface{}{
			"count": val.Len(),
			"type":  val.Type().String(),
		}
	case reflect.Map:
		return map[string]interface{}{
			"count": val.Len(),
			"type":  val.Type().String(),
		}
	default:
		return map[string]interface{}{
			"type":  val.Type().String(),
			"value": daw.sanitizeValue(results),
		}
	}
}

// GetAdapter returns the wrapped adapter
func (daw *DebugAdapterWrapper) GetAdapter() interface{} {
	return daw.adapter
}

// GetDebugLogger returns the debug logger
func (daw *DebugAdapterWrapper) GetDebugLogger() *debug.DebugLogger {
	return daw.debugLogger
}
