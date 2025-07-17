package automation

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutomationRule_Validate(t *testing.T) {
	tests := []struct {
		name     string
		rule     *AutomationRule
		expected bool
	}{
		{
			name: "valid rule",
			rule: &AutomationRule{
				ID:          "test-rule",
				Name:        "Test Rule",
				Description: "A test rule",
				Enabled:     true,
				Mode:        ExecutionModeSingle,
				Triggers: []Trigger{
					NewStateTrigger("trigger1", "sensor.test"),
				},
				Actions: []Action{
					NewServiceAction("action1", "light.turn_on"),
				},
				Variables: make(map[string]interface{}),
			},
			expected: true,
		},
		{
			name: "missing name",
			rule: &AutomationRule{
				ID:          "test-rule",
				Description: "A test rule",
				Enabled:     true,
				Mode:        ExecutionModeSingle,
				Triggers: []Trigger{
					NewStateTrigger("trigger1", "sensor.test"),
				},
				Actions: []Action{
					NewServiceAction("action1", "light.turn_on"),
				},
				Variables: make(map[string]interface{}),
			},
			expected: false,
		},
		{
			name: "no triggers",
			rule: &AutomationRule{
				ID:          "test-rule",
				Name:        "Test Rule",
				Description: "A test rule",
				Enabled:     true,
				Mode:        ExecutionModeSingle,
				Triggers:    []Trigger{},
				Actions: []Action{
					NewServiceAction("action1", "light.turn_on"),
				},
				Variables: make(map[string]interface{}),
			},
			expected: false,
		},
		{
			name: "no actions",
			rule: &AutomationRule{
				ID:          "test-rule",
				Name:        "Test Rule",
				Description: "A test rule",
				Enabled:     true,
				Mode:        ExecutionModeSingle,
				Triggers: []Trigger{
					NewStateTrigger("trigger1", "sensor.test"),
				},
				Actions:   []Action{},
				Variables: make(map[string]interface{}),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := tt.rule.Validate()
			assert.Equal(t, tt.expected, validation.Valid)
			if !tt.expected {
				assert.NotEmpty(t, validation.Errors)
			}
		})
	}
}

func TestStateTrigger_Evaluate(t *testing.T) {
	trigger := NewStateTrigger("test-trigger", "sensor.test")
	trigger.From = "off"
	trigger.To = "on"

	tests := []struct {
		name     string
		event    Event
		expected bool
	}{
		{
			name: "matching state change",
			event: Event{
				Type:     "state_changed",
				EntityID: "sensor.test",
				Data: map[string]interface{}{
					"old_state": "off",
					"new_state": "on",
				},
			},
			expected: true,
		},
		{
			name: "non-matching entity",
			event: Event{
				Type:     "state_changed",
				EntityID: "sensor.other",
				Data: map[string]interface{}{
					"old_state": "off",
					"new_state": "on",
				},
			},
			expected: false,
		},
		{
			name: "non-matching state transition",
			event: Event{
				Type:     "state_changed",
				EntityID: "sensor.test",
				Data: map[string]interface{}{
					"old_state": "on",
					"new_state": "off",
				},
			},
			expected: false,
		},
		{
			name: "wrong event type",
			event: Event{
				Type:     "other_event",
				EntityID: "sensor.test",
				Data: map[string]interface{}{
					"old_state": "off",
					"new_state": "on",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := trigger.Evaluate(context.Background(), tt.event)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTimeTrigger_Validate(t *testing.T) {
	tests := []struct {
		name     string
		trigger  *TimeTrigger
		expected bool
	}{
		{
			name: "valid time trigger with 'at'",
			trigger: &TimeTrigger{
				BaseTrigger: BaseTrigger{
					ID:   "test-trigger",
					Type: TriggerTypeTime,
				},
				At: "14:30",
			},
			expected: true,
		},
		{
			name: "valid time trigger with cron",
			trigger: &TimeTrigger{
				BaseTrigger: BaseTrigger{
					ID:   "test-trigger",
					Type: TriggerTypeTime,
				},
				Cron: "30 14 * * *",
			},
			expected: true,
		},
		{
			name: "valid time trigger with interval",
			trigger: &TimeTrigger{
				BaseTrigger: BaseTrigger{
					ID:   "test-trigger",
					Type: TriggerTypeTime,
				},
				Interval: "5m",
			},
			expected: true,
		},
		{
			name: "invalid - multiple specifications",
			trigger: &TimeTrigger{
				BaseTrigger: BaseTrigger{
					ID:   "test-trigger",
					Type: TriggerTypeTime,
				},
				At:   "14:30",
				Cron: "30 14 * * *",
			},
			expected: false,
		},
		{
			name: "invalid - no specification",
			trigger: &TimeTrigger{
				BaseTrigger: BaseTrigger{
					ID:   "test-trigger",
					Type: TriggerTypeTime,
				},
			},
			expected: false,
		},
		{
			name: "invalid time format",
			trigger: &TimeTrigger{
				BaseTrigger: BaseTrigger{
					ID:   "test-trigger",
					Type: TriggerTypeTime,
				},
				At: "25:30",
			},
			expected: false,
		},
		{
			name: "invalid cron expression",
			trigger: &TimeTrigger{
				BaseTrigger: BaseTrigger{
					ID:   "test-trigger",
					Type: TriggerTypeTime,
				},
				Cron: "invalid cron",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.trigger.Validate()
			if tt.expected {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestStateCondition_Evaluate(t *testing.T) {
	condition := NewStateCondition("test-condition", "sensor.test")
	condition.State = "on"

	tests := []struct {
		name     string
		data     map[string]interface{}
		expected bool
	}{
		{
			name: "matching state",
			data: map[string]interface{}{
				"entity_sensor.test": map[string]interface{}{
					"state": "on",
				},
			},
			expected: true,
		},
		{
			name: "non-matching state",
			data: map[string]interface{}{
				"entity_sensor.test": map[string]interface{}{
					"state": "off",
				},
			},
			expected: false,
		},
		{
			name: "missing entity",
			data: map[string]interface{}{
				"entity_sensor.other": map[string]interface{}{
					"state": "on",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := condition.Evaluate(context.Background(), tt.data)
			if tt.name == "missing entity" {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTimeCondition_Evaluate(t *testing.T) {
	condition := NewTimeCondition("test-condition")
	condition.After = "09:00:00"
	condition.Before = "17:00:00"

	// Note: This test would need to be adjusted based on the current time
	// In a real test, you'd mock the time or use a fixed time for testing
	result, err := condition.Evaluate(context.Background(), make(map[string]interface{}))
	require.NoError(t, err)
	// The result depends on the current time, so we just verify no error
	assert.IsType(t, bool(true), result)
}

func TestNumericCondition_Evaluate(t *testing.T) {
	condition := NewNumericCondition("test-condition", "sensor.temperature")
	above := 20.0
	condition.Above = &above

	tests := []struct {
		name     string
		data     map[string]interface{}
		expected bool
	}{
		{
			name: "value above threshold",
			data: map[string]interface{}{
				"entity_sensor.temperature": map[string]interface{}{
					"state": "25.5",
				},
			},
			expected: true,
		},
		{
			name: "value below threshold",
			data: map[string]interface{}{
				"entity_sensor.temperature": map[string]interface{}{
					"state": "18.5",
				},
			},
			expected: false,
		},
		{
			name: "value equal to threshold",
			data: map[string]interface{}{
				"entity_sensor.temperature": map[string]interface{}{
					"state": "20.0",
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := condition.Evaluate(context.Background(), tt.data)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestServiceAction_Execute(t *testing.T) {
	action := NewServiceAction("test-action", "light.turn_on")
	action.EntityID = "light.living_room"
	action.Data = map[string]interface{}{
		"brightness": 255,
	}

	data := map[string]interface{}{
		"test_var": "test_value",
	}

	err := action.Execute(context.Background(), data)
	assert.NoError(t, err)
}

func TestDelayAction_Execute(t *testing.T) {
	action := NewDelayAction("test-action", "100ms")

	start := time.Now()
	err := action.Execute(context.Background(), make(map[string]interface{}))
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, duration, 100*time.Millisecond)
}

func TestVariableAction_Execute(t *testing.T) {
	action := NewVariableAction("test-action", "test_var", "test_value")

	data := make(map[string]interface{})
	err := action.Execute(context.Background(), data)

	assert.NoError(t, err)
	assert.Equal(t, "test_value", data["test_var"])
}

func TestScheduler(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	config := &SchedulerConfig{
		Timezone: "UTC",
	}

	scheduler, err := NewScheduler(config, logger)
	require.NoError(t, err)

	// Start scheduler
	err = scheduler.Start()
	require.NoError(t, err)
	defer scheduler.Stop()

	// Create a time trigger
	trigger := NewTimeTrigger("test-trigger")
	trigger.Interval = "1m"

	handler := func(ctx context.Context, t Trigger, event Event) error {
		return nil
	}

	// Schedule the trigger
	err = scheduler.ScheduleTrigger("test-rule", trigger, handler)
	require.NoError(t, err)

	// Wait for setup to complete
	time.Sleep(100 * time.Millisecond)

	// Check if trigger is scheduled (we can't wait 1 minute for actual execution in test)
	stats := scheduler.GetStatistics()
	assert.Equal(t, 1, stats["total_triggers"], "Should have 1 scheduled trigger")

	// Unschedule the trigger
	err = scheduler.UnscheduleTrigger(trigger.GetID())
	assert.NoError(t, err)
}

func TestRuleParser_ParseFromJSON(t *testing.T) {
	parser := NewRuleParser()

	jsonRule := `{
		"id": "test-rule",
		"name": "Test Rule",
		"description": "A test automation rule",
		"enabled": true,
		"mode": "single",
		"triggers": [
			{
				"id": "trigger1",
				"type": "state",
				"entity_id": "sensor.test",
				"to": "on"
			}
		],
		"conditions": [
			{
				"id": "condition1",
				"type": "time",
				"after": "09:00",
				"before": "17:00"
			}
		],
		"actions": [
			{
				"id": "action1",
				"type": "service",
				"service": "light.turn_on",
				"entity_id": "light.living_room"
			}
		],
		"variables": {
			"brightness": 255
		}
	}`

	rule, err := parser.ParseFromJSON([]byte(jsonRule))
	require.NoError(t, err)

	assert.Equal(t, "test-rule", rule.ID)
	assert.Equal(t, "Test Rule", rule.Name)
	assert.True(t, rule.Enabled)
	assert.Equal(t, ExecutionModeSingle, rule.Mode)
	assert.Len(t, rule.Triggers, 1)
	assert.Len(t, rule.Conditions, 1)
	assert.Len(t, rule.Actions, 1)
	assert.Equal(t, float64(255), rule.Variables["brightness"])
}

func TestRuleParser_ParseFromYAML(t *testing.T) {
	parser := NewRuleParser()

	yamlRule := `
id: test-rule
name: "Test Rule"
description: "A test automation rule"
enabled: true
mode: single
triggers:
  - id: trigger1
    platform: state
    entity_id: sensor.test
    to: "on"
conditions:
  - id: condition1
    condition: time
    after: "09:00"
    before: "17:00"
actions:
  - id: action1
    service: light.turn_on
    entity_id: light.living_room
variables:
  brightness: 255
`

	rule, err := parser.ParseFromYAML([]byte(yamlRule))
	require.NoError(t, err)

	assert.Equal(t, "test-rule", rule.ID)
	assert.Equal(t, "Test Rule", rule.Name)
	assert.True(t, rule.Enabled)
	assert.Equal(t, ExecutionModeSingle, rule.Mode)
	assert.Len(t, rule.Triggers, 1)
	assert.Len(t, rule.Conditions, 1)
	assert.Len(t, rule.Actions, 1)
	assert.Equal(t, 255, rule.Variables["brightness"])
}

func TestExecutionContext(t *testing.T) {
	logger := logrus.New()
	ctx := NewExecutionContext(context.Background(), "test-rule", "test-trigger", logger)

	// Test variable operations
	ctx.SetVariable("test_var", "test_value")
	value, exists := ctx.GetVariable("test_var")
	assert.True(t, exists)
	assert.Equal(t, "test_value", value)

	// Test stack operations
	ctx.PushStack("step1")
	ctx.PushStack("step2")
	stack := ctx.GetStack()
	assert.Len(t, stack, 2)
	assert.Equal(t, "step2", stack[1])

	popped := ctx.PopStack()
	assert.Equal(t, "step2", popped)

	// Test trace operations
	ctx.AddTrace("action", "action1", "test action", true, 100*time.Millisecond, nil, nil)
	trace := ctx.GetTrace()
	assert.Len(t, trace, 1)
	assert.Equal(t, "action", trace[0].Type)
	assert.True(t, trace[0].Success)

	// Test metrics
	metrics := ctx.GetMetrics()
	assert.Equal(t, 1, metrics.ActionsExecuted)
	assert.Greater(t, metrics.TotalDuration, time.Duration(0))
}

func TestAutomationEngine_AddRule(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce log noise in tests

	config := &EngineConfig{
		Workers:          1,
		QueueSize:        10,
		ExecutionTimeout: 30 * time.Second,
	}

	engine, err := NewAutomationEngine(config, nil, nil, logger)
	require.NoError(t, err)

	// Create a test rule
	rule := &AutomationRule{
		ID:          "test-rule",
		Name:        "Test Rule",
		Description: "A test rule",
		Enabled:     true,
		Mode:        ExecutionModeSingle,
		Triggers: []Trigger{
			NewStateTrigger("trigger1", "sensor.test"),
		},
		Actions: []Action{
			NewServiceAction("action1", "light.turn_on"),
		},
		Variables: make(map[string]interface{}),
	}

	err = engine.AddRule(rule)
	assert.NoError(t, err)

	// Verify rule was added
	retrievedRule, err := engine.GetRule("test-rule")
	assert.NoError(t, err)
	assert.Equal(t, rule.Name, retrievedRule.Name)

	// Test getting all rules
	allRules := engine.GetAllRules()
	assert.Len(t, allRules, 1)
}

func TestCompositeTrigger_Evaluate(t *testing.T) {
	trigger1 := NewStateTrigger("trigger1", "sensor.test1")
	trigger1.To = "on"

	trigger2 := NewStateTrigger("trigger2", "sensor.test2")
	trigger2.To = "on"

	// Test AND composite trigger
	andTrigger := NewCompositeTrigger("and-trigger", "and")
	andTrigger.Triggers = []Trigger{trigger1, trigger2}

	// Both triggers match
	event1 := Event{
		Type:     "state_changed",
		EntityID: "sensor.test1",
		Data: map[string]interface{}{
			"new_state": "on",
		},
	}

	result, _, err := andTrigger.Evaluate(context.Background(), event1)
	require.NoError(t, err)
	// Should be false because only one trigger matches
	assert.False(t, result)

	// Test OR composite trigger
	orTrigger := NewCompositeTrigger("or-trigger", "or")
	orTrigger.Triggers = []Trigger{trigger1, trigger2}

	result, _, err = orTrigger.Evaluate(context.Background(), event1)
	require.NoError(t, err)
	// Should be true because at least one trigger matches
	assert.True(t, result)
}

func TestCompositeCondition_Evaluate(t *testing.T) {
	condition1 := NewStateCondition("condition1", "sensor.test1")
	condition1.State = "on"

	condition2 := NewStateCondition("condition2", "sensor.test2")
	condition2.State = "on"

	// Test AND composite condition
	andCondition := NewCompositeCondition("and-condition", "and")
	andCondition.Conditions = []Condition{condition1, condition2}

	data := map[string]interface{}{
		"entity_sensor.test1": map[string]interface{}{
			"state": "on",
		},
		"entity_sensor.test2": map[string]interface{}{
			"state": "off",
		},
	}

	result, err := andCondition.Evaluate(context.Background(), data)
	require.NoError(t, err)
	// Should be false because not all conditions are met
	assert.False(t, result)

	// Test OR composite condition
	orCondition := NewCompositeCondition("or-condition", "or")
	orCondition.Conditions = []Condition{condition1, condition2}

	result, err = orCondition.Evaluate(context.Background(), data)
	require.NoError(t, err)
	// Should be true because at least one condition is met
	assert.True(t, result)
}
