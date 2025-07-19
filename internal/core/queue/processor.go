package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/sqlite"
	"github.com/sirupsen/logrus"
)

type QueueProcessor struct {
	queueRepo   *sqlite.QueueRepository
	service     QueueServiceInterface
	logger      *logrus.Logger
	workers     []*QueueWorker
	workerCount int
	stopChan    chan bool
	wg          sync.WaitGroup
	mu          sync.RWMutex
	running     bool
	handlers    map[string]ActionHandler

	// Settings that can be updated dynamically
	pollInterval    time.Duration
	maxConcurrent   int
	cleanupInterval time.Duration
}

// QueueServiceInterface defines the interface for the queue service
type QueueServiceInterface interface {
	NotifyActionStatusChange(action *models.QueuedAction, oldStatus, newStatus string)
}

// ActionHandler defines the interface for handling different action types
type ActionHandler interface {
	Execute(ctx context.Context, action *models.QueuedAction) (*ActionExecutionResult, error)
	GetHandlerName() string
	GetTimeout() time.Duration
}

// ActionExecutionResult represents the result of executing an action
type ActionExecutionResult struct {
	Success      bool                   `json:"success"`
	Data         map[string]interface{} `json:"data,omitempty"`
	ErrorCode    string                 `json:"error_code,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	ErrorDetails map[string]interface{} `json:"error_details,omitempty"`
	ShouldRetry  bool                   `json:"should_retry"`
}

// QueueWorker represents a single worker processing actions
type QueueWorker struct {
	ID              string
	processor       *QueueProcessor
	logger          *logrus.Entry
	status          string
	currentActionID *int
	lastActivity    time.Time
	processedCount  int
	errorCount      int
	stopChan        chan bool
}

func NewQueueProcessor(queueRepo *sqlite.QueueRepository, service QueueServiceInterface, logger *logrus.Logger) *QueueProcessor {
	processor := &QueueProcessor{
		queueRepo:       queueRepo,
		service:         service,
		logger:          logger,
		workerCount:     5, // Default worker count
		stopChan:        make(chan bool),
		handlers:        make(map[string]ActionHandler),
		pollInterval:    time.Second * 1,  // Default 1 second
		maxConcurrent:   5,                // Default 5 concurrent workers
		cleanupInterval: time.Minute * 10, // Default 10 minute cleanup interval
	}

	// Register default action handlers
	processor.registerDefaultHandlers()

	return processor
}

func (p *QueueProcessor) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return fmt.Errorf("processor is already running")
	}

	p.logger.Info("Starting queue processor")

	// Load settings from database
	if err := p.loadSettings(ctx); err != nil {
		p.logger.WithError(err).Error("Failed to load settings, using defaults")
	}

	// Start workers
	p.workers = make([]*QueueWorker, p.workerCount)
	for i := 0; i < p.workerCount; i++ {
		workerID := fmt.Sprintf("worker_%d", i+1)
		worker := &QueueWorker{
			ID:           workerID,
			processor:    p,
			logger:       p.logger.WithField("worker_id", workerID),
			status:       "idle",
			lastActivity: time.Now(),
			stopChan:     make(chan bool),
		}
		p.workers[i] = worker

		p.wg.Add(1)
		go worker.run(ctx)
	}

	// Start cleanup goroutine
	p.wg.Add(1)
	go p.runCleanup(ctx)

	p.running = true
	p.logger.WithField("worker_count", p.workerCount).Info("Queue processor started")

	return nil
}

func (p *QueueProcessor) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return nil
	}

	p.logger.Info("Stopping queue processor")

	// Signal all workers to stop
	close(p.stopChan)

	// Stop individual workers
	for _, worker := range p.workers {
		close(worker.stopChan)
	}

	// Wait for all workers to finish with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.Info("All workers stopped successfully")
	case <-time.After(30 * time.Second):
		p.logger.Warn("Timeout waiting for workers to stop")
	}

	p.running = false
	p.logger.Info("Queue processor stopped")

	return nil
}

func (p *QueueProcessor) ProcessManually(ctx context.Context, req *models.QueueProcessRequest) (int, error) {
	p.logger.Info("Processing queue manually")

	filter := &models.QueueFilter{
		Status: []string{"pending", "retrying"},
		Limit:  req.MaxActions,
	}

	if req.Priority != "" {
		filter.Priority = []string{req.Priority}
	}

	if req.ActionType != "" {
		filter.ActionType = []string{req.ActionType}
	}

	if len(req.ActionIDs) > 0 {
		// Process specific actions
		processedCount := 0
		for _, actionID := range req.ActionIDs {
			action, err := p.queueRepo.GetQueuedAction(ctx, actionID)
			if err != nil {
				p.logger.WithError(err).WithField("action_id", actionID).Error("Failed to get action for manual processing")
				continue
			}

			if action == nil {
				p.logger.WithField("action_id", actionID).Warn("Action not found for manual processing")
				continue
			}

			// Force retry if requested
			if req.ForceRetry && action.IsFailed() {
				if err := p.queueRepo.UpdateActionStatus(ctx, actionID, "pending", nil, nil); err != nil {
					p.logger.WithError(err).WithField("action_id", actionID).Error("Failed to reset action for retry")
					continue
				}
			}

			if p.processAction(ctx, action) {
				processedCount++
			}
		}
		return processedCount, nil
	}

	// Process actions matching filter
	actions, err := p.queueRepo.GetQueuedActions(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to get actions for processing: %w", err)
	}

	processedCount := 0
	for _, action := range actions {
		if p.processAction(ctx, action) {
			processedCount++
		}
	}

	p.logger.WithField("processed_count", processedCount).Info("Manual processing completed")
	return processedCount, nil
}

func (p *QueueProcessor) GetWorkerStatus() []models.WorkerStatus {
	p.mu.RLock()
	defer p.mu.RUnlock()

	status := make([]models.WorkerStatus, len(p.workers))
	for i, worker := range p.workers {
		status[i] = models.WorkerStatus{
			WorkerID:        worker.ID,
			Status:          worker.status,
			CurrentActionID: worker.currentActionID,
			LastActivity:    worker.lastActivity,
			ProcessedCount:  worker.processedCount,
			ErrorCount:      worker.errorCount,
		}
	}

	return status
}

func (p *QueueProcessor) OnSettingChanged(key, value string) {
	switch key {
	case "max_concurrent_workers":
		var newCount int
		if _, err := fmt.Sscanf(value, "%d", &newCount); err == nil && newCount > 0 && newCount <= 20 {
			p.logger.WithFields(logrus.Fields{
				"old_count": p.workerCount,
				"new_count": newCount,
			}).Info("Worker count will be updated on next restart")
			// Note: Changing worker count requires restart for safety
		}
	case "worker_poll_interval_ms":
		var intervalMs int
		if _, err := fmt.Sscanf(value, "%d", &intervalMs); err == nil && intervalMs > 0 {
			p.pollInterval = time.Duration(intervalMs) * time.Millisecond
			p.logger.WithField("poll_interval", p.pollInterval).Info("Poll interval updated")
		}
	}
}

func (p *QueueProcessor) loadSettings(ctx context.Context) error {
	settings, err := p.queueRepo.GetQueueSettings(ctx)
	if err != nil {
		return err
	}

	for _, setting := range settings {
		switch setting.Key {
		case "max_concurrent_workers":
			var count int
			if _, err := fmt.Sscanf(setting.Value, "%d", &count); err == nil && count > 0 && count <= 20 {
				p.workerCount = count
				p.maxConcurrent = count
			}
		case "worker_poll_interval_ms":
			var intervalMs int
			if _, err := fmt.Sscanf(setting.Value, "%d", &intervalMs); err == nil && intervalMs > 0 {
				p.pollInterval = time.Duration(intervalMs) * time.Millisecond
			}
		}
	}

	return nil
}

func (p *QueueProcessor) registerDefaultHandlers() {
	// Register handlers for different action types
	p.handlers["entity_state_change"] = &EntityStateHandler{logger: p.logger}
	p.handlers["service_call"] = &ServiceCallHandler{logger: p.logger}
	p.handlers["scene_activation"] = &SceneHandler{logger: p.logger}
	p.handlers["automation_trigger"] = &AutomationHandler{logger: p.logger}
	p.handlers["system_command"] = &SystemCommandHandler{logger: p.logger}
	p.handlers["script_execution"] = &ScriptHandler{logger: p.logger}
	p.handlers["notification_send"] = &NotificationHandler{logger: p.logger}
	p.handlers["bulk_operation"] = &BulkOperationHandler{logger: p.logger}
}

func (p *QueueProcessor) processAction(ctx context.Context, action *models.QueuedAction) bool {
	// Check if dependencies are met
	if dependenciesMet, err := p.queueRepo.CheckDependenciesMet(ctx, action.ID); err != nil {
		p.logger.WithError(err).WithField("action_id", action.ID).Error("Failed to check dependencies")
		return false
	} else if !dependenciesMet {
		p.logger.WithField("action_id", action.ID).Debug("Action dependencies not met, skipping")
		return false
	}

	// Check if action should execute now
	if !action.ShouldExecuteNow() {
		return false
	}

	// Mark action as processing
	if err := p.queueRepo.MarkActionAsProcessing(ctx, action.ID, "manual"); err != nil {
		p.logger.WithError(err).WithField("action_id", action.ID).Error("Failed to mark action as processing")
		return false
	}

	// Get handler for action type
	handler, exists := p.handlers[action.ActionType.HandlerName]
	if !exists {
		p.logger.WithField("handler_name", action.ActionType.HandlerName).Error("No handler found for action type")
		p.queueRepo.CompleteAction(ctx, action.ID, false, nil,
			stringPtr("No handler found"), stringPtr("Handler not registered"), 0)
		return false
	}

	// Execute the action
	startTime := time.Now()
	result, err := handler.Execute(ctx, action)
	duration := time.Since(startTime)

	if err != nil {
		p.logger.WithError(err).WithField("action_id", action.ID).Error("Action execution failed")

		// Check if we should retry
		if result != nil && result.ShouldRetry && action.CanRetry() {
			p.scheduleRetry(ctx, action, err)
		} else {
			// Mark as failed
			errorMsg := err.Error()
			var errorDetails *string
			if result != nil && result.ErrorDetails != nil {
				if detailsJSON, jsonErr := json.Marshal(result.ErrorDetails); jsonErr == nil {
					errorDetails = stringPtr(string(detailsJSON))
				}
			}
			p.queueRepo.CompleteAction(ctx, action.ID, false, nil,
				&errorMsg, errorDetails, duration.Milliseconds())
		}
		return false
	}

	// Mark as completed
	var resultData *string
	if result.Data != nil {
		if dataJSON, jsonErr := json.Marshal(result.Data); jsonErr == nil {
			resultData = stringPtr(string(dataJSON))
		}
	}

	p.queueRepo.CompleteAction(ctx, action.ID, result.Success, resultData,
		nil, nil, duration.Milliseconds())

	// Notify service of status change
	if p.service != nil {
		p.service.NotifyActionStatusChange(action, "processing", "completed")
	}

	return true
}

func (p *QueueProcessor) scheduleRetry(ctx context.Context, action *models.QueuedAction, execErr error) {
	retryCount := action.RetryCount + 1

	// Calculate backoff time
	backoffMs := float64(1000) * math.Pow(action.RetryBackoffFactor, float64(retryCount-1))

	// Cap at maximum backoff
	maxBackoffMs := float64(300000) // 5 minutes
	if backoffMs > maxBackoffMs {
		backoffMs = maxBackoffMs
	}

	nextRetryAt := time.Now().Add(time.Duration(backoffMs) * time.Millisecond)

	if err := p.queueRepo.ScheduleRetry(ctx, action.ID, nextRetryAt); err != nil {
		p.logger.WithError(err).WithField("action_id", action.ID).Error("Failed to schedule retry")
		return
	}

	// Record the attempt
	result := &models.ActionResult{
		ActionID:      action.ID,
		AttemptNumber: retryCount,
		StatusID:      5, // retrying status
		StartedAt:     time.Now(),
		Success:       false,
	}
	
	// Set error message
	errorMessage := execErr.Error()
	result.ErrorMessage.String = errorMessage
	result.ErrorMessage.Valid = true
	
	// Set worker ID
	result.WorkerID.String = "manual"
	result.WorkerID.Valid = true

	p.queueRepo.CreateActionResult(ctx, result)

	p.logger.WithFields(logrus.Fields{
		"action_id":     action.ID,
		"retry_count":   retryCount,
		"next_retry_at": nextRetryAt,
		"backoff_ms":    backoffMs,
	}).Info("Action scheduled for retry")

	// Notify service of status change
	if p.service != nil {
		p.service.NotifyActionStatusChange(action, "processing", "retrying")
	}
}

func (p *QueueProcessor) runCleanup(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.performCleanup(ctx)
		}
	}
}

func (p *QueueProcessor) performCleanup(ctx context.Context) {
	// This could be expanded to include various cleanup tasks
	// For now, we'll just log that cleanup is running
	p.logger.Debug("Performing periodic cleanup")

	// Cleanup actions past their deadline
	filter := &models.QueueFilter{
		Status: []string{"pending", "retrying"},
	}

	actions, err := p.queueRepo.GetQueuedActions(ctx, filter)
	if err != nil {
		p.logger.WithError(err).Error("Failed to get actions for cleanup")
		return
	}

	timeoutCount := 0

	for _, action := range actions {
		if action.IsOverdue() {
			// Mark as timeout
			if err := p.queueRepo.UpdateActionStatus(ctx, action.ID, "timeout",
				stringPtr("Action exceeded deadline"), nil); err != nil {
				p.logger.WithError(err).WithField("action_id", action.ID).Error("Failed to mark overdue action as timeout")
			} else {
				timeoutCount++
				if p.service != nil {
					p.service.NotifyActionStatusChange(action, action.Status.Name, "timeout")
				}
			}
		}
	}

	if timeoutCount > 0 {
		p.logger.WithField("timeout_count", timeoutCount).Info("Marked overdue actions as timeout")
	}
}

// Worker Methods

func (w *QueueWorker) run(ctx context.Context) {
	defer w.processor.wg.Done()

	w.logger.Info("Queue worker started")
	ticker := time.NewTicker(w.processor.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			w.logger.Info("Queue worker stopping")
			return
		case <-w.processor.stopChan:
			w.logger.Info("Queue worker stopping due to processor shutdown")
			return
		case <-ticker.C:
			w.processNextAction(ctx)
		}
	}
}

func (w *QueueWorker) processNextAction(ctx context.Context) {
	w.lastActivity = time.Now()

	// Get next action to process
	actions, err := w.processor.queueRepo.GetNextActionsToProcess(ctx, 1)
	if err != nil {
		w.logger.WithError(err).Error("Failed to get next action")
		w.errorCount++
		return
	}

	if len(actions) == 0 {
		w.status = "idle"
		w.currentActionID = nil
		return
	}

	action := actions[0]
	w.status = "processing"
	w.currentActionID = &action.ID

	// Process the action
	if w.processor.processAction(ctx, action) {
		w.processedCount++
	} else {
		w.errorCount++
	}

	w.status = "idle"
	w.currentActionID = nil
}

// Helper functions

func stringPtr(s string) *string {
	return &s
}

func stringToNullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
