package automation

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ExecutionContext represents the execution context for a rule
type ExecutionContext struct {
	ID        string                 `json:"id"`
	RuleID    string                 `json:"rule_id"`
	TriggerID string                 `json:"trigger_id"`
	StartTime time.Time              `json:"start_time"`
	Variables map[string]interface{} `json:"variables"`
	Stack     []string               `json:"stack"`
	Trace     []TraceEntry           `json:"trace"`
	Metrics   *ExecutionMetrics      `json:"metrics"`

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	logger *logrus.Logger
}

// TraceEntry represents a single execution trace entry
type TraceEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type"` // trigger, condition, action
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// ExecutionMetrics contains performance metrics for rule execution
type ExecutionMetrics struct {
	TriggersEvaluated   int           `json:"triggers_evaluated"`
	ConditionsEvaluated int           `json:"conditions_evaluated"`
	ActionsExecuted     int           `json:"actions_executed"`
	TotalDuration       time.Duration `json:"total_duration"`
	TriggerDuration     time.Duration `json:"trigger_duration"`
	ConditionDuration   time.Duration `json:"condition_duration"`
	ActionDuration      time.Duration `json:"action_duration"`

	// Memory and performance
	MemoryUsage    int64 `json:"memory_usage"`
	PeakMemory     int64 `json:"peak_memory"`
	GoroutineCount int   `json:"goroutine_count"`
}

// NewExecutionContext creates a new execution context
func NewExecutionContext(ctx context.Context, ruleID, triggerID string, logger *logrus.Logger) *ExecutionContext {
	childCtx, cancel := context.WithCancel(ctx)

	return &ExecutionContext{
		ID:        uuid.New().String(),
		RuleID:    ruleID,
		TriggerID: triggerID,
		StartTime: time.Now(),
		Variables: make(map[string]interface{}),
		Stack:     make([]string, 0),
		Trace:     make([]TraceEntry, 0),
		Metrics: &ExecutionMetrics{
			MemoryUsage:    0,
			PeakMemory:     0,
			GoroutineCount: 0,
		},
		ctx:    childCtx,
		cancel: cancel,
		logger: logger,
	}
}

// Context returns the underlying context
func (ec *ExecutionContext) Context() context.Context {
	return ec.ctx
}

// Cancel cancels the execution context
func (ec *ExecutionContext) Cancel() {
	ec.cancel()
}

// IsCancelled checks if the context is cancelled
func (ec *ExecutionContext) IsCancelled() bool {
	select {
	case <-ec.ctx.Done():
		return true
	default:
		return false
	}
}

// SetVariable sets a variable in the context
func (ec *ExecutionContext) SetVariable(key string, value interface{}) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.Variables[key] = value
}

// GetVariable gets a variable from the context
func (ec *ExecutionContext) GetVariable(key string) (interface{}, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	value, exists := ec.Variables[key]
	return value, exists
}

// GetAllVariables returns all variables
func (ec *ExecutionContext) GetAllVariables() map[string]interface{} {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	result := make(map[string]interface{})
	for k, v := range ec.Variables {
		result[k] = v
	}
	return result
}

// PushStack pushes an item onto the execution stack
func (ec *ExecutionContext) PushStack(item string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.Stack = append(ec.Stack, item)
}

// PopStack pops an item from the execution stack
func (ec *ExecutionContext) PopStack() string {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if len(ec.Stack) == 0 {
		return ""
	}

	item := ec.Stack[len(ec.Stack)-1]
	ec.Stack = ec.Stack[:len(ec.Stack)-1]
	return item
}

// GetStack returns the current execution stack
func (ec *ExecutionContext) GetStack() []string {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	result := make([]string, len(ec.Stack))
	copy(result, ec.Stack)
	return result
}

// AddTrace adds a trace entry
func (ec *ExecutionContext) AddTrace(entryType, id, name string, success bool, duration time.Duration, err error, data map[string]interface{}) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	entry := TraceEntry{
		Timestamp: time.Now(),
		Type:      entryType,
		ID:        id,
		Name:      name,
		Success:   success,
		Duration:  duration,
		Data:      data,
	}

	if err != nil {
		entry.Error = err.Error()
	}

	ec.Trace = append(ec.Trace, entry)

	// Update metrics based on entry type
	switch entryType {
	case "trigger":
		ec.Metrics.TriggersEvaluated++
		ec.Metrics.TriggerDuration += duration
	case "condition":
		ec.Metrics.ConditionsEvaluated++
		ec.Metrics.ConditionDuration += duration
	case "action":
		ec.Metrics.ActionsExecuted++
		ec.Metrics.ActionDuration += duration
	}
}

// GetTrace returns the execution trace
func (ec *ExecutionContext) GetTrace() []TraceEntry {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	result := make([]TraceEntry, len(ec.Trace))
	copy(result, ec.Trace)
	return result
}

// GetMetrics returns execution metrics
func (ec *ExecutionContext) GetMetrics() *ExecutionMetrics {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	// Update total duration
	ec.Metrics.TotalDuration = time.Since(ec.StartTime)

	// Create a copy
	metrics := *ec.Metrics
	return &metrics
}

// UpdateMemoryUsage updates memory usage metrics
func (ec *ExecutionContext) UpdateMemoryUsage(current int64) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.Metrics.MemoryUsage = current
	if current > ec.Metrics.PeakMemory {
		ec.Metrics.PeakMemory = current
	}
}

// SetGoroutineCount sets the current goroutine count
func (ec *ExecutionContext) SetGoroutineCount(count int) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.Metrics.GoroutineCount = count
}

// Log logs a message with context information
func (ec *ExecutionContext) Log() *logrus.Entry {
	return ec.logger.WithFields(logrus.Fields{
		"execution_id": ec.ID,
		"rule_id":      ec.RuleID,
		"trigger_id":   ec.TriggerID,
	})
}

// LogTrace logs the current trace for debugging
func (ec *ExecutionContext) LogTrace() {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	entry := ec.Log()
	entry.Info("Execution trace:")

	for i, trace := range ec.Trace {
		entry.WithFields(logrus.Fields{
			"step":     i + 1,
			"type":     trace.Type,
			"id":       trace.ID,
			"name":     trace.Name,
			"success":  trace.Success,
			"duration": trace.Duration,
			"error":    trace.Error,
		}).Info("Trace entry")
	}
}

// GetSummary returns a summary of the execution context
func (ec *ExecutionContext) GetSummary() map[string]interface{} {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	successCount := 0
	errorCount := 0

	for _, trace := range ec.Trace {
		if trace.Success {
			successCount++
		} else {
			errorCount++
		}
	}

	return map[string]interface{}{
		"id":               ec.ID,
		"rule_id":          ec.RuleID,
		"trigger_id":       ec.TriggerID,
		"start_time":       ec.StartTime,
		"duration":         time.Since(ec.StartTime),
		"total_steps":      len(ec.Trace),
		"successful_steps": successCount,
		"failed_steps":     errorCount,
		"variables_count":  len(ec.Variables),
		"stack_depth":      len(ec.Stack),
		"metrics":          ec.GetMetrics(),
	}
}

// ExecutionContextManager manages execution contexts
type ExecutionContextManager struct {
	contexts map[string]*ExecutionContext
	mu       sync.RWMutex
	logger   *logrus.Logger
}

// NewExecutionContextManager creates a new execution context manager
func NewExecutionContextManager(logger *logrus.Logger) *ExecutionContextManager {
	return &ExecutionContextManager{
		contexts: make(map[string]*ExecutionContext),
		logger:   logger,
	}
}

// CreateContext creates a new execution context
func (ecm *ExecutionContextManager) CreateContext(ctx context.Context, ruleID, triggerID string) *ExecutionContext {
	execCtx := NewExecutionContext(ctx, ruleID, triggerID, ecm.logger)

	ecm.mu.Lock()
	ecm.contexts[execCtx.ID] = execCtx
	ecm.mu.Unlock()

	return execCtx
}

// GetContext retrieves an execution context by ID
func (ecm *ExecutionContextManager) GetContext(id string) (*ExecutionContext, bool) {
	ecm.mu.RLock()
	defer ecm.mu.RUnlock()

	ctx, exists := ecm.contexts[id]
	return ctx, exists
}

// RemoveContext removes an execution context
func (ecm *ExecutionContextManager) RemoveContext(id string) {
	ecm.mu.Lock()
	defer ecm.mu.Unlock()

	if ctx, exists := ecm.contexts[id]; exists {
		ctx.Cancel()
		delete(ecm.contexts, id)
	}
}

// GetActiveContexts returns all active execution contexts
func (ecm *ExecutionContextManager) GetActiveContexts() []*ExecutionContext {
	ecm.mu.RLock()
	defer ecm.mu.RUnlock()

	contexts := make([]*ExecutionContext, 0, len(ecm.contexts))
	for _, ctx := range ecm.contexts {
		contexts = append(contexts, ctx)
	}

	return contexts
}

// GetContextsForRule returns all contexts for a specific rule
func (ecm *ExecutionContextManager) GetContextsForRule(ruleID string) []*ExecutionContext {
	ecm.mu.RLock()
	defer ecm.mu.RUnlock()

	var contexts []*ExecutionContext
	for _, ctx := range ecm.contexts {
		if ctx.RuleID == ruleID {
			contexts = append(contexts, ctx)
		}
	}

	return contexts
}

// CancelContextsForRule cancels all contexts for a specific rule
func (ecm *ExecutionContextManager) CancelContextsForRule(ruleID string) {
	ecm.mu.Lock()
	defer ecm.mu.Unlock()

	for id, ctx := range ecm.contexts {
		if ctx.RuleID == ruleID {
			ctx.Cancel()
			delete(ecm.contexts, id)
		}
	}
}

// CleanupExpiredContexts removes contexts that have been completed or are too old
func (ecm *ExecutionContextManager) CleanupExpiredContexts(maxAge time.Duration) {
	ecm.mu.Lock()
	defer ecm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)

	for id, ctx := range ecm.contexts {
		if ctx.StartTime.Before(cutoff) || ctx.IsCancelled() {
			ctx.Cancel()
			delete(ecm.contexts, id)
		}
	}
}

// GetStatistics returns statistics about execution contexts
func (ecm *ExecutionContextManager) GetStatistics() map[string]interface{} {
	ecm.mu.RLock()
	defer ecm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_contexts": len(ecm.contexts),
		"by_rule":        make(map[string]int),
		"by_status":      make(map[string]int),
	}

	byRule := stats["by_rule"].(map[string]int)
	byStatus := stats["by_status"].(map[string]int)

	for _, ctx := range ecm.contexts {
		byRule[ctx.RuleID]++

		if ctx.IsCancelled() {
			byStatus["cancelled"]++
		} else {
			byStatus["active"]++
		}
	}

	return stats
}
