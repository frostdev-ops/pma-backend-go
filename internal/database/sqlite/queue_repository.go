package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type QueueRepository struct {
	db  *sqlx.DB
	log *logrus.Logger
}

func NewQueueRepository(db *sqlx.DB, log *logrus.Logger) *QueueRepository {
	return &QueueRepository{
		db:  db,
		log: log,
	}
}

// Action Type Management
func (r *QueueRepository) GetActionTypes(ctx context.Context) ([]*models.ActionType, error) {
	query := `SELECT id, name, description, handler_name, default_timeout, max_retries, 
			  retry_backoff_factor, enabled, created_at, updated_at FROM action_types ORDER BY name`

	var actionTypes []*models.ActionType
	err := r.db.SelectContext(ctx, &actionTypes, query)
	if err != nil {
		r.log.WithError(err).Error("Failed to get action types")
		return nil, fmt.Errorf("failed to get action types: %w", err)
	}

	return actionTypes, nil
}

func (r *QueueRepository) GetActionTypeByName(ctx context.Context, name string) (*models.ActionType, error) {
	query := `SELECT id, name, description, handler_name, default_timeout, max_retries, 
			  retry_backoff_factor, enabled, created_at, updated_at FROM action_types WHERE name = ?`

	var actionType models.ActionType
	err := r.db.GetContext(ctx, &actionType, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.log.WithError(err).WithField("name", name).Error("Failed to get action type")
		return nil, fmt.Errorf("failed to get action type: %w", err)
	}

	return &actionType, nil
}

// Priority Management
func (r *QueueRepository) GetActionPriorities(ctx context.Context) ([]*models.ActionPriority, error) {
	query := `SELECT id, name, weight, description FROM action_priorities ORDER BY weight DESC`

	var priorities []*models.ActionPriority
	err := r.db.SelectContext(ctx, &priorities, query)
	if err != nil {
		r.log.WithError(err).Error("Failed to get action priorities")
		return nil, fmt.Errorf("failed to get action priorities: %w", err)
	}

	return priorities, nil
}

func (r *QueueRepository) GetActionPriorityByName(ctx context.Context, name string) (*models.ActionPriority, error) {
	query := `SELECT id, name, weight, description FROM action_priorities WHERE name = ?`

	var priority models.ActionPriority
	err := r.db.GetContext(ctx, &priority, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.log.WithError(err).WithField("name", name).Error("Failed to get action priority")
		return nil, fmt.Errorf("failed to get action priority: %w", err)
	}

	return &priority, nil
}

// Status Management
func (r *QueueRepository) GetActionStatuses(ctx context.Context) ([]*models.ActionStatus, error) {
	query := `SELECT id, name, description, is_terminal FROM action_statuses ORDER BY id`

	var statuses []*models.ActionStatus
	err := r.db.SelectContext(ctx, &statuses, query)
	if err != nil {
		r.log.WithError(err).Error("Failed to get action statuses")
		return nil, fmt.Errorf("failed to get action statuses: %w", err)
	}

	return statuses, nil
}

func (r *QueueRepository) GetActionStatusByName(ctx context.Context, name string) (*models.ActionStatus, error) {
	query := `SELECT id, name, description, is_terminal FROM action_statuses WHERE name = ?`

	var status models.ActionStatus
	err := r.db.GetContext(ctx, &status, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.log.WithError(err).WithField("name", name).Error("Failed to get action status")
		return nil, fmt.Errorf("failed to get action status: %w", err)
	}

	return &status, nil
}

// Queued Action CRUD Operations
func (r *QueueRepository) CreateQueuedAction(ctx context.Context, action *models.QueuedAction) error {
	query := `
		INSERT INTO queued_actions (
			action_type_id, priority_id, status_id, name, description, user_id, 
			correlation_id, parent_action_id, action_data, target_entity_id,
			timeout_seconds, max_retries, retry_count, retry_backoff_factor,
			scheduled_at, execute_after, deadline, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		action.ActionTypeID, action.PriorityID, action.StatusID,
		action.Name, action.Description, action.UserID,
		action.CorrelationID, action.ParentActionID, action.ActionData,
		action.TargetEntityID, action.TimeoutSeconds, action.MaxRetries,
		action.RetryCount, action.RetryBackoffFactor, action.ScheduledAt,
		action.ExecuteAfter, action.Deadline, action.CreatedBy,
	)

	if err != nil {
		r.log.WithError(err).Error("Failed to create queued action")
		return fmt.Errorf("failed to create queued action: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	action.ID = int(id)
	action.CreatedAt = time.Now()
	action.UpdatedAt = time.Now()

	return nil
}

func (r *QueueRepository) GetQueuedAction(ctx context.Context, id int) (*models.QueuedAction, error) {
	query := `
		SELECT 
			qa.id, qa.action_type_id, qa.priority_id, qa.status_id, qa.name, qa.description,
			qa.user_id, qa.correlation_id, qa.parent_action_id, qa.action_data, qa.target_entity_id,
			qa.timeout_seconds, qa.max_retries, qa.retry_count, qa.retry_backoff_factor,
			qa.scheduled_at, qa.execute_after, qa.deadline, qa.started_at, qa.completed_at,
			qa.last_attempt_at, qa.next_retry_at, qa.result_data, qa.error_message, qa.error_details,
			qa.created_at, qa.updated_at, qa.created_by, qa.execution_duration_ms,
			at.name as action_type_name, at.description as action_type_description, at.handler_name,
			ap.name as priority_name, ap.weight as priority_weight,
			ast.name as status_name, ast.description as status_description, ast.is_terminal
		FROM queued_actions qa
		JOIN action_types at ON qa.action_type_id = at.id
		JOIN action_priorities ap ON qa.priority_id = ap.id
		JOIN action_statuses ast ON qa.status_id = ast.id
		WHERE qa.id = ?
	`

	row := r.db.QueryRowxContext(ctx, query, id)

	var action models.QueuedAction
	var actionTypeName, actionTypeDescription, handlerName string
	var priorityName string
	var priorityWeight int
	var statusName, statusDescription string
	var isTerminal bool

	err := row.Scan(
		&action.ID, &action.ActionTypeID, &action.PriorityID, &action.StatusID,
		&action.Name, &action.Description, &action.UserID, &action.CorrelationID,
		&action.ParentActionID, &action.ActionData, &action.TargetEntityID,
		&action.TimeoutSeconds, &action.MaxRetries, &action.RetryCount, &action.RetryBackoffFactor,
		&action.ScheduledAt, &action.ExecuteAfter, &action.Deadline, &action.StartedAt,
		&action.CompletedAt, &action.LastAttemptAt, &action.NextRetryAt, &action.ResultData,
		&action.ErrorMessage, &action.ErrorDetails, &action.CreatedAt, &action.UpdatedAt,
		&action.CreatedBy, &action.ExecutionDurationMS,
		&actionTypeName, &actionTypeDescription, &handlerName,
		&priorityName, &priorityWeight,
		&statusName, &statusDescription, &isTerminal,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.log.WithError(err).WithField("id", id).Error("Failed to get queued action")
		return nil, fmt.Errorf("failed to get queued action: %w", err)
	}

	// Populate joined objects
	action.ActionType = &models.ActionType{
		ID:          action.ActionTypeID,
		Name:        actionTypeName,
		Description: actionTypeDescription,
		HandlerName: handlerName,
	}
	action.Priority = &models.ActionPriority{
		ID:     action.PriorityID,
		Name:   priorityName,
		Weight: priorityWeight,
	}
	action.Status = &models.ActionStatus{
		ID:          action.StatusID,
		Name:        statusName,
		Description: statusDescription,
		IsTerminal:  isTerminal,
	}

	return &action, nil
}

func (r *QueueRepository) GetQueuedActions(ctx context.Context, filter *models.QueueFilter) ([]*models.QueuedAction, error) {
	baseQuery := `
		SELECT 
			qa.id, qa.action_type_id, qa.priority_id, qa.status_id, qa.name, qa.description,
			qa.user_id, qa.correlation_id, qa.parent_action_id, qa.action_data, qa.target_entity_id,
			qa.timeout_seconds, qa.max_retries, qa.retry_count, qa.retry_backoff_factor,
			qa.scheduled_at, qa.execute_after, qa.deadline, qa.started_at, qa.completed_at,
			qa.last_attempt_at, qa.next_retry_at, qa.result_data, qa.error_message, qa.error_details,
			qa.created_at, qa.updated_at, qa.created_by, qa.execution_duration_ms,
			at.name as action_type_name, at.description as action_type_description, at.handler_name,
			ap.name as priority_name, ap.weight as priority_weight,
			ast.name as status_name, ast.description as status_description, ast.is_terminal
		FROM queued_actions qa
		JOIN action_types at ON qa.action_type_id = at.id
		JOIN action_priorities ap ON qa.priority_id = ap.id
		JOIN action_statuses ast ON qa.status_id = ast.id
	`

	conditions := []string{}
	args := []interface{}{}

	if filter != nil {
		if len(filter.Status) > 0 {
			placeholders := strings.Repeat("?,", len(filter.Status))
			placeholders = placeholders[:len(placeholders)-1]
			conditions = append(conditions, fmt.Sprintf("ast.name IN (%s)", placeholders))
			for _, status := range filter.Status {
				args = append(args, status)
			}
		}

		if len(filter.Priority) > 0 {
			placeholders := strings.Repeat("?,", len(filter.Priority))
			placeholders = placeholders[:len(placeholders)-1]
			conditions = append(conditions, fmt.Sprintf("ap.name IN (%s)", placeholders))
			for _, priority := range filter.Priority {
				args = append(args, priority)
			}
		}

		if len(filter.ActionType) > 0 {
			placeholders := strings.Repeat("?,", len(filter.ActionType))
			placeholders = placeholders[:len(placeholders)-1]
			conditions = append(conditions, fmt.Sprintf("at.name IN (%s)", placeholders))
			for _, actionType := range filter.ActionType {
				args = append(args, actionType)
			}
		}

		if filter.UserID != nil {
			conditions = append(conditions, "qa.user_id = ?")
			args = append(args, *filter.UserID)
		}

		if filter.CorrelationID != "" {
			conditions = append(conditions, "qa.correlation_id = ?")
			args = append(args, filter.CorrelationID)
		}

		if filter.TargetEntityID != "" {
			conditions = append(conditions, "qa.target_entity_id = ?")
			args = append(args, filter.TargetEntityID)
		}

		if filter.CreatedAfter != nil {
			conditions = append(conditions, "qa.created_at > ?")
			args = append(args, *filter.CreatedAfter)
		}

		if filter.CreatedBefore != nil {
			conditions = append(conditions, "qa.created_at < ?")
			args = append(args, *filter.CreatedBefore)
		}

		if filter.ScheduledAfter != nil {
			conditions = append(conditions, "qa.scheduled_at > ?")
			args = append(args, *filter.ScheduledAfter)
		}

		if filter.ScheduledBefore != nil {
			conditions = append(conditions, "qa.scheduled_at < ?")
			args = append(args, *filter.ScheduledBefore)
		}
	}

	// Add WHERE clause if we have conditions
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Add ORDER BY clause
	orderBy := "qa.created_at"
	if filter != nil && filter.OrderBy != "" {
		switch filter.OrderBy {
		case "priority":
			orderBy = "ap.weight"
		case "scheduled_at":
			orderBy = "qa.scheduled_at"
		case "updated_at":
			orderBy = "qa.updated_at"
		default:
			orderBy = "qa.created_at"
		}
	}

	orderDirection := "DESC"
	if filter != nil && filter.OrderDirection == "asc" {
		orderDirection = "ASC"
	}

	baseQuery += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDirection)

	// Add LIMIT and OFFSET
	if filter != nil {
		if filter.Limit > 0 {
			baseQuery += " LIMIT ?"
			args = append(args, filter.Limit)
		}
		if filter.Offset > 0 {
			baseQuery += " OFFSET ?"
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.db.QueryxContext(ctx, baseQuery, args...)
	if err != nil {
		r.log.WithError(err).Error("Failed to get queued actions")
		return nil, fmt.Errorf("failed to get queued actions: %w", err)
	}
	defer rows.Close()

	var actions []*models.QueuedAction
	for rows.Next() {
		var action models.QueuedAction
		var actionTypeName, actionTypeDescription, handlerName string
		var priorityName string
		var priorityWeight int
		var statusName, statusDescription string
		var isTerminal bool

		err := rows.Scan(
			&action.ID, &action.ActionTypeID, &action.PriorityID, &action.StatusID,
			&action.Name, &action.Description, &action.UserID, &action.CorrelationID,
			&action.ParentActionID, &action.ActionData, &action.TargetEntityID,
			&action.TimeoutSeconds, &action.MaxRetries, &action.RetryCount, &action.RetryBackoffFactor,
			&action.ScheduledAt, &action.ExecuteAfter, &action.Deadline, &action.StartedAt,
			&action.CompletedAt, &action.LastAttemptAt, &action.NextRetryAt, &action.ResultData,
			&action.ErrorMessage, &action.ErrorDetails, &action.CreatedAt, &action.UpdatedAt,
			&action.CreatedBy, &action.ExecutionDurationMS,
			&actionTypeName, &actionTypeDescription, &handlerName,
			&priorityName, &priorityWeight,
			&statusName, &statusDescription, &isTerminal,
		)

		if err != nil {
			r.log.WithError(err).Error("Failed to scan queued action")
			return nil, fmt.Errorf("failed to scan queued action: %w", err)
		}

		// Populate joined objects
		action.ActionType = &models.ActionType{
			ID:          action.ActionTypeID,
			Name:        actionTypeName,
			Description: actionTypeDescription,
			HandlerName: handlerName,
		}
		action.Priority = &models.ActionPriority{
			ID:     action.PriorityID,
			Name:   priorityName,
			Weight: priorityWeight,
		}
		action.Status = &models.ActionStatus{
			ID:          action.StatusID,
			Name:        statusName,
			Description: statusDescription,
			IsTerminal:  isTerminal,
		}

		actions = append(actions, &action)
	}

	return actions, nil
}

func (r *QueueRepository) UpdateQueuedAction(ctx context.Context, action *models.QueuedAction) error {
	query := `
		UPDATE queued_actions SET
			action_type_id = ?, priority_id = ?, status_id = ?, name = ?, description = ?,
			user_id = ?, correlation_id = ?, parent_action_id = ?, action_data = ?,
			target_entity_id = ?, timeout_seconds = ?, max_retries = ?, retry_count = ?,
			retry_backoff_factor = ?, scheduled_at = ?, execute_after = ?, deadline = ?,
			started_at = ?, completed_at = ?, last_attempt_at = ?, next_retry_at = ?,
			result_data = ?, error_message = ?, error_details = ?, created_by = ?,
			execution_duration_ms = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query,
		action.ActionTypeID, action.PriorityID, action.StatusID,
		action.Name, action.Description, action.UserID, action.CorrelationID,
		action.ParentActionID, action.ActionData, action.TargetEntityID,
		action.TimeoutSeconds, action.MaxRetries, action.RetryCount,
		action.RetryBackoffFactor, action.ScheduledAt, action.ExecuteAfter,
		action.Deadline, action.StartedAt, action.CompletedAt,
		action.LastAttemptAt, action.NextRetryAt, action.ResultData,
		action.ErrorMessage, action.ErrorDetails, action.CreatedBy,
		action.ExecutionDurationMS, action.ID,
	)

	if err != nil {
		r.log.WithError(err).WithField("id", action.ID).Error("Failed to update queued action")
		return fmt.Errorf("failed to update queued action: %w", err)
	}

	return nil
}

func (r *QueueRepository) DeleteQueuedAction(ctx context.Context, id int) error {
	query := `DELETE FROM queued_actions WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.log.WithError(err).WithField("id", id).Error("Failed to delete queued action")
		return fmt.Errorf("failed to delete queued action: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("action with ID %d not found", id)
	}

	return nil
}

// Queue Processing Methods
func (r *QueueRepository) GetNextActionsToProcess(ctx context.Context, limit int) ([]*models.QueuedAction, error) {
	query := `
		SELECT 
			qa.id, qa.action_type_id, qa.priority_id, qa.status_id, qa.name, qa.description,
			qa.user_id, qa.correlation_id, qa.parent_action_id, qa.action_data, qa.target_entity_id,
			qa.timeout_seconds, qa.max_retries, qa.retry_count, qa.retry_backoff_factor,
			qa.scheduled_at, qa.execute_after, qa.deadline, qa.started_at, qa.completed_at,
			qa.last_attempt_at, qa.next_retry_at, qa.result_data, qa.error_message, qa.error_details,
			qa.created_at, qa.updated_at, qa.created_by, qa.execution_duration_ms,
			at.name as action_type_name, at.description as action_type_description, at.handler_name,
			ap.name as priority_name, ap.weight as priority_weight,
			ast.name as status_name, ast.description as status_description, ast.is_terminal
		FROM queued_actions qa
		JOIN action_types at ON qa.action_type_id = at.id
		JOIN action_priorities ap ON qa.priority_id = ap.id
		JOIN action_statuses ast ON qa.status_id = ast.id
		WHERE ast.name IN ('pending', 'retrying')
		  AND (qa.scheduled_at IS NULL OR qa.scheduled_at <= CURRENT_TIMESTAMP)
		  AND (qa.execute_after IS NULL OR qa.execute_after <= CURRENT_TIMESTAMP)
		  AND (qa.next_retry_at IS NULL OR qa.next_retry_at <= CURRENT_TIMESTAMP)
		  AND (qa.deadline IS NULL OR qa.deadline > CURRENT_TIMESTAMP)
		ORDER BY ap.weight DESC, qa.created_at ASC
		LIMIT ?
	`

	rows, err := r.db.QueryxContext(ctx, query, limit)
	if err != nil {
		r.log.WithError(err).Error("Failed to get next actions to process")
		return nil, fmt.Errorf("failed to get next actions to process: %w", err)
	}
	defer rows.Close()

	var actions []*models.QueuedAction
	for rows.Next() {
		var action models.QueuedAction
		var actionTypeName, actionTypeDescription, handlerName string
		var priorityName string
		var priorityWeight int
		var statusName, statusDescription string
		var isTerminal bool

		err := rows.Scan(
			&action.ID, &action.ActionTypeID, &action.PriorityID, &action.StatusID,
			&action.Name, &action.Description, &action.UserID, &action.CorrelationID,
			&action.ParentActionID, &action.ActionData, &action.TargetEntityID,
			&action.TimeoutSeconds, &action.MaxRetries, &action.RetryCount, &action.RetryBackoffFactor,
			&action.ScheduledAt, &action.ExecuteAfter, &action.Deadline, &action.StartedAt,
			&action.CompletedAt, &action.LastAttemptAt, &action.NextRetryAt, &action.ResultData,
			&action.ErrorMessage, &action.ErrorDetails, &action.CreatedAt, &action.UpdatedAt,
			&action.CreatedBy, &action.ExecutionDurationMS,
			&actionTypeName, &actionTypeDescription, &handlerName,
			&priorityName, &priorityWeight,
			&statusName, &statusDescription, &isTerminal,
		)

		if err != nil {
			r.log.WithError(err).Error("Failed to scan next action to process")
			return nil, fmt.Errorf("failed to scan next action to process: %w", err)
		}

		// Populate joined objects
		action.ActionType = &models.ActionType{
			ID:          action.ActionTypeID,
			Name:        actionTypeName,
			Description: actionTypeDescription,
			HandlerName: handlerName,
		}
		action.Priority = &models.ActionPriority{
			ID:     action.PriorityID,
			Name:   priorityName,
			Weight: priorityWeight,
		}
		action.Status = &models.ActionStatus{
			ID:          action.StatusID,
			Name:        statusName,
			Description: statusDescription,
			IsTerminal:  isTerminal,
		}

		actions = append(actions, &action)
	}

	return actions, nil
}

func (r *QueueRepository) UpdateActionStatus(ctx context.Context, actionID int, statusName string, errorMessage, errorDetails *string) error {
	// First get the status ID
	statusQuery := `SELECT id FROM action_statuses WHERE name = ?`
	var statusID int
	err := r.db.GetContext(ctx, &statusID, statusQuery, statusName)
	if err != nil {
		return fmt.Errorf("failed to get status ID for status %s: %w", statusName, err)
	}

	// Update the action
	updateQuery := `
		UPDATE queued_actions SET 
			status_id = ?, 
			error_message = ?, 
			error_details = ?,
			last_attempt_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err = r.db.ExecContext(ctx, updateQuery, statusID, errorMessage, errorDetails, actionID)
	if err != nil {
		return fmt.Errorf("failed to update action status: %w", err)
	}

	return nil
}

func (r *QueueRepository) MarkActionAsProcessing(ctx context.Context, actionID int, workerID string) error {
	// Get processing status ID
	statusQuery := `SELECT id FROM action_statuses WHERE name = 'processing'`
	var statusID int
	err := r.db.GetContext(ctx, &statusID, statusQuery)
	if err != nil {
		return fmt.Errorf("failed to get processing status ID: %w", err)
	}

	// Update action to processing status
	updateQuery := `
		UPDATE queued_actions SET 
			status_id = ?, 
			started_at = CURRENT_TIMESTAMP,
			last_attempt_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status_id IN (SELECT id FROM action_statuses WHERE name IN ('pending', 'retrying'))
	`

	result, err := r.db.ExecContext(ctx, updateQuery, statusID, actionID)
	if err != nil {
		return fmt.Errorf("failed to mark action as processing: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("action %d not found or not in processable state", actionID)
	}

	return nil
}

func (r *QueueRepository) CompleteAction(ctx context.Context, actionID int, success bool, resultData *string, errorMessage, errorDetails *string, durationMS int64) error {
	statusName := "completed"
	if !success {
		statusName = "failed"
	}

	// Get status ID
	statusQuery := `SELECT id FROM action_statuses WHERE name = ?`
	var statusID int
	err := r.db.GetContext(ctx, &statusID, statusQuery, statusName)
	if err != nil {
		return fmt.Errorf("failed to get status ID for %s: %w", statusName, err)
	}

	// Update action
	updateQuery := `
		UPDATE queued_actions SET 
			status_id = ?, 
			completed_at = CURRENT_TIMESTAMP,
			result_data = ?,
			error_message = ?,
			error_details = ?,
			execution_duration_ms = ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err = r.db.ExecContext(ctx, updateQuery, statusID, resultData, errorMessage, errorDetails, durationMS, actionID)
	if err != nil {
		return fmt.Errorf("failed to complete action: %w", err)
	}

	return nil
}

func (r *QueueRepository) ScheduleRetry(ctx context.Context, actionID int, nextRetryAt time.Time) error {
	// Get retrying status ID
	statusQuery := `SELECT id FROM action_statuses WHERE name = 'retrying'`
	var statusID int
	err := r.db.GetContext(ctx, &statusID, statusQuery)
	if err != nil {
		return fmt.Errorf("failed to get retrying status ID: %w", err)
	}

	// Update action for retry
	updateQuery := `
		UPDATE queued_actions SET 
			status_id = ?, 
			retry_count = retry_count + 1,
			next_retry_at = ?,
			last_attempt_at = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err = r.db.ExecContext(ctx, updateQuery, statusID, nextRetryAt, actionID)
	if err != nil {
		return fmt.Errorf("failed to schedule retry: %w", err)
	}

	return nil
}

// Action Result Management
func (r *QueueRepository) CreateActionResult(ctx context.Context, result *models.ActionResult) error {
	query := `
		INSERT INTO action_results (
			action_id, attempt_number, status_id, started_at, completed_at, duration_ms,
			success, result_data, error_code, error_message, error_details,
			worker_id, execution_context
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	execResult, err := r.db.ExecContext(ctx, query,
		result.ActionID, result.AttemptNumber, result.StatusID, result.StartedAt,
		result.CompletedAt, result.DurationMS, result.Success, result.ResultData,
		result.ErrorCode, result.ErrorMessage, result.ErrorDetails, result.WorkerID,
		result.ExecutionContext,
	)

	if err != nil {
		r.log.WithError(err).Error("Failed to create action result")
		return fmt.Errorf("failed to create action result: %w", err)
	}

	id, err := execResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	result.ID = int(id)
	result.CreatedAt = time.Now()

	return nil
}

func (r *QueueRepository) GetActionResults(ctx context.Context, actionID int) ([]*models.ActionResult, error) {
	query := `
		SELECT 
			ar.id, ar.action_id, ar.attempt_number, ar.status_id, ar.started_at, ar.completed_at,
			ar.duration_ms, ar.success, ar.result_data, ar.error_code, ar.error_message,
			ar.error_details, ar.worker_id, ar.execution_context, ar.created_at,
			ast.name as status_name, ast.description as status_description, ast.is_terminal
		FROM action_results ar
		JOIN action_statuses ast ON ar.status_id = ast.id
		WHERE ar.action_id = ?
		ORDER BY ar.attempt_number DESC
	`

	rows, err := r.db.QueryxContext(ctx, query, actionID)
	if err != nil {
		r.log.WithError(err).WithField("action_id", actionID).Error("Failed to get action results")
		return nil, fmt.Errorf("failed to get action results: %w", err)
	}
	defer rows.Close()

	var results []*models.ActionResult
	for rows.Next() {
		var result models.ActionResult
		var statusName, statusDescription string
		var isTerminal bool

		err := rows.Scan(
			&result.ID, &result.ActionID, &result.AttemptNumber, &result.StatusID,
			&result.StartedAt, &result.CompletedAt, &result.DurationMS, &result.Success,
			&result.ResultData, &result.ErrorCode, &result.ErrorMessage, &result.ErrorDetails,
			&result.WorkerID, &result.ExecutionContext, &result.CreatedAt,
			&statusName, &statusDescription, &isTerminal,
		)

		if err != nil {
			r.log.WithError(err).Error("Failed to scan action result")
			return nil, fmt.Errorf("failed to scan action result: %w", err)
		}

		result.Status = &models.ActionStatus{
			ID:          result.StatusID,
			Name:        statusName,
			Description: statusDescription,
			IsTerminal:  isTerminal,
		}

		results = append(results, &result)
	}

	return results, nil
}

// Action Dependencies
func (r *QueueRepository) CreateActionDependency(ctx context.Context, dependency *models.ActionDependency) error {
	query := `
		INSERT INTO action_dependencies (action_id, depends_on_action_id, dependency_type)
		VALUES (?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		dependency.ActionID, dependency.DependsOnActionID, dependency.DependencyType,
	)

	if err != nil {
		r.log.WithError(err).Error("Failed to create action dependency")
		return fmt.Errorf("failed to create action dependency: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	dependency.ID = int(id)
	dependency.CreatedAt = time.Now()

	return nil
}

func (r *QueueRepository) GetActionDependencies(ctx context.Context, actionID int) ([]*models.ActionDependency, error) {
	query := `
		SELECT id, action_id, depends_on_action_id, dependency_type, created_at
		FROM action_dependencies
		WHERE action_id = ?
		ORDER BY created_at
	`

	var dependencies []*models.ActionDependency
	err := r.db.SelectContext(ctx, &dependencies, query, actionID)
	if err != nil {
		r.log.WithError(err).WithField("action_id", actionID).Error("Failed to get action dependencies")
		return nil, fmt.Errorf("failed to get action dependencies: %w", err)
	}

	return dependencies, nil
}

func (r *QueueRepository) CheckDependenciesMet(ctx context.Context, actionID int) (bool, error) {
	query := `
		SELECT COUNT(*) as unmet_count
		FROM action_dependencies ad
		JOIN queued_actions qa ON ad.depends_on_action_id = qa.id
		JOIN action_statuses ast ON qa.status_id = ast.id
		WHERE ad.action_id = ?
		  AND (
		    (ad.dependency_type = 'completion' AND ast.is_terminal = FALSE) OR
		    (ad.dependency_type = 'success' AND ast.name != 'completed') OR
		    (ad.dependency_type = 'failure' AND ast.name NOT IN ('failed', 'timeout', 'cancelled'))
		  )
	`

	var unmetCount int
	err := r.db.GetContext(ctx, &unmetCount, query, actionID)
	if err != nil {
		r.log.WithError(err).WithField("action_id", actionID).Error("Failed to check dependencies")
		return false, fmt.Errorf("failed to check dependencies: %w", err)
	}

	return unmetCount == 0, nil
}

// Queue Settings
func (r *QueueRepository) GetQueueSettings(ctx context.Context) ([]*models.QueueSetting, error) {
	query := `SELECT id, key, value, data_type, description, category, is_readonly, created_at, updated_at 
			  FROM queue_settings ORDER BY category, key`

	var settings []*models.QueueSetting
	err := r.db.SelectContext(ctx, &settings, query)
	if err != nil {
		r.log.WithError(err).Error("Failed to get queue settings")
		return nil, fmt.Errorf("failed to get queue settings: %w", err)
	}

	return settings, nil
}

func (r *QueueRepository) GetQueueSetting(ctx context.Context, key string) (*models.QueueSetting, error) {
	query := `SELECT id, key, value, data_type, description, category, is_readonly, created_at, updated_at 
			  FROM queue_settings WHERE key = ?`

	var setting models.QueueSetting
	err := r.db.GetContext(ctx, &setting, query, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		r.log.WithError(err).WithField("key", key).Error("Failed to get queue setting")
		return nil, fmt.Errorf("failed to get queue setting: %w", err)
	}

	return &setting, nil
}

func (r *QueueRepository) UpdateQueueSetting(ctx context.Context, key, value string) error {
	query := `UPDATE queue_settings SET value = ?, updated_at = CURRENT_TIMESTAMP 
			  WHERE key = ? AND is_readonly = FALSE`

	result, err := r.db.ExecContext(ctx, query, value, key)
	if err != nil {
		r.log.WithError(err).WithFields(logrus.Fields{
			"key":   key,
			"value": value,
		}).Error("Failed to update queue setting")
		return fmt.Errorf("failed to update queue setting: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("setting %s not found or is readonly", key)
	}

	return nil
}

// Statistics and Health
func (r *QueueRepository) GetQueueStatistics(ctx context.Context) (*models.QueueStatistics, error) {
	stats := &models.QueueStatistics{
		ActionsByPriority: make(map[string]int),
		ActionsByType:     make(map[string]int),
		WorkerStatus:      []models.WorkerStatus{},
	}

	// Get total counts by status
	statusQuery := `
		SELECT ast.name, COUNT(*) as count
		FROM queued_actions qa
		JOIN action_statuses ast ON qa.status_id = ast.id
		GROUP BY ast.name
	`

	rows, err := r.db.QueryxContext(ctx, statusQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get status statistics: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan status count: %w", err)
		}

		stats.TotalActions += count
		switch status {
		case "pending":
			stats.PendingActions = count
		case "processing":
			stats.ProcessingActions = count
		case "completed":
			stats.CompletedActions = count
		case "failed":
			stats.FailedActions = count
		case "retrying":
			stats.RetryingActions = count
		case "cancelled":
			stats.CancelledActions = count
		}
	}

	// Get counts by priority
	priorityQuery := `
		SELECT ap.name, COUNT(*) as count
		FROM queued_actions qa
		JOIN action_priorities ap ON qa.priority_id = ap.id
		GROUP BY ap.name
	`

	rows, err = r.db.QueryxContext(ctx, priorityQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get priority statistics: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var priority string
		var count int
		if err := rows.Scan(&priority, &count); err != nil {
			return nil, fmt.Errorf("failed to scan priority count: %w", err)
		}
		stats.ActionsByPriority[priority] = count
	}

	// Get counts by action type
	typeQuery := `
		SELECT at.name, COUNT(*) as count
		FROM queued_actions qa
		JOIN action_types at ON qa.action_type_id = at.id
		GROUP BY at.name
	`

	rows, err = r.db.QueryxContext(ctx, typeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get type statistics: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var actionType string
		var count int
		if err := rows.Scan(&actionType, &count); err != nil {
			return nil, fmt.Errorf("failed to scan type count: %w", err)
		}
		stats.ActionsByType[actionType] = count
	}

	// Get average execution time
	avgTimeQuery := `
		SELECT AVG(execution_duration_ms) 
		FROM queued_actions 
		WHERE execution_duration_ms IS NOT NULL AND completed_at IS NOT NULL
	`

	var avgTime sql.NullFloat64
	err = r.db.GetContext(ctx, &avgTime, avgTimeQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get average execution time: %w", err)
	}

	if avgTime.Valid {
		stats.AvgExecutionTime = avgTime.Float64
	}

	// Calculate success rate
	if stats.CompletedActions > 0 || stats.FailedActions > 0 {
		totalCompleted := stats.CompletedActions + stats.FailedActions
		stats.SuccessRate = float64(stats.CompletedActions) / float64(totalCompleted) * 100
	}

	// Get oldest pending action
	oldestQuery := `
		SELECT MIN(created_at) 
		FROM queued_actions qa
		JOIN action_statuses ast ON qa.status_id = ast.id
		WHERE ast.name = 'pending'
	`

	var oldestTime sql.NullTime
	err = r.db.GetContext(ctx, &oldestTime, oldestQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get oldest pending action: %w", err)
	}

	if oldestTime.Valid {
		stats.OldestPendingAction = &oldestTime.Time
	}

	// Get last processed time
	lastProcessedQuery := `
		SELECT MAX(completed_at) 
		FROM queued_actions 
		WHERE completed_at IS NOT NULL
	`

	var lastProcessed sql.NullTime
	err = r.db.GetContext(ctx, &lastProcessed, lastProcessedQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get last processed time: %w", err)
	}

	if lastProcessed.Valid {
		stats.LastProcessedAt = &lastProcessed.Time
	}

	// Determine queue health
	if stats.PendingActions > 100 {
		stats.QueueHealth = "critical"
	} else if stats.PendingActions > 50 {
		stats.QueueHealth = "warning"
	} else if stats.ProcessingActions > 0 || stats.PendingActions > 0 {
		stats.QueueHealth = "active"
	} else {
		stats.QueueHealth = "healthy"
	}

	return stats, nil
}

// Cleanup Operations
func (r *QueueRepository) CleanupOldActions(ctx context.Context, olderThan time.Time, statuses []string) (int, error) {
	placeholders := strings.Repeat("?,", len(statuses))
	placeholders = placeholders[:len(placeholders)-1]

	query := fmt.Sprintf(`
		DELETE FROM queued_actions 
		WHERE id IN (
			SELECT qa.id FROM queued_actions qa
			JOIN action_statuses ast ON qa.status_id = ast.id
			WHERE qa.created_at < ? AND ast.name IN (%s)
		)
	`, placeholders)

	args := []interface{}{olderThan}
	for _, status := range statuses {
		args = append(args, status)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		r.log.WithError(err).Error("Failed to cleanup old actions")
		return 0, fmt.Errorf("failed to cleanup old actions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

func (r *QueueRepository) ClearQueue(ctx context.Context, filter *models.QueueClearRequest) (int, error) {
	conditions := []string{}
	args := []interface{}{}

	if len(filter.Status) > 0 {
		placeholders := strings.Repeat("?,", len(filter.Status))
		placeholders = placeholders[:len(placeholders)-1]
		conditions = append(conditions, fmt.Sprintf("ast.name IN (%s)", placeholders))
		for _, status := range filter.Status {
			args = append(args, status)
		}
	}

	if filter.OlderThan != nil {
		conditions = append(conditions, "qa.created_at < ?")
		args = append(args, *filter.OlderThan)
	}

	if filter.ActionType != "" {
		conditions = append(conditions, "at.name = ?")
		args = append(args, filter.ActionType)
	}

	if filter.CorrelationID != "" {
		conditions = append(conditions, "qa.correlation_id = ?")
		args = append(args, filter.CorrelationID)
	}

	baseQuery := `
		DELETE FROM queued_actions 
		WHERE id IN (
			SELECT qa.id FROM queued_actions qa
			JOIN action_statuses ast ON qa.status_id = ast.id
			JOIN action_types at ON qa.action_type_id = at.id
	`

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	baseQuery += ")"

	result, err := r.db.ExecContext(ctx, baseQuery, args...)
	if err != nil {
		r.log.WithError(err).Error("Failed to clear queue")
		return 0, fmt.Errorf("failed to clear queue: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}
