package debug

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// ServiceLogger provides comprehensive logging utilities for any component.
type ServiceLogger struct {
	logger    *DebugLogger
	component string
}

// NewServiceLogger creates a new service logger utility.
func NewServiceLogger(component string, logger *DebugLogger) *ServiceLogger {
	return &ServiceLogger{
		component: component,
		logger:    logger,
	}
}

// LogCall logs a method/function call with parameters and timing.
// It returns a function that should be deferred to log completion.
func (sl *ServiceLogger) LogCall(ctx context.Context, method string, args ...interface{}) func() {
	if sl.logger == nil || !sl.logger.IsComponentEnabled(sl.component) {
		return func() {}
	}

	start := time.Now()
	fields := map[string]interface{}{
		"method":    method,
		"arguments": sl.sanitize(args),
	}

	sl.logger.LogWithContext(ctx, "debug", sl.component, "Call started", fields)

	return func() {
		duration := time.Since(start)
		sl.logger.LogDuration(sl.component, "Call completed", duration, fields)
	}
}

// LogError logs an error within a component's method.
func (sl *ServiceLogger) LogError(ctx context.Context, method string, err error, details map[string]interface{}) {
	if sl.logger == nil || !sl.logger.IsComponentEnabled(sl.component) {
		return
	}

	fields := map[string]interface{}{
		"method": method,
	}
	for k, v := range details {
		fields[k] = v
	}

	sl.logger.LogError(sl.component, "An error occurred", err, fields)
}

// LogInfo logs an informational message.
func (sl *ServiceLogger) LogInfo(ctx context.Context, method string, message string, details map[string]interface{}) {
	if sl.logger == nil || !sl.logger.IsComponentEnabled(sl.component) {
		return
	}

	fields := map[string]interface{}{
		"method": method,
	}
	for k, v := range details {
		fields[k] = v
	}

	sl.logger.LogWithContext(ctx, "info", sl.component, message, fields)
}

// LogData logs a data payload for debugging purposes.
func (sl *ServiceLogger) LogData(ctx context.Context, method string, operation string, data interface{}) {
	if sl.logger == nil || !sl.logger.IsComponentEnabled(sl.component) {
		return
	}

	fields := map[string]interface{}{
		"method":    method,
		"operation": operation,
		"data_type": reflect.TypeOf(data).String(),
		"data_size": sl.getDataSize(data),
		"data":      sl.getDataSample(data),
	}

	sl.logger.LogWithContext(ctx, "debug", sl.component, "Data details", fields)
}

// sanitize sanitizes arguments for logging.
func (sl *ServiceLogger) sanitize(args []interface{}) []interface{} {
	sanitized := make([]interface{}, len(args))
	for i, arg := range args {
		sanitized[i] = sl.sanitizeValue(arg)
	}
	return sanitized
}

// sanitizeValue sanitizes a single value, redacting sensitive information.
func (sl *ServiceLogger) sanitizeValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr && val.IsNil() {
		return nil
	}

	str := fmt.Sprintf("%#v", value)
	lowerStr := strings.ToLower(str)

	sensitivePatterns := []string{"password", "token", "secret", "key", "auth", "credential", "pin"}
	for _, pattern := range sensitivePatterns {
		if strings.Contains(lowerStr, pattern) {
			return "[REDACTED]"
		}
	}

	if len(str) > 1000 {
		return str[:1000] + "..."
	}

	return value
}

// getDataSize estimates the size of the data.
func (sl *ServiceLogger) getDataSize(data interface{}) interface{} {
	if data == nil {
		return 0
	}
	val := reflect.ValueOf(data)
	switch val.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
		return val.Len()
	default:
		return "unknown"
	}
}

// getDataSample returns a limited-size sample of the data for logging.
func (sl *ServiceLogger) getDataSample(data interface{}) interface{} {
	if data == nil {
		return nil
	}

	// Attempt to marshal to JSON first for a clean representation
	jsonData, err := json.Marshal(data)
	if err == nil {
		if len(jsonData) > 500 {
			return string(jsonData[:500]) + "..."
		}
		return json.RawMessage(jsonData) // Return as RawMessage to keep it as JSON
	}

	// Fallback to string representation if JSON marshaling fails
	strData := fmt.Sprintf("%#v", data)
	if len(strData) > 500 {
		return strData[:500] + "..."
	}
	return strData
}
