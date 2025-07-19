package automation

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ExecutionRequest represents a request to execute a rule
type ExecutionRequest struct {
	RuleID    string
	TriggerID string
	Event     Event
	Context   context.Context
}

// AutomationEngine manages the entire automation system
type AutomationEngine struct {
	// Core components
	rules          map[string]*AutomationRule
	scheduler      *Scheduler
	parser         *RuleParser
	contextManager *ExecutionContextManager

	// External dependencies
	unifiedService *unified.UnifiedEntityService
	wsHub          *websocket.Hub
	logger         *logrus.Logger

	// Execution management
	executionQueue chan *ExecutionRequest
	workers        int
	workerPool     []*worker

	// State management
	mu      sync.RWMutex
	running bool

	// Configuration
	config *EngineConfig

	// Statistics
	stats *EngineStatistics
}

// EngineConfig contains automation engine configuration
type EngineConfig struct {
	Workers              int                   `json:"workers"`
	QueueSize            int                   `json:"queue_size"`
	ExecutionTimeout     time.Duration         `json:"execution_timeout"`
	MaxConcurrentRules   int                   `json:"max_concurrent_rules"`
	EnableCircuitBreaker bool                  `json:"enable_circuit_breaker"`
	CircuitBreakerConfig *CircuitBreakerConfig `json:"circuit_breaker"`
	SchedulerConfig      *SchedulerConfig      `json:"scheduler"`
}

// CircuitBreakerConfig contains circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int           `json:"failure_threshold"`
	ResetTimeout     time.Duration `json:"reset_timeout"`
	MaxRequests      int           `json:"max_requests"`
}

// EngineStatistics contains engine performance statistics
type EngineStatistics struct {
	TotalRules           int64         `json:"total_rules"`
	ActiveRules          int64         `json:"active_rules"`
	TotalExecutions      int64         `json:"total_executions"`
	SuccessfulExecutions int64         `json:"successful_executions"`
	FailedExecutions     int64         `json:"failed_executions"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	QueueLength          int           `json:"queue_length"`
	ActiveWorkers        int           `json:"active_workers"`

	mu sync.RWMutex
}

// worker represents a worker that executes automation rules
type worker struct {
	id     int
	engine *AutomationEngine
	stop   chan bool
	active bool
	mu     sync.RWMutex
}

// NewAutomationEngine creates a new automation engine
func NewAutomationEngine(config *EngineConfig, unifiedService *unified.UnifiedEntityService, wsHub *websocket.Hub, logger *logrus.Logger) (*AutomationEngine, error) {
	if config == nil {
		config = &EngineConfig{
			Workers:              runtime.NumCPU(),
			QueueSize:            1000,
			ExecutionTimeout:     30 * time.Second,
			MaxConcurrentRules:   100,
			EnableCircuitBreaker: true,
			CircuitBreakerConfig: &CircuitBreakerConfig{
				FailureThreshold: 5,
				ResetTimeout:     60 * time.Second,
				MaxRequests:      10,
			},
			SchedulerConfig: &SchedulerConfig{
				Timezone: "UTC",
			},
		}
	}

	// Create scheduler
	scheduler, err := NewScheduler(config.SchedulerConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create scheduler: %v", err)
	}

	engine := &AutomationEngine{
		rules:          make(map[string]*AutomationRule),
		scheduler:      scheduler,
		parser:         NewRuleParser(),
		contextManager: NewExecutionContextManager(logger),
		unifiedService: unifiedService,
		wsHub:          wsHub,
		logger:         logger,
		executionQueue: make(chan *ExecutionRequest, config.QueueSize),
		workers:        config.Workers,
		config:         config,
		stats:          &EngineStatistics{},
	}

	// Create worker pool
	engine.workerPool = make([]*worker, config.Workers)
	for i := 0; i < config.Workers; i++ {
		engine.workerPool[i] = &worker{
			id:     i,
			engine: engine,
			stop:   make(chan bool),
		}
	}

	return engine, nil
}

// Start starts the automation engine
func (ae *AutomationEngine) Start(ctx context.Context) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	if ae.running {
		return fmt.Errorf("automation engine is already running")
	}

	ae.logger.Info("Starting automation engine")

	// Start scheduler
	if err := ae.scheduler.Start(); err != nil {
		return fmt.Errorf("failed to start scheduler: %v", err)
	}

	// Start workers
	for _, worker := range ae.workerPool {
		go worker.start()
	}

	// Start event processor
	go ae.processEvents(ctx)

	// Start context cleanup routine
	go ae.cleanupRoutine(ctx)

	ae.running = true
	ae.logger.WithField("workers", ae.workers).Info("Automation engine started")

	return nil
}

// Stop stops the automation engine
func (ae *AutomationEngine) Stop() error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	if !ae.running {
		return fmt.Errorf("automation engine is not running")
	}

	ae.logger.Info("Stopping automation engine")

	// Stop scheduler
	if err := ae.scheduler.Stop(); err != nil {
		ae.logger.WithError(err).Warn("Error stopping scheduler")
	}

	// Stop workers
	for _, worker := range ae.workerPool {
		worker.stop <- true
	}

	// Wait for workers to finish
	time.Sleep(100 * time.Millisecond)

	ae.running = false
	ae.logger.Info("Automation engine stopped")

	return nil
}

// AddRule adds a new automation rule
func (ae *AutomationEngine) AddRule(rule *AutomationRule) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}

	// Validate rule
	if validation := rule.Validate(); !validation.Valid {
		return fmt.Errorf("rule validation failed: %v", validation.Errors)
	}

	// Generate ID if not provided
	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = now
	}
	rule.UpdatedAt = now

	// Add rule to collection
	ae.rules[rule.ID] = rule

	// Set up triggers
	if err := ae.setupTriggers(rule); err != nil {
		delete(ae.rules, rule.ID)
		return fmt.Errorf("failed to setup triggers: %v", err)
	}

	ae.updateStats()
	ae.logger.WithFields(logrus.Fields{
		"rule_id":   rule.ID,
		"rule_name": rule.Name,
	}).Info("Automation rule added")

	return nil
}

// UpdateRule updates an existing automation rule
func (ae *AutomationEngine) UpdateRule(rule *AutomationRule) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	if rule == nil {
		return fmt.Errorf("rule cannot be nil")
	}

	if rule.ID == "" {
		return fmt.Errorf("rule ID is required for update")
	}

	// Check if rule exists
	oldRule, exists := ae.rules[rule.ID]
	if !exists {
		return fmt.Errorf("rule %s not found", rule.ID)
	}

	// Validate updated rule
	if validation := rule.Validate(); !validation.Valid {
		return fmt.Errorf("rule validation failed: %v", validation.Errors)
	}

	// Preserve timestamps
	rule.CreatedAt = oldRule.CreatedAt
	rule.UpdatedAt = time.Now()

	// Clean up old triggers
	ae.cleanupTriggers(oldRule)

	// Cancel existing executions
	ae.contextManager.CancelContextsForRule(rule.ID)

	// Update rule
	ae.rules[rule.ID] = rule

	// Set up new triggers
	if err := ae.setupTriggers(rule); err != nil {
		// Restore old rule on failure
		ae.rules[rule.ID] = oldRule
		ae.setupTriggers(oldRule)
		return fmt.Errorf("failed to setup triggers: %v", err)
	}

	ae.updateStats()
	ae.logger.WithFields(logrus.Fields{
		"rule_id":   rule.ID,
		"rule_name": rule.Name,
	}).Info("Automation rule updated")

	return nil
}

// RemoveRule removes an automation rule
func (ae *AutomationEngine) RemoveRule(ruleID string) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	rule, exists := ae.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	// Clean up triggers
	ae.cleanupTriggers(rule)

	// Cancel existing executions
	ae.contextManager.CancelContextsForRule(ruleID)

	// Remove rule
	delete(ae.rules, ruleID)

	ae.updateStats()
	ae.logger.WithFields(logrus.Fields{
		"rule_id":   ruleID,
		"rule_name": rule.Name,
	}).Info("Automation rule removed")

	return nil
}

// GetRule retrieves a rule by ID
func (ae *AutomationEngine) GetRule(ruleID string) (*AutomationRule, error) {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	rule, exists := ae.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("rule %s not found", ruleID)
	}

	return rule.Clone(), nil
}

// GetAllRules returns all rules
func (ae *AutomationEngine) GetAllRules() []*AutomationRule {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	rules := make([]*AutomationRule, 0, len(ae.rules))
	for _, rule := range ae.rules {
		rules = append(rules, rule.Clone())
	}

	return rules
}

// EnableRule enables a rule
func (ae *AutomationEngine) EnableRule(ruleID string) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	rule, exists := ae.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	if rule.Enabled {
		return nil // Already enabled
	}

	rule.Enabled = true
	rule.UpdatedAt = time.Now()

	// Set up triggers
	if err := ae.setupTriggers(rule); err != nil {
		rule.Enabled = false
		return fmt.Errorf("failed to setup triggers: %v", err)
	}

	ae.updateStats()
	ae.logger.WithField("rule_id", ruleID).Info("Rule enabled")

	return nil
}

// DisableRule disables a rule
func (ae *AutomationEngine) DisableRule(ruleID string) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	rule, exists := ae.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	if !rule.Enabled {
		return nil // Already disabled
	}

	rule.Enabled = false
	rule.UpdatedAt = time.Now()

	// Clean up triggers
	ae.cleanupTriggers(rule)

	// Cancel existing executions
	ae.contextManager.CancelContextsForRule(ruleID)

	ae.updateStats()
	ae.logger.WithField("rule_id", ruleID).Info("Rule disabled")

	return nil
}

// TestRule executes a rule manually for testing
func (ae *AutomationEngine) TestRule(ruleID string, testData map[string]interface{}) (*ExecutionContext, error) {
	ae.mu.RLock()
	rule, exists := ae.rules[ruleID]
	ae.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("rule %s not found", ruleID)
	}

	// Create test event
	event := Event{
		Type:      "manual_test",
		Source:    "automation_engine",
		Data:      testData,
		Timestamp: time.Now(),
	}

	// Create execution context
	ctx := ae.contextManager.CreateContext(context.Background(), ruleID, "manual_test")

	// Execute rule
	err := ae.executeRule(ctx, rule, event)

	return ctx, err
}

// GetStatistics returns engine statistics
func (ae *AutomationEngine) GetStatistics() *EngineStatistics {
	ae.stats.mu.RLock()
	defer ae.stats.mu.RUnlock()

	// Create a copy without copying the mutex
	stats := EngineStatistics{
		TotalRules:           ae.stats.TotalRules,
		ActiveRules:          ae.stats.ActiveRules,
		TotalExecutions:      ae.stats.TotalExecutions,
		SuccessfulExecutions: ae.stats.SuccessfulExecutions,
		FailedExecutions:     ae.stats.FailedExecutions,
		AverageExecutionTime: ae.stats.AverageExecutionTime,
		QueueLength:          len(ae.executionQueue),
		ActiveWorkers:        0, // Will be calculated below
	}

	// Count active workers
	activeWorkers := 0
	for _, worker := range ae.workerPool {
		worker.mu.RLock()
		if worker.active {
			activeWorkers++
		}
		worker.mu.RUnlock()
	}
	stats.ActiveWorkers = activeWorkers

	return &stats
}

// setupTriggers sets up triggers for a rule
func (ae *AutomationEngine) setupTriggers(rule *AutomationRule) error {
	if !rule.Enabled {
		return nil
	}

	for _, trigger := range rule.Triggers {
		// Set up trigger handler
		handler := func(ctx context.Context, t Trigger, event Event) error {
			// Create execution request
			request := &ExecutionRequest{
				RuleID:    rule.ID,
				TriggerID: t.GetID(),
				Event:     event,
				Context:   ctx,
			}

			// Queue for execution
			select {
			case ae.executionQueue <- request:
				return nil
			default:
				return fmt.Errorf("execution queue full")
			}
		}

		// Subscribe trigger
		if err := trigger.Subscribe(handler); err != nil {
			return fmt.Errorf("failed to subscribe trigger %s: %v", trigger.GetID(), err)
		}

		// Schedule time-based triggers
		if timeTrigger, ok := trigger.(*TimeTrigger); ok {
			if err := ae.scheduler.ScheduleTrigger(rule.ID, timeTrigger, handler); err != nil {
				return fmt.Errorf("failed to schedule trigger %s: %v", trigger.GetID(), err)
			}
		}
	}

	return nil
}

// cleanupTriggers cleans up triggers for a rule
func (ae *AutomationEngine) cleanupTriggers(rule *AutomationRule) {
	// Unsubscribe all triggers
	for _, trigger := range rule.Triggers {
		trigger.Unsubscribe()
	}

	// Unschedule time-based triggers
	ae.scheduler.UnscheduleTriggersForRule(rule.ID)
}

// processEvents processes incoming events
func (ae *AutomationEngine) processEvents(ctx context.Context) {
	// Listen to scheduler events
	go func() {
		for event := range ae.scheduler.GetEventChannel() {
			ae.handleEvent(event)
		}
	}()

	// Listen to WebSocket events (Home Assistant, etc.)
	// This would be implemented based on the WebSocket hub
}

// handleEvent handles an incoming event
func (ae *AutomationEngine) handleEvent(event Event) {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	// Process event for all rules
	for _, rule := range ae.rules {
		if !rule.Enabled {
			continue
		}

		// Check if any trigger matches this event
		for _, trigger := range rule.Triggers {
			go func(r *AutomationRule, t Trigger) {
				ctx := context.Background()
				matches, _, err := t.Evaluate(ctx, event)
				if err != nil {
					ae.logger.WithError(err).WithFields(logrus.Fields{
						"rule_id":    r.ID,
						"trigger_id": t.GetID(),
					}).Error("Trigger evaluation failed")
					return
				}

				if matches {
					request := &ExecutionRequest{
						RuleID:    r.ID,
						TriggerID: t.GetID(),
						Event:     event,
						Context:   ctx,
					}

					select {
					case ae.executionQueue <- request:
					default:
						ae.logger.Warn("Execution queue full, dropping request")
					}
				}
			}(rule, trigger)
		}
	}
}

// executeRule executes a rule
func (ae *AutomationEngine) executeRule(execCtx *ExecutionContext, rule *AutomationRule, event Event) error {
	start := time.Now()

	ae.logger.WithFields(logrus.Fields{
		"rule_id":      rule.ID,
		"rule_name":    rule.Name,
		"execution_id": execCtx.ID,
	}).Info("Executing automation rule")

	// Update statistics
	ae.stats.mu.Lock()
	ae.stats.TotalExecutions++
	ae.stats.mu.Unlock()

	// Set up execution context variables
	for k, v := range rule.Variables {
		execCtx.SetVariable(k, v)
	}

	// Add event data to context
	execCtx.SetVariable("trigger_event", event.Data)
	execCtx.SetVariable("trigger_type", event.Type)
	execCtx.SetVariable("trigger_source", event.Source)

	// Check if rule can execute
	if !rule.CanExecute() {
		return fmt.Errorf("rule cannot execute in current state")
	}

	// Update rule status
	rule.Status = RuleStatusRunning

	defer func() {
		rule.Status = RuleStatusIdle
		rule.LastRun = &start
		rule.RunCount++

		duration := time.Since(start)

		// Update average execution time
		ae.stats.mu.Lock()
		if ae.stats.AverageExecutionTime == 0 {
			ae.stats.AverageExecutionTime = duration
		} else {
			ae.stats.AverageExecutionTime = (ae.stats.AverageExecutionTime + duration) / 2
		}
		ae.stats.mu.Unlock()
	}()

	// Evaluate conditions
	for i, condition := range rule.Conditions {
		condStart := time.Now()

		result, err := condition.Evaluate(execCtx.Context(), execCtx.GetAllVariables())
		condDuration := time.Since(condStart)

		execCtx.AddTrace("condition", condition.GetID(), fmt.Sprintf("condition_%d", i), result && err == nil, condDuration, err, nil)

		if err != nil {
			ae.stats.mu.Lock()
			ae.stats.FailedExecutions++
			ae.stats.mu.Unlock()
			return fmt.Errorf("condition %d failed: %v", i, err)
		}

		if !result {
			ae.logger.WithFields(logrus.Fields{
				"rule_id":      rule.ID,
				"condition_id": condition.GetID(),
			}).Debug("Rule condition not met")
			return nil // Conditions not met, but not an error
		}
	}

	// Execute actions
	for i, action := range rule.Actions {
		actionStart := time.Now()

		err := action.Execute(execCtx.Context(), execCtx.GetAllVariables())
		actionDuration := time.Since(actionStart)

		execCtx.AddTrace("action", action.GetID(), fmt.Sprintf("action_%d", i), err == nil, actionDuration, err, nil)

		if err != nil {
			ae.stats.mu.Lock()
			ae.stats.FailedExecutions++
			ae.stats.mu.Unlock()
			return fmt.Errorf("action %d failed: %v", i, err)
		}
	}

	// Update success statistics
	ae.stats.mu.Lock()
	ae.stats.SuccessfulExecutions++
	ae.stats.mu.Unlock()

	ae.logger.WithFields(logrus.Fields{
		"rule_id":      rule.ID,
		"execution_id": execCtx.ID,
		"duration":     time.Since(start),
	}).Info("Rule execution completed successfully")

	return nil
}

// updateStats updates engine statistics
func (ae *AutomationEngine) updateStats() {
	ae.stats.mu.Lock()
	defer ae.stats.mu.Unlock()

	ae.stats.TotalRules = int64(len(ae.rules))

	activeRules := int64(0)
	for _, rule := range ae.rules {
		if rule.Enabled {
			activeRules++
		}
	}
	ae.stats.ActiveRules = activeRules
}

// cleanupRoutine performs periodic cleanup tasks
func (ae *AutomationEngine) cleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ae.contextManager.CleanupExpiredContexts(30 * time.Minute)
		}
	}
}

// worker methods
func (w *worker) start() {
	w.engine.logger.WithField("worker_id", w.id).Debug("Worker started")

	for {
		select {
		case <-w.stop:
			w.engine.logger.WithField("worker_id", w.id).Debug("Worker stopped")
			return
		case request := <-w.engine.executionQueue:
			w.processRequest(request)
		}
	}
}

func (w *worker) processRequest(request *ExecutionRequest) {
	w.mu.Lock()
	w.active = true
	w.mu.Unlock()

	defer func() {
		w.mu.Lock()
		w.active = false
		w.mu.Unlock()
	}()

	// Get rule
	w.engine.mu.RLock()
	rule, exists := w.engine.rules[request.RuleID]
	w.engine.mu.RUnlock()

	if !exists {
		w.engine.logger.WithField("rule_id", request.RuleID).Warn("Rule not found for execution")
		return
	}

	// Create execution context
	execCtx := w.engine.contextManager.CreateContext(request.Context, request.RuleID, request.TriggerID)
	defer w.engine.contextManager.RemoveContext(execCtx.ID)

	// Execute rule with timeout
	ctx, cancel := context.WithTimeout(request.Context, w.engine.config.ExecutionTimeout)
	defer cancel()

	execCtx = w.engine.contextManager.CreateContext(ctx, request.RuleID, request.TriggerID)

	if err := w.engine.executeRule(execCtx, rule, request.Event); err != nil {
		w.engine.logger.WithError(err).WithFields(logrus.Fields{
			"rule_id":      request.RuleID,
			"trigger_id":   request.TriggerID,
			"execution_id": execCtx.ID,
		}).Error("Rule execution failed")
	}
}
