package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// ActionPriority represents the priority levels for queued actions
type ActionPriority struct {
	ID          int    `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Weight      int    `json:"weight" db:"weight"`
	Description string `json:"description" db:"description"`
}

// ActionStatus represents the possible statuses of queued actions
type ActionStatus struct {
	ID          int    `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	IsTerminal  bool   `json:"is_terminal" db:"is_terminal"`
}

// ActionType represents the types of actions that can be queued
type ActionType struct {
	ID                 int       `json:"id" db:"id"`
	Name               string    `json:"name" db:"name"`
	Description        string    `json:"description" db:"description"`
	HandlerName        string    `json:"handler_name" db:"handler_name"`
	DefaultTimeout     int       `json:"default_timeout" db:"default_timeout"`
	MaxRetries         int       `json:"max_retries" db:"max_retries"`
	RetryBackoffFactor float64   `json:"retry_backoff_factor" db:"retry_backoff_factor"`
	Enabled            bool      `json:"enabled" db:"enabled"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// QueuedAction represents a single action in the queue
type QueuedAction struct {
	ID                  int            `json:"id" db:"id"`
	ActionTypeID        int            `json:"action_type_id" db:"action_type_id"`
	PriorityID          int            `json:"priority_id" db:"priority_id"`
	StatusID            int            `json:"status_id" db:"status_id"`
	Name                string         `json:"name" db:"name"`
	Description         sql.NullString `json:"description" db:"description"`
	UserID              sql.NullInt64  `json:"user_id" db:"user_id"`
	CorrelationID       sql.NullString `json:"correlation_id" db:"correlation_id"`
	ParentActionID      sql.NullInt64  `json:"parent_action_id" db:"parent_action_id"`
	ActionData          string         `json:"action_data" db:"action_data"`
	TargetEntityID      sql.NullString `json:"target_entity_id" db:"target_entity_id"`
	TimeoutSeconds      sql.NullInt64  `json:"timeout_seconds" db:"timeout_seconds"`
	MaxRetries          sql.NullInt64  `json:"max_retries" db:"max_retries"`
	RetryCount          int            `json:"retry_count" db:"retry_count"`
	RetryBackoffFactor  float64        `json:"retry_backoff_factor" db:"retry_backoff_factor"`
	ScheduledAt         sql.NullTime   `json:"scheduled_at" db:"scheduled_at"`
	ExecuteAfter        sql.NullTime   `json:"execute_after" db:"execute_after"`
	Deadline            sql.NullTime   `json:"deadline" db:"deadline"`
	StartedAt           sql.NullTime   `json:"started_at" db:"started_at"`
	CompletedAt         sql.NullTime   `json:"completed_at" db:"completed_at"`
	LastAttemptAt       sql.NullTime   `json:"last_attempt_at" db:"last_attempt_at"`
	NextRetryAt         sql.NullTime   `json:"next_retry_at" db:"next_retry_at"`
	ResultData          sql.NullString `json:"result_data" db:"result_data"`
	ErrorMessage        sql.NullString `json:"error_message" db:"error_message"`
	ErrorDetails        sql.NullString `json:"error_details" db:"error_details"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`
	CreatedBy           sql.NullString `json:"created_by" db:"created_by"`
	ExecutionDurationMS sql.NullInt64  `json:"execution_duration_ms" db:"execution_duration_ms"`

	// Joined fields (not in database)
	ActionType   *ActionType        `json:"action_type,omitempty" db:"-"`
	Priority     *ActionPriority    `json:"priority,omitempty" db:"-"`
	Status       *ActionStatus      `json:"status,omitempty" db:"-"`
	Dependencies []ActionDependency `json:"dependencies,omitempty" db:"-"`
	Results      []ActionResult     `json:"results,omitempty" db:"-"`
}

// ActionResult represents the result of an action execution attempt
type ActionResult struct {
	ID               int            `json:"id" db:"id"`
	ActionID         int            `json:"action_id" db:"action_id"`
	AttemptNumber    int            `json:"attempt_number" db:"attempt_number"`
	StatusID         int            `json:"status_id" db:"status_id"`
	StartedAt        time.Time      `json:"started_at" db:"started_at"`
	CompletedAt      sql.NullTime   `json:"completed_at" db:"completed_at"`
	DurationMS       sql.NullInt64  `json:"duration_ms" db:"duration_ms"`
	Success          bool           `json:"success" db:"success"`
	ResultData       sql.NullString `json:"result_data" db:"result_data"`
	ErrorCode        sql.NullString `json:"error_code" db:"error_code"`
	ErrorMessage     sql.NullString `json:"error_message" db:"error_message"`
	ErrorDetails     sql.NullString `json:"error_details" db:"error_details"`
	WorkerID         sql.NullString `json:"worker_id" db:"worker_id"`
	ExecutionContext sql.NullString `json:"execution_context" db:"execution_context"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`

	// Joined fields
	Status *ActionStatus `json:"status,omitempty" db:"-"`
}

// QueueSetting represents a configuration setting for the queue system
type QueueSetting struct {
	ID          int       `json:"id" db:"id"`
	Key         string    `json:"key" db:"key"`
	Value       string    `json:"value" db:"value"`
	DataType    string    `json:"data_type" db:"data_type"`
	Description string    `json:"description" db:"description"`
	Category    string    `json:"category" db:"category"`
	IsReadonly  bool      `json:"is_readonly" db:"is_readonly"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ActionDependency represents a dependency between actions
type ActionDependency struct {
	ID                int       `json:"id" db:"id"`
	ActionID          int       `json:"action_id" db:"action_id"`
	DependsOnActionID int       `json:"depends_on_action_id" db:"depends_on_action_id"`
	DependencyType    string    `json:"dependency_type" db:"dependency_type"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`

	// Joined fields
	DependsOnAction *QueuedAction `json:"depends_on_action,omitempty" db:"-"`
}

// QueueStatistics represents overall queue health and statistics
type QueueStatistics struct {
	TotalActions        int            `json:"total_actions"`
	PendingActions      int            `json:"pending_actions"`
	ProcessingActions   int            `json:"processing_actions"`
	CompletedActions    int            `json:"completed_actions"`
	FailedActions       int            `json:"failed_actions"`
	RetryingActions     int            `json:"retrying_actions"`
	CancelledActions    int            `json:"cancelled_actions"`
	ActionsByPriority   map[string]int `json:"actions_by_priority"`
	ActionsByType       map[string]int `json:"actions_by_type"`
	AvgExecutionTime    float64        `json:"avg_execution_time_ms"`
	SuccessRate         float64        `json:"success_rate"`
	QueueHealth         string         `json:"queue_health"`
	WorkerStatus        []WorkerStatus `json:"worker_status"`
	OldestPendingAction *time.Time     `json:"oldest_pending_action"`
	LastProcessedAt     *time.Time     `json:"last_processed_at"`
}

// WorkerStatus represents the status of individual queue workers
type WorkerStatus struct {
	WorkerID        string    `json:"worker_id"`
	Status          string    `json:"status"` // idle, processing, error
	CurrentActionID *int      `json:"current_action_id"`
	LastActivity    time.Time `json:"last_activity"`
	ProcessedCount  int       `json:"processed_count"`
	ErrorCount      int       `json:"error_count"`
}

// CreateActionRequest represents a request to create a new queued action
type CreateActionRequest struct {
	ActionType     string                    `json:"action_type" binding:"required"`
	Name           string                    `json:"name" binding:"required"`
	Description    string                    `json:"description"`
	Priority       string                    `json:"priority"` // low, normal, high, urgent, critical
	ActionData     json.RawMessage           `json:"action_data" binding:"required"`
	TargetEntityID string                    `json:"target_entity_id"`
	CorrelationID  string                    `json:"correlation_id"`
	ParentActionID *int                      `json:"parent_action_id"`
	TimeoutSeconds *int                      `json:"timeout_seconds"`
	MaxRetries     *int                      `json:"max_retries"`
	ScheduledAt    *time.Time                `json:"scheduled_at"`
	ExecuteAfter   *time.Time                `json:"execute_after"`
	Deadline       *time.Time                `json:"deadline"`
	Dependencies   []ActionDependencyRequest `json:"dependencies"`
}

// ActionDependencyRequest represents a dependency when creating an action
type ActionDependencyRequest struct {
	DependsOnActionID int    `json:"depends_on_action_id" binding:"required"`
	DependencyType    string `json:"dependency_type"` // completion, success, failure
}

// UpdateActionRequest represents a request to update a queued action
type UpdateActionRequest struct {
	Name         *string    `json:"name"`
	Description  *string    `json:"description"`
	Priority     *string    `json:"priority"`
	Status       *string    `json:"status"`
	ScheduledAt  *time.Time `json:"scheduled_at"`
	ExecuteAfter *time.Time `json:"execute_after"`
	Deadline     *time.Time `json:"deadline"`
}

// QueueFilter represents filters for querying queued actions
type QueueFilter struct {
	Status          []string   `json:"status"`
	Priority        []string   `json:"priority"`
	ActionType      []string   `json:"action_type"`
	UserID          *int       `json:"user_id"`
	CorrelationID   string     `json:"correlation_id"`
	TargetEntityID  string     `json:"target_entity_id"`
	CreatedAfter    *time.Time `json:"created_after"`
	CreatedBefore   *time.Time `json:"created_before"`
	ScheduledAfter  *time.Time `json:"scheduled_after"`
	ScheduledBefore *time.Time `json:"scheduled_before"`
	Limit           int        `json:"limit"`
	Offset          int        `json:"offset"`
	OrderBy         string     `json:"order_by"`        // created_at, priority, scheduled_at
	OrderDirection  string     `json:"order_direction"` // asc, desc
}

// BulkActionRequest represents a request to perform bulk operations
type BulkActionRequest struct {
	Actions []CreateActionRequest `json:"actions" binding:"required"`
	Options BulkActionOptions     `json:"options"`
}

// BulkActionOptions represents options for bulk operations
type BulkActionOptions struct {
	Sequential    bool   `json:"sequential"`     // Execute actions sequentially
	StopOnError   bool   `json:"stop_on_error"`  // Stop if any action fails
	CorrelationID string `json:"correlation_id"` // Group all actions under this ID
	Priority      string `json:"priority"`       // Override priority for all actions
	MaxConcurrent int    `json:"max_concurrent"` // Limit concurrent execution
}

// QueueProcessRequest represents a request to manually process the queue
type QueueProcessRequest struct {
	ActionIDs  []int  `json:"action_ids"`  // Specific actions to process
	Priority   string `json:"priority"`    // Process only actions with this priority
	ActionType string `json:"action_type"` // Process only actions of this type
	MaxActions int    `json:"max_actions"` // Limit number of actions to process
	ForceRetry bool   `json:"force_retry"` // Force retry failed actions
}

// QueueClearRequest represents a request to clear queue items
type QueueClearRequest struct {
	Status        []string   `json:"status"`                           // Clear actions with these statuses
	OlderThan     *time.Time `json:"older_than"`                       // Clear actions older than this
	ActionType    string     `json:"action_type"`                      // Clear actions of this type
	CorrelationID string     `json:"correlation_id"`                   // Clear actions with this correlation ID
	ConfirmClear  bool       `json:"confirm_clear" binding:"required"` // Safety confirmation
}

// RetryPolicy represents the retry configuration for an action
type RetryPolicy struct {
	MaxRetries         int      `json:"max_retries"`
	BackoffFactor      float64  `json:"backoff_factor"`
	InitialBackoffMS   int      `json:"initial_backoff_ms"`
	MaxBackoffMS       int      `json:"max_backoff_ms"`
	RetryableErrors    []string `json:"retryable_errors"`
	NonRetryableErrors []string `json:"non_retryable_errors"`
}

// ActionExecutionContext represents the context in which an action is executed
type ActionExecutionContext struct {
	WorkerID        string                 `json:"worker_id"`
	ExecutionStart  time.Time              `json:"execution_start"`
	Environment     map[string]string      `json:"environment"`
	ServiceVersions map[string]string      `json:"service_versions"`
	RequestID       string                 `json:"request_id"`
	UserContext     map[string]interface{} `json:"user_context"`
}

// ActionPayload represents the standard structure for action data
type ActionPayload struct {
	Type       string                 `json:"type"`
	Target     string                 `json:"target"`
	Parameters map[string]interface{} `json:"parameters"`
	Options    map[string]interface{} `json:"options"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Utility methods for status checking
func (qa *QueuedAction) IsTerminal() bool {
	if qa.Status != nil {
		return qa.Status.IsTerminal
	}
	// Fallback to status ID check
	return qa.StatusID == 3 || qa.StatusID == 4 || qa.StatusID == 6 || qa.StatusID == 7 // completed, failed, cancelled, timeout
}

func (qa *QueuedAction) IsPending() bool {
	return qa.StatusID == 1 // pending
}

func (qa *QueuedAction) IsProcessing() bool {
	return qa.StatusID == 2 // processing
}

func (qa *QueuedAction) IsCompleted() bool {
	return qa.StatusID == 3 // completed
}

func (qa *QueuedAction) IsFailed() bool {
	return qa.StatusID == 4 // failed
}

func (qa *QueuedAction) IsRetrying() bool {
	return qa.StatusID == 5 // retrying
}

func (qa *QueuedAction) CanRetry() bool {
	if qa.MaxRetries.Valid {
		return qa.RetryCount < int(qa.MaxRetries.Int64)
	}
	return qa.RetryCount < 3 // Default max retries
}

func (qa *QueuedAction) ShouldExecuteNow() bool {
	now := time.Now()

	// Check if scheduled for future execution
	if qa.ScheduledAt.Valid && qa.ScheduledAt.Time.After(now) {
		return false
	}

	// Check if should not execute yet
	if qa.ExecuteAfter.Valid && qa.ExecuteAfter.Time.After(now) {
		return false
	}

	// Check if past deadline
	if qa.Deadline.Valid && qa.Deadline.Time.Before(now) {
		return false
	}

	// Check if it's a retry and not time yet
	if qa.NextRetryAt.Valid && qa.NextRetryAt.Time.After(now) {
		return false
	}

	return true
}

func (qa *QueuedAction) IsOverdue() bool {
	if qa.Deadline.Valid {
		return qa.Deadline.Time.Before(time.Now())
	}
	return false
}

// Helper methods for ActionPayload
func (ap *ActionPayload) GetParameter(key string) (interface{}, bool) {
	if ap.Parameters == nil {
		return nil, false
	}
	val, exists := ap.Parameters[key]
	return val, exists
}

func (ap *ActionPayload) SetParameter(key string, value interface{}) {
	if ap.Parameters == nil {
		ap.Parameters = make(map[string]interface{})
	}
	ap.Parameters[key] = value
}

func (ap *ActionPayload) GetOption(key string) (interface{}, bool) {
	if ap.Options == nil {
		return nil, false
	}
	val, exists := ap.Options[key]
	return val, exists
}

func (ap *ActionPayload) SetOption(key string, value interface{}) {
	if ap.Options == nil {
		ap.Options = make(map[string]interface{})
	}
	ap.Options[key] = value
}
