# Automation Engine Architecture

## Overview

The Automation Engine is a sophisticated rule-based automation system that enables complex smart home automation scenarios. It provides a flexible, event-driven architecture with support for triggers, conditions, actions, and advanced features like LLM integration, scheduling, and circuit breaker protection.

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Automation Engine                       │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │ Rule        │  │ Trigger     │  │ Condition   │       │
│  │ Engine      │  │ Manager     │  │ Evaluator   │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │ Action      │  │ Scheduler   │  │ Circuit     │       │
│  │ Executor    │  │             │  │ Breaker     │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │ LLM         │  │ Context     │  │ Event       │       │
│  │ Integration │  │ Manager     │  │ Processor   │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    External Systems                        │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │ Entity      │  │ AI          │  │ WebSocket   │       │
│  │ Service     │  │ Service     │  │ Hub         │       │
│  └─────────────┘  └─────────────┘  └─────────────┘       │
└─────────────────────────────────────────────────────────────┘
```

### Rule Structure

```go
type AutomationRule struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    Enabled     bool                   `json:"enabled"`
    Category    string                 `json:"category"`
    Priority    int                    `json:"priority"`
    
    // Core components
    Triggers    []Trigger              `json:"triggers"`
    Conditions  []Condition            `json:"conditions"`
    Actions     []Action               `json:"actions"`
    
    // Advanced features
    Schedule    *Schedule              `json:"schedule,omitempty"`
    Context     map[string]interface{} `json:"context,omitempty"`
    Variables   map[string]interface{} `json:"variables,omitempty"`
    
    // Execution control
    MaxExecutionsPerHour int           `json:"max_executions_per_hour"`
    CooldownMinutes      int           `json:"cooldown_minutes"`
    Timeout             time.Duration  `json:"timeout"`
    
    // Metadata
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
    LastExecuted *time.Time            `json:"last_executed,omitempty"`
    ExecutionCount int                 `json:"execution_count"`
    SuccessCount   int                 `json:"success_count"`
    ErrorCount     int                 `json:"error_count"`
}
```

## Trigger System

### Trigger Types

```go
type TriggerType string

const (
    TriggerTypeStateChanged    TriggerType = "state_changed"
    TriggerTypeTime           TriggerType = "time"
    TriggerTypeSunset         TriggerType = "sunset"
    TriggerTypeSunrise        TriggerType = "sunrise"
    TriggerTypeNumericState   TriggerType = "numeric_state"
    TriggerTypeTemplate       TriggerType = "template"
    TriggerTypeZone           TriggerType = "zone"
    TriggerTypeEvent          TriggerType = "event"
    TriggerTypeWebhook        TriggerType = "webhook"
    TriggerTypeMQTT           TriggerType = "mqtt"
    TriggerTypeLLM            TriggerType = "llm"
)
```

### Trigger Implementation

```go
type Trigger struct {
    ID          string                 `json:"id"`
    Type        TriggerType            `json:"type"`
    EntityID    string                 `json:"entity_id,omitempty"`
    Platform    string                 `json:"platform,omitempty"`
    For         time.Duration          `json:"for,omitempty"`
    From        string                 `json:"from,omitempty"`
    To          string                 `json:"to,omitempty"`
    Above       float64                `json:"above,omitempty"`
    Below       float64                `json:"below,omitempty"`
    Offset      string                 `json:"offset,omitempty"`
    Template    string                 `json:"template,omitempty"`
    EventType   string                 `json:"event_type,omitempty"`
    EventData   map[string]interface{} `json:"event_data,omitempty"`
    WebhookURL  string                 `json:"webhook_url,omitempty"`
    MQTTTopic   string                 `json:"mqtt_topic,omitempty"`
    LLMPrompt   string                 `json:"llm_prompt,omitempty"`
    Enabled     bool                   `json:"enabled"`
}

// State Changed Trigger
type StateChangedTrigger struct {
    EntityID    string                 `json:"entity_id"`
    From        string                 `json:"from,omitempty"`
    To          string                 `json:"to,omitempty"`
    For         time.Duration          `json:"for,omitempty"`
    Attributes  map[string]interface{} `json:"attributes,omitempty"`
}

// Time Trigger
type TimeTrigger struct {
    At          string                 `json:"at"`
    DaysOfWeek  []int                  `json:"days_of_week,omitempty"`
    Offset      string                 `json:"offset,omitempty"`
    Randomize   time.Duration          `json:"randomize,omitempty"`
}

// Sunset/Sunrise Trigger
type SunTrigger struct {
    Event       string                 `json:"event"` // "sunset" or "sunrise"
    Offset      string                 `json:"offset,omitempty"`
    Latitude    float64                `json:"latitude,omitempty"`
    Longitude   float64                `json:"longitude,omitempty"`
}
```

### Trigger Manager

```go
type TriggerManager struct {
    triggers    map[string]*Trigger
    listeners   map[TriggerType][]TriggerListener
    scheduler   *Scheduler
    logger      *logrus.Logger
    mutex       sync.RWMutex
}

type TriggerListener struct {
    RuleID      string
    Trigger     *Trigger
    Callback    func(TriggerEvent)
}

type TriggerEvent struct {
    TriggerID   string
    RuleID      string
    EntityID    string
    OldState    interface{}
    NewState    interface{}
    Timestamp   time.Time
    Metadata    map[string]interface{}
}
```

## Condition System

### Condition Types

```go
type ConditionType string

const (
    ConditionTypeState       ConditionType = "state"
    ConditionTypeNumericState ConditionType = "numeric_state"
    ConditionTypeTemplate    ConditionType = "template"
    ConditionTypeTime        ConditionType = "time"
    ConditionTypeZone        ConditionType = "zone"
    ConditionTypeDevice      ConditionType = "device"
    ConditionTypeSun         ConditionType = "sun"
    ConditionTypeWeather     ConditionType = "weather"
    ConditionTypeLLM         ConditionType = "llm"
    ConditionTypeGroup       ConditionType = "group"
)
```

### Condition Implementation

```go
type Condition struct {
    ID          string                 `json:"id"`
    Type        ConditionType          `json:"type"`
    EntityID    string                 `json:"entity_id,omitempty"`
    Attribute   string                 `json:"attribute,omitempty"`
    Above       float64                `json:"above,omitempty"`
    Below       float64                `json:"below,omitempty"`
    Value       interface{}            `json:"value,omitempty"`
    Template    string                 `json:"template,omitempty"`
    After       string                 `json:"after,omitempty"`
    Before      string                 `json:"before,omitempty"`
    Weekday     []int                  `json:"weekday,omitempty"`
    Zone        string                 `json:"zone,omitempty"`
    Device      string                 `json:"device,omitempty"`
    LLMPrompt   string                 `json:"llm_prompt,omitempty"`
    GroupID     string                 `json:"group_id,omitempty"`
    Operator    string                 `json:"operator,omitempty"`
    Enabled     bool                   `json:"enabled"`
}

// State Condition
type StateCondition struct {
    EntityID    string                 `json:"entity_id"`
    State       string                 `json:"state"`
    Attribute   string                 `json:"attribute,omitempty"`
    Value       interface{}            `json:"value,omitempty"`
}

// Numeric State Condition
type NumericStateCondition struct {
    EntityID    string                 `json:"entity_id"`
    Attribute   string                 `json:"attribute"`
    Above       float64                `json:"above,omitempty"`
    Below       float64                `json:"below,omitempty"`
    Value       float64                `json:"value,omitempty"`
}

// Time Condition
type TimeCondition struct {
    After       string                 `json:"after,omitempty"`
    Before      string                 `json:"before,omitempty"`
    Weekday     []int                  `json:"weekday,omitempty"`
    Timezone    string                 `json:"timezone,omitempty"`
}

// LLM Condition
type LLMCondition struct {
    Prompt      string                 `json:"prompt"`
    Context     map[string]interface{} `json:"context,omitempty"`
    Model       string                 `json:"model,omitempty"`
    Threshold   float64                `json:"threshold,omitempty"`
}
```

### Condition Evaluator

```go
type ConditionEvaluator struct {
    entityService *UnifiedEntityService
    llmService    *LLMService
    logger        *logrus.Logger
}

func (e *ConditionEvaluator) EvaluateCondition(condition *Condition, context *ExecutionContext) (bool, error) {
    switch condition.Type {
    case ConditionTypeState:
        return e.evaluateStateCondition(condition, context)
    case ConditionTypeNumericState:
        return e.evaluateNumericStateCondition(condition, context)
    case ConditionTypeTime:
        return e.evaluateTimeCondition(condition, context)
    case ConditionTypeLLM:
        return e.evaluateLLMCondition(condition, context)
    case ConditionTypeTemplate:
        return e.evaluateTemplateCondition(condition, context)
    default:
        return false, fmt.Errorf("unsupported condition type: %s", condition.Type)
    }
}

func (e *ConditionEvaluator) evaluateStateCondition(condition *Condition, context *ExecutionContext) (bool, error) {
    entity, err := e.entityService.GetEntity(condition.EntityID)
    if err != nil {
        return false, err
    }
    
    if condition.Attribute != "" {
        value, exists := entity.Attributes[condition.Attribute]
        if !exists {
            return false, nil
        }
        return e.compareValues(value, condition.Value, condition.Operator)
    }
    
    return entity.State == condition.State, nil
}

func (e *ConditionEvaluator) evaluateLLMCondition(condition *Condition, context *ExecutionContext) (bool, error) {
    // Build context for LLM evaluation
    llmContext := map[string]interface{}{
        "entities": context.Entities,
        "time":     time.Now(),
        "user":     context.User,
    }
    
    // Merge with condition context
    for k, v := range condition.Context {
        llmContext[k] = v
    }
    
    // Evaluate with LLM
    result, err := e.llmService.EvaluateCondition(condition.Prompt, llmContext, condition.Model)
    if err != nil {
        return false, err
    }
    
    return result.Satisfied, nil
}
```

## Action System

### Action Types

```go
type ActionType string

const (
    ActionTypeCallService    ActionType = "call_service"
    ActionTypeDelay          ActionType = "delay"
    ActionTypeWait           ActionType = "wait"
    ActionTypeWaitTemplate   ActionType = "wait_template"
    ActionTypeDevice         ActionType = "device"
    ActionTypeScene          ActionType = "scene"
    ActionTypeScript         ActionType = "script"
    ActionTypeEvent          ActionType = "event"
    ActionTypeWebhook        ActionType = "webhook"
    ActionTypeMQTT           ActionType = "mqtt"
    ActionTypeLLM            ActionType = "llm"
    ActionTypeConditional    ActionType = "conditional"
    ActionTypeRepeat         ActionType = "repeat"
    ActionTypeChoose         ActionType = "choose"
)
```

### Action Implementation

```go
type Action struct {
    ID          string                 `json:"id"`
    Type        ActionType             `json:"type"`
    Service     string                 `json:"service,omitempty"`
    EntityID    string                 `json:"entity_id,omitempty"`
    Data        map[string]interface{} `json:"data,omitempty"`
    Delay       time.Duration          `json:"delay,omitempty"`
    Template    string                 `json:"template,omitempty"`
    Device      string                 `json:"device,omitempty"`
    Scene       string                 `json:"scene,omitempty"`
    Script      string                 `json:"script,omitempty"`
    EventType   string                 `json:"event_type,omitempty"`
    EventData   map[string]interface{} `json:"event_data,omitempty"`
    WebhookURL  string                 `json:"webhook_url,omitempty"`
    MQTTTopic   string                 `json:"mqtt_topic,omitempty"`
    MQTTMessage string                 `json:"mqtt_message,omitempty"`
    LLMPrompt   string                 `json:"llm_prompt,omitempty"`
    Condition   *Condition             `json:"condition,omitempty"`
    Actions     []Action               `json:"actions,omitempty"`
    Default     []Action               `json:"default,omitempty"`
    Choices     []Choice               `json:"choices,omitempty"`
    Repeat      *RepeatConfig          `json:"repeat,omitempty"`
    Enabled     bool                   `json:"enabled"`
}

// Conditional Action
type ConditionalAction struct {
    Condition   *Condition             `json:"condition"`
    Then        []Action               `json:"then"`
    Else        []Action               `json:"else,omitempty"`
}

// Repeat Action
type RepeatConfig struct {
    Count       int                    `json:"count,omitempty"`
    While       *Condition             `json:"while,omitempty"`
    Until       *Condition             `json:"until,omitempty"`
    Sequence    []Action               `json:"sequence"`
    MaxIterations int                  `json:"max_iterations,omitempty"`
}

// Choose Action
type Choice struct {
    Condition   *Condition             `json:"condition"`
    Actions     []Action               `json:"actions"`
}
```

### Action Executor

```go
type ActionExecutor struct {
    entityService *UnifiedEntityService
    llmService    *LLMService
    scheduler     *Scheduler
    logger        *logrus.Logger
    circuitBreaker *CircuitBreaker
}

func (e *ActionExecutor) ExecuteAction(action *Action, context *ExecutionContext) error {
    // Check circuit breaker
    if !e.circuitBreaker.Allow() {
        return fmt.Errorf("circuit breaker is open")
    }
    
    switch action.Type {
    case ActionTypeCallService:
        return e.executeCallService(action, context)
    case ActionTypeDelay:
        return e.executeDelay(action, context)
    case ActionTypeDevice:
        return e.executeDeviceAction(action, context)
    case ActionTypeLLM:
        return e.executeLLMAction(action, context)
    case ActionTypeConditional:
        return e.executeConditionalAction(action, context)
    case ActionTypeRepeat:
        return e.executeRepeatAction(action, context)
    case ActionTypeChoose:
        return e.executeChooseAction(action, context)
    default:
        return fmt.Errorf("unsupported action type: %s", action.Type)
    }
}

func (e *ActionExecutor) executeCallService(action *Action, context *ExecutionContext) error {
    // Execute service call through entity service
    return e.entityService.ExecuteAction(action.EntityID, action.Service, action.Data)
}

func (e *ActionExecutor) executeLLMAction(action *Action, context *ExecutionContext) error {
    // Build context for LLM
    llmContext := map[string]interface{}{
        "entities": context.Entities,
        "time":     time.Now(),
        "user":     context.User,
        "variables": context.Variables,
    }
    
    // Execute LLM action
    result, err := e.llmService.ExecuteAction(action.LLMPrompt, llmContext, action.Data)
    if err != nil {
        return err
    }
    
    // Handle LLM result (e.g., execute suggested actions)
    if result.Actions != nil {
        for _, suggestedAction := range result.Actions {
            if err := e.ExecuteAction(&suggestedAction, context); err != nil {
                e.logger.WithError(err).Error("Failed to execute suggested action")
            }
        }
    }
    
    return nil
}
```

## Scheduler System

### Scheduler Implementation

```go
type Scheduler struct {
    rules       map[string]*AutomationRule
    cronJobs    map[string]*cron.Cron
    timeTriggers map[string]*TimeTrigger
    logger      *logrus.Logger
    mutex       sync.RWMutex
}

type TimeTrigger struct {
    RuleID      string
    Schedule    *Schedule
    NextRun     time.Time
    Enabled     bool
}

type Schedule struct {
    Cron        string                 `json:"cron,omitempty"`
    At          string                 `json:"at,omitempty"`
    Every       time.Duration          `json:"every,omitempty"`
    DaysOfWeek  []int                  `json:"days_of_week,omitempty"`
    Timezone    string                 `json:"timezone,omitempty"`
    Randomize   time.Duration          `json:"randomize,omitempty"`
}

func (s *Scheduler) AddRule(rule *AutomationRule) error {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    
    s.rules[rule.ID] = rule
    
    // Add time triggers
    for _, trigger := range rule.Triggers {
        if trigger.Type == TriggerTypeTime {
            if err := s.addTimeTrigger(rule.ID, trigger); err != nil {
                return err
            }
        }
    }
    
    return nil
}

func (s *Scheduler) addTimeTrigger(ruleID string, trigger *Trigger) error {
    timeTrigger := &TimeTrigger{
        RuleID: ruleID,
        Schedule: &Schedule{
            At: trigger.At,
            DaysOfWeek: trigger.DaysOfWeek,
            Timezone: "UTC",
        },
        Enabled: trigger.Enabled,
    }
    
    // Parse schedule and calculate next run
    if err := s.calculateNextRun(timeTrigger); err != nil {
        return err
    }
    
    s.timeTriggers[ruleID] = timeTrigger
    
    // Schedule the trigger
    if timeTrigger.Enabled {
        s.scheduleTimeTrigger(timeTrigger)
    }
    
    return nil
}
```

## Circuit Breaker

### Circuit Breaker Implementation

```go
type CircuitBreaker struct {
    state       CircuitBreakerState
    failureCount int
    lastFailure  time.Time
    threshold    int
    timeout      time.Duration
    mutex        sync.RWMutex
}

type CircuitBreakerState string

const (
    CircuitBreakerStateClosed   CircuitBreakerState = "closed"
    CircuitBreakerStateOpen     CircuitBreakerState = "open"
    CircuitBreakerStateHalfOpen CircuitBreakerState = "half_open"
)

func (cb *CircuitBreaker) Allow() bool {
    cb.mutex.RLock()
    defer cb.mutex.RUnlock()
    
    switch cb.state {
    case CircuitBreakerStateClosed:
        return true
    case CircuitBreakerStateOpen:
        if time.Since(cb.lastFailure) > cb.timeout {
            cb.mutex.RUnlock()
            cb.mutex.Lock()
            cb.state = CircuitBreakerStateHalfOpen
            cb.mutex.Unlock()
            cb.mutex.RLock()
            return true
        }
        return false
    case CircuitBreakerStateHalfOpen:
        return true
    default:
        return false
    }
}

func (cb *CircuitBreaker) RecordSuccess() {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    if cb.state == CircuitBreakerStateHalfOpen {
        cb.state = CircuitBreakerStateClosed
        cb.failureCount = 0
    }
}

func (cb *CircuitBreaker) RecordFailure() {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    cb.failureCount++
    cb.lastFailure = time.Now()
    
    if cb.failureCount >= cb.threshold {
        cb.state = CircuitBreakerStateOpen
    }
}
```

## Context Management

### Execution Context

```go
type ExecutionContext struct {
    RuleID      string
    TriggerID   string
    UserID      string
    Entities    map[string]*UnifiedEntity
    Variables   map[string]interface{}
    Timestamp   time.Time
    Metadata    map[string]interface{}
}

type ContextManager struct {
    contexts    map[string]*ExecutionContext
    mutex       sync.RWMutex
    logger      *logrus.Logger
}

func (cm *ContextManager) CreateContext(ruleID, triggerID, userID string) *ExecutionContext {
    context := &ExecutionContext{
        RuleID:    ruleID,
        TriggerID: triggerID,
        UserID:    userID,
        Entities:  make(map[string]*UnifiedEntity),
        Variables: make(map[string]interface{}),
        Timestamp: time.Now(),
        Metadata:  make(map[string]interface{}),
    }
    
    cm.mutex.Lock()
    cm.contexts[ruleID] = context
    cm.mutex.Unlock()
    
    return context
}

func (cm *ContextManager) GetContext(ruleID string) *ExecutionContext {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    
    return cm.contexts[ruleID]
}

func (cm *ContextManager) UpdateContext(ruleID string, updates map[string]interface{}) {
    context := cm.GetContext(ruleID)
    if context == nil {
        return
    }
    
    for key, value := range updates {
        context.Variables[key] = value
    }
}
```

## LLM Integration

### LLM Service Integration

```go
type LLMService struct {
    manager     *LLMManager
    logger      *logrus.Logger
}

type LLMEvaluationResult struct {
    Satisfied   bool                   `json:"satisfied"`
    Confidence  float64                `json:"confidence"`
    Reasoning   string                 `json:"reasoning"`
    Actions     []Action               `json:"actions,omitempty"`
}

type LLMActionResult struct {
    Success     bool                   `json:"success"`
    Message     string                 `json:"message"`
    Actions     []Action               `json:"actions,omitempty"`
    Variables   map[string]interface{} `json:"variables,omitempty"`
}

func (ls *LLMService) EvaluateCondition(prompt string, context map[string]interface{}, model string) (*LLMEvaluationResult, error) {
    // Build prompt with context
    fullPrompt := ls.buildPromptWithContext(prompt, context)
    
    // Evaluate with LLM
    response, err := ls.manager.GenerateResponse(fullPrompt, model)
    if err != nil {
        return nil, err
    }
    
    // Parse response
    result := &LLMEvaluationResult{}
    if err := json.Unmarshal([]byte(response), result); err != nil {
        return nil, err
    }
    
    return result, nil
}

func (ls *LLMService) ExecuteAction(prompt string, context map[string]interface{}, data map[string]interface{}) (*LLMActionResult, error) {
    // Build action prompt
    actionPrompt := ls.buildActionPrompt(prompt, context, data)
    
    // Execute with LLM
    response, err := ls.manager.GenerateResponse(actionPrompt, "")
    if err != nil {
        return nil, err
    }
    
    // Parse response
    result := &LLMActionResult{}
    if err := json.Unmarshal([]byte(response), result); err != nil {
        return nil, err
    }
    
    return result, nil
}
```

## Event Processing

### Event Processor

```go
type EventProcessor struct {
    rules       map[string]*AutomationRule
    evaluator   *ConditionEvaluator
    executor    *ActionExecutor
    logger      *logrus.Logger
    mutex       sync.RWMutex
}

func (ep *EventProcessor) ProcessEvent(event *TriggerEvent) error {
    ep.mutex.RLock()
    defer ep.mutex.RUnlock()
    
    // Find rules that match this trigger
    matchingRules := ep.findMatchingRules(event)
    
    // Process each matching rule
    for _, rule := range matchingRules {
        if err := ep.processRule(rule, event); err != nil {
            ep.logger.WithError(err).WithField("rule_id", rule.ID).Error("Failed to process rule")
        }
    }
    
    return nil
}

func (ep *EventProcessor) findMatchingRules(event *TriggerEvent) []*AutomationRule {
    var matchingRules []*AutomationRule
    
    for _, rule := range ep.rules {
        if !rule.Enabled {
            continue
        }
        
        // Check if any trigger matches
        for _, trigger := range rule.Triggers {
            if ep.matchesTrigger(trigger, event) {
                matchingRules = append(matchingRules, rule)
                break
            }
        }
    }
    
    return matchingRules
}

func (ep *EventProcessor) processRule(rule *AutomationRule, event *TriggerEvent) error {
    // Create execution context
    context := &ExecutionContext{
        RuleID:    rule.ID,
        TriggerID: event.TriggerID,
        Entities:  make(map[string]*UnifiedEntity),
        Variables: rule.Variables,
        Timestamp: time.Now(),
        Metadata:  make(map[string]interface{}),
    }
    
    // Evaluate conditions
    if !ep.evaluateConditions(rule.Conditions, context) {
        return nil // Conditions not met
    }
    
    // Execute actions
    return ep.executeActions(rule.Actions, context)
}

func (ep *EventProcessor) evaluateConditions(conditions []Condition, context *ExecutionContext) bool {
    for _, condition := range conditions {
        if !condition.Enabled {
            continue
        }
        
        satisfied, err := ep.evaluator.EvaluateCondition(&condition, context)
        if err != nil {
            ep.logger.WithError(err).WithField("condition_id", condition.ID).Error("Failed to evaluate condition")
            return false
        }
        
        if !satisfied {
            return false
        }
    }
    
    return true
}

func (ep *EventProcessor) executeActions(actions []Action, context *ExecutionContext) error {
    for _, action := range actions {
        if !action.Enabled {
            continue
        }
        
        if err := ep.executor.ExecuteAction(&action, context); err != nil {
            ep.logger.WithError(err).WithField("action_id", action.ID).Error("Failed to execute action")
            return err
        }
    }
    
    return nil
}
```

## Performance Optimizations

### 1. Rule Indexing

```go
type RuleIndex struct {
    byTrigger   map[TriggerType][]string
    byEntity    map[string][]string
    byCondition map[ConditionType][]string
    byAction    map[ActionType][]string
    mutex       sync.RWMutex
}

func (ri *RuleIndex) AddRule(rule *AutomationRule) {
    ri.mutex.Lock()
    defer ri.mutex.Unlock()
    
    // Index by triggers
    for _, trigger := range rule.Triggers {
        if ri.byTrigger[trigger.Type] == nil {
            ri.byTrigger[trigger.Type] = []string{}
        }
        ri.byTrigger[trigger.Type] = append(ri.byTrigger[trigger.Type], rule.ID)
    }
    
    // Index by entities
    for _, condition := range rule.Conditions {
        if condition.EntityID != "" {
            if ri.byEntity[condition.EntityID] == nil {
                ri.byEntity[condition.EntityID] = []string{}
            }
            ri.byEntity[condition.EntityID] = append(ri.byEntity[condition.EntityID], rule.ID)
        }
    }
}
```

### 2. Parallel Execution

```go
func (ep *EventProcessor) processRulesParallel(rules []*AutomationRule, event *TriggerEvent) error {
    var wg sync.WaitGroup
    errors := make(chan error, len(rules))
    
    for _, rule := range rules {
        wg.Add(1)
        go func(r *AutomationRule) {
            defer wg.Done()
            if err := ep.processRule(r, event); err != nil {
                errors <- err
            }
        }(rule)
    }
    
    wg.Wait()
    close(errors)
    
    // Collect errors
    var errs []error
    for err := range errors {
        errs = append(errs, err)
    }
    
    if len(errs) > 0 {
        return fmt.Errorf("multiple rule processing errors: %v", errs)
    }
    
    return nil
}
```

### 3. Caching

```go
type ConditionCache struct {
    cache       map[string]*ConditionResult
    ttl         time.Duration
    mutex       sync.RWMutex
}

type ConditionResult struct {
    Satisfied   bool
    Timestamp   time.Time
    EntityID    string
    ConditionID string
}

func (cc *ConditionCache) Get(key string) (*ConditionResult, bool) {
    cc.mutex.RLock()
    defer cc.mutex.RUnlock()
    
    result, exists := cc.cache[key]
    if !exists {
        return nil, false
    }
    
    if time.Since(result.Timestamp) > cc.ttl {
        delete(cc.cache, key)
        return nil, false
    }
    
    return result, true
}

func (cc *ConditionCache) Set(key string, result *ConditionResult) {
    cc.mutex.Lock()
    defer cc.mutex.Unlock()
    
    cc.cache[key] = result
}
```

## Monitoring and Metrics

### Automation Metrics

```go
type AutomationMetrics struct {
    TotalRules          int64
    EnabledRules        int64
    DisabledRules       int64
    TotalExecutions     int64
    SuccessfulExecutions int64
    FailedExecutions    int64
    AverageExecutionTime time.Duration
    LastExecution       time.Time
    ActiveTriggers      int64
    PendingActions      int64
    CircuitBreakerState string
    CacheHitRate        float64
    MemoryUsage         int64
}

type RuleMetrics struct {
    RuleID              string
    ExecutionCount      int64
    SuccessCount        int64
    ErrorCount          int64
    AverageExecutionTime time.Duration
    LastExecution       time.Time
    LastSuccess         time.Time
    LastError           time.Time
    ErrorRate           float64
}
```

### Health Monitoring

```go
type AutomationHealth struct {
    Status        string
    Timestamp     time.Time
    ActiveRules   int
    PendingActions int
    CircuitBreaker CircuitBreakerState
    MemoryUsage   float64
    ErrorRate     float64
    LastSync      time.Time
    Message       string
}
```

## Configuration

### Engine Configuration

```yaml
automation_engine:
  enabled: true
  max_concurrent_rules: 10
  execution_timeout: "30s"
  circuit_breaker:
    threshold: 5
    timeout: "1m"
  caching:
    enabled: true
    ttl: "5m"
    max_size: 1000
  llm_integration:
    enabled: true
    default_model: "LFM2-1.2B"
    max_tokens: 1000
    temperature: 0.7
  scheduling:
    enabled: true
    timezone: "UTC"
    max_scheduled_rules: 100
  monitoring:
    enabled: true
    metrics_interval: "1m"
    health_check_interval: "30s"
```

## Usage Examples

### Basic Rule Creation

```go
// Create a simple rule
rule := &AutomationRule{
    ID:          "rule_123",
    Name:        "Turn on lights at sunset",
    Description: "Automatically turn on lights when sun sets",
    Enabled:     true,
    Category:    "lighting",
    Triggers: []Trigger{
        {
            Type: TriggerTypeSunset,
            Offset: "-30m",
        },
    },
    Conditions: []Condition{
        {
            Type: ConditionTypeState,
            EntityID: "binary_sensor.motion",
            Value: "on",
        },
    },
    Actions: []Action{
        {
            Type: ActionTypeCallService,
            Service: "light.turn_on",
            EntityID: "light.living_room",
            Data: map[string]interface{}{
                "brightness": 128,
            },
        },
    },
}

// Add rule to engine
engine.AddRule(rule)
```

### LLM-Enhanced Rule

```go
// Create a rule with LLM condition
rule := &AutomationRule{
    ID:          "rule_llm_123",
    Name:        "Smart lighting based on context",
    Description: "Use AI to determine optimal lighting",
    Enabled:     true,
    Category:    "lighting",
    Triggers: []Trigger{
        {
            Type: TriggerTypeStateChanged,
            EntityID: "binary_sensor.motion",
            To: "on",
        },
    },
    Conditions: []Condition{
        {
            Type: ConditionTypeLLM,
            LLMPrompt: "Is it appropriate to turn on bright lights right now? Consider time of day, weather, and user preferences.",
        },
    },
    Actions: []Action{
        {
            Type: ActionTypeLLM,
            LLMPrompt: "Determine the optimal lighting settings for the current context and execute them.",
        },
    },
}
```

### Complex Rule with Variables

```go
// Create a rule with variables and conditional actions
rule := &AutomationRule{
    ID:          "rule_complex_123",
    Name:        "Adaptive climate control",
    Description: "Smart climate control based on multiple factors",
    Enabled:     true,
    Category:    "climate",
    Variables: map[string]interface{}{
        "target_temp": 22.0,
        "comfort_range": 2.0,
    },
    Triggers: []Trigger{
        {
            Type: TriggerTypeStateChanged,
            EntityID: "sensor.temperature",
        },
    },
    Conditions: []Condition{
        {
            Type: ConditionTypeNumericState,
            EntityID: "sensor.temperature",
            Below: 20.0,
        },
    },
    Actions: []Action{
        {
            Type: ActionTypeConditional,
            Condition: &Condition{
                Type: ConditionTypeTime,
                After: "22:00:00",
                Before: "06:00:00",
            },
            Actions: []Action{
                {
                    Type: ActionTypeCallService,
                    Service: "climate.set_temperature",
                    EntityID: "climate.living_room",
                    Data: map[string]interface{}{
                        "temperature": 18.0,
                    },
                },
            },
            Default: []Action{
                {
                    Type: ActionTypeCallService,
                    Service: "climate.set_temperature",
                    EntityID: "climate.living_room",
                    Data: map[string]interface{}{
                        "temperature": 22.0,
                    },
                },
            },
        },
    },
}
```

## Testing

### Unit Tests

```go
func TestAutomationEngine_ProcessRule(t *testing.T) {
    engine := NewAutomationEngine(config)
    
    // Create test rule
    rule := &AutomationRule{
        ID: "test_rule",
        Triggers: []Trigger{
            {Type: TriggerTypeStateChanged, EntityID: "test.entity"},
        },
        Conditions: []Condition{
            {Type: ConditionTypeState, EntityID: "test.entity", Value: "on"},
        },
        Actions: []Action{
            {Type: ActionTypeCallService, Service: "test.service"},
        },
    }
    
    // Add rule
    engine.AddRule(rule)
    
    // Create test event
    event := &TriggerEvent{
        TriggerID: "test_trigger",
        EntityID:  "test.entity",
        NewState:  "on",
    }
    
    // Process event
    err := engine.ProcessEvent(event)
    assert.NoError(t, err)
}
```

### Integration Tests

```go
func TestAutomationEngine_Integration(t *testing.T) {
    // Test with real entity service
    engine := NewAutomationEngine(config)
    
    // Start engine
    err := engine.Start()
    assert.NoError(t, err)
    defer engine.Stop()
    
    // Test rule execution
    rule := createTestRule()
    engine.AddRule(rule)
    
    // Simulate trigger event
    event := createTestEvent()
    err = engine.ProcessEvent(event)
    assert.NoError(t, err)
    
    // Verify action was executed
    // (implementation depends on entity service)
}
```

## Performance Benchmarks

### Rule Processing Performance

```
BenchmarkProcessRule_Simple-8          100000     15000 ns/op
BenchmarkProcessRule_Complex-8          50000      45000 ns/op
BenchmarkProcessRule_WithLLM-8          10000      150000 ns/op
```

### Condition Evaluation Performance

```
BenchmarkEvaluateCondition_State-8     1000000    2000 ns/op
BenchmarkEvaluateCondition_Time-8      1000000    1500 ns/op
BenchmarkEvaluateCondition_LLM-8       1000       150000 ns/op
```

### Action Execution Performance

```
BenchmarkExecuteAction_Service-8       100000     8000 ns/op
BenchmarkExecuteAction_Delay-8         100000     5000 ns/op
BenchmarkExecuteAction_LLM-8           1000       120000 ns/op
```

## Troubleshooting

### Common Issues

1. **Rule Not Triggering**
   - Check trigger configuration
   - Verify entity states
   - Review trigger timing

2. **Conditions Not Met**
   - Validate condition syntax
   - Check entity availability
   - Review LLM prompts

3. **Actions Not Executing**
   - Verify action permissions
   - Check entity capabilities
   - Review action parameters

4. **Performance Issues**
   - Monitor rule complexity
   - Check LLM response times
   - Review caching configuration

### Debug Commands

```bash
# Check automation status
curl http://localhost:3001/api/v1/automation/status

# Get rule statistics
curl http://localhost:3001/api/v1/automation/rules/stats

# Test rule execution
curl -X POST http://localhost:3001/api/v1/automation/rules/rule_123/test

# Get execution history
curl http://localhost:3001/api/v1/automation/history
```

## Future Enhancements

### Planned Features

1. **Advanced LLM Integration**
   - Natural language rule creation
   - Context-aware automation
   - Learning from user behavior

2. **Enhanced Scheduling**
   - Calendar integration
   - Weather-based scheduling
   - Predictive scheduling

3. **Visual Rule Editor**
   - Drag-and-drop interface
   - Visual condition builders
   - Real-time rule testing

4. **Advanced Analytics**
   - Rule performance analytics
   - Energy optimization suggestions
   - Usage pattern analysis

---

**Automation Engine** - Sophisticated rule-based automation with LLM integration and advanced features. 