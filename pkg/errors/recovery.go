package errors

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateHalfOpen
	StateOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern for error recovery
type CircuitBreaker struct {
	name              string
	maxFailures       int
	timeout           time.Duration
	resetTimeout      time.Duration
	halfOpenMaxCalls  int
	state             CircuitBreakerState
	failures          int64
	lastFailTime      time.Time
	halfOpenCalls     int64
	halfOpenSuccesses int64
	mu                sync.RWMutex
	onStateChange     func(name string, from CircuitBreakerState, to CircuitBreakerState)
	logger            *logrus.Logger
}

// CircuitBreakerConfig contains configuration for a circuit breaker
type CircuitBreakerConfig struct {
	Name             string        `yaml:"name"`
	MaxFailures      int           `yaml:"max_failures"`
	Timeout          time.Duration `yaml:"timeout"`
	ResetTimeout     time.Duration `yaml:"reset_timeout"`
	HalfOpenMaxCalls int           `yaml:"half_open_max_calls"`
	OnStateChange    func(name string, from CircuitBreakerState, to CircuitBreakerState)
	Logger           *logrus.Logger
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.MaxFailures <= 0 {
		config.MaxFailures = 5
	}
	if config.Timeout <= 0 {
		config.Timeout = time.Second * 60
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = time.Second * 30
	}
	if config.HalfOpenMaxCalls <= 0 {
		config.HalfOpenMaxCalls = 3
	}

	return &CircuitBreaker{
		name:             config.Name,
		maxFailures:      config.MaxFailures,
		timeout:          config.Timeout,
		resetTimeout:     config.ResetTimeout,
		halfOpenMaxCalls: config.HalfOpenMaxCalls,
		state:            StateClosed,
		onStateChange:    config.OnStateChange,
		logger:           config.Logger,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check if circuit breaker allows execution
	if !cb.allowRequest() {
		return NewEnhanced(503, "Circuit breaker is open", CategoryUnavailable, SeverityHigh).
			WithContext(&ErrorContext{
				Component: cb.name,
				Operation: "circuit_breaker_check",
				Metadata: map[string]interface{}{
					"state":    cb.getState().String(),
					"failures": atomic.LoadInt64(&cb.failures),
				},
			})
	}

	// Execute the function
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	// Record the result
	cb.recordResult(err, duration)

	return err
}

// allowRequest checks if a request should be allowed through the circuit breaker
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if we should transition to half-open
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			// Double-check after acquiring write lock
			if cb.state == StateOpen && time.Since(cb.lastFailTime) > cb.resetTimeout {
				cb.setState(StateHalfOpen)
				cb.halfOpenCalls = 0
				cb.halfOpenSuccesses = 0
			}
			cb.mu.Unlock()
			cb.mu.RLock()
		}
		return cb.state == StateHalfOpen
	case StateHalfOpen:
		return atomic.LoadInt64(&cb.halfOpenCalls) < int64(cb.halfOpenMaxCalls)
	default:
		return false
	}
}

// recordResult records the result of a function execution
func (cb *CircuitBreaker) recordResult(err error, duration time.Duration) {
	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	// Log metrics
	if cb.logger != nil {
		cb.logger.WithFields(logrus.Fields{
			"circuit_breaker": cb.name,
			"state":           cb.getState().String(),
			"success":         err == nil,
			"duration":        duration,
			"failures":        atomic.LoadInt64(&cb.failures),
		}).Debug("Circuit breaker execution recorded")
	}
}

// recordSuccess records a successful execution
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Reset failure count on success
		atomic.StoreInt64(&cb.failures, 0)
	case StateHalfOpen:
		atomic.AddInt64(&cb.halfOpenSuccesses, 1)
		atomic.AddInt64(&cb.halfOpenCalls, 1)

		// Check if we should close the circuit
		if cb.halfOpenSuccesses >= int64(cb.halfOpenMaxCalls/2) {
			cb.setState(StateClosed)
			atomic.StoreInt64(&cb.failures, 0)
		}
	}
}

// recordFailure records a failed execution
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		failures := atomic.AddInt64(&cb.failures, 1)
		if failures >= int64(cb.maxFailures) {
			cb.setState(StateOpen)
			cb.lastFailTime = time.Now()
		}
	case StateHalfOpen:
		cb.setState(StateOpen)
		cb.lastFailTime = time.Now()
		atomic.AddInt64(&cb.failures, 1)
	}
}

// setState changes the circuit breaker state and notifies listeners
func (cb *CircuitBreaker) setState(newState CircuitBreakerState) {
	oldState := cb.state
	cb.state = newState

	if cb.onStateChange != nil && oldState != newState {
		go cb.onStateChange(cb.name, oldState, newState)
	}

	if cb.logger != nil {
		cb.logger.WithFields(logrus.Fields{
			"circuit_breaker": cb.name,
			"old_state":       oldState.String(),
			"new_state":       newState.String(),
			"failures":        atomic.LoadInt64(&cb.failures),
		}).Info("Circuit breaker state changed")
	}
}

// getState safely gets the current state
func (cb *CircuitBreaker) getState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"name":                cb.name,
		"state":               cb.state.String(),
		"failures":            atomic.LoadInt64(&cb.failures),
		"max_failures":        cb.maxFailures,
		"last_fail_time":      cb.lastFailTime,
		"half_open_calls":     atomic.LoadInt64(&cb.halfOpenCalls),
		"half_open_successes": atomic.LoadInt64(&cb.halfOpenSuccesses),
		"timeout":             cb.timeout,
		"reset_timeout":       cb.resetTimeout,
	}
}

// RetryPolicy defines retry behavior for operations
type RetryPolicy struct {
	MaxAttempts     int             `yaml:"max_attempts"`
	InitialDelay    time.Duration   `yaml:"initial_delay"`
	MaxDelay        time.Duration   `yaml:"max_delay"`
	BackoffFactor   float64         `yaml:"backoff_factor"`
	Jitter          bool            `yaml:"jitter"`
	RetryableErrors []ErrorCategory `yaml:"retryable_errors"`
}

// DefaultRetryPolicy returns a sensible default retry policy
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxAttempts:   3,
		InitialDelay:  time.Millisecond * 100,
		MaxDelay:      time.Second * 30,
		BackoffFactor: 2.0,
		Jitter:        true,
		RetryableErrors: []ErrorCategory{
			CategoryNetwork,
			CategoryTimeout,
			CategoryUnavailable,
			CategoryRateLimit,
		},
	}
}

// ShouldRetry determines if an error should be retried
func (rp *RetryPolicy) ShouldRetry(err error, attempt int) bool {
	if attempt >= rp.MaxAttempts {
		return false
	}

	if enhancedErr, ok := err.(*EnhancedError); ok {
		// Check if error is explicitly retryable
		if !enhancedErr.Retryable {
			return false
		}

		// Check if error category is in retryable list
		for _, category := range rp.RetryableErrors {
			if enhancedErr.Category == category {
				return true
			}
		}
		return false
	}

	// For non-enhanced errors, be conservative
	return false
}

// GetDelay calculates the delay before the next retry attempt
func (rp *RetryPolicy) GetDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return rp.InitialDelay
	}

	delay := float64(rp.InitialDelay) * math.Pow(rp.BackoffFactor, float64(attempt-1))

	if rp.Jitter {
		// Add up to 25% jitter
		jitter := delay * 0.25 * (math.Mod(float64(time.Now().UnixNano()), 1.0))
		delay += jitter
	}

	if time.Duration(delay) > rp.MaxDelay {
		delay = float64(rp.MaxDelay)
	}

	return time.Duration(delay)
}

// RetryExecutor executes functions with retry logic
type RetryExecutor struct {
	policy *RetryPolicy
	logger *logrus.Logger
}

// NewRetryExecutor creates a new retry executor
func NewRetryExecutor(policy *RetryPolicy, logger *logrus.Logger) *RetryExecutor {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}
	return &RetryExecutor{
		policy: policy,
		logger: logger,
	}
}

// Execute executes a function with retry logic
func (re *RetryExecutor) Execute(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= re.policy.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the function
		err := fn()
		if err == nil {
			// Success
			if attempt > 1 && re.logger != nil {
				re.logger.WithFields(logrus.Fields{
					"operation": operation,
					"attempt":   attempt,
				}).Info("Operation succeeded after retry")
			}
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !re.policy.ShouldRetry(err, attempt) {
			if re.logger != nil {
				re.logger.WithFields(logrus.Fields{
					"operation": operation,
					"attempt":   attempt,
					"error":     err.Error(),
				}).Debug("Error not retryable, stopping")
			}
			break
		}

		// If this is the last attempt, don't wait
		if attempt >= re.policy.MaxAttempts {
			break
		}

		// Calculate delay and wait
		delay := re.policy.GetDelay(attempt)
		if re.logger != nil {
			re.logger.WithFields(logrus.Fields{
				"operation": operation,
				"attempt":   attempt,
				"delay":     delay,
				"error":     err.Error(),
			}).Debug("Retrying operation after delay")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	// All attempts failed
	if enhancedErr, ok := lastErr.(*EnhancedError); ok {
		// Add retry information to the error
		enhancedErr.Context.Metadata["retry_attempts"] = re.policy.MaxAttempts
		enhancedErr.Context.Metadata["operation"] = operation
		return enhancedErr
	}

	return lastErr
}

// ErrorReporter collects and reports errors for monitoring and alerting
type ErrorReporter struct {
	errors     []ErrorReport
	mu         sync.RWMutex
	maxErrors  int
	logger     *logrus.Logger
	onError    func(ErrorReport)
	onCritical func(ErrorReport)
}

// ErrorReport contains information about a reported error
type ErrorReport struct {
	Error          *EnhancedError         `json:"error"`
	Timestamp      time.Time              `json:"timestamp"`
	Count          int64                  `json:"count"`
	LastSeen       time.Time              `json:"last_seen"`
	FirstSeen      time.Time              `json:"first_seen"`
	Context        map[string]interface{} `json:"context"`
	Resolved       bool                   `json:"resolved"`
	ResolutionTime *time.Time             `json:"resolution_time,omitempty"`
}

// NewErrorReporter creates a new error reporter
func NewErrorReporter(maxErrors int, logger *logrus.Logger) *ErrorReporter {
	if maxErrors <= 0 {
		maxErrors = 1000
	}

	return &ErrorReporter{
		errors:    make([]ErrorReport, 0),
		maxErrors: maxErrors,
		logger:    logger,
	}
}

// SetCallbacks sets callback functions for error reporting
func (er *ErrorReporter) SetCallbacks(onError func(ErrorReport), onCritical func(ErrorReport)) {
	er.onError = onError
	er.onCritical = onCritical
}

// ReportError reports an error to the error reporter
func (er *ErrorReporter) ReportError(err *EnhancedError) {
	if err == nil {
		return
	}

	er.mu.Lock()
	defer er.mu.Unlock()

	now := time.Now()

	// Look for existing error report
	for i := range er.errors {
		report := &er.errors[i]
		if report.Error.Code == err.Code &&
			report.Error.Category == err.Category &&
			report.Error.Message == err.Message {
			// Update existing report
			report.Count++
			report.LastSeen = now

			// Trigger callbacks
			er.triggerCallbacks(*report)
			return
		}
	}

	// Create new error report
	report := ErrorReport{
		Error:     err,
		Timestamp: now,
		Count:     1,
		LastSeen:  now,
		FirstSeen: now,
		Context:   make(map[string]interface{}),
	}

	// Add contextual information
	if err.Context != nil {
		report.Context["operation"] = err.Context.Operation
		report.Context["component"] = err.Context.Component
		report.Context["entity_id"] = err.Context.EntityID
		report.Context["request_id"] = err.Context.RequestID
	}

	er.errors = append(er.errors, report)

	// Limit the number of stored errors
	if len(er.errors) > er.maxErrors {
		er.errors = er.errors[1:]
	}

	// Trigger callbacks
	er.triggerCallbacks(report)

	// Log the error
	if er.logger != nil {
		er.logger.WithFields(logrus.Fields{
			"error_id":  err.ErrorID,
			"category":  err.Category,
			"severity":  err.Severity,
			"retryable": err.Retryable,
			"operation": err.Context.Operation,
			"component": err.Context.Component,
		}).Error("Error reported")
	}
}

// triggerCallbacks triggers appropriate callbacks based on error severity
func (er *ErrorReporter) triggerCallbacks(report ErrorReport) {
	if er.onError != nil {
		go er.onError(report)
	}

	if report.Error.Severity == SeverityCritical && er.onCritical != nil {
		go er.onCritical(report)
	}
}

// GetErrorReports returns all error reports
func (er *ErrorReporter) GetErrorReports() []ErrorReport {
	er.mu.RLock()
	defer er.mu.RUnlock()

	// Return a copy to avoid race conditions
	reports := make([]ErrorReport, len(er.errors))
	copy(reports, er.errors)
	return reports
}

// GetErrorStats returns error statistics
func (er *ErrorReporter) GetErrorStats() map[string]interface{} {
	er.mu.RLock()
	defer er.mu.RUnlock()

	stats := map[string]interface{}{
		"total_errors":     len(er.errors),
		"by_category":      make(map[string]int),
		"by_severity":      make(map[string]int),
		"critical_count":   0,
		"unresolved_count": 0,
	}

	categoryStats := stats["by_category"].(map[string]int)
	severityStats := stats["by_severity"].(map[string]int)

	for _, report := range er.errors {
		categoryStats[string(report.Error.Category)]++
		severityStats[string(report.Error.Severity)]++

		if report.Error.Severity == SeverityCritical {
			stats["critical_count"] = stats["critical_count"].(int) + 1
		}

		if !report.Resolved {
			stats["unresolved_count"] = stats["unresolved_count"].(int) + 1
		}
	}

	return stats
}

// ResolveError marks an error as resolved
func (er *ErrorReporter) ResolveError(errorID string) bool {
	er.mu.Lock()
	defer er.mu.Unlock()

	for i := range er.errors {
		if er.errors[i].Error.ErrorID == errorID {
			now := time.Now()
			er.errors[i].Resolved = true
			er.errors[i].ResolutionTime = &now
			return true
		}
	}

	return false
}

// ClearOldErrors removes resolved errors older than the specified duration
func (er *ErrorReporter) ClearOldErrors(maxAge time.Duration) int {
	er.mu.Lock()
	defer er.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	var kept []ErrorReport
	removed := 0

	for _, report := range er.errors {
		// Keep unresolved errors or recently resolved ones
		if !report.Resolved || (report.ResolutionTime != nil && report.ResolutionTime.After(cutoff)) {
			kept = append(kept, report)
		} else {
			removed++
		}
	}

	er.errors = kept
	return removed
}

// RecoveryManager coordinates all error recovery mechanisms
type RecoveryManager struct {
	circuitBreakers map[string]*CircuitBreaker
	retryExecutor   *RetryExecutor
	errorReporter   *ErrorReporter
	logger          *logrus.Logger
	mu              sync.RWMutex
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(logger *logrus.Logger) *RecoveryManager {
	return &RecoveryManager{
		circuitBreakers: make(map[string]*CircuitBreaker),
		retryExecutor:   NewRetryExecutor(DefaultRetryPolicy(), logger),
		errorReporter:   NewErrorReporter(1000, logger),
		logger:          logger,
	}
}

// AddCircuitBreaker adds a circuit breaker for a specific operation
func (rm *RecoveryManager) AddCircuitBreaker(name string, config CircuitBreakerConfig) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	config.Name = name
	config.Logger = rm.logger
	config.OnStateChange = rm.onCircuitBreakerStateChange

	rm.circuitBreakers[name] = NewCircuitBreaker(config)
}

// ExecuteWithRecovery executes an operation with full error recovery
func (rm *RecoveryManager) ExecuteWithRecovery(ctx context.Context, operation string, fn func() error) error {
	// Wrap the function with retry logic
	retryFn := func() error {
		return rm.retryExecutor.Execute(ctx, operation, fn)
	}

	// Execute with circuit breaker if available
	if cb, exists := rm.getCircuitBreaker(operation); exists {
		err := cb.Execute(ctx, retryFn)
		if err != nil {
			// Report the error
			if enhancedErr, ok := err.(*EnhancedError); ok {
				rm.errorReporter.ReportError(enhancedErr)
			}
		}
		return err
	}

	// Execute with retry only
	err := retryFn()
	if err != nil {
		// Report the error
		if enhancedErr, ok := err.(*EnhancedError); ok {
			rm.errorReporter.ReportError(enhancedErr)
		}
	}
	return err
}

// getCircuitBreaker safely gets a circuit breaker
func (rm *RecoveryManager) getCircuitBreaker(name string) (*CircuitBreaker, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	cb, exists := rm.circuitBreakers[name]
	return cb, exists
}

// onCircuitBreakerStateChange handles circuit breaker state changes
func (rm *RecoveryManager) onCircuitBreakerStateChange(name string, from, to CircuitBreakerState) {
	if rm.logger != nil {
		rm.logger.WithFields(logrus.Fields{
			"circuit_breaker": name,
			"from_state":      from.String(),
			"to_state":        to.String(),
		}).Warn("Circuit breaker state changed")
	}

	// Report critical errors when circuit breaker opens
	if to == StateOpen {
		err := NewEnhanced(503,
			fmt.Sprintf("Circuit breaker '%s' opened due to failures", name),
			CategoryUnavailable, SeverityCritical)
		err.Context = &ErrorContext{
			Component: name,
			Operation: "circuit_breaker_state_change",
			Metadata: map[string]interface{}{
				"from_state": from.String(),
				"to_state":   to.String(),
			},
		}
		rm.errorReporter.ReportError(err)
	}
}

// GetMetrics returns comprehensive recovery metrics
func (rm *RecoveryManager) GetMetrics() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	metrics := map[string]interface{}{
		"circuit_breakers": make(map[string]interface{}),
		"error_stats":      rm.errorReporter.GetErrorStats(),
	}

	// Add circuit breaker metrics
	cbMetrics := metrics["circuit_breakers"].(map[string]interface{})
	for name, cb := range rm.circuitBreakers {
		cbMetrics[name] = cb.GetMetrics()
	}

	return metrics
}

// SetErrorCallbacks sets callback functions for error reporting
func (rm *RecoveryManager) SetErrorCallbacks(onError func(ErrorReport), onCritical func(ErrorReport)) {
	rm.errorReporter.SetCallbacks(onError, onCritical)
}

// GetErrorReporter returns the error reporter for direct access
func (rm *RecoveryManager) GetErrorReporter() *ErrorReporter {
	return rm.errorReporter
}
