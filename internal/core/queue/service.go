package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/sqlite"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/sirupsen/logrus"
)

type QueueService struct {
	queueRepo *sqlite.QueueRepository
	wsHub     *websocket.Hub
	logger    *logrus.Logger
	processor *QueueProcessor
}

func NewQueueService(queueRepo *sqlite.QueueRepository, wsHub *websocket.Hub, logger *logrus.Logger) *QueueService {
	service := &QueueService{
		queueRepo: queueRepo,
		wsHub:     wsHub,
		logger:    logger,
	}

	// Initialize processor
	service.processor = NewQueueProcessor(queueRepo, service, logger)

	return service
}

// Start initializes and starts the queue processing workers
func (s *QueueService) Start(ctx context.Context) error {
	s.logger.Info("Starting queue service")

	// Start the queue processor
	if err := s.processor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start queue processor: %w", err)
	}

	s.logger.Info("Queue service started successfully")
	return nil
}

// Stop gracefully shuts down the queue service
func (s *QueueService) Stop(ctx context.Context) error {
	s.logger.Info("Stopping queue service")

	// Stop the queue processor
	if err := s.processor.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop queue processor: %w", err)
	}

	s.logger.Info("Queue service stopped successfully")
	return nil
}

// EnqueueAction adds a new action to the queue
func (s *QueueService) EnqueueAction(ctx context.Context, req *models.CreateActionRequest, userID *int) (*models.QueuedAction, error) {
	// Validate action type
	actionType, err := s.queueRepo.GetActionTypeByName(ctx, req.ActionType)
	if err != nil {
		return nil, fmt.Errorf("failed to get action type: %w", err)
	}
	if actionType == nil {
		return nil, fmt.Errorf("action type '%s' not found", req.ActionType)
	}
	if !actionType.Enabled {
		return nil, fmt.Errorf("action type '%s' is disabled", req.ActionType)
	}

	// Get priority ID
	priorityName := req.Priority
	if priorityName == "" {
		priorityName = "normal"
	}
	priority, err := s.queueRepo.GetActionPriorityByName(ctx, priorityName)
	if err != nil {
		return nil, fmt.Errorf("failed to get priority: %w", err)
	}
	if priority == nil {
		return nil, fmt.Errorf("priority '%s' not found", priorityName)
	}

	// Get pending status ID
	status, err := s.queueRepo.GetActionStatusByName(ctx, "pending")
	if err != nil {
		return nil, fmt.Errorf("failed to get pending status: %w", err)
	}

	// Validate action data
	if err := s.validateActionData(req.ActionType, req.ActionData); err != nil {
		return nil, fmt.Errorf("invalid action data: %w", err)
	}

	// Create the queued action
	action := &models.QueuedAction{
		ActionTypeID:       actionType.ID,
		PriorityID:         priority.ID,
		StatusID:           status.ID,
		Name:               req.Name,
		ActionData:         string(req.ActionData),
		RetryBackoffFactor: actionType.RetryBackoffFactor,
	}

	// Set optional fields
	if req.Description != "" {
		action.Description.String = req.Description
		action.Description.Valid = true
	}

	if userID != nil {
		action.UserID.Int64 = int64(*userID)
		action.UserID.Valid = true
		action.CreatedBy.String = fmt.Sprintf("user_%d", *userID)
		action.CreatedBy.Valid = true
	} else {
		action.CreatedBy.String = "system"
		action.CreatedBy.Valid = true
	}

	if req.CorrelationID != "" {
		action.CorrelationID.String = req.CorrelationID
		action.CorrelationID.Valid = true
	}

	if req.ParentActionID != nil {
		action.ParentActionID.Int64 = int64(*req.ParentActionID)
		action.ParentActionID.Valid = true
	}

	if req.TargetEntityID != "" {
		action.TargetEntityID.String = req.TargetEntityID
		action.TargetEntityID.Valid = true
	}

	if req.TimeoutSeconds != nil {
		action.TimeoutSeconds.Int64 = int64(*req.TimeoutSeconds)
		action.TimeoutSeconds.Valid = true
	} else {
		action.TimeoutSeconds.Int64 = int64(actionType.DefaultTimeout)
		action.TimeoutSeconds.Valid = true
	}

	if req.MaxRetries != nil {
		action.MaxRetries.Int64 = int64(*req.MaxRetries)
		action.MaxRetries.Valid = true
	} else {
		action.MaxRetries.Int64 = int64(actionType.MaxRetries)
		action.MaxRetries.Valid = true
	}

	if req.ScheduledAt != nil {
		action.ScheduledAt.Time = *req.ScheduledAt
		action.ScheduledAt.Valid = true
	}

	if req.ExecuteAfter != nil {
		action.ExecuteAfter.Time = *req.ExecuteAfter
		action.ExecuteAfter.Valid = true
	}

	if req.Deadline != nil {
		action.Deadline.Time = *req.Deadline
		action.Deadline.Valid = true
	}

	// Create the action in database
	if err := s.queueRepo.CreateQueuedAction(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to create queued action: %w", err)
	}

	// Create dependencies if specified
	for _, depReq := range req.Dependencies {
		dependency := &models.ActionDependency{
			ActionID:          action.ID,
			DependsOnActionID: depReq.DependsOnActionID,
			DependencyType:    depReq.DependencyType,
		}
		if dependency.DependencyType == "" {
			dependency.DependencyType = "completion"
		}

		if err := s.queueRepo.CreateActionDependency(ctx, dependency); err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"action_id":            action.ID,
				"depends_on_action_id": depReq.DependsOnActionID,
			}).Error("Failed to create action dependency")
			// Continue - dependency creation failure shouldn't fail the entire action
		}
	}

	// Get the full action with joined data
	fullAction, err := s.queueRepo.GetQueuedAction(ctx, action.ID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get created action")
		// Return the basic action if we can't get the full one
		fullAction = action
	}

	// Send WebSocket notification
	s.sendWebSocketNotification("action_queued", map[string]interface{}{
		"action": fullAction,
	})

	s.logger.WithFields(logrus.Fields{
		"action_id":   action.ID,
		"action_type": req.ActionType,
		"name":        req.Name,
		"priority":    priorityName,
	}).Info("Action queued successfully")

	return fullAction, nil
}

// EnqueueBulkActions creates multiple actions at once
func (s *QueueService) EnqueueBulkActions(ctx context.Context, req *models.BulkActionRequest, userID *int) ([]*models.QueuedAction, error) {
	var actions []*models.QueuedAction
	correlationID := req.Options.CorrelationID

	// Generate correlation ID if not provided
	if correlationID == "" {
		correlationID = fmt.Sprintf("bulk_%d", time.Now().UnixNano())
	}

	for i, actionReq := range req.Actions {
		// Apply bulk options
		if req.Options.Priority != "" {
			actionReq.Priority = req.Options.Priority
		}
		if actionReq.CorrelationID == "" {
			actionReq.CorrelationID = correlationID
		}

		// For sequential actions, make each depend on the previous
		if req.Options.Sequential && i > 0 {
			prevAction := actions[i-1]
			actionReq.Dependencies = append(actionReq.Dependencies, models.ActionDependencyRequest{
				DependsOnActionID: prevAction.ID,
				DependencyType:    "completion",
			})
		}

		action, err := s.EnqueueAction(ctx, &actionReq, userID)
		if err != nil {
			if req.Options.StopOnError {
				return actions, fmt.Errorf("failed to create action %d: %w", i, err)
			}
			s.logger.WithError(err).WithField("action_index", i).Error("Failed to create bulk action")
			continue
		}

		actions = append(actions, action)
	}

	s.logger.WithFields(logrus.Fields{
		"total_actions":   len(req.Actions),
		"created_actions": len(actions),
		"correlation_id":  correlationID,
		"sequential":      req.Options.Sequential,
	}).Info("Bulk actions queued")

	return actions, nil
}

// GetAction retrieves a single queued action
func (s *QueueService) GetAction(ctx context.Context, id int) (*models.QueuedAction, error) {
	action, err := s.queueRepo.GetQueuedAction(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}
	if action == nil {
		return nil, fmt.Errorf("action with ID %d not found", id)
	}

	// Load dependencies
	dependencies, err := s.queueRepo.GetActionDependencies(ctx, id)
	if err != nil {
		s.logger.WithError(err).WithField("action_id", id).Error("Failed to load dependencies")
	} else {
		// Convert slice of pointers to slice of values
		depValues := make([]models.ActionDependency, len(dependencies))
		for i, dep := range dependencies {
			depValues[i] = *dep
		}
		action.Dependencies = depValues
	}

	// Load results
	results, err := s.queueRepo.GetActionResults(ctx, id)
	if err != nil {
		s.logger.WithError(err).WithField("action_id", id).Error("Failed to load results")
	} else {
		// Convert slice of pointers to slice of values
		resultValues := make([]models.ActionResult, len(results))
		for i, result := range results {
			resultValues[i] = *result
		}
		action.Results = resultValues
	}

	return action, nil
}

// GetActions retrieves queued actions with filtering
func (s *QueueService) GetActions(ctx context.Context, filter *models.QueueFilter) ([]*models.QueuedAction, error) {
	return s.queueRepo.GetQueuedActions(ctx, filter)
}

// UpdateAction updates a queued action
func (s *QueueService) UpdateAction(ctx context.Context, id int, req *models.UpdateActionRequest) (*models.QueuedAction, error) {
	// Get existing action
	action, err := s.queueRepo.GetQueuedAction(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get action: %w", err)
	}
	if action == nil {
		return nil, fmt.Errorf("action with ID %d not found", id)
	}

	// Check if action can be updated
	if action.IsTerminal() {
		return nil, fmt.Errorf("cannot update terminal action (status: %s)", action.Status.Name)
	}

	// Apply updates
	if req.Name != nil {
		action.Name = *req.Name
	}

	if req.Description != nil {
		action.Description.String = *req.Description
		action.Description.Valid = true
	}

	if req.Priority != nil {
		priority, err := s.queueRepo.GetActionPriorityByName(ctx, *req.Priority)
		if err != nil {
			return nil, fmt.Errorf("failed to get priority: %w", err)
		}
		if priority == nil {
			return nil, fmt.Errorf("priority '%s' not found", *req.Priority)
		}
		action.PriorityID = priority.ID
	}

	if req.Status != nil {
		status, err := s.queueRepo.GetActionStatusByName(ctx, *req.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to get status: %w", err)
		}
		if status == nil {
			return nil, fmt.Errorf("status '%s' not found", *req.Status)
		}
		action.StatusID = status.ID
	}

	if req.ScheduledAt != nil {
		action.ScheduledAt.Time = *req.ScheduledAt
		action.ScheduledAt.Valid = true
	}

	if req.ExecuteAfter != nil {
		action.ExecuteAfter.Time = *req.ExecuteAfter
		action.ExecuteAfter.Valid = true
	}

	if req.Deadline != nil {
		action.Deadline.Time = *req.Deadline
		action.Deadline.Valid = true
	}

	// Update in database
	if err := s.queueRepo.UpdateQueuedAction(ctx, action); err != nil {
		return nil, fmt.Errorf("failed to update action: %w", err)
	}

	// Get updated action with fresh joined data
	updatedAction, err := s.queueRepo.GetQueuedAction(ctx, id)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get updated action")
		updatedAction = action
	}

	// Send WebSocket notification
	s.sendWebSocketNotification("action_updated", map[string]interface{}{
		"action": updatedAction,
	})

	s.logger.WithFields(logrus.Fields{
		"action_id": id,
		"name":      updatedAction.Name,
		"status":    updatedAction.Status.Name,
	}).Info("Action updated successfully")

	return updatedAction, nil
}

// CancelAction cancels a queued action
func (s *QueueService) CancelAction(ctx context.Context, id int) error {
	// Get the action
	action, err := s.queueRepo.GetQueuedAction(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get action: %w", err)
	}
	if action == nil {
		return fmt.Errorf("action with ID %d not found", id)
	}

	// Check if action can be cancelled
	if action.IsTerminal() {
		return fmt.Errorf("cannot cancel terminal action (status: %s)", action.Status.Name)
	}

	// Update status to cancelled
	if err := s.queueRepo.UpdateActionStatus(ctx, id, "cancelled", nil, nil); err != nil {
		return fmt.Errorf("failed to cancel action: %w", err)
	}

	// Send WebSocket notification
	s.sendWebSocketNotification("action_cancelled", map[string]interface{}{
		"action_id": id,
	})

	s.logger.WithField("action_id", id).Info("Action cancelled")

	return nil
}

// DeleteAction removes a queued action
func (s *QueueService) DeleteAction(ctx context.Context, id int) error {
	// Get the action first to check if it can be deleted
	action, err := s.queueRepo.GetQueuedAction(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get action: %w", err)
	}
	if action == nil {
		return fmt.Errorf("action with ID %d not found", id)
	}

	// Only allow deletion of terminal actions or pending actions
	if !action.IsTerminal() && !action.IsPending() {
		return fmt.Errorf("cannot delete non-terminal action (status: %s)", action.Status.Name)
	}

	if err := s.queueRepo.DeleteQueuedAction(ctx, id); err != nil {
		return fmt.Errorf("failed to delete action: %w", err)
	}

	// Send WebSocket notification
	s.sendWebSocketNotification("action_deleted", map[string]interface{}{
		"action_id": id,
	})

	s.logger.WithField("action_id", id).Info("Action deleted")

	return nil
}

// ProcessQueue manually triggers queue processing
func (s *QueueService) ProcessQueue(ctx context.Context, req *models.QueueProcessRequest) (int, error) {
	return s.processor.ProcessManually(ctx, req)
}

// ClearQueue removes actions from the queue based on criteria
func (s *QueueService) ClearQueue(ctx context.Context, req *models.QueueClearRequest) (int, error) {
	if !req.ConfirmClear {
		return 0, fmt.Errorf("clear operation must be confirmed")
	}

	deletedCount, err := s.queueRepo.ClearQueue(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("failed to clear queue: %w", err)
	}

	// Send WebSocket notification
	s.sendWebSocketNotification("queue_cleared", map[string]interface{}{
		"deleted_count": deletedCount,
		"criteria":      req,
	})

	s.logger.WithFields(logrus.Fields{
		"deleted_count": deletedCount,
		"status":        req.Status,
		"action_type":   req.ActionType,
	}).Info("Queue cleared")

	return deletedCount, nil
}

// GetStatistics returns queue statistics and health information
func (s *QueueService) GetStatistics(ctx context.Context) (*models.QueueStatistics, error) {
	stats, err := s.queueRepo.GetQueueStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	// Add worker status from processor
	if s.processor != nil {
		stats.WorkerStatus = s.processor.GetWorkerStatus()
	}

	return stats, nil
}

// GetActionTypes returns available action types
func (s *QueueService) GetActionTypes(ctx context.Context) ([]*models.ActionType, error) {
	return s.queueRepo.GetActionTypes(ctx)
}

// GetSettings returns queue configuration settings
func (s *QueueService) GetSettings(ctx context.Context) ([]*models.QueueSetting, error) {
	return s.queueRepo.GetQueueSettings(ctx)
}

// UpdateSetting updates a queue configuration setting
func (s *QueueService) UpdateSetting(ctx context.Context, key, value string) error {
	if err := s.queueRepo.UpdateQueueSetting(ctx, key, value); err != nil {
		return fmt.Errorf("failed to update setting: %w", err)
	}

	// Notify processor if it's a processing-related setting
	if s.processor != nil {
		s.processor.OnSettingChanged(key, value)
	}

	s.logger.WithFields(logrus.Fields{
		"key":   key,
		"value": value,
	}).Info("Queue setting updated")

	return nil
}

// CleanupOldActions removes old completed/failed actions
func (s *QueueService) CleanupOldActions(ctx context.Context) (int, error) {
	// Get retention settings
	completedRetentionSetting, err := s.queueRepo.GetQueueSetting(ctx, "completed_action_retention_days")
	if err != nil {
		return 0, fmt.Errorf("failed to get completed action retention setting: %w", err)
	}

	deadLetterRetentionSetting, err := s.queueRepo.GetQueueSetting(ctx, "dead_letter_retention_days")
	if err != nil {
		return 0, fmt.Errorf("failed to get dead letter retention setting: %w", err)
	}

	// Default to 30 days for completed, 7 days for failed
	completedRetentionDays := 30
	deadLetterRetentionDays := 7

	if completedRetentionSetting != nil {
		fmt.Sscanf(completedRetentionSetting.Value, "%d", &completedRetentionDays)
	}
	if deadLetterRetentionSetting != nil {
		fmt.Sscanf(deadLetterRetentionSetting.Value, "%d", &deadLetterRetentionDays)
	}

	totalDeleted := 0

	// Cleanup completed actions
	completedCutoff := time.Now().AddDate(0, 0, -completedRetentionDays)
	completedDeleted, err := s.queueRepo.CleanupOldActions(ctx, completedCutoff, []string{"completed"})
	if err != nil {
		s.logger.WithError(err).Error("Failed to cleanup old completed actions")
	} else {
		totalDeleted += completedDeleted
	}

	// Cleanup failed actions
	failedCutoff := time.Now().AddDate(0, 0, -deadLetterRetentionDays)
	failedDeleted, err := s.queueRepo.CleanupOldActions(ctx, failedCutoff, []string{"failed", "cancelled", "timeout"})
	if err != nil {
		s.logger.WithError(err).Error("Failed to cleanup old failed actions")
	} else {
		totalDeleted += failedDeleted
	}

	if totalDeleted > 0 {
		s.logger.WithFields(logrus.Fields{
			"completed_deleted": completedDeleted,
			"failed_deleted":    failedDeleted,
			"total_deleted":     totalDeleted,
		}).Info("Old actions cleaned up")

		// Send WebSocket notification
		s.sendWebSocketNotification("actions_cleaned", map[string]interface{}{
			"deleted_count": totalDeleted,
		})
	}

	return totalDeleted, nil
}

// Helper Methods

func (s *QueueService) validateActionData(actionType string, actionData json.RawMessage) error {
	// Basic JSON validation
	var payload map[string]interface{}
	if err := json.Unmarshal(actionData, &payload); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Type-specific validation
	switch actionType {
	case "entity_state_change":
		if _, ok := payload["entity_id"]; !ok {
			return fmt.Errorf("entity_id is required for entity_state_change")
		}
		if _, ok := payload["state"]; !ok {
			return fmt.Errorf("state is required for entity_state_change")
		}

	case "service_call":
		if _, ok := payload["domain"]; !ok {
			return fmt.Errorf("domain is required for service_call")
		}
		if _, ok := payload["service"]; !ok {
			return fmt.Errorf("service is required for service_call")
		}

	case "scene_activation":
		if _, ok := payload["scene_id"]; !ok {
			return fmt.Errorf("scene_id is required for scene_activation")
		}

	case "notification_send":
		if _, ok := payload["message"]; !ok {
			return fmt.Errorf("message is required for notification_send")
		}
	}

	return nil
}

func (s *QueueService) sendWebSocketNotification(eventType string, data map[string]interface{}) {
	if s.wsHub == nil {
		return
	}

	notification := map[string]interface{}{
		"type":      "queue_event",
		"event":     eventType,
		"data":      data,
		"timestamp": time.Now(),
	}

	s.wsHub.BroadcastToTopic("queue_notifications", "queue_event", notification)
}

// NotifyActionStatusChange is called by the processor when action status changes
func (s *QueueService) NotifyActionStatusChange(action *models.QueuedAction, oldStatus, newStatus string) {
	s.sendWebSocketNotification("action_status_changed", map[string]interface{}{
		"action_id":  action.ID,
		"old_status": oldStatus,
		"new_status": newStatus,
		"action":     action,
	})
}
