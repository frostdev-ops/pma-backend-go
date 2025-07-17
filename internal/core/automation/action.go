package automation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// Action interface defines action behavior
type Action interface {
	GetType() string
	GetID() string
	Execute(ctx context.Context, data map[string]interface{}) error
	Validate() error
	EstimateExecutionTime() time.Duration
	Clone() Action
}

// ActionType represents different action types
type ActionType string

const (
	ActionTypeService      ActionType = "service"
	ActionTypeNotification ActionType = "notification"
	ActionTypeDelay        ActionType = "delay"
	ActionTypeVariable     ActionType = "variable"
	ActionTypeHTTP         ActionType = "http"
	ActionTypeScript       ActionType = "script"
	ActionTypeConditional  ActionType = "conditional"
	ActionTypeWebSocket    ActionType = "websocket"
)

// BaseAction provides common action functionality
type BaseAction struct {
	ID          string                 `json:"id"`
	Type        ActionType             `json:"type"`
	Description string                 `json:"description"`
	Config      map[string]interface{} `json:"config"`
	Enabled     bool                   `json:"enabled"`
}

func (ba *BaseAction) GetID() string {
	return ba.ID
}

func (ba *BaseAction) GetType() string {
	return string(ba.Type)
}

func (ba *BaseAction) Validate() error {
	if ba.ID == "" {
		return fmt.Errorf("action ID is required")
	}
	if ba.Type == "" {
		return fmt.Errorf("action type is required")
	}
	return nil
}

func (ba *BaseAction) EstimateExecutionTime() time.Duration {
	return time.Millisecond * 100 // Default estimation
}

// ServiceAction calls Home Assistant or PMA services
type ServiceAction struct {
	BaseAction
	Service  string                 `json:"service"`
	EntityID string                 `json:"entity_id,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
	Target   map[string]interface{} `json:"target,omitempty"`
}

func NewServiceAction(id, service string) *ServiceAction {
	return &ServiceAction{
		BaseAction: BaseAction{
			ID:      id,
			Type:    ActionTypeService,
			Enabled: true,
		},
		Service: service,
	}
}

func (sa *ServiceAction) Execute(ctx context.Context, data map[string]interface{}) error {
	if !sa.Enabled {
		return nil
	}

	// Merge template data with action data
	actionData := make(map[string]interface{})
	for k, v := range sa.Data {
		actionData[k] = v
	}

	// Process templates in action data
	processedData, err := processTemplates(actionData, data)
	if err != nil {
		return fmt.Errorf("template processing failed: %v", err)
	}

	// Build service call payload
	payload := map[string]interface{}{
		"service": sa.Service,
		"data":    processedData,
	}

	if sa.EntityID != "" {
		payload["entity_id"] = sa.EntityID
	}

	if sa.Target != nil {
		payload["target"] = sa.Target
	}

	// This would be implemented to actually call the Home Assistant service
	// For now, we'll just log the action
	fmt.Printf("Executing service action: %s with data: %+v\n", sa.Service, payload)

	return nil
}

func (sa *ServiceAction) Clone() Action {
	data, _ := json.Marshal(sa)
	var clone ServiceAction
	json.Unmarshal(data, &clone)
	return &clone
}

func (sa *ServiceAction) Validate() error {
	if err := sa.BaseAction.Validate(); err != nil {
		return err
	}
	if sa.Service == "" {
		return fmt.Errorf("service is required for service action")
	}
	return nil
}

func (sa *ServiceAction) EstimateExecutionTime() time.Duration {
	return time.Millisecond * 500 // Service calls typically take longer
}

// NotificationAction sends notifications
type NotificationAction struct {
	BaseAction
	Title   string                 `json:"title"`
	Message string                 `json:"message"`
	Target  string                 `json:"target,omitempty"` // websocket, email, push, etc.
	Data    map[string]interface{} `json:"data,omitempty"`
}

func NewNotificationAction(id, title, message string) *NotificationAction {
	return &NotificationAction{
		BaseAction: BaseAction{
			ID:      id,
			Type:    ActionTypeNotification,
			Enabled: true,
		},
		Title:   title,
		Message: message,
		Target:  "websocket",
	}
}

func (na *NotificationAction) Execute(ctx context.Context, data map[string]interface{}) error {
	if !na.Enabled {
		return nil
	}

	// Process templates in title and message
	title, err := processStringTemplate(na.Title, data)
	if err != nil {
		return fmt.Errorf("title template processing failed: %v", err)
	}

	message, err := processStringTemplate(na.Message, data)
	if err != nil {
		return fmt.Errorf("message template processing failed: %v", err)
	}

	notification := map[string]interface{}{
		"title":   title,
		"message": message,
		"target":  na.Target,
		"data":    na.Data,
	}

	// Send notification based on target
	switch na.Target {
	case "websocket":
		return sendWebSocketNotification(ctx, notification)
	case "email":
		return sendEmailNotification(ctx, notification)
	default:
		fmt.Printf("Sending notification: %+v\n", notification)
	}

	return nil
}

func (na *NotificationAction) Clone() Action {
	data, _ := json.Marshal(na)
	var clone NotificationAction
	json.Unmarshal(data, &clone)
	return &clone
}

func (na *NotificationAction) Validate() error {
	if err := na.BaseAction.Validate(); err != nil {
		return err
	}
	if na.Title == "" && na.Message == "" {
		return fmt.Errorf("either title or message is required for notification action")
	}
	return nil
}

// DelayAction introduces delays
type DelayAction struct {
	BaseAction
	Duration string `json:"duration"` // e.g., "5s", "2m", "1h"
}

func NewDelayAction(id, duration string) *DelayAction {
	return &DelayAction{
		BaseAction: BaseAction{
			ID:      id,
			Type:    ActionTypeDelay,
			Enabled: true,
		},
		Duration: duration,
	}
}

func (da *DelayAction) Execute(ctx context.Context, data map[string]interface{}) error {
	if !da.Enabled {
		return nil
	}

	duration, err := time.ParseDuration(da.Duration)
	if err != nil {
		return fmt.Errorf("invalid duration: %v", err)
	}

	select {
	case <-time.After(duration):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (da *DelayAction) Clone() Action {
	data, _ := json.Marshal(da)
	var clone DelayAction
	json.Unmarshal(data, &clone)
	return &clone
}

func (da *DelayAction) Validate() error {
	if err := da.BaseAction.Validate(); err != nil {
		return err
	}
	if da.Duration == "" {
		return fmt.Errorf("duration is required for delay action")
	}
	if _, err := time.ParseDuration(da.Duration); err != nil {
		return fmt.Errorf("invalid duration format: %s", da.Duration)
	}
	return nil
}

func (da *DelayAction) EstimateExecutionTime() time.Duration {
	duration, _ := time.ParseDuration(da.Duration)
	return duration
}

// VariableAction manipulates variables
type VariableAction struct {
	BaseAction
	Variable string      `json:"variable"`
	Value    interface{} `json:"value"`
	Scope    string      `json:"scope,omitempty"` // "rule", "global"
}

func NewVariableAction(id, variable string, value interface{}) *VariableAction {
	return &VariableAction{
		BaseAction: BaseAction{
			ID:      id,
			Type:    ActionTypeVariable,
			Enabled: true,
		},
		Variable: variable,
		Value:    value,
		Scope:    "rule",
	}
}

func (va *VariableAction) Execute(ctx context.Context, data map[string]interface{}) error {
	if !va.Enabled {
		return nil
	}

	// Process template in value
	processedValue, err := processTemplate(va.Value, data)
	if err != nil {
		return fmt.Errorf("value template processing failed: %v", err)
	}

	// Set variable in appropriate scope
	switch va.Scope {
	case "global":
		// Set in global variables (would need global variable store)
		fmt.Printf("Setting global variable %s = %v\n", va.Variable, processedValue)
	default:
		// Set in rule context
		data[va.Variable] = processedValue
	}

	return nil
}

func (va *VariableAction) Clone() Action {
	data, _ := json.Marshal(va)
	var clone VariableAction
	json.Unmarshal(data, &clone)
	return &clone
}

func (va *VariableAction) Validate() error {
	if err := va.BaseAction.Validate(); err != nil {
		return err
	}
	if va.Variable == "" {
		return fmt.Errorf("variable name is required for variable action")
	}
	return nil
}

// HTTPAction makes HTTP requests
type HTTPAction struct {
	BaseAction
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    interface{}       `json:"body,omitempty"`
	Timeout string            `json:"timeout,omitempty"`
}

func NewHTTPAction(id, url string) *HTTPAction {
	return &HTTPAction{
		BaseAction: BaseAction{
			ID:      id,
			Type:    ActionTypeHTTP,
			Enabled: true,
		},
		URL:     url,
		Method:  "GET",
		Timeout: "30s",
	}
}

func (ha *HTTPAction) Execute(ctx context.Context, data map[string]interface{}) error {
	if !ha.Enabled {
		return nil
	}

	// Process URL template
	url, err := processStringTemplate(ha.URL, data)
	if err != nil {
		return fmt.Errorf("URL template processing failed: %v", err)
	}

	// Process body template
	var bodyReader io.Reader
	if ha.Body != nil {
		processedBody, err := processTemplate(ha.Body, data)
		if err != nil {
			return fmt.Errorf("body template processing failed: %v", err)
		}

		bodyBytes, err := json.Marshal(processedBody)
		if err != nil {
			return fmt.Errorf("body serialization failed: %v", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, ha.Method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("request creation failed: %v", err)
	}

	// Add headers
	for key, value := range ha.Headers {
		processedValue, err := processStringTemplate(value, data)
		if err != nil {
			return fmt.Errorf("header template processing failed: %v", err)
		}
		req.Header.Set(key, processedValue)
	}

	// Set content type for JSON body
	if ha.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Create client with timeout
	timeout, err := time.ParseDuration(ha.Timeout)
	if err != nil {
		timeout = 30 * time.Second
	}

	client := &http.Client{Timeout: timeout}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	return nil
}

func (ha *HTTPAction) Clone() Action {
	data, _ := json.Marshal(ha)
	var clone HTTPAction
	json.Unmarshal(data, &clone)
	return &clone
}

func (ha *HTTPAction) Validate() error {
	if err := ha.BaseAction.Validate(); err != nil {
		return err
	}
	if ha.URL == "" {
		return fmt.Errorf("URL is required for HTTP action")
	}
	return nil
}

func (ha *HTTPAction) EstimateExecutionTime() time.Duration {
	timeout, err := time.ParseDuration(ha.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return timeout
}

// ScriptAction executes shell scripts
type ScriptAction struct {
	BaseAction
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Timeout string   `json:"timeout,omitempty"`
}

func NewScriptAction(id, command string) *ScriptAction {
	return &ScriptAction{
		BaseAction: BaseAction{
			ID:      id,
			Type:    ActionTypeScript,
			Enabled: true,
		},
		Command: command,
		Timeout: "60s",
	}
}

func (sa *ScriptAction) Execute(ctx context.Context, data map[string]interface{}) error {
	if !sa.Enabled {
		return nil
	}

	// Process command template
	command, err := processStringTemplate(sa.Command, data)
	if err != nil {
		return fmt.Errorf("command template processing failed: %v", err)
	}

	// Process args templates
	var processedArgs []string
	for _, arg := range sa.Args {
		processedArg, err := processStringTemplate(arg, data)
		if err != nil {
			return fmt.Errorf("arg template processing failed: %v", err)
		}
		processedArgs = append(processedArgs, processedArg)
	}

	// Create command with timeout
	timeout, err := time.ParseDuration(sa.Timeout)
	if err != nil {
		timeout = 60 * time.Second
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, command, processedArgs...)

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("script execution failed: %v, output: %s", err, string(output))
	}

	return nil
}

func (sa *ScriptAction) Clone() Action {
	data, _ := json.Marshal(sa)
	var clone ScriptAction
	json.Unmarshal(data, &clone)
	return &clone
}

func (sa *ScriptAction) Validate() error {
	if err := sa.BaseAction.Validate(); err != nil {
		return err
	}
	if sa.Command == "" {
		return fmt.Errorf("command is required for script action")
	}
	return nil
}

func (sa *ScriptAction) EstimateExecutionTime() time.Duration {
	timeout, err := time.ParseDuration(sa.Timeout)
	if err != nil {
		return 60 * time.Second
	}
	return timeout
}

// ConditionalAction executes actions based on conditions
type ConditionalAction struct {
	BaseAction
	Conditions  []Condition `json:"conditions"`
	ThenActions []Action    `json:"then_actions"`
	ElseActions []Action    `json:"else_actions,omitempty"`
}

func NewConditionalAction(id string) *ConditionalAction {
	return &ConditionalAction{
		BaseAction: BaseAction{
			ID:      id,
			Type:    ActionTypeConditional,
			Enabled: true,
		},
	}
}

func (ca *ConditionalAction) Execute(ctx context.Context, data map[string]interface{}) error {
	if !ca.Enabled {
		return nil
	}

	// Evaluate all conditions
	conditionMet := true
	for _, condition := range ca.Conditions {
		result, err := condition.Evaluate(ctx, data)
		if err != nil {
			return fmt.Errorf("condition evaluation failed: %v", err)
		}
		if !result {
			conditionMet = false
			break
		}
	}

	// Execute appropriate actions
	var actionsToExecute []Action
	if conditionMet {
		actionsToExecute = ca.ThenActions
	} else {
		actionsToExecute = ca.ElseActions
	}

	for _, action := range actionsToExecute {
		if err := action.Execute(ctx, data); err != nil {
			return fmt.Errorf("action execution failed: %v", err)
		}
	}

	return nil
}

func (ca *ConditionalAction) Clone() Action {
	clone := &ConditionalAction{
		BaseAction:  ca.BaseAction,
		Conditions:  make([]Condition, len(ca.Conditions)),
		ThenActions: make([]Action, len(ca.ThenActions)),
		ElseActions: make([]Action, len(ca.ElseActions)),
	}

	for i, condition := range ca.Conditions {
		clone.Conditions[i] = condition.Clone()
	}
	for i, action := range ca.ThenActions {
		clone.ThenActions[i] = action.Clone()
	}
	for i, action := range ca.ElseActions {
		clone.ElseActions[i] = action.Clone()
	}

	return clone
}

func (ca *ConditionalAction) Validate() error {
	if err := ca.BaseAction.Validate(); err != nil {
		return err
	}

	if len(ca.Conditions) == 0 {
		return fmt.Errorf("at least one condition is required for conditional action")
	}

	if len(ca.ThenActions) == 0 {
		return fmt.Errorf("at least one then action is required for conditional action")
	}

	for i, condition := range ca.Conditions {
		if err := condition.Validate(); err != nil {
			return fmt.Errorf("condition %d: %v", i, err)
		}
	}

	for i, action := range ca.ThenActions {
		if err := action.Validate(); err != nil {
			return fmt.Errorf("then action %d: %v", i, err)
		}
	}

	for i, action := range ca.ElseActions {
		if err := action.Validate(); err != nil {
			return fmt.Errorf("else action %d: %v", i, err)
		}
	}

	return nil
}

func (ca *ConditionalAction) EstimateExecutionTime() time.Duration {
	var maxTime time.Duration

	// Estimate then actions
	var thenTime time.Duration
	for _, action := range ca.ThenActions {
		thenTime += action.EstimateExecutionTime()
	}
	if thenTime > maxTime {
		maxTime = thenTime
	}

	// Estimate else actions
	var elseTime time.Duration
	for _, action := range ca.ElseActions {
		elseTime += action.EstimateExecutionTime()
	}
	if elseTime > maxTime {
		maxTime = elseTime
	}

	return maxTime
}

// Helper functions
func processTemplates(data map[string]interface{}, context map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for key, value := range data {
		processedValue, err := processTemplate(value, context)
		if err != nil {
			return nil, err
		}
		result[key] = processedValue
	}

	return result, nil
}

func processTemplate(value interface{}, context map[string]interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return processStringTemplate(v, context)
	case map[string]interface{}:
		return processTemplates(v, context)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			processedItem, err := processTemplate(item, context)
			if err != nil {
				return nil, err
			}
			result[i] = processedItem
		}
		return result, nil
	default:
		return value, nil
	}
}

func processStringTemplate(template string, context map[string]interface{}) (string, error) {
	// Simple template processing - replace {{ variable }} with context values
	// In production, use a proper template engine
	result := template

	// Find all template variables
	for key, value := range context {
		placeholder := fmt.Sprintf("{{ %s }}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	return result, nil
}

func sendWebSocketNotification(ctx context.Context, notification map[string]interface{}) error {
	// Implementation would send via WebSocket hub
	fmt.Printf("WebSocket notification: %+v\n", notification)
	return nil
}

func sendEmailNotification(ctx context.Context, notification map[string]interface{}) error {
	// Implementation would send via email service
	fmt.Printf("Email notification: %+v\n", notification)
	return nil
}

// ActionFactory creates actions from configuration
type ActionFactory struct{}

func (af *ActionFactory) CreateAction(config map[string]interface{}) (Action, error) {
	actionType, ok := config["type"].(string)
	if !ok {
		return nil, fmt.Errorf("action type is required")
	}

	id, ok := config["id"].(string)
	if !ok {
		return nil, fmt.Errorf("action id is required")
	}

	switch ActionType(actionType) {
	case ActionTypeService:
		service, ok := config["service"].(string)
		if !ok {
			return nil, fmt.Errorf("service is required for service action")
		}
		action := NewServiceAction(id, service)
		if entityID, exists := config["entity_id"].(string); exists {
			action.EntityID = entityID
		}
		if data, exists := config["data"].(map[string]interface{}); exists {
			action.Data = data
		}
		return action, nil

	case ActionTypeNotification:
		title, _ := config["title"].(string)
		message, _ := config["message"].(string)
		action := NewNotificationAction(id, title, message)
		if target, exists := config["target"].(string); exists {
			action.Target = target
		}
		return action, nil

	case ActionTypeDelay:
		duration, ok := config["duration"].(string)
		if !ok {
			return nil, fmt.Errorf("duration is required for delay action")
		}
		return NewDelayAction(id, duration), nil

	case ActionTypeHTTP:
		url, ok := config["url"].(string)
		if !ok {
			return nil, fmt.Errorf("url is required for HTTP action")
		}
		action := NewHTTPAction(id, url)
		if method, exists := config["method"].(string); exists {
			action.Method = method
		}
		if body, exists := config["body"]; exists {
			action.Body = body
		}
		return action, nil

	default:
		return nil, fmt.Errorf("unsupported action type: %s", actionType)
	}
}
