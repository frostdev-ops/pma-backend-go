package automation

import (
	"encoding/json"
	"fmt"
	"time"
)

// ExecutionMode defines how a rule should be executed
type ExecutionMode string

const (
	ExecutionModeSingle   ExecutionMode = "single"   // Only one instance at a time
	ExecutionModeParallel ExecutionMode = "parallel" // Multiple instances can run
	ExecutionModeQueued   ExecutionMode = "queued"   // Queue executions if busy
)

// RuleStatus represents the current status of a rule
type RuleStatus string

const (
	RuleStatusIdle    RuleStatus = "idle"
	RuleStatusRunning RuleStatus = "running"
	RuleStatusWaiting RuleStatus = "waiting"
	RuleStatusError   RuleStatus = "error"
)

// AutomationRule represents a complete automation rule
type AutomationRule struct {
	ID          string                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Enabled     bool                   `json:"enabled" db:"enabled"`
	Triggers    []Trigger              `json:"triggers" db:"triggers"`
	Conditions  []Condition            `json:"conditions" db:"conditions"`
	Actions     []Action               `json:"actions" db:"actions"`
	Mode        ExecutionMode          `json:"mode" db:"mode"`
	LastRun     *time.Time             `json:"last_run" db:"last_run"`
	NextRun     *time.Time             `json:"next_run" db:"next_run"`
	RunCount    int64                  `json:"run_count" db:"run_count"`
	Variables   map[string]interface{} `json:"variables" db:"variables"`
	Status      RuleStatus             `json:"status" db:"status"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`

	// Runtime fields (not stored in DB)
	Priority int      `json:"priority" db:"-"`
	Tags     []string `json:"tags" db:"-"`
	Category string   `json:"category" db:"-"`
}

// RuleExecution represents a single execution of a rule
type RuleExecution struct {
	ID        string                 `json:"id" db:"id"`
	RuleID    string                 `json:"rule_id" db:"rule_id"`
	StartTime time.Time              `json:"start_time" db:"start_time"`
	EndTime   *time.Time             `json:"end_time" db:"end_time"`
	Success   bool                   `json:"success" db:"success"`
	Error     string                 `json:"error" db:"error"`
	Context   map[string]interface{} `json:"context" db:"context"`
	Variables map[string]interface{} `json:"variables" db:"variables"`
	Duration  time.Duration          `json:"duration" db:"duration"`
}

// RuleValidationError represents validation errors for rules
type RuleValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// RuleValidationResult contains validation results
type RuleValidationResult struct {
	Valid  bool                  `json:"valid"`
	Errors []RuleValidationError `json:"errors,omitempty"`
}

// Validate performs comprehensive validation of the rule
func (r *AutomationRule) Validate() *RuleValidationResult {
	result := &RuleValidationResult{Valid: true, Errors: []RuleValidationError{}}

	// Validate basic fields
	if r.Name == "" {
		result.Errors = append(result.Errors, RuleValidationError{
			Field:   "name",
			Message: "Rule name is required",
		})
	}

	if r.ID == "" {
		result.Errors = append(result.Errors, RuleValidationError{
			Field:   "id",
			Message: "Rule ID is required",
		})
	}

	// Validate execution mode
	switch r.Mode {
	case ExecutionModeSingle, ExecutionModeParallel, ExecutionModeQueued:
		// Valid modes
	default:
		result.Errors = append(result.Errors, RuleValidationError{
			Field:   "mode",
			Message: "Invalid execution mode",
		})
	}

	// Validate triggers
	if len(r.Triggers) == 0 {
		result.Errors = append(result.Errors, RuleValidationError{
			Field:   "triggers",
			Message: "At least one trigger is required",
		})
	}

	for i, trigger := range r.Triggers {
		if err := trigger.Validate(); err != nil {
			result.Errors = append(result.Errors, RuleValidationError{
				Field:   fmt.Sprintf("triggers[%d]", i),
				Message: err.Error(),
			})
		}
	}

	// Validate conditions
	for i, condition := range r.Conditions {
		if err := condition.Validate(); err != nil {
			result.Errors = append(result.Errors, RuleValidationError{
				Field:   fmt.Sprintf("conditions[%d]", i),
				Message: err.Error(),
			})
		}
	}

	// Validate actions
	if len(r.Actions) == 0 {
		result.Errors = append(result.Errors, RuleValidationError{
			Field:   "actions",
			Message: "At least one action is required",
		})
	}

	for i, action := range r.Actions {
		if err := action.Validate(); err != nil {
			result.Errors = append(result.Errors, RuleValidationError{
				Field:   fmt.Sprintf("actions[%d]", i),
				Message: err.Error(),
			})
		}
	}

	if len(result.Errors) > 0 {
		result.Valid = false
	}

	return result
}

// GetEstimatedExecutionTime calculates estimated execution time for the rule
func (r *AutomationRule) GetEstimatedExecutionTime() time.Duration {
	var total time.Duration

	for _, action := range r.Actions {
		total += action.EstimateExecutionTime()
	}

	return total
}

// Clone creates a deep copy of the rule
func (r *AutomationRule) Clone() *AutomationRule {
	data, _ := json.Marshal(r)
	var clone AutomationRule
	json.Unmarshal(data, &clone)
	return &clone
}

// HasTag checks if the rule has a specific tag
func (r *AutomationRule) HasTag(tag string) bool {
	for _, t := range r.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// CanExecute checks if the rule can be executed based on its current state
func (r *AutomationRule) CanExecute() bool {
	if !r.Enabled {
		return false
	}

	switch r.Mode {
	case ExecutionModeSingle:
		return r.Status != RuleStatusRunning
	case ExecutionModeParallel:
		return true
	case ExecutionModeQueued:
		return true
	default:
		return false
	}
}
