# WebSocket Home Assistant Integration

This document describes the WebSocket Home Assistant (HA) Integration that enables real-time forwarding of Home Assistant events to connected WebSocket clients.

## Overview

The WebSocket HA Integration bridges Home Assistant events with frontend clients through WebSocket connections. It provides:

- **Real-time event forwarding** from Home Assistant to WebSocket clients
- **Selective event subscriptions** by event type, entity, and room
- **Rate limiting and batching** for performance optimization
- **Comprehensive filtering** to reduce unnecessary traffic
- **Detailed statistics and monitoring** for troubleshooting

## Architecture

```
Home Assistant → Sync Service → HA Event Forwarder → WebSocket Hub → Frontend Clients
```

### Components

1. **HA Event Forwarder** (`internal/websocket/ha_forwarder.go`)
   - Core component that receives HA events and forwards them to clients
   - Handles filtering, rate limiting, and batching
   - Provides statistics and error tracking

2. **Enhanced WebSocket Client** (`internal/websocket/client.go`)
   - Extended to support HA event subscriptions
   - Manages event type, entity, and room filters
   - Handles subscription state management

3. **Message Types** (`internal/websocket/message.go`)
   - Defines HA-specific message structures
   - Provides type safety for event forwarding
   - Supports various HA event types

4. **WebSocket API Handlers** (`internal/api/handlers/websocket.go`)
   - HTTP API endpoints for subscription management
   - Testing and statistics endpoints
   - Client subscription lifecycle management

## Event Types

The integration supports the following Home Assistant event types:

- `ha_state_changed` - Entity state changes
- `ha_entity_added` - New entities discovered
- `ha_entity_removed` - Entities removed
- `ha_area_updated` - Area/room updates
- `ha_service_called` - Service calls
- `ha_sync_status` - Sync service status updates

## Configuration

### YAML Configuration

```yaml
websocket:
  homeassistant:
    enabled: true
    max_events_per_second: 50        # Rate limiting
    batch_events: true               # Enable event batching
    batch_window: "100ms"            # Batching window
    default_subscriptions:           # Default event types for new clients
      - "ha_state_changed"
      - "ha_sync_status"
    forward_all_entities: false      # Bypass entity filtering
    max_errors_retained: 100         # Maximum error history
```

### Programmatic Configuration

```go
config := &websocket.HAEventForwarderConfig{
    MaxEventsPerSecond:   50,
    BatchEvents:          true,
    BatchWindow:          100 * time.Millisecond,
    DefaultSubscriptions: []string{
        websocket.MessageTypeHAStateChanged,
        websocket.MessageTypeHASyncStatus,
    },
    ForwardAllEntities:   false,
    MaxErrorsRetained:    100,
}

forwarder := websocket.NewHAEventForwarder(hub, logger, config)
```

## Usage

### Setting Up the Integration

```go
// Create WebSocket Hub
wsHub := websocket.NewHub(logger)

// Create HA Event Forwarder
haForwarder := websocket.NewHAEventForwarder(wsHub, logger, config)

// Connect to sync service
syncService.SetEventForwarder(haForwarder)

// Start components
go wsHub.Run()
haForwarder.Start()
```

### Client Subscription (WebSocket Messages)

Clients can subscribe to HA events by sending WebSocket messages:

#### Subscribe to Event Types
```json
{
  "type": "subscribe_ha_events",
  "data": {
    "event_types": ["ha_state_changed", "ha_sync_status"]
  }
}
```

#### Subscribe to Specific Entities
```json
{
  "type": "subscribe_ha_entities",
  "data": {
    "entity_ids": ["light.living_room", "sensor.temperature"]
  }
}
```

#### Subscribe to Room Events
```json
{
  "type": "subscribe_ha_rooms",
  "data": {
    "room_ids": ["living_room", "bedroom"]
  }
}
```

### HTTP API Endpoints

#### Subscribe to HA Events
```http
POST /api/v1/websocket/ha/subscribe?client_id=<client_id>
Content-Type: application/json

{
  "event_types": ["ha_state_changed"],
  "room_ids": ["living_room"],
  "entity_ids": ["light.living_room"]
}
```

#### Get Subscription Status
```http
GET /api/v1/websocket/ha/subscriptions?client_id=<client_id>
```

#### Get Forwarding Statistics
```http
GET /api/v1/websocket/ha/stats
```

#### Test Event Forwarding
```http
POST /api/v1/websocket/ha/test
Content-Type: application/json

{
  "event_type": "ha_state_changed",
  "entity_id": "light.test",
  "test_data": {
    "brightness": 255
  }
}
```

## Message Formats

### State Changed Message
```json
{
  "type": "ha_state_changed",
  "data": {
    "entity_id": "light.living_room",
    "old_state": "off",
    "new_state": "on",
    "attributes": {
      "brightness": 255,
      "color_temp": 154,
      "friendly_name": "Living Room Light"
    },
    "room_id": "living_room"
  },
  "timestamp": "2023-12-07T10:30:00Z"
}
```

### Sync Status Message
```json
{
  "type": "ha_sync_status",
  "data": {
    "status": "connected",
    "message": "Successfully connected to Home Assistant",
    "entity_count": 42
  },
  "timestamp": "2023-12-07T10:30:00Z"
}
```

### Entity Added Message
```json
{
  "type": "ha_entity_added",
  "data": {
    "entity_id": "light.new_bulb",
    "entity_data": {
      "friendly_name": "New Smart Bulb",
      "state": "off",
      "domain": "light"
    },
    "room_id": "bedroom"
  },
  "timestamp": "2023-12-07T10:30:00Z"
}
```

## Filtering and Subscriptions

### Event Type Filtering
Clients subscribe to specific event types. Only subscribed event types are forwarded.

### Entity Filtering
- If no entity filters are set, all entities are forwarded
- If entity filters are set, only specified entities are forwarded
- Useful for dashboard views focused on specific devices

### Room Filtering
- If no room filters are set, all rooms are forwarded
- If room filters are set, only entities in specified rooms are forwarded
- Requires entity-to-room mapping to be configured

### Combined Filtering
All filters are applied with AND logic:
- Event type must be subscribed
- Entity must pass entity filter (if set)
- Entity's room must pass room filter (if set)

## Performance Features

### Rate Limiting
- Configurable maximum events per second
- Prevents overwhelming clients during HA sync operations
- Drops excess events with statistics tracking

### Event Batching
- Groups rapid events into batches
- Reduces WebSocket traffic overhead
- Configurable batch window (default 100ms)
- Maintains event ordering within batches

### Statistics Tracking
- Events forwarded/dropped counters
- Per-event-type statistics
- Client connection metrics
- Error tracking and reporting

## Error Handling

### Client Disconnection
- Automatic cleanup of disconnected clients
- No memory leaks from stale subscriptions
- Graceful handling of send channel full conditions

### Event Processing Errors
- Failed events are logged but don't stop processing
- Error details tracked for debugging
- Configurable error history retention

### Rate Limit Exceeded
- Events are dropped silently when rate limit exceeded
- Statistics track dropped event count
- No client disconnection on rate limit

## Monitoring and Debugging

### Statistics API
The `/api/v1/websocket/ha/stats` endpoint provides:

```json
{
  "events_forwarded": 1234,
  "events_dropped": 5,
  "connected_clients": 3,
  "subscribed_clients": 2,
  "last_event_time": "2023-12-07T10:30:00Z",
  "batched_events": 500,
  "batches_processed": 50,
  "event_type_stats": {
    "ha_state_changed": 800,
    "ha_sync_status": 10
  },
  "forwarding_errors": []
}
```

### Testing
Use the test endpoint to verify event forwarding:

```bash
curl -X POST http://localhost:3001/api/v1/websocket/ha/test \
  -H "Content-Type: application/json" \
  -d '{
    "event_type": "ha_state_changed",
    "entity_id": "light.test",
    "test_data": {"brightness": 255}
  }'
```

## Integration with Sync Service

The sync service forwards events to the WebSocket integration:

```go
// In sync service event handlers
func (s *SyncService) handleStateChanged(event homeassistant.Event) error {
    // ... existing sync logic ...
    
    // Forward to WebSocket clients
    if s.eventForwarder != nil {
        s.forwardStateChangeEvent(entityID, oldState, newState)
    }
    
    return nil
}
```

## Example Implementation

See `examples/websocket_ha_integration.go` for a complete working example that demonstrates:

- Setting up the integration
- Configuring event types and filters
- Simulating HA events
- Monitoring statistics
- Client subscription patterns

## Testing

Run the comprehensive test suite:

```bash
# Unit tests
go test ./internal/websocket/...

# Benchmark tests
go test -bench=. ./internal/websocket/

# Integration tests with real WebSocket connections
go test -tags=integration ./internal/websocket/...
```

### Test Coverage

The test suite covers:
- Event forwarding logic
- Client filtering behavior
- Rate limiting functionality
- Event batching performance
- Error handling scenarios
- Statistics accuracy
- Memory leak prevention

## Best Practices

### For Developers

1. **Always check if forwarder is available** before calling methods
2. **Use appropriate event types** for different scenarios
3. **Configure rate limits** based on expected event volume
4. **Monitor statistics** for performance optimization
5. **Test with multiple clients** to verify filtering

### For Frontend Clients

1. **Subscribe only to needed event types** to reduce traffic
2. **Use entity/room filters** for focused views
3. **Handle connection errors gracefully**
4. **Implement reconnection logic** for reliability
5. **Process batched events efficiently**

### For Operations

1. **Monitor forwarding statistics** for anomalies
2. **Adjust rate limits** based on client capacity
3. **Review error logs** for integration issues
4. **Test event forwarding** after configuration changes
5. **Monitor memory usage** with many connected clients

## Troubleshooting

### Common Issues

**Events not being forwarded:**
- Check client subscription status
- Verify event type is enabled
- Check entity/room filters
- Review rate limiting settings

**High memory usage:**
- Check for client connection leaks
- Review error retention settings
- Monitor batch sizes
- Verify client cleanup

**Performance issues:**
- Adjust rate limiting
- Enable/optimize batching
- Review filter efficiency
- Check client processing speed

### Debug Logging

Enable debug logging for detailed event flow:

```go
logger.SetLevel(logrus.DebugLevel)
```

This provides detailed logs for:
- Event forwarding decisions
- Client filtering results
- Batch processing details
- Error conditions 