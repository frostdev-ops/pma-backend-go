package automation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// ScheduledTrigger represents a trigger that should be scheduled
type ScheduledTrigger struct {
	ID       string
	RuleID   string
	Trigger  *TimeTrigger
	Handler  TriggerHandler
	EntryID  cron.EntryID
	NextRun  time.Time
	LastRun  *time.Time
	RunCount int64
}

// Scheduler manages time-based triggers and their execution
type Scheduler struct {
	cron     *cron.Cron
	triggers map[string]*ScheduledTrigger
	timezone *time.Location
	logger   *logrus.Logger
	mu       sync.RWMutex
	running  bool

	// Event channel for notifying about trigger events
	eventChan chan Event
}

// SchedulerConfig contains scheduler configuration
type SchedulerConfig struct {
	Timezone          string `json:"timezone"`
	MissedJobMaxAge   string `json:"missed_job_max_age"`
	MaxConcurrentJobs int    `json:"max_concurrent_jobs"`
}

// NewScheduler creates a new scheduler instance
func NewScheduler(config *SchedulerConfig, logger *logrus.Logger) (*Scheduler, error) {
	// Parse timezone
	timezone := time.UTC
	if config != nil && config.Timezone != "" {
		tz, err := time.LoadLocation(config.Timezone)
		if err != nil {
			logger.WithError(err).Warnf("Invalid timezone %s, using UTC", config.Timezone)
		} else {
			timezone = tz
		}
	}

	// Create cron instance with timezone and second precision
	cronInstance := cron.New(
		cron.WithLocation(timezone),
		cron.WithSeconds(),
		cron.WithChain(
			cron.SkipIfStillRunning(cron.DefaultLogger),
			cron.Recover(cron.DefaultLogger),
		),
	)

	return &Scheduler{
		cron:      cronInstance,
		triggers:  make(map[string]*ScheduledTrigger),
		timezone:  timezone,
		logger:    logger,
		eventChan: make(chan Event, 100),
	}, nil
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	s.cron.Start()
	s.running = true
	s.logger.Info("Automation scheduler started")

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler is not running")
	}

	// Stop the cron scheduler
	ctx := s.cron.Stop()

	// Wait for running jobs to complete (with timeout)
	select {
	case <-ctx.Done():
		s.logger.Info("All scheduled jobs completed")
	case <-time.After(30 * time.Second):
		s.logger.Warn("Timeout waiting for scheduled jobs to complete")
	}

	s.running = false
	s.logger.Info("Automation scheduler stopped")

	return nil
}

// IsRunning returns whether the scheduler is running
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// ScheduleTrigger schedules a time-based trigger
func (s *Scheduler) ScheduleTrigger(ruleID string, trigger *TimeTrigger, handler TriggerHandler) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if trigger == nil {
		return fmt.Errorf("trigger cannot be nil")
	}

	if err := trigger.Validate(); err != nil {
		return fmt.Errorf("invalid trigger: %v", err)
	}

	scheduledTrigger := &ScheduledTrigger{
		ID:      trigger.GetID(),
		RuleID:  ruleID,
		Trigger: trigger,
		Handler: handler,
	}

	// Generate cron expression based on trigger type
	cronExpr, err := s.generateCronExpression(trigger)
	if err != nil {
		return fmt.Errorf("failed to generate cron expression: %v", err)
	}

	// Schedule the job
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.executeTrigger(scheduledTrigger)
	})
	if err != nil {
		return fmt.Errorf("failed to schedule trigger: %v", err)
	}

	scheduledTrigger.EntryID = entryID
	scheduledTrigger.NextRun = s.cron.Entry(entryID).Next

	s.triggers[trigger.GetID()] = scheduledTrigger

	s.logger.WithFields(map[string]interface{}{
		"trigger_id": trigger.GetID(),
		"rule_id":    ruleID,
		"cron_expr":  cronExpr,
		"next_run":   scheduledTrigger.NextRun,
	}).Info("Trigger scheduled successfully")

	return nil
}

// UnscheduleTrigger removes a scheduled trigger
func (s *Scheduler) UnscheduleTrigger(triggerID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	scheduledTrigger, exists := s.triggers[triggerID]
	if !exists {
		return fmt.Errorf("trigger %s not found", triggerID)
	}

	// Remove from cron
	s.cron.Remove(scheduledTrigger.EntryID)

	// Remove from our map
	delete(s.triggers, triggerID)

	s.logger.WithField("trigger_id", triggerID).Info("Trigger unscheduled")

	return nil
}

// GetScheduledTriggers returns all scheduled triggers
func (s *Scheduler) GetScheduledTriggers() []*ScheduledTrigger {
	s.mu.RLock()
	defer s.mu.RUnlock()

	triggers := make([]*ScheduledTrigger, 0, len(s.triggers))
	for _, trigger := range s.triggers {
		// Update next run time
		if entry := s.cron.Entry(trigger.EntryID); entry.ID != 0 {
			trigger.NextRun = entry.Next
		}
		triggers = append(triggers, trigger)
	}

	return triggers
}

// GetScheduledTriggersForRule returns scheduled triggers for a specific rule
func (s *Scheduler) GetScheduledTriggersForRule(ruleID string) []*ScheduledTrigger {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var triggers []*ScheduledTrigger
	for _, trigger := range s.triggers {
		if trigger.RuleID == ruleID {
			triggers = append(triggers, trigger)
		}
	}

	return triggers
}

// UnscheduleTriggersForRule removes all scheduled triggers for a rule
func (s *Scheduler) UnscheduleTriggersForRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var removed []string
	for triggerID, scheduledTrigger := range s.triggers {
		if scheduledTrigger.RuleID == ruleID {
			s.cron.Remove(scheduledTrigger.EntryID)
			delete(s.triggers, triggerID)
			removed = append(removed, triggerID)
		}
	}

	s.logger.WithFields(map[string]interface{}{
		"rule_id":          ruleID,
		"removed_triggers": removed,
	}).Info("Unscheduled triggers for rule")

	return nil
}

// GetEventChannel returns the event channel for trigger events
func (s *Scheduler) GetEventChannel() <-chan Event {
	return s.eventChan
}

// GetStatistics returns scheduler statistics
func (s *Scheduler) GetStatistics() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"running":        s.running,
		"total_triggers": len(s.triggers),
		"timezone":       s.timezone.String(),
		"cron_entries":   len(s.cron.Entries()),
		"next_schedules": make([]map[string]interface{}, 0),
	}

	// Get next few schedules
	entries := s.cron.Entries()
	nextSchedules := stats["next_schedules"].([]map[string]interface{})

	for _, entry := range entries {
		if len(nextSchedules) >= 10 { // Limit to next 10 schedules
			break
		}

		// Find corresponding trigger
		var triggerID, ruleID string
		for id, trigger := range s.triggers {
			if trigger.EntryID == entry.ID {
				triggerID = id
				ruleID = trigger.RuleID
				break
			}
		}

		nextSchedules = append(nextSchedules, map[string]interface{}{
			"trigger_id": triggerID,
			"rule_id":    ruleID,
			"next_run":   entry.Next,
			"prev_run":   entry.Prev,
		})
	}

	stats["next_schedules"] = nextSchedules

	return stats
}

// generateCronExpression converts a TimeTrigger to a cron expression
func (s *Scheduler) generateCronExpression(trigger *TimeTrigger) (string, error) {
	if trigger.Cron != "" {
		// Validate the cron expression
		if _, err := cron.ParseStandard(trigger.Cron); err != nil {
			return "", fmt.Errorf("invalid cron expression: %s", trigger.Cron)
		}
		return trigger.Cron, nil
	}

	if trigger.At != "" {
		// Convert time of day to cron expression
		t, err := time.Parse("15:04", trigger.At)
		if err != nil {
			// Try with seconds
			t, err = time.Parse("15:04:05", trigger.At)
			if err != nil {
				return "", fmt.Errorf("invalid time format: %s", trigger.At)
			}
		}

		// Generate cron expression for daily execution at specified time
		return fmt.Sprintf("%d %d %d * * *", t.Second(), t.Minute(), t.Hour()), nil
	}

	if trigger.Interval != "" {
		// Convert interval to cron expression
		duration, err := time.ParseDuration(trigger.Interval)
		if err != nil {
			return "", fmt.Errorf("invalid interval: %s", trigger.Interval)
		}

		// For intervals, we'll use a simple approach
		// Note: This is simplified - real cron intervals would need more complex logic
		if duration.Minutes() == 1 {
			return "0 * * * * *", nil // Every minute
		} else if duration.Minutes() == 5 {
			return "0 */5 * * * *", nil // Every 5 minutes
		} else if duration.Minutes() == 15 {
			return "0 */15 * * * *", nil // Every 15 minutes
		} else if duration.Minutes() == 30 {
			return "0 */30 * * * *", nil // Every 30 minutes
		} else if duration.Hours() == 1 {
			return "0 0 * * * *", nil // Every hour
		} else {
			// For other intervals, we'll need a different approach
			// This is a limitation of cron - it doesn't handle arbitrary intervals well
			return "", fmt.Errorf("unsupported interval: %s (use cron expression instead)", trigger.Interval)
		}
	}

	return "", fmt.Errorf("no scheduling method specified in trigger")
}

// executeTrigger executes a scheduled trigger
func (s *Scheduler) executeTrigger(scheduledTrigger *ScheduledTrigger) {
	start := time.Now()

	s.logger.WithFields(map[string]interface{}{
		"trigger_id": scheduledTrigger.ID,
		"rule_id":    scheduledTrigger.RuleID,
	}).Debug("Executing scheduled trigger")

	// Update statistics
	s.mu.Lock()
	scheduledTrigger.RunCount++
	now := time.Now()
	scheduledTrigger.LastRun = &now
	s.mu.Unlock()

	// Create trigger event
	event := Event{
		Type:   "time_trigger",
		Source: "scheduler",
		Data: map[string]interface{}{
			"trigger_id":   scheduledTrigger.ID,
			"rule_id":      scheduledTrigger.RuleID,
			"trigger_time": start,
			"run_count":    scheduledTrigger.RunCount,
		},
		Timestamp: start,
	}

	// Send event to channel (non-blocking)
	select {
	case s.eventChan <- event:
	default:
		s.logger.Warn("Event channel full, dropping trigger event")
	}

	// Call the trigger handler if set
	if scheduledTrigger.Handler != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := scheduledTrigger.Handler(ctx, scheduledTrigger.Trigger, event); err != nil {
			s.logger.WithError(err).WithFields(map[string]interface{}{
				"trigger_id": scheduledTrigger.ID,
				"rule_id":    scheduledTrigger.RuleID,
			}).Error("Trigger handler failed")
		}
	}

	duration := time.Since(start)
	s.logger.WithFields(map[string]interface{}{
		"trigger_id": scheduledTrigger.ID,
		"rule_id":    scheduledTrigger.RuleID,
		"duration":   duration,
	}).Debug("Scheduled trigger execution completed")
}

// ValidateSchedule validates if a schedule configuration is valid
func (s *Scheduler) ValidateSchedule(trigger *TimeTrigger) error {
	if trigger == nil {
		return fmt.Errorf("trigger cannot be nil")
	}

	if err := trigger.Validate(); err != nil {
		return err
	}

	// Try to generate cron expression to validate
	_, err := s.generateCronExpression(trigger)
	return err
}

// GetNextRuns returns the next N scheduled executions for a trigger
func (s *Scheduler) GetNextRuns(triggerID string, count int) ([]time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	scheduledTrigger, exists := s.triggers[triggerID]
	if !exists {
		return nil, fmt.Errorf("trigger %s not found", triggerID)
	}

	// Get the cron entry
	entry := s.cron.Entry(scheduledTrigger.EntryID)
	if entry.ID == 0 {
		return nil, fmt.Errorf("cron entry not found for trigger %s", triggerID)
	}

	// Calculate next runs
	nextRuns := make([]time.Time, 0, count)
	current := entry.Next

	for i := 0; i < count && !current.IsZero(); i++ {
		nextRuns = append(nextRuns, current)

		// This is a simplified approach - in reality, we'd need to
		// properly calculate the next execution time based on the schedule
		current = current.Add(24 * time.Hour) // Placeholder logic
	}

	return nextRuns, nil
}

// RescheduleAllTriggers reschedules all triggers (useful after timezone changes)
func (s *Scheduler) RescheduleAllTriggers() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info("Rescheduling all triggers")

	// Store current triggers
	currentTriggers := make(map[string]*ScheduledTrigger)
	for id, trigger := range s.triggers {
		currentTriggers[id] = trigger
	}

	// Remove all current schedules
	for _, trigger := range currentTriggers {
		s.cron.Remove(trigger.EntryID)
	}
	s.triggers = make(map[string]*ScheduledTrigger)

	// Re-add all triggers
	for _, trigger := range currentTriggers {
		cronExpr, err := s.generateCronExpression(trigger.Trigger)
		if err != nil {
			s.logger.WithError(err).WithField("trigger_id", trigger.ID).Error("Failed to reschedule trigger")
			continue
		}

		entryID, err := s.cron.AddFunc(cronExpr, func() {
			s.executeTrigger(trigger)
		})
		if err != nil {
			s.logger.WithError(err).WithField("trigger_id", trigger.ID).Error("Failed to add rescheduled trigger")
			continue
		}

		trigger.EntryID = entryID
		trigger.NextRun = s.cron.Entry(entryID).Next
		s.triggers[trigger.ID] = trigger
	}

	s.logger.WithField("rescheduled_count", len(s.triggers)).Info("Trigger rescheduling completed")

	return nil
}
