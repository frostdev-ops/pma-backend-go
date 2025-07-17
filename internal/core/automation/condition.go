package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Condition interface defines condition behavior
type Condition interface {
	GetType() string
	GetID() string
	Evaluate(ctx context.Context, data map[string]interface{}) (bool, error)
	Validate() error
	Clone() Condition
}

// ConditionType represents different condition types
type ConditionType string

const (
	ConditionTypeState     ConditionType = "state"
	ConditionTypeTime      ConditionType = "time"
	ConditionTypeNumeric   ConditionType = "numeric"
	ConditionTypeTemplate  ConditionType = "template"
	ConditionTypeHistory   ConditionType = "history"
	ConditionTypeComposite ConditionType = "composite"
	ConditionTypeDevice    ConditionType = "device"
)

// BaseCondition provides common condition functionality
type BaseCondition struct {
	ID          string                 `json:"id"`
	Type        ConditionType          `json:"type"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Enabled     bool                   `json:"enabled"`
}

func (bc *BaseCondition) GetID() string {
	return bc.ID
}

func (bc *BaseCondition) GetType() string {
	return string(bc.Type)
}

func (bc *BaseCondition) Validate() error {
	if bc.ID == "" {
		return fmt.Errorf("condition ID is required")
	}
	if bc.Type == "" {
		return fmt.Errorf("condition type is required")
	}
	return nil
}

// StateCondition checks entity state
type StateCondition struct {
	BaseCondition
	EntityID  string      `json:"entity_id"`
	State     interface{} `json:"state,omitempty"`
	Attribute string      `json:"attribute,omitempty"`
	For       *Duration   `json:"for,omitempty"`
}

func NewStateCondition(id, entityID string) *StateCondition {
	return &StateCondition{
		BaseCondition: BaseCondition{
			ID:      id,
			Type:    ConditionTypeState,
			Enabled: true,
		},
		EntityID: entityID,
	}
}

func (sc *StateCondition) Evaluate(ctx context.Context, data map[string]interface{}) (bool, error) {
	if !sc.Enabled {
		return true, nil // Disabled conditions are considered true
	}

	// Get entity state from context data
	entityKey := fmt.Sprintf("entity_%s", sc.EntityID)
	entityData, exists := data[entityKey]
	if !exists {
		// Try to get from Home Assistant client
		// This would need to be implemented with actual HA client
		return false, fmt.Errorf("entity %s not found in context", sc.EntityID)
	}

	entityMap, ok := entityData.(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("invalid entity data format")
	}

	var currentValue interface{}
	if sc.Attribute != "" {
		// Check attribute value
		attributes, ok := entityMap["attributes"].(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("entity attributes not found")
		}
		currentValue = attributes[sc.Attribute]
	} else {
		// Check state value
		currentValue = entityMap["state"]
	}

	// Compare state
	if sc.State != nil {
		return compareValues(currentValue, sc.State), nil
	}

	// If no specific state is specified, just check if entity exists
	return true, nil
}

func (sc *StateCondition) Clone() Condition {
	data, _ := json.Marshal(sc)
	var clone StateCondition
	json.Unmarshal(data, &clone)
	return &clone
}

func (sc *StateCondition) Validate() error {
	if err := sc.BaseCondition.Validate(); err != nil {
		return err
	}
	if sc.EntityID == "" {
		return fmt.Errorf("entity_id is required for state condition")
	}
	return nil
}

// TimeCondition checks time-based conditions
type TimeCondition struct {
	BaseCondition
	Before    string   `json:"before,omitempty"`     // Time of day (HH:MM:SS)
	After     string   `json:"after,omitempty"`      // Time of day (HH:MM:SS)
	Weekdays  []string `json:"weekdays,omitempty"`   // ["mon", "tue", "wed", ...]
	StartDate string   `json:"start_date,omitempty"` // Date (YYYY-MM-DD)
	EndDate   string   `json:"end_date,omitempty"`   // Date (YYYY-MM-DD)
}

func NewTimeCondition(id string) *TimeCondition {
	return &TimeCondition{
		BaseCondition: BaseCondition{
			ID:      id,
			Type:    ConditionTypeTime,
			Enabled: true,
		},
	}
}

func (tc *TimeCondition) Evaluate(ctx context.Context, data map[string]interface{}) (bool, error) {
	if !tc.Enabled {
		return true, nil
	}

	now := time.Now()

	// Check time range
	if tc.Before != "" || tc.After != "" {
		currentTime := now.Format("15:04:05")

		if tc.After != "" && currentTime < tc.After {
			return false, nil
		}
		if tc.Before != "" && currentTime > tc.Before {
			return false, nil
		}
	}

	// Check weekdays
	if len(tc.Weekdays) > 0 {
		currentWeekday := strings.ToLower(now.Weekday().String()[:3])
		found := false
		for _, day := range tc.Weekdays {
			if strings.ToLower(day) == currentWeekday {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}

	// Check date range
	if tc.StartDate != "" || tc.EndDate != "" {
		currentDate := now.Format("2006-01-02")

		if tc.StartDate != "" && currentDate < tc.StartDate {
			return false, nil
		}
		if tc.EndDate != "" && currentDate > tc.EndDate {
			return false, nil
		}
	}

	return true, nil
}

func (tc *TimeCondition) Clone() Condition {
	data, _ := json.Marshal(tc)
	var clone TimeCondition
	json.Unmarshal(data, &clone)
	return &clone
}

func (tc *TimeCondition) Validate() error {
	if err := tc.BaseCondition.Validate(); err != nil {
		return err
	}

	// Validate time formats
	if tc.Before != "" && !isValidTimeFormat(tc.Before) {
		return fmt.Errorf("invalid before time format: %s", tc.Before)
	}
	if tc.After != "" && !isValidTimeFormat(tc.After) {
		return fmt.Errorf("invalid after time format: %s", tc.After)
	}

	// Validate date formats
	if tc.StartDate != "" {
		if _, err := time.Parse("2006-01-02", tc.StartDate); err != nil {
			return fmt.Errorf("invalid start_date format: %s", tc.StartDate)
		}
	}
	if tc.EndDate != "" {
		if _, err := time.Parse("2006-01-02", tc.EndDate); err != nil {
			return fmt.Errorf("invalid end_date format: %s", tc.EndDate)
		}
	}

	return nil
}

// NumericCondition performs numeric comparisons
type NumericCondition struct {
	BaseCondition
	EntityID  string      `json:"entity_id"`
	Attribute string      `json:"attribute,omitempty"`
	Above     *float64    `json:"above,omitempty"`
	Below     *float64    `json:"below,omitempty"`
	Equals    *float64    `json:"equals,omitempty"`
	NotEquals *float64    `json:"not_equals,omitempty"`
	Value     interface{} `json:"value,omitempty"` // For direct value comparison
}

func NewNumericCondition(id, entityID string) *NumericCondition {
	return &NumericCondition{
		BaseCondition: BaseCondition{
			ID:      id,
			Type:    ConditionTypeNumeric,
			Enabled: true,
		},
		EntityID: entityID,
	}
}

func (nc *NumericCondition) Evaluate(ctx context.Context, data map[string]interface{}) (bool, error) {
	if !nc.Enabled {
		return true, nil
	}

	var value float64
	var err error

	if nc.Value != nil {
		// Use direct value
		value, err = convertToFloat(nc.Value)
		if err != nil {
			return false, fmt.Errorf("invalid value: %v", nc.Value)
		}
	} else {
		// Get value from entity
		entityKey := fmt.Sprintf("entity_%s", nc.EntityID)
		entityData, exists := data[entityKey]
		if !exists {
			return false, fmt.Errorf("entity %s not found in context", nc.EntityID)
		}

		entityMap, ok := entityData.(map[string]interface{})
		if !ok {
			return false, fmt.Errorf("invalid entity data format")
		}

		var rawValue interface{}
		if nc.Attribute != "" {
			attributes, ok := entityMap["attributes"].(map[string]interface{})
			if !ok {
				return false, fmt.Errorf("entity attributes not found")
			}
			rawValue = attributes[nc.Attribute]
		} else {
			rawValue = entityMap["state"]
		}

		value, err = convertToFloat(rawValue)
		if err != nil {
			return false, fmt.Errorf("cannot convert value to number: %v", rawValue)
		}
	}

	// Perform comparisons
	if nc.Above != nil && value <= *nc.Above {
		return false, nil
	}
	if nc.Below != nil && value >= *nc.Below {
		return false, nil
	}
	if nc.Equals != nil && value != *nc.Equals {
		return false, nil
	}
	if nc.NotEquals != nil && value == *nc.NotEquals {
		return false, nil
	}

	return true, nil
}

func (nc *NumericCondition) Clone() Condition {
	data, _ := json.Marshal(nc)
	var clone NumericCondition
	json.Unmarshal(data, &clone)
	return &clone
}

func (nc *NumericCondition) Validate() error {
	if err := nc.BaseCondition.Validate(); err != nil {
		return err
	}
	if nc.EntityID == "" && nc.Value == nil {
		return fmt.Errorf("either entity_id or value is required for numeric condition")
	}
	return nil
}

// TemplateCondition evaluates template expressions
type TemplateCondition struct {
	BaseCondition
	ValueTemplate string `json:"value_template"`
}

func NewTemplateCondition(id, template string) *TemplateCondition {
	return &TemplateCondition{
		BaseCondition: BaseCondition{
			ID:      id,
			Type:    ConditionTypeTemplate,
			Enabled: true,
		},
		ValueTemplate: template,
	}
}

func (tc *TemplateCondition) Evaluate(ctx context.Context, data map[string]interface{}) (bool, error) {
	if !tc.Enabled {
		return true, nil
	}

	// This is a simplified template evaluation
	// In a real implementation, you'd use a proper template engine
	result, err := evaluateSimpleTemplate(tc.ValueTemplate, data)
	if err != nil {
		return false, err
	}

	// Convert result to boolean
	switch v := result.(type) {
	case bool:
		return v, nil
	case string:
		return strings.ToLower(v) == "true", nil
	case float64:
		return v != 0, nil
	case int:
		return v != 0, nil
	default:
		return false, fmt.Errorf("template result cannot be converted to boolean: %v", result)
	}
}

func (tc *TemplateCondition) Clone() Condition {
	data, _ := json.Marshal(tc)
	var clone TemplateCondition
	json.Unmarshal(data, &clone)
	return &clone
}

func (tc *TemplateCondition) Validate() error {
	if err := tc.BaseCondition.Validate(); err != nil {
		return err
	}
	if tc.ValueTemplate == "" {
		return fmt.Errorf("value_template is required for template condition")
	}
	return nil
}

// CompositeCondition combines multiple conditions with AND/OR logic
type CompositeCondition struct {
	BaseCondition
	Operator   string      `json:"operator"` // "and" or "or"
	Conditions []Condition `json:"conditions"`
}

func NewCompositeCondition(id, operator string) *CompositeCondition {
	return &CompositeCondition{
		BaseCondition: BaseCondition{
			ID:      id,
			Type:    ConditionTypeComposite,
			Enabled: true,
		},
		Operator: operator,
	}
}

func (cc *CompositeCondition) Evaluate(ctx context.Context, data map[string]interface{}) (bool, error) {
	if !cc.Enabled {
		return true, nil
	}

	results := make([]bool, len(cc.Conditions))

	for i, condition := range cc.Conditions {
		result, err := condition.Evaluate(ctx, data)
		if err != nil {
			return false, err
		}
		results[i] = result
	}

	switch strings.ToLower(cc.Operator) {
	case "and":
		for _, result := range results {
			if !result {
				return false, nil
			}
		}
		return true, nil
	case "or":
		for _, result := range results {
			if result {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf("invalid operator: %s", cc.Operator)
	}
}

func (cc *CompositeCondition) Clone() Condition {
	clone := &CompositeCondition{
		BaseCondition: cc.BaseCondition,
		Operator:      cc.Operator,
		Conditions:    make([]Condition, len(cc.Conditions)),
	}

	for i, condition := range cc.Conditions {
		clone.Conditions[i] = condition.Clone()
	}

	return clone
}

func (cc *CompositeCondition) Validate() error {
	if err := cc.BaseCondition.Validate(); err != nil {
		return err
	}

	if cc.Operator != "and" && cc.Operator != "or" {
		return fmt.Errorf("operator must be 'and' or 'or'")
	}

	if len(cc.Conditions) < 2 {
		return fmt.Errorf("composite condition must have at least 2 conditions")
	}

	for i, condition := range cc.Conditions {
		if err := condition.Validate(); err != nil {
			return fmt.Errorf("condition %d: %v", i, err)
		}
	}

	return nil
}

// Helper functions
func convertToFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

func evaluateSimpleTemplate(template string, data map[string]interface{}) (interface{}, error) {
	// This is a very simple template evaluator
	// In production, you'd want to use a proper template engine like text/template

	// Handle simple variable substitution like {{ entity_light_living_room.state }}
	re := regexp.MustCompile(`\{\{\s*([^}]+)\s*\}\}`)
	matches := re.FindAllStringSubmatch(template, -1)

	result := template
	for _, match := range matches {
		if len(match) > 1 {
			path := strings.TrimSpace(match[1])
			value := getValueFromPath(data, path)
			result = strings.ReplaceAll(result, match[0], fmt.Sprintf("%v", value))
		}
	}

	// Try to parse as boolean
	if result == "true" {
		return true, nil
	}
	if result == "false" {
		return false, nil
	}

	// Try to parse as number
	if num, err := strconv.ParseFloat(result, 64); err == nil {
		return num, nil
	}

	// Return as string
	return result, nil
}

func getValueFromPath(data map[string]interface{}, path string) interface{} {
	// Handle paths like "entity_light_living_room.state" or "entity_light_living_room.attributes.brightness"
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if value, exists := current[part]; exists {
			if nextMap, ok := value.(map[string]interface{}); ok {
				current = nextMap
			} else {
				return value
			}
		} else {
			return nil
		}
	}

	return current
}

// ConditionFactory creates conditions from configuration
type ConditionFactory struct{}

func (cf *ConditionFactory) CreateCondition(config map[string]interface{}) (Condition, error) {
	conditionType, ok := config["type"].(string)
	if !ok {
		return nil, fmt.Errorf("condition type is required")
	}

	id, ok := config["id"].(string)
	if !ok {
		return nil, fmt.Errorf("condition id is required")
	}

	switch ConditionType(conditionType) {
	case ConditionTypeState:
		entityID, ok := config["entity_id"].(string)
		if !ok {
			return nil, fmt.Errorf("entity_id is required for state condition")
		}
		condition := NewStateCondition(id, entityID)
		if state, exists := config["state"]; exists {
			condition.State = state
		}
		if attr, exists := config["attribute"].(string); exists {
			condition.Attribute = attr
		}
		return condition, nil

	case ConditionTypeTime:
		condition := NewTimeCondition(id)
		if before, exists := config["before"].(string); exists {
			condition.Before = before
		}
		if after, exists := config["after"].(string); exists {
			condition.After = after
		}
		if weekdays, exists := config["weekdays"].([]interface{}); exists {
			condition.Weekdays = make([]string, len(weekdays))
			for i, day := range weekdays {
				condition.Weekdays[i] = day.(string)
			}
		}
		return condition, nil

	case ConditionTypeNumeric:
		entityID, ok := config["entity_id"].(string)
		if !ok {
			return nil, fmt.Errorf("entity_id is required for numeric condition")
		}
		condition := NewNumericCondition(id, entityID)
		if above, exists := config["above"].(float64); exists {
			condition.Above = &above
		}
		if below, exists := config["below"].(float64); exists {
			condition.Below = &below
		}
		return condition, nil

	case ConditionTypeTemplate:
		template, ok := config["value_template"].(string)
		if !ok {
			return nil, fmt.Errorf("value_template is required for template condition")
		}
		return NewTemplateCondition(id, template), nil

	default:
		return nil, fmt.Errorf("unsupported condition type: %s", conditionType)
	}
}
