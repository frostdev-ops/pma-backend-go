package automation

import (
	"encoding/json"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// RuleParser handles parsing automation rules from different formats
type RuleParser struct {
	triggerFactory   *TriggerFactory
	conditionFactory *ConditionFactory
	actionFactory    *ActionFactory
}

// NewRuleParser creates a new rule parser
func NewRuleParser() *RuleParser {
	return &RuleParser{
		triggerFactory:   &TriggerFactory{},
		conditionFactory: &ConditionFactory{},
		actionFactory:    &ActionFactory{},
	}
}

// ParseFromYAML parses an automation rule from YAML
func (rp *RuleParser) ParseFromYAML(yamlData []byte) (*AutomationRule, error) {
	var rawRule map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &rawRule); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v", err)
	}

	return rp.parseFromMap(rawRule)
}

// ParseFromJSON parses an automation rule from JSON
func (rp *RuleParser) ParseFromJSON(jsonData []byte) (*AutomationRule, error) {
	var rawRule map[string]interface{}
	if err := json.Unmarshal(jsonData, &rawRule); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	return rp.parseFromMap(rawRule)
}

// parseFromMap parses an automation rule from a map
func (rp *RuleParser) parseFromMap(rawRule map[string]interface{}) (*AutomationRule, error) {
	rule := &AutomationRule{
		Variables: make(map[string]interface{}),
	}

	// Parse basic fields
	if id, ok := rawRule["id"].(string); ok {
		rule.ID = id
	}

	if name, ok := rawRule["name"].(string); ok {
		rule.Name = name
	} else {
		return nil, fmt.Errorf("rule name is required")
	}

	if description, ok := rawRule["description"].(string); ok {
		rule.Description = description
	}

	if enabled, ok := rawRule["enabled"].(bool); ok {
		rule.Enabled = enabled
	} else {
		rule.Enabled = true // Default to enabled
	}

	if mode, ok := rawRule["mode"].(string); ok {
		rule.Mode = ExecutionMode(mode)
	} else {
		rule.Mode = ExecutionModeSingle // Default mode
	}

	// Parse variables
	if variables, ok := rawRule["variables"].(map[string]interface{}); ok {
		rule.Variables = variables
	}

	// Parse triggers
	triggersData, ok := rawRule["triggers"]
	if !ok {
		return nil, fmt.Errorf("triggers are required")
	}

	triggers, err := rp.parseTriggers(triggersData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse triggers: %v", err)
	}
	rule.Triggers = triggers

	// Parse conditions (optional)
	if conditionsData, ok := rawRule["conditions"]; ok {
		conditions, err := rp.parseConditions(conditionsData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse conditions: %v", err)
		}
		rule.Conditions = conditions
	}

	// Parse actions
	actionsData, ok := rawRule["actions"]
	if !ok {
		return nil, fmt.Errorf("actions are required")
	}

	actions, err := rp.parseActions(actionsData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse actions: %v", err)
	}
	rule.Actions = actions

	return rule, nil
}

// parseTriggers parses trigger configurations
func (rp *RuleParser) parseTriggers(triggersData interface{}) ([]Trigger, error) {
	triggersList, ok := triggersData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("triggers must be an array")
	}

	var triggers []Trigger
	for i, triggerData := range triggersList {
		triggerMap, ok := triggerData.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("trigger %d must be an object", i)
		}

		// Ensure trigger has an ID
		if _, hasID := triggerMap["id"]; !hasID {
			triggerMap["id"] = fmt.Sprintf("trigger_%d", i)
		}

		trigger, err := rp.parseSingleTrigger(triggerMap)
		if err != nil {
			return nil, fmt.Errorf("trigger %d: %v", i, err)
		}

		triggers = append(triggers, trigger)
	}

	return triggers, nil
}

// parseSingleTrigger parses a single trigger configuration
func (rp *RuleParser) parseSingleTrigger(triggerMap map[string]interface{}) (Trigger, error) {
	// Handle different trigger formats
	platform, ok := triggerMap["platform"].(string)
	if ok {
		// Home Assistant style trigger
		return rp.parseHomeAssistantTrigger(platform, triggerMap)
	}

	if _, ok := triggerMap["type"]; ok {
		// Standard trigger format
		return rp.triggerFactory.CreateTrigger(triggerMap)
	}

	return nil, fmt.Errorf("trigger must have either 'platform' or 'type' field")
}

// parseHomeAssistantTrigger parses Home Assistant style triggers
func (rp *RuleParser) parseHomeAssistantTrigger(platform string, triggerMap map[string]interface{}) (Trigger, error) {
	id, _ := triggerMap["id"].(string)
	if id == "" {
		id = fmt.Sprintf("ha_%s_trigger", platform)
	}

	switch platform {
	case "state":
		entityID, ok := triggerMap["entity_id"].(string)
		if !ok {
			return nil, fmt.Errorf("entity_id is required for state trigger")
		}

		trigger := NewStateTrigger(id, entityID)

		if from, exists := triggerMap["from"]; exists {
			trigger.From = from
		}
		if to, exists := triggerMap["to"]; exists {
			trigger.To = to
		}
		if attr, exists := triggerMap["attribute"].(string); exists {
			trigger.Attribute = attr
		}

		return trigger, nil

	case "time":
		trigger := NewTimeTrigger(id)

		if at, exists := triggerMap["at"].(string); exists {
			trigger.At = at
		}

		return trigger, nil

	case "sun":
		trigger := NewEventTrigger(id, "sun_event")

		event, ok := triggerMap["event"].(string)
		if !ok {
			return nil, fmt.Errorf("event is required for sun trigger")
		}

		trigger.EventData = map[string]interface{}{
			"event": event,
		}

		if offset, exists := triggerMap["offset"].(string); exists {
			trigger.EventData["offset"] = offset
		}

		return trigger, nil

	case "webhook":
		webhookID, ok := triggerMap["webhook_id"].(string)
		if !ok {
			return nil, fmt.Errorf("webhook_id is required for webhook trigger")
		}

		trigger := NewWebhookTrigger(id, webhookID)

		if method, exists := triggerMap["method"].(string); exists {
			trigger.Method = method
		}

		return trigger, nil

	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

// parseConditions parses condition configurations
func (rp *RuleParser) parseConditions(conditionsData interface{}) ([]Condition, error) {
	conditionsList, ok := conditionsData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("conditions must be an array")
	}

	var conditions []Condition
	for i, conditionData := range conditionsList {
		conditionMap, ok := conditionData.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("condition %d must be an object", i)
		}

		// Ensure condition has an ID
		if _, hasID := conditionMap["id"]; !hasID {
			conditionMap["id"] = fmt.Sprintf("condition_%d", i)
		}

		condition, err := rp.parseSingleCondition(conditionMap)
		if err != nil {
			return nil, fmt.Errorf("condition %d: %v", i, err)
		}

		conditions = append(conditions, condition)
	}

	return conditions, nil
}

// parseSingleCondition parses a single condition configuration
func (rp *RuleParser) parseSingleCondition(conditionMap map[string]interface{}) (Condition, error) {
	// Handle different condition formats
	conditionType, ok := conditionMap["condition"].(string)
	if ok {
		// Home Assistant style condition
		return rp.parseHomeAssistantCondition(conditionType, conditionMap)
	}

	// Standard condition format
	if _, hasType := conditionMap["type"]; hasType {
		return rp.conditionFactory.CreateCondition(conditionMap)
	}

	return nil, fmt.Errorf("condition must have either 'condition' or 'type' field")
}

// parseHomeAssistantCondition parses Home Assistant style conditions
func (rp *RuleParser) parseHomeAssistantCondition(conditionType string, conditionMap map[string]interface{}) (Condition, error) {
	id, _ := conditionMap["id"].(string)
	if id == "" {
		id = fmt.Sprintf("ha_%s_condition", conditionType)
	}

	switch conditionType {
	case "state":
		entityID, ok := conditionMap["entity_id"].(string)
		if !ok {
			return nil, fmt.Errorf("entity_id is required for state condition")
		}

		condition := NewStateCondition(id, entityID)

		if state, exists := conditionMap["state"]; exists {
			condition.State = state
		}
		if attr, exists := conditionMap["attribute"].(string); exists {
			condition.Attribute = attr
		}

		return condition, nil

	case "time":
		condition := NewTimeCondition(id)

		if before, exists := conditionMap["before"].(string); exists {
			condition.Before = before
		}
		if after, exists := conditionMap["after"].(string); exists {
			condition.After = after
		}
		if weekdays, exists := conditionMap["weekday"].([]interface{}); exists {
			condition.Weekdays = make([]string, len(weekdays))
			for i, day := range weekdays {
				condition.Weekdays[i] = day.(string)
			}
		}

		return condition, nil

	case "numeric_state":
		entityID, ok := conditionMap["entity_id"].(string)
		if !ok {
			return nil, fmt.Errorf("entity_id is required for numeric_state condition")
		}

		condition := NewNumericCondition(id, entityID)

		if above, exists := conditionMap["above"].(float64); exists {
			condition.Above = &above
		}
		if below, exists := conditionMap["below"].(float64); exists {
			condition.Below = &below
		}
		if attr, exists := conditionMap["attribute"].(string); exists {
			condition.Attribute = attr
		}

		return condition, nil

	case "template":
		template, ok := conditionMap["value_template"].(string)
		if !ok {
			return nil, fmt.Errorf("value_template is required for template condition")
		}

		return NewTemplateCondition(id, template), nil

	case "and":
		conditions, ok := conditionMap["conditions"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("conditions are required for 'and' condition")
		}

		compositeCondition := NewCompositeCondition(id, "and")

		for i, condData := range conditions {
			condMap, ok := condData.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("condition %d must be an object", i)
			}

			if _, hasID := condMap["id"]; !hasID {
				condMap["id"] = fmt.Sprintf("%s_sub_%d", id, i)
			}

			subCondition, err := rp.parseSingleCondition(condMap)
			if err != nil {
				return nil, fmt.Errorf("sub-condition %d: %v", i, err)
			}

			compositeCondition.Conditions = append(compositeCondition.Conditions, subCondition)
		}

		return compositeCondition, nil

	case "or":
		conditions, ok := conditionMap["conditions"].([]interface{})
		if !ok {
			return nil, fmt.Errorf("conditions are required for 'or' condition")
		}

		compositeCondition := NewCompositeCondition(id, "or")

		for i, condData := range conditions {
			condMap, ok := condData.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("condition %d must be an object", i)
			}

			if _, hasID := condMap["id"]; !hasID {
				condMap["id"] = fmt.Sprintf("%s_sub_%d", id, i)
			}

			subCondition, err := rp.parseSingleCondition(condMap)
			if err != nil {
				return nil, fmt.Errorf("sub-condition %d: %v", i, err)
			}

			compositeCondition.Conditions = append(compositeCondition.Conditions, subCondition)
		}

		return compositeCondition, nil

	default:
		return nil, fmt.Errorf("unsupported condition type: %s", conditionType)
	}
}

// parseActions parses action configurations
func (rp *RuleParser) parseActions(actionsData interface{}) ([]Action, error) {
	actionsList, ok := actionsData.([]interface{})
	if !ok {
		return nil, fmt.Errorf("actions must be an array")
	}

	var actions []Action
	for i, actionData := range actionsList {
		actionMap, ok := actionData.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("action %d must be an object", i)
		}

		// Ensure action has an ID
		if _, hasID := actionMap["id"]; !hasID {
			actionMap["id"] = fmt.Sprintf("action_%d", i)
		}

		action, err := rp.parseSingleAction(actionMap)
		if err != nil {
			return nil, fmt.Errorf("action %d: %v", i, err)
		}

		actions = append(actions, action)
	}

	return actions, nil
}

// parseSingleAction parses a single action configuration
func (rp *RuleParser) parseSingleAction(actionMap map[string]interface{}) (Action, error) {
	// Handle different action formats
	if _, ok := actionMap["service"]; ok {
		// Home Assistant style service call
		return rp.parseServiceAction(actionMap)
	}

	// Standard action format
	if _, hasType := actionMap["type"]; hasType {
		return rp.actionFactory.CreateAction(actionMap)
	}

	// Check for other Home Assistant action types
	if _, hasDelay := actionMap["delay"]; hasDelay {
		return rp.parseDelayAction(actionMap)
	}

	return nil, fmt.Errorf("action must have 'service', 'type', or 'delay' field")
}

// parseServiceAction parses a Home Assistant style service action
func (rp *RuleParser) parseServiceAction(actionMap map[string]interface{}) (Action, error) {
	service, ok := actionMap["service"].(string)
	if !ok {
		return nil, fmt.Errorf("service is required for service action")
	}

	id, _ := actionMap["id"].(string)
	if id == "" {
		id = fmt.Sprintf("service_%s", strings.ReplaceAll(service, ".", "_"))
	}

	action := NewServiceAction(id, service)

	if entityID, exists := actionMap["entity_id"].(string); exists {
		action.EntityID = entityID
	}

	if data, exists := actionMap["data"].(map[string]interface{}); exists {
		action.Data = data
	}

	if target, exists := actionMap["target"].(map[string]interface{}); exists {
		action.Target = target
	}

	return action, nil
}

// parseDelayAction parses a delay action
func (rp *RuleParser) parseDelayAction(actionMap map[string]interface{}) (Action, error) {
	id, _ := actionMap["id"].(string)
	if id == "" {
		id = "delay_action"
	}

	delay, ok := actionMap["delay"].(string)
	if !ok {
		// Handle different delay formats
		if delayMap, isMap := actionMap["delay"].(map[string]interface{}); isMap {
			delay = rp.parseDelayFromMap(delayMap)
		} else {
			return nil, fmt.Errorf("invalid delay format")
		}
	}

	return NewDelayAction(id, delay), nil
}

// parseDelayFromMap parses delay from a map format (e.g., {hours: 1, minutes: 30})
func (rp *RuleParser) parseDelayFromMap(delayMap map[string]interface{}) string {
	var parts []string

	if hours, ok := delayMap["hours"].(float64); ok && hours > 0 {
		parts = append(parts, fmt.Sprintf("%.0fh", hours))
	}
	if minutes, ok := delayMap["minutes"].(float64); ok && minutes > 0 {
		parts = append(parts, fmt.Sprintf("%.0fm", minutes))
	}
	if seconds, ok := delayMap["seconds"].(float64); ok && seconds > 0 {
		parts = append(parts, fmt.Sprintf("%.0fs", seconds))
	}

	if len(parts) == 0 {
		return "1s" // Default delay
	}

	return strings.Join(parts, "")
}

// SerializeToYAML serializes an automation rule to YAML
func (rp *RuleParser) SerializeToYAML(rule *AutomationRule) ([]byte, error) {
	ruleMap, err := rp.serializeToMap(rule)
	if err != nil {
		return nil, err
	}

	return yaml.Marshal(ruleMap)
}

// SerializeToJSON serializes an automation rule to JSON
func (rp *RuleParser) SerializeToJSON(rule *AutomationRule) ([]byte, error) {
	ruleMap, err := rp.serializeToMap(rule)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(ruleMap, "", "  ")
}

// serializeToMap converts an automation rule to a map
func (rp *RuleParser) serializeToMap(rule *AutomationRule) (map[string]interface{}, error) {
	ruleMap := map[string]interface{}{
		"id":          rule.ID,
		"name":        rule.Name,
		"description": rule.Description,
		"enabled":     rule.Enabled,
		"mode":        string(rule.Mode),
	}

	if len(rule.Variables) > 0 {
		ruleMap["variables"] = rule.Variables
	}

	// Serialize triggers
	triggers := make([]interface{}, len(rule.Triggers))
	for i, trigger := range rule.Triggers {
		triggerData, _ := json.Marshal(trigger)
		var triggerMap map[string]interface{}
		json.Unmarshal(triggerData, &triggerMap)
		triggers[i] = triggerMap
	}
	ruleMap["triggers"] = triggers

	// Serialize conditions
	if len(rule.Conditions) > 0 {
		conditions := make([]interface{}, len(rule.Conditions))
		for i, condition := range rule.Conditions {
			conditionData, _ := json.Marshal(condition)
			var conditionMap map[string]interface{}
			json.Unmarshal(conditionData, &conditionMap)
			conditions[i] = conditionMap
		}
		ruleMap["conditions"] = conditions
	}

	// Serialize actions
	actions := make([]interface{}, len(rule.Actions))
	for i, action := range rule.Actions {
		actionData, _ := json.Marshal(action)
		var actionMap map[string]interface{}
		json.Unmarshal(actionData, &actionMap)
		actions[i] = actionMap
	}
	ruleMap["actions"] = actions

	return ruleMap, nil
}

// ValidateRuleSyntax validates rule syntax without creating the full rule
func (rp *RuleParser) ValidateRuleSyntax(data []byte, format string) *RuleValidationResult {
	var rawRule map[string]interface{}
	var err error

	switch strings.ToLower(format) {
	case "yaml", "yml":
		err = yaml.Unmarshal(data, &rawRule)
	case "json":
		err = json.Unmarshal(data, &rawRule)
	default:
		return &RuleValidationResult{
			Valid: false,
			Errors: []RuleValidationError{{
				Field:   "format",
				Message: "unsupported format, use 'yaml' or 'json'",
			}},
		}
	}

	if err != nil {
		return &RuleValidationResult{
			Valid: false,
			Errors: []RuleValidationError{{
				Field:   "syntax",
				Message: fmt.Sprintf("parse error: %v", err),
			}},
		}
	}

	// Try to parse the rule
	rule, err := rp.parseFromMap(rawRule)
	if err != nil {
		return &RuleValidationResult{
			Valid: false,
			Errors: []RuleValidationError{{
				Field:   "structure",
				Message: err.Error(),
			}},
		}
	}

	// Validate the parsed rule
	return rule.Validate()
}
