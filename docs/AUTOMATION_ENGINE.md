# PMA Backend Go - Automation Engine

The PMA Backend Go Automation Engine is a powerful, extensible automation system that allows you to create complex automation rules using triggers, conditions, and actions. It integrates seamlessly with Home Assistant, WebSocket events, and various external services.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Core Components](#core-components)
- [API Endpoints](#api-endpoints)
- [Configuration](#configuration)
- [Automation Rules](#automation-rules)
- [Examples](#examples)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

The Automation Engine provides:

- **Event-driven automation** based on state changes, time, webhooks, and custom events
- **Complex condition evaluation** with logical operators (AND/OR) and various condition types
- **Powerful action execution** supporting services, notifications, delays, and variables
- **Concurrent execution** with configurable worker pools and execution modes
- **Scheduling system** with cron expressions and time-based triggers
- **Circuit breaker protection** for problematic rules
- **Performance monitoring** and detailed statistics
- **YAML/JSON rule format** compatible with Home Assistant

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP API      │    │  WebSocket Hub  │    │ Home Assistant  │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          ▼                      ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Automation Engine                            │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   Rule Parser   │   Scheduler     │     Execution Context      │
├─────────────────┼─────────────────┼─────────────────────────────┤
│    Triggers     │   Conditions    │         Actions             │
├─────────────────┼─────────────────┼─────────────────────────────┤
│  Worker Pool    │   Statistics    │     Circuit Breaker        │
└─────────────────┴─────────────────┴─────────────────────────────┘
```

## Core Components

### 1. Automation Engine (`engine.go`)

The main orchestrator that manages rules, workers, and execution.

**Key Features:**
- Rule lifecycle management (add, update, remove, enable/disable)
- Worker pool for concurrent execution
- Event queue and processing
- Statistics and performance monitoring
- Circuit breaker pattern for problematic rules

**Configuration:**
```go
type EngineConfig struct {
    Workers              int
    QueueSize           int
    ExecutionTimeout    time.Duration
    MaxConcurrentRules  int
    EnableCircuitBreaker bool
    CircuitBreakerConfig *CircuitBreakerConfig
    SchedulerConfig     *SchedulerConfig
}
```

### 2. Rule Structure (`rule.go`)

Defines automation rules with validation.

```go
type AutomationRule struct {
    ID            string        `json:"id"`
    Name          string        `json:"name"`
    Description   string        `json:"description"`
    Enabled       bool          `json:"enabled"`
    ExecutionMode ExecutionMode `json:"execution_mode"`
    Triggers      []Trigger     `json:"triggers"`
    Conditions    []Condition   `json:"conditions"`
    Actions       []Action      `json:"actions"`
    Variables     Variables     `json:"variables"`
    CreatedAt     time.Time     `json:"created_at"`
    UpdatedAt     time.Time     `json:"updated_at"`
    LastRun       *time.Time    `json:"last_run,omitempty"`
    RunCount      int64         `json:"run_count"`
}
```

**Execution Modes:**
- `single`: Only one instance can run at a time
- `parallel`: Multiple instances can run simultaneously  
- `queued`: Instances are queued and executed sequentially

### 3. Triggers (`trigger.go`)

Define when automation rules should be executed.

**Trigger Types:**

#### State Trigger
Triggers when an entity's state changes.
```yaml
triggers:
  - type: "state"
    entity_id: "binary_sensor.motion"
    from: "off"
    to: "on"
    for: "5m"
    attribute: "temperature"
    above: 25
```

#### Time Trigger
Triggers at specific times or intervals.
```yaml
triggers:
  - type: "time"
    at: "07:00:00"           # Specific time
    # OR
    cron: "0 */15 * * * *"   # Cron expression
    # OR  
    interval: "5m"           # Interval (1m, 5m, 15m, 30m, 1h)
```

#### Event Trigger
Triggers on custom events.
```yaml
triggers:
  - type: "event"
    event_type: "garage_door_opened"
    event_data:
      user: "john"
```

#### Webhook Trigger
Triggers via HTTP webhooks.
```yaml
triggers:
  - type: "webhook"
    webhook_id: "automation_webhook"
    method: "POST"
```

#### Composite Trigger
Combines multiple triggers with logical operators.
```yaml
triggers:
  - type: "composite"
    logic: "or"
    triggers:
      - type: "state"
        entity_id: "sensor.temperature"
        above: 25
      - type: "time"
        at: "14:00:00"
```

### 4. Conditions (`condition.go`)

Define when actions should be executed after triggers fire.

**Condition Types:**

#### State Condition
```yaml
conditions:
  - type: "state"
    entity_id: "sun.sun"
    state: "below_horizon"
```

#### Time Condition
```yaml
conditions:
  - type: "time"
    after: "sunset"
    before: "23:00:00"
    weekday: ["mon", "tue", "wed", "thu", "fri"]
```

#### Numeric Condition
```yaml
conditions:
  - type: "numeric"
    entity_id: "sensor.temperature"
    above: 20
    below: 30
```

#### Template Condition
```yaml
conditions:
  - type: "template"
    template: "{{ states('sensor.temperature') | float > 25 }}"
```

#### Composite Condition
```yaml
conditions:
  - type: "and"
    conditions:
      - type: "state"
        entity_id: "binary_sensor.someone_home"
        state: "on"
      - type: "time"
        after: "18:00:00"
```

### 5. Actions (`action.go`)

Define what should happen when triggers fire and conditions are met.

**Action Types:**

#### Service Action
Call Home Assistant or PMA services.
```yaml
actions:
  - type: "service"
    service: "light.turn_on"
    data:
      entity_id: "light.living_room"
      brightness: 255
      color_name: "blue"
```

#### Notification Action
Send notifications via various channels.
```yaml
actions:
  - type: "notification"
    message: "Motion detected in living room!"
    target: "mobile"
    priority: "high"
    data:
      tag: "motion_alert"
```

#### Delay Action
Add delays between actions.
```yaml
actions:
  - type: "delay"
    duration: "30s"
```

#### Variable Action
Manipulate variables.
```yaml
actions:
  - type: "variable"
    action: "set"
    variable: "last_motion_time"
    value: "{{ now() }}"
```

#### HTTP Action
Make HTTP requests.
```yaml
actions:
  - type: "http"
    url: "https://api.example.com/webhook"
    method: "POST"
    headers:
      Authorization: "Bearer {{ token }}"
    data:
      message: "Automation triggered"
```

#### Script Action
Execute shell commands.
```yaml
actions:
  - type: "script"
    command: "/usr/local/bin/backup.sh"
    args: ["--quick"]
    timeout: "5m"
```

#### Conditional Action
Execute actions based on conditions.
```yaml
actions:
  - type: "conditional"
    condition:
      type: "state"
      entity_id: "sun.sun"
      state: "below_horizon"
    actions:
      - type: "service"
        service: "light.turn_on"
        data:
          entity_id: "light.entrance"
```

### 6. Scheduler (`scheduler.go`)

Manages time-based triggers and scheduling.

**Features:**
- Cron expression support (5-field format)
- Timezone handling
- Trigger scheduling and unscheduling
- Next execution time calculation
- Statistics and monitoring

### 7. Parser (`parser.go`)

Parses YAML and JSON automation rules.

**Features:**
- Home Assistant compatibility
- Rule validation
- Import/export functionality
- Template processing
- Error reporting

### 8. Execution Context (`context.go`)

Manages execution state and variables.

**Features:**
- Variable scoping
- Execution tracing
- Performance metrics
- Context cleanup
- Stack management

## API Endpoints

### Rule Management

#### Get All Rules
```http
GET /api/v1/automation/rules
```

Query parameters:
- `enabled`: Filter by enabled status
- `category`: Filter by category
- `page`: Page number for pagination
- `limit`: Number of rules per page

#### Get Single Rule
```http
GET /api/v1/automation/rules/{id}
```

#### Create Rule
```http
POST /api/v1/automation/rules
Content-Type: application/json

{
  "name": "My Automation",
  "description": "Sample automation rule",
  "enabled": true,
  "execution_mode": "single",
  "triggers": [...],
  "conditions": [...],
  "actions": [...]
}
```

#### Update Rule
```http
PUT /api/v1/automation/rules/{id}
Content-Type: application/json

{
  "name": "Updated Automation",
  "enabled": false
}
```

#### Delete Rule
```http
DELETE /api/v1/automation/rules/{id}
```

### Rule Control

#### Enable Rule
```http
POST /api/v1/automation/rules/{id}/enable
```

#### Disable Rule
```http
POST /api/v1/automation/rules/{id}/disable
```

#### Test Rule
```http
POST /api/v1/automation/rules/{id}/test
Content-Type: application/json

{
  "trigger_data": {
    "entity_id": "binary_sensor.test",
    "new_state": "on"
  }
}
```

### Import/Export

#### Import Rules
```http
POST /api/v1/automation/rules/import
Content-Type: multipart/form-data

file: automation_rules.yaml
```

#### Export Rules
```http
GET /api/v1/automation/rules/export?format=yaml
```

### Validation

#### Validate Rule
```http
POST /api/v1/automation/rules/validate
Content-Type: application/json

{
  "rule": {...}
}
```

### Statistics and Templates

#### Get Statistics
```http
GET /api/v1/automation/statistics
```

#### Get Templates
```http
GET /api/v1/automation/templates
```

#### Get Execution History
```http
GET /api/v1/automation/history?rule_id={id}&limit=50
```

## Configuration

### Engine Configuration

Create an automation engine with custom configuration:

```go
config := &automation.EngineConfig{
    Workers:              10,
    QueueSize:           1000,
    ExecutionTimeout:    30 * time.Second,
    MaxConcurrentRules:  100,
    EnableCircuitBreaker: true,
    CircuitBreakerConfig: &automation.CircuitBreakerConfig{
        FailureThreshold: 5,
        ResetTimeout:     60 * time.Second,
        MaxRequests:      10,
    },
    SchedulerConfig: &automation.SchedulerConfig{
        Timezone: "America/New_York",
    },
}

engine, err := automation.NewAutomationEngine(config, haClient, wsHub, logger)
```

### Scheduler Configuration

```go
schedulerConfig := &automation.SchedulerConfig{
    Timezone: "UTC",
    // Additional scheduler settings
}
```

## Automation Rules

### Rule Format

Automation rules can be defined in YAML or JSON format:

```yaml
# YAML Format
id: "sample_automation"
name: "Sample Automation"
description: "A sample automation rule"
enabled: true
execution_mode: "single"

triggers:
  - type: "state"
    entity_id: "binary_sensor.motion"
    to: "on"

conditions:
  - type: "state"
    entity_id: "sun.sun"
    state: "below_horizon"

actions:
  - type: "service"
    service: "light.turn_on"
    data:
      entity_id: "light.living_room"
      brightness: 255

variables:
  last_triggered: "{{ now() }}"
```

```json
// JSON Format
{
  "id": "sample_automation",
  "name": "Sample Automation",
  "description": "A sample automation rule",
  "enabled": true,
  "execution_mode": "single",
  "triggers": [
    {
      "type": "state",
      "entity_id": "binary_sensor.motion",
      "to": "on"
    }
  ],
  "conditions": [
    {
      "type": "state",
      "entity_id": "sun.sun",
      "state": "below_horizon"
    }
  ],
  "actions": [
    {
      "type": "service",
      "service": "light.turn_on",
      "data": {
        "entity_id": "light.living_room",
        "brightness": 255
      }
    }
  ],
  "variables": {
    "last_triggered": "{{ now() }}"
  }
}
```

### Rule Validation

Rules are automatically validated when created or updated. Validation checks:

- Required fields (id, name, triggers, actions)
- Trigger type and configuration
- Condition syntax and logic
- Action type and parameters
- Variable references
- Template syntax

### Variables and Templates

Rules support variables and template expressions:

```yaml
variables:
  user_name: "John"
  temperature_threshold: 25

actions:
  - type: "notification"
    message: "Hello {{ variables.user_name }}, temperature is {{ states('sensor.temperature') }}°C"
    
  - type: "service"
    service: "climate.set_temperature"
    data:
      entity_id: "climate.living_room"
      temperature: "{{ variables.temperature_threshold }}"
```

## Examples

See `examples/automation_rules.yaml` for comprehensive examples including:

1. **Basic Light Control** - Motion-activated lighting
2. **Security System** - Door alerts with notifications
3. **Morning Routine** - Scheduled morning automation
4. **Energy Management** - Power consumption optimization
5. **Weather Response** - Weather-based environment control
6. **Presence Detection** - Welcome home automation
7. **Night Security** - Comprehensive night routine
8. **Complex Automation** - Multi-trigger composite rules

## Best Practices

### Rule Design

1. **Keep rules focused** - One rule should handle one specific scenario
2. **Use descriptive names** - Make rules easy to understand and maintain
3. **Add conditions wisely** - Prevent unwanted executions
4. **Handle edge cases** - Consider what happens when sensors are unavailable

### Performance

1. **Limit concurrent rules** - Use appropriate execution modes
2. **Optimize conditions** - Put fast conditions first in AND chains
3. **Use circuit breakers** - Protect against problematic rules
4. **Monitor statistics** - Track execution times and failure rates

### Security

1. **Validate inputs** - Always validate webhook and external data
2. **Limit script actions** - Restrict shell command execution
3. **Use secure templates** - Sanitize template inputs
4. **Audit rule changes** - Track who modifies rules

### Maintenance

1. **Version control** - Keep rules in version control
2. **Test rules** - Use the test endpoint before deployment
3. **Monitor logs** - Watch for errors and warnings
4. **Regular backups** - Export rules regularly

## Troubleshooting

### Common Issues

#### Rule Not Triggering

1. Check if rule is enabled
2. Verify trigger configuration
3. Check condition evaluation
4. Review entity states
5. Check scheduler status (for time triggers)

#### Actions Not Executing

1. Verify action syntax
2. Check service availability
3. Review execution logs
4. Check circuit breaker status
5. Verify permissions

#### Performance Issues

1. Review worker pool size
2. Check queue length
3. Analyze execution times
4. Look for blocking actions
5. Consider rule optimization

### Debug Tools

#### Get Rule Statistics
```http
GET /api/v1/automation/statistics
```

Returns execution counts, timing, and error information.

#### Test Rule Execution
```http
POST /api/v1/automation/rules/{id}/test
```

Test rules without affecting the real system.

#### View Execution History
```http
GET /api/v1/automation/history?rule_id={id}
```

See detailed execution logs and results.

### Logging

The automation engine provides detailed logging at different levels:

- **DEBUG**: Detailed execution flow
- **INFO**: Rule lifecycle events
- **WARN**: Non-critical issues
- **ERROR**: Failures and exceptions

Configure logging in your application:

```go
logger := logrus.New()
logger.SetLevel(logrus.DebugLevel)
```

### Monitoring

Monitor key metrics:

- **Rule execution count** - How often rules run
- **Execution duration** - How long rules take
- **Success/failure rates** - Rule reliability
- **Queue length** - System load
- **Worker utilization** - Performance optimization

Use the statistics endpoint to gather metrics:

```go
stats := engine.GetStatistics()
fmt.Printf("Total executions: %d\n", stats.TotalExecutions)
fmt.Printf("Average duration: %v\n", stats.AverageExecutionTime)
```

---

For more information and examples, see:
- `examples/automation_rules.yaml` - Sample automation rules
- `internal/core/automation/` - Source code
- API documentation - Interactive API reference 