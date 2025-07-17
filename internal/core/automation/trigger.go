package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// Event represents an automation event that can trigger rules
type Event struct {
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	EntityID  string                 `json:"entity_id,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// TriggerHandler is called when a trigger fires
type TriggerHandler func(ctx context.Context, trigger Trigger, event Event) error

// Trigger interface defines trigger behavior
type Trigger interface {
	GetType() string
	GetID() string
	Evaluate(ctx context.Context, event Event) (bool, map[string]interface{}, error)
	Subscribe(handler TriggerHandler) error
	Unsubscribe() error
	Validate() error
	Clone() Trigger
}

// TriggerType represents different trigger types
type TriggerType string

const (
	TriggerTypeState     TriggerType = "state"
	TriggerTypeTime      TriggerType = "time"
	TriggerTypeEvent     TriggerType = "event"
	TriggerTypeWebhook   TriggerType = "webhook"
	TriggerTypeSystem    TriggerType = "system"
	TriggerTypeComposite TriggerType = "composite"
	TriggerTypeManual    TriggerType = "manual"
)

// BaseTrigger provides common trigger functionality
type BaseTrigger struct {
	ID          string                 `json:"id"`
	Type        TriggerType            `json:"type"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Enabled     bool                   `json:"enabled"`

	handler TriggerHandler
}

func (bt *BaseTrigger) GetID() string {
	return bt.ID
}

func (bt *BaseTrigger) GetType() string {
	return string(bt.Type)
}

func (bt *BaseTrigger) Subscribe(handler TriggerHandler) error {
	bt.handler = handler
	return nil
}

func (bt *BaseTrigger) Unsubscribe() error {
	bt.handler = nil
	return nil
}

func (bt *BaseTrigger) Validate() error {
	if bt.ID == "" {
		return fmt.Errorf("trigger ID is required")
	}
	if bt.Type == "" {
		return fmt.Errorf("trigger type is required")
	}
	return nil
}

// StateTrigger triggers when entity state changes
type StateTrigger struct {
	BaseTrigger
	EntityID  string      `json:"entity_id"`
	From      interface{} `json:"from,omitempty"`
	To        interface{} `json:"to,omitempty"`
	Attribute string      `json:"attribute,omitempty"`
	For       *Duration   `json:"for,omitempty"`
}

func NewStateTrigger(id, entityID string) *StateTrigger {
	return &StateTrigger{
		BaseTrigger: BaseTrigger{
			ID:      id,
			Type:    TriggerTypeState,
			Enabled: true,
		},
		EntityID: entityID,
	}
}

func (st *StateTrigger) Evaluate(ctx context.Context, event Event) (bool, map[string]interface{}, error) {
	if !st.Enabled {
		return false, nil, nil
	}

	if event.Type != "state_changed" || event.EntityID != st.EntityID {
		return false, nil, nil
	}

	data := map[string]interface{}{
		"entity_id": event.EntityID,
		"event":     event.Data,
	}

	// Check state transition
	if st.From != nil || st.To != nil {
		oldState, _ := event.Data["old_state"]
		newState, _ := event.Data["new_state"]

		if st.From != nil && !compareValues(oldState, st.From) {
			return false, nil, nil
		}
		if st.To != nil && !compareValues(newState, st.To) {
			return false, nil, nil
		}
	}

	// TODO: Implement duration check for "for" field

	return true, data, nil
}

func (st *StateTrigger) Clone() Trigger {
	data, _ := json.Marshal(st)
	var clone StateTrigger
	json.Unmarshal(data, &clone)
	return &clone
}

func (st *StateTrigger) Validate() error {
	if err := st.BaseTrigger.Validate(); err != nil {
		return err
	}
	if st.EntityID == "" {
		return fmt.Errorf("entity_id is required for state trigger")
	}
	return nil
}

// TimeTrigger triggers at specific times
type TimeTrigger struct {
	BaseTrigger
	At       string `json:"at,omitempty"`       // Time of day (HH:MM:SS)
	Cron     string `json:"cron,omitempty"`     // Cron expression
	Interval string `json:"interval,omitempty"` // Interval (e.g., "5m", "1h")

	cronJob   cron.EntryID
	scheduler *cron.Cron
}

func NewTimeTrigger(id string) *TimeTrigger {
	return &TimeTrigger{
		BaseTrigger: BaseTrigger{
			ID:      id,
			Type:    TriggerTypeTime,
			Enabled: true,
		},
		scheduler: cron.New(cron.WithSeconds()),
	}
}

func (tt *TimeTrigger) Evaluate(ctx context.Context, event Event) (bool, map[string]interface{}, error) {
	if !tt.Enabled {
		return false, nil, nil
	}

	if event.Type != "time_trigger" {
		return false, nil, nil
	}

	data := map[string]interface{}{
		"trigger_time": time.Now(),
		"trigger_type": "time",
	}

	return true, data, nil
}

func (tt *TimeTrigger) Clone() Trigger {
	data, _ := json.Marshal(tt)
	var clone TimeTrigger
	json.Unmarshal(data, &clone)
	clone.scheduler = cron.New(cron.WithSeconds())
	return &clone
}

func (tt *TimeTrigger) Validate() error {
	if err := tt.BaseTrigger.Validate(); err != nil {
		return err
	}

	count := 0
	if tt.At != "" {
		count++
		if !isValidTimeFormat(tt.At) {
			return fmt.Errorf("invalid time format for 'at': %s", tt.At)
		}
	}
	if tt.Cron != "" {
		count++
		if _, err := cron.ParseStandard(tt.Cron); err != nil {
			return fmt.Errorf("invalid cron expression: %s", tt.Cron)
		}
	}
	if tt.Interval != "" {
		count++
		if _, err := time.ParseDuration(tt.Interval); err != nil {
			return fmt.Errorf("invalid interval: %s", tt.Interval)
		}
	}

	if count != 1 {
		return fmt.Errorf("exactly one of 'at', 'cron', or 'interval' must be specified")
	}

	return nil
}

// EventTrigger triggers on specific events
type EventTrigger struct {
	BaseTrigger
	EventType string                 `json:"event_type"`
	EventData map[string]interface{} `json:"event_data,omitempty"`
}

func NewEventTrigger(id, eventType string) *EventTrigger {
	return &EventTrigger{
		BaseTrigger: BaseTrigger{
			ID:      id,
			Type:    TriggerTypeEvent,
			Enabled: true,
		},
		EventType: eventType,
	}
}

func (et *EventTrigger) Evaluate(ctx context.Context, event Event) (bool, map[string]interface{}, error) {
	if !et.Enabled {
		return false, nil, nil
	}

	if event.Type != et.EventType {
		return false, nil, nil
	}

	// Check event data matches if specified
	if et.EventData != nil {
		for key, expectedValue := range et.EventData {
			if actualValue, exists := event.Data[key]; !exists || !compareValues(actualValue, expectedValue) {
				return false, nil, nil
			}
		}
	}

	data := map[string]interface{}{
		"event_type": event.Type,
		"event_data": event.Data,
	}

	return true, data, nil
}

func (et *EventTrigger) Clone() Trigger {
	data, _ := json.Marshal(et)
	var clone EventTrigger
	json.Unmarshal(data, &clone)
	return &clone
}

func (et *EventTrigger) Validate() error {
	if err := et.BaseTrigger.Validate(); err != nil {
		return err
	}
	if et.EventType == "" {
		return fmt.Errorf("event_type is required for event trigger")
	}
	return nil
}

// WebhookTrigger triggers on webhook calls
type WebhookTrigger struct {
	BaseTrigger
	WebhookID string            `json:"webhook_id"`
	Method    string            `json:"method,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

func NewWebhookTrigger(id, webhookID string) *WebhookTrigger {
	return &WebhookTrigger{
		BaseTrigger: BaseTrigger{
			ID:      id,
			Type:    TriggerTypeWebhook,
			Enabled: true,
		},
		WebhookID: webhookID,
		Method:    "POST",
	}
}

func (wt *WebhookTrigger) Evaluate(ctx context.Context, event Event) (bool, map[string]interface{}, error) {
	if !wt.Enabled {
		return false, nil, nil
	}

	if event.Type != "webhook" {
		return false, nil, nil
	}

	webhookID, _ := event.Data["webhook_id"].(string)
	if webhookID != wt.WebhookID {
		return false, nil, nil
	}

	method, _ := event.Data["method"].(string)
	if wt.Method != "" && method != wt.Method {
		return false, nil, nil
	}

	data := map[string]interface{}{
		"webhook_id": webhookID,
		"method":     method,
		"payload":    event.Data["payload"],
		"headers":    event.Data["headers"],
	}

	return true, data, nil
}

func (wt *WebhookTrigger) Clone() Trigger {
	data, _ := json.Marshal(wt)
	var clone WebhookTrigger
	json.Unmarshal(data, &clone)
	return &clone
}

func (wt *WebhookTrigger) Validate() error {
	if err := wt.BaseTrigger.Validate(); err != nil {
		return err
	}
	if wt.WebhookID == "" {
		return fmt.Errorf("webhook_id is required for webhook trigger")
	}
	return nil
}

// CompositeTrigger combines multiple triggers with AND/OR logic
type CompositeTrigger struct {
	BaseTrigger
	Operator string    `json:"operator"` // "and" or "or"
	Triggers []Trigger `json:"triggers"`
}

func NewCompositeTrigger(id, operator string) *CompositeTrigger {
	return &CompositeTrigger{
		BaseTrigger: BaseTrigger{
			ID:      id,
			Type:    TriggerTypeComposite,
			Enabled: true,
		},
		Operator: operator,
	}
}

func (ct *CompositeTrigger) Evaluate(ctx context.Context, event Event) (bool, map[string]interface{}, error) {
	if !ct.Enabled {
		return false, nil, nil
	}

	results := make([]bool, len(ct.Triggers))
	data := make(map[string]interface{})

	for i, trigger := range ct.Triggers {
		result, triggerData, err := trigger.Evaluate(ctx, event)
		if err != nil {
			return false, nil, err
		}
		results[i] = result

		// Merge trigger data
		for key, value := range triggerData {
			data[fmt.Sprintf("trigger_%d_%s", i, key)] = value
		}
	}

	var finalResult bool
	switch strings.ToLower(ct.Operator) {
	case "and":
		finalResult = true
		for _, result := range results {
			if !result {
				finalResult = false
				break
			}
		}
	case "or":
		finalResult = false
		for _, result := range results {
			if result {
				finalResult = true
				break
			}
		}
	default:
		return false, nil, fmt.Errorf("invalid operator: %s", ct.Operator)
	}

	return finalResult, data, nil
}

func (ct *CompositeTrigger) Clone() Trigger {
	clone := &CompositeTrigger{
		BaseTrigger: ct.BaseTrigger,
		Operator:    ct.Operator,
		Triggers:    make([]Trigger, len(ct.Triggers)),
	}

	for i, trigger := range ct.Triggers {
		clone.Triggers[i] = trigger.Clone()
	}

	return clone
}

func (ct *CompositeTrigger) Validate() error {
	if err := ct.BaseTrigger.Validate(); err != nil {
		return err
	}

	if ct.Operator != "and" && ct.Operator != "or" {
		return fmt.Errorf("operator must be 'and' or 'or'")
	}

	if len(ct.Triggers) < 2 {
		return fmt.Errorf("composite trigger must have at least 2 triggers")
	}

	for i, trigger := range ct.Triggers {
		if err := trigger.Validate(); err != nil {
			return fmt.Errorf("trigger %d: %v", i, err)
		}
	}

	return nil
}

// Duration represents a time duration with string parsing
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = duration
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration.String())
}

// Helper functions
func compareValues(actual, expected interface{}) bool {
	return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
}

func isValidTimeFormat(timeStr string) bool {
	// Support HH:MM and HH:MM:SS formats
	timeFormats := []string{"15:04", "15:04:05"}
	for _, format := range timeFormats {
		if _, err := time.Parse(format, timeStr); err == nil {
			return true
		}
	}
	return false
}

// TriggerFactory creates triggers from configuration
type TriggerFactory struct{}

func (tf *TriggerFactory) CreateTrigger(config map[string]interface{}) (Trigger, error) {
	triggerType, ok := config["type"].(string)
	if !ok {
		return nil, fmt.Errorf("trigger type is required")
	}

	id, ok := config["id"].(string)
	if !ok {
		return nil, fmt.Errorf("trigger id is required")
	}

	switch TriggerType(triggerType) {
	case TriggerTypeState:
		entityID, ok := config["entity_id"].(string)
		if !ok {
			return nil, fmt.Errorf("entity_id is required for state trigger")
		}
		trigger := NewStateTrigger(id, entityID)
		if from, exists := config["from"]; exists {
			trigger.From = from
		}
		if to, exists := config["to"]; exists {
			trigger.To = to
		}
		if attr, exists := config["attribute"].(string); exists {
			trigger.Attribute = attr
		}
		return trigger, nil

	case TriggerTypeTime:
		trigger := NewTimeTrigger(id)
		if at, exists := config["at"].(string); exists {
			trigger.At = at
		}
		if cron, exists := config["cron"].(string); exists {
			trigger.Cron = cron
		}
		if interval, exists := config["interval"].(string); exists {
			trigger.Interval = interval
		}
		return trigger, nil

	case TriggerTypeEvent:
		eventType, ok := config["event_type"].(string)
		if !ok {
			return nil, fmt.Errorf("event_type is required for event trigger")
		}
		trigger := NewEventTrigger(id, eventType)
		if eventData, exists := config["event_data"].(map[string]interface{}); exists {
			trigger.EventData = eventData
		}
		return trigger, nil

	case TriggerTypeWebhook:
		webhookID, ok := config["webhook_id"].(string)
		if !ok {
			return nil, fmt.Errorf("webhook_id is required for webhook trigger")
		}
		trigger := NewWebhookTrigger(id, webhookID)
		if method, exists := config["method"].(string); exists {
			trigger.Method = method
		}
		return trigger, nil

	default:
		return nil, fmt.Errorf("unsupported trigger type: %s", triggerType)
	}
}
