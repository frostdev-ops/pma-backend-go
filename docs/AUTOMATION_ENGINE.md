# PMA Automation Engine Documentation

## Overview

The PMA Automation Engine is a powerful, flexible system for creating and executing complex automation rules based on triggers, conditions, and actions. It integrates seamlessly with Home Assistant entities and provides extensive customization options for home automation scenarios.

## Features

- **Multiple Trigger Types**: State changes, time-based scheduling, events, webhooks, and composite triggers
- **Flexible Conditions**: State conditions, time conditions, numeric comparisons, templates, and composite logic
- **Comprehensive Actions**: Service calls, notifications, delays, variables, HTTP requests, scripts, and conditional execution
- **Execution Modes**: Single, parallel, and queued execution strategies
- **Real-time Scheduling**: Cron-based scheduling with timezone support
- **Rule Validation**: Comprehensive syntax and semantic validation
- **Execution Tracing**: Detailed execution logs and performance metrics
- **YAML/JSON Support**: Import/export rules in both formats
- **Thread-safe**: Concurrent rule execution with worker pools
- **Circuit Breaker**: Automatic protection against problematic rules

## Architecture

### Core Components

1. **AutomationEngine**: Main orchestrator managing rules and execution
2. **Scheduler**: Time-based trigger management with cron support
3. **RuleParser**: YAML/JSON rule parsing and validation
4. **ExecutionContext**: Rule execution tracking and debugging
5. **Triggers**: Event detection and subscription management
6. **Conditions**: Rule condition evaluation
7. **Actions**: Rule action execution

### File Structure

```
internal/core/automation/
├── engine.go           # Main automation engine
├── rule.go            # Rule definition and models
├── trigger.go         # Trigger types and handlers
├── condition.go       # Condition evaluator
├── action.go          # Action executor
├── scheduler.go       # Time-based scheduling
├── parser.go          # Rule parser (YAML/JSON)
├── context.go         # Execution context
└── automation_test.go # Comprehensive tests
```

## Rule Structure

### Basic Rule Format

```yaml
id: unique_rule_id
name: "Human Readable Name"
description: "Rule description"
enabled: true
mode: single  # single, parallel, queued
triggers:
  - # Trigger definitions
conditions:
  - # Condition definitions (optional)
actions:
  - # Action definitions
variables:
  # Rule-specific variables (optional)
```

### Execution Modes

- **single**: Only one instance of the rule can run at a time
- **parallel**: Multiple instances can run simultaneously
- **queued**: Queue executions if the rule is busy

## Trigger Types

### State Triggers

Trigger when entity state changes:

```yaml
triggers:
  - platform: state
    entity_id: sensor.temperature
    from: "20"
    to: "25"
    for: "00:05:00"  # Optional duration
    attribute: "battery_level"  # Optional attribute
```

### Time Triggers

Trigger at specific times:

```yaml
triggers:
  # Time of day
  - platform: time
    at: "07:00:00"
  
  # Cron expression
  - platform: time
    cron: "0 0 */2 * * *"  # Every 2 hours
  
  # Interval
  - platform: time
    interval: "5m"
```

### Event Triggers

Trigger on specific events:

```yaml
triggers:
  - platform: event
    event_type: "automation_triggered"
    event_data:
      source: "motion_sensor"
```

### Sun Triggers

Trigger based on sunrise/sunset:

```yaml
triggers:
  - platform: sun
    event: sunset
    offset: "-00:30:00"  # 30 minutes before sunset
```

### Webhook Triggers

Trigger via HTTP webhooks:

```yaml
triggers:
  - platform: webhook
    webhook_id: "my_webhook"
    method: POST
```

### Composite Triggers

Combine multiple triggers with AND/OR logic:

```yaml
triggers:
  - platform: composite
    operator: and  # or "or"
    triggers:
      - platform: state
        entity_id: sensor.motion
        to: "on"
      - platform: time
        after: "sunset"
```

## Condition Types

### State Conditions

Check entity states:

```yaml
conditions:
  - condition: state
    entity_id: person.john
    state: "home"
    attribute: "battery_level"  # Optional
    for: "00:10:00"  # Optional duration
```

### Time Conditions

Check time-based criteria:

```yaml
conditions:
  - condition: time
    after: "08:00:00"
    before: "22:00:00"
    weekday:
      - mon
      - tue
      - wed
      - thu
      - fri
```

### Numeric Conditions

Compare numeric values:

```yaml
conditions:
  - condition: numeric_state
    entity_id: sensor.temperature
    above: 20.0
    below: 30.0
    attribute: "humidity"  # Optional
```

### Template Conditions

Use template expressions:

```yaml
conditions:
  - condition: template
    value_template: "{{ states('sensor.temperature') | float > 25 }}"
```

### Composite Conditions

Combine conditions with AND/OR logic:

```yaml
conditions:
  - condition: and
    conditions:
      - condition: state
        entity_id: person.john
        state: "home"
      - condition: time
        after: "18:00"
```

## Action Types

### Service Actions

Call Home Assistant or PMA services:

```yaml
actions:
  - service: light.turn_on
    entity_id: light.living_room
    data:
      brightness: 255
      color_name: "blue"
    target:
      area_id: living_room
```

### Notification Actions

Send notifications:

```yaml
actions:
  - service: notify.mobile_app
    data:
      title: "Alert"
      message: "Temperature is {{ states('sensor.temp') }}°C"
      data:
        priority: high
```

### Delay Actions

Introduce delays:

```yaml
actions:
  - delay: "00:05:00"  # 5 minutes
  # or
  - delay:
      hours: 1
      minutes: 30
      seconds: 15
```

### Variable Actions

Set variables:

```yaml
actions:
  - service: variable.set
    data:
      variable: "last_motion_time"
      value: "{{ now() }}"
      scope: "global"  # or "rule"
```

### HTTP Actions

Make HTTP requests:

```yaml
actions:
  - service: http.request
    data:
      url: "https://api.example.com/webhook"
      method: POST
      headers:
        Authorization: "Bearer {{ token }}"
      body:
        temperature: "{{ states('sensor.temp') }}"
```

### Script Actions

Execute shell commands:

```yaml
actions:
  - service: script.execute
    data:
      command: "/usr/bin/backup_script.sh"
      args:
        - "--config"
        - "/etc/backup.conf"
      timeout: "300s"
```

### Conditional Actions

Execute actions based on conditions:

```yaml
actions:
  - service: conditional
    conditions:
      - condition: state
        entity_id: light.living_room
        state: "off"
    then_actions:
      - service: light.turn_on
        entity_id: light.living_room
    else_actions:
      - service: light.turn_off
        entity_id: light.living_room
```

## API Endpoints

### Rule Management

- `GET /api/v1/automations` - List all rules
- `POST /api/v1/automations` - Create new rule
- `GET /api/v1/automations/{id}` - Get rule details
- `PUT /api/v1/automations/{id}` - Update rule
- `DELETE /api/v1/automations/{id}` - Delete rule

### Rule Operations

- `POST /api/v1/automations/{id}/enable` - Enable rule
- `POST /api/v1/automations/{id}/disable` - Disable rule
- `POST /api/v1/automations/{id}/test` - Test rule execution
- `GET /api/v1/automations/{id}/history` - Get execution history

### Import/Export

- `GET /api/v1/automations/{id}/export?format=yaml` - Export rule
- `POST /api/v1/automations/import` - Import rule
- `POST /api/v1/automations/validate` - Validate rule syntax

### Utilities

- `GET /api/v1/automations/templates` - Get rule templates
- `GET /api/v1/automations/statistics` - Get engine statistics

## Usage Examples

### Basic Motion Light

```yaml
id: motion_light
name: "Motion Activated Light"
description: "Turn on light when motion detected"
enabled: true
triggers:
  - platform: state
    entity_id: binary_sensor.motion
    to: "on"
conditions:
  - condition: state
    entity_id: sun.sun
    state: "below_horizon"
actions:
  - service: light.turn_on
    entity_id: light.living_room
    data:
      brightness: 255
```

### Advanced Morning Routine

```yaml
id: morning_routine
name: "Weekday Morning Routine"
description: "Complex morning automation"
enabled: true
mode: single
variables:
  coffee_time: "07:00"
  wake_brightness: 30
triggers:
  - platform: time
    at: "{{ variables.coffee_time }}"
conditions:
  - condition: time
    weekday: [mon, tue, wed, thu, fri]
  - condition: state
    entity_id: person.john
    state: "home"
actions:
  - service: light.turn_on
    target:
      area_id: bedroom
    data:
      brightness: "{{ variables.wake_brightness }}"
      transition: 300
  - delay: "00:05:00"
  - service: switch.turn_on
    entity_id: switch.coffee_maker
  - service: media_player.play_media
    entity_id: media_player.bedroom
    data:
      media_content_id: "spotify:playlist:morning"
      media_content_type: "music"
```

## Configuration

### Engine Configuration

```go
config := &EngineConfig{
    Workers:              4,
    QueueSize:           1000,
    ExecutionTimeout:    30 * time.Second,
    MaxConcurrentRules:  100,
    EnableCircuitBreaker: true,
    CircuitBreakerConfig: &CircuitBreakerConfig{
        FailureThreshold: 5,
        ResetTimeout:     60 * time.Second,
        MaxRequests:      10,
    },
    SchedulerConfig: &SchedulerConfig{
        Timezone: "America/New_York",
    },
}
```

### Scheduler Configuration

```go
schedulerConfig := &SchedulerConfig{
    Timezone:         "UTC",
    MissedJobMaxAge:  "1h",
    MaxConcurrentJobs: 10,
}
```

## Monitoring and Debugging

### Execution Context

Every rule execution creates an ExecutionContext that tracks:

- Variable state
- Execution stack
- Performance metrics
- Detailed trace logs
- Error information

### Statistics

The engine provides comprehensive statistics:

```json
{
  "total_rules": 25,
  "active_rules": 20,
  "total_executions": 1500,
  "successful_executions": 1485,
  "failed_executions": 15,
  "average_execution_time": "150ms",
  "queue_length": 3,
  "active_workers": 2
}
```

### Logging

The system provides structured logging with multiple levels:

- DEBUG: Detailed execution traces
- INFO: Rule lifecycle events
- WARN: Non-critical issues
- ERROR: Execution failures

## Best Practices

### Rule Design

1. **Keep rules simple**: Break complex logic into multiple rules
2. **Use descriptive names**: Make rules self-documenting
3. **Test thoroughly**: Use the test endpoint before enabling
4. **Handle edge cases**: Consider failure scenarios
5. **Use appropriate modes**: Choose single/parallel/queued based on needs

### Performance

1. **Minimize conditions**: Evaluate expensive conditions last
2. **Use efficient triggers**: Avoid overly broad state triggers
3. **Batch actions**: Group related actions together
4. **Monitor execution times**: Watch for slow rules
5. **Set reasonable timeouts**: Prevent runaway executions

### Security

1. **Validate inputs**: Check webhook and HTTP data
2. **Limit script execution**: Restrict shell command access
3. **Use HTTPS**: Encrypt external communications
4. **Monitor logs**: Watch for suspicious activity
5. **Regular backups**: Export rules regularly

## Troubleshooting

### Common Issues

1. **Rule not triggering**: Check trigger conditions and entity states
2. **Actions failing**: Verify service availability and parameters
3. **Template errors**: Test templates in development environment
4. **Timing issues**: Check timezone configuration
5. **Memory usage**: Monitor execution context cleanup

### Debugging Tools

1. **Test endpoint**: Manually test rules with custom data
2. **Execution trace**: Review detailed execution logs
3. **Statistics endpoint**: Monitor engine performance
4. **Validation endpoint**: Check rule syntax before deployment
5. **Log analysis**: Search structured logs for patterns

## Integration Examples

### Home Assistant Integration

```go
// Initialize automation engine with HA client
engine, err := automation.NewAutomationEngine(
    config,
    haClient,
    wsHub,
    logger,
)

// Start the engine
err = engine.Start(context.Background())

// Add rules from Home Assistant
rule, err := parser.ParseFromYAML(yamlData)
err = engine.AddRule(rule)
```

### WebSocket Events

```go
// Listen for Home Assistant events
go func() {
    for event := range haEventChan {
        automationEvent := automation.Event{
            Type:      "state_changed",
            EntityID:  event.EntityID,
            Data:      event.Data,
            Timestamp: time.Now(),
        }
        engine.HandleEvent(automationEvent)
    }
}()
```

## Migration and Backup

### Export Rules

```bash
curl -X GET "http://localhost:3001/api/v1/automations/export?format=yaml" \
  -H "Authorization: Bearer $TOKEN" \
  -o backup.yaml
```

### Import Rules

```bash
curl -X POST "http://localhost:3001/api/v1/automations/import" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/x-yaml" \
  --data-binary @backup.yaml
```

### Bulk Operations

```bash
# Export all rules
for id in $(curl -s "http://localhost:3001/api/v1/automations" | jq -r '.data.rules[].id'); do
  curl -s "http://localhost:3001/api/v1/automations/$id/export?format=yaml" > "rule_$id.yaml"
done
```

## Future Enhancements

1. **Visual Rule Editor**: Web-based drag-and-drop interface
2. **Rule Dependencies**: Manage rule execution order
3. **Advanced Templates**: Extended templating engine
4. **Rule Versioning**: Track and rollback rule changes
5. **Performance Analytics**: Detailed execution metrics
6. **Machine Learning**: Predictive automation suggestions
7. **Mobile App**: Remote rule management
8. **Cloud Sync**: Multi-instance rule synchronization 