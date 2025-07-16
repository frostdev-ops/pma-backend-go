# Home Assistant Synchronization Service

This package provides comprehensive synchronization between Home Assistant and the PMA backend system, enabling real-time bidirectional updates and state consistency.

## Overview

The Home Assistant Synchronization Service acts as a bridge between Home Assistant and PMA, ensuring that entity states, room configurations, and other data remain synchronized across both systems.

## Architecture

### Core Components

1. **SyncService** (`sync_service.go`) - Main orchestrator for synchronization operations
2. **EntityMapper** (`entity_mapper.go`) - Handles mapping between HA and PMA entity formats
3. **EventProcessor** (`event_handler.go`) - Processes real-time WebSocket events from Home Assistant
4. **API Handlers** (`../api/handlers/homeassistant_sync.go`) - REST API endpoints for sync management

### Data Flow

```
Home Assistant ←→ HA Client ←→ Sync Service ←→ Entity/Room Services ←→ Database
                      ↓
                 Event Processor ←→ WebSocket Events
                      ↓
                 PMA WebSocket Hub ←→ Frontend Clients
```

## Features

### Synchronization Capabilities

- **Full Sync**: Complete synchronization of all supported entities
- **Incremental Sync**: Real-time updates via WebSocket events
- **Selective Sync**: Sync specific entities or rooms
- **Bidirectional Updates**: PMA → HA and HA → PMA

### Supported Entity Domains

- `light` - Lights and dimmable devices
- `switch` - Simple on/off switches
- `sensor` - Read-only sensors
- `binary_sensor` - Binary state sensors
- `climate` - HVAC and climate control
- `cover` - Blinds, curtains, garage doors
- `fan` - Fans and ventilation
- `lock` - Smart locks
- `vacuum` - Vacuum cleaners
- `media_player` - Audio/video devices

### Error Handling & Resilience

- **Retry Logic**: Configurable retry attempts with exponential backoff
- **Conflict Resolution**: Multiple strategies (HA wins, PMA wins, timestamp-based)
- **Graceful Degradation**: Service continues operating if HA is temporarily unavailable
- **Error Tracking**: Comprehensive error logging and statistics

## Configuration

### Sync Configuration

```yaml
homeassistant:
  sync:
    enabled: true
    full_sync_interval: "1h"
    supported_domains:
      - "light"
      - "switch"
      - "sensor"
      - "binary_sensor"
      - "climate"
      - "cover"
    conflict_resolution: "homeassistant_wins"  # or "pma_wins", "timestamp"
    batch_size: 100
    retry_attempts: 3
    retry_delay: "5s"
    event_buffer_size: 1000
    event_processing_delay: "100ms"
```

### Conflict Resolution Strategies

1. **homeassistant_wins**: Home Assistant state always takes precedence
2. **pma_wins**: PMA state takes precedence
3. **timestamp**: Most recently updated state wins

## Usage

### Initialization

```go
// Create sync service
syncService := homeassistant.NewSyncService(
    haClient,
    entityService,
    roomService,
    configRepo,
    wsHub,
    logger,
    syncConfig,
)

// Start the service
if err := syncService.Start(ctx); err != nil {
    log.Fatal("Failed to start sync service:", err)
}

// Graceful shutdown
defer syncService.Stop(ctx)
```

### Manual Operations

```go
// Trigger full synchronization
err := syncService.FullSync(ctx)

// Sync specific entity
err := syncService.SyncEntity(ctx, "light.living_room")

// Sync all entities in a room
err := syncService.SyncRoom(ctx, "1")

// Update entity state in HA
err := syncService.UpdateEntityState(ctx, "light.bedroom", "on", map[string]interface{}{
    "brightness": 200,
})

// Call HA service
err := syncService.CallService(ctx, "light", "turn_on", "light.kitchen", map[string]interface{}{
    "brightness": 150,
    "color_temp": 4000,
})
```

## API Endpoints

### Sync Management

- `POST /api/v1/ha/sync/full` - Trigger full synchronization
- `GET /api/v1/ha/sync/status` - Get synchronization status and statistics
- `POST /api/v1/ha/sync/entity/{id}` - Sync specific entity
- `POST /api/v1/ha/sync/room/{id}` - Sync all entities in room

### Entity Control

- `PUT /api/v1/ha/entity/{id}/state` - Update entity state
- `POST /api/v1/ha/service/{domain}/{service}` - Call HA service

### Example API Usage

```bash
# Trigger full sync
curl -X POST http://localhost:3001/api/v1/ha/sync/full

# Get sync status
curl http://localhost:3001/api/v1/ha/sync/status

# Sync specific entity
curl -X POST http://localhost:3001/api/v1/ha/sync/entity/light.living_room

# Update entity state
curl -X PUT http://localhost:3001/api/v1/ha/entity/light.bedroom/state \
  -H "Content-Type: application/json" \
  -d '{"state": "on", "attributes": {"brightness": 200}}'

# Call service
curl -X POST http://localhost:3001/api/v1/ha/service/light/turn_on \
  -H "Content-Type: application/json" \
  -d '{"entity_id": "light.kitchen", "data": {"brightness": 150}}'
```

## Monitoring & Statistics

### Sync Statistics

The service provides comprehensive statistics via the `GetSyncStats()` method:

```go
type SyncStats struct {
    LastFullSync     time.Time   `json:"last_full_sync"`
    EntitiesSynced   int         `json:"entities_synced"`
    SyncErrors       []SyncError `json:"sync_errors"`
    IsConnected      bool        `json:"is_connected"`
    EventsProcessed  int64       `json:"events_processed"`
    LastEventTime    time.Time   `json:"last_event_time"`
    CurrentOperation string      `json:"current_operation"`
}
```

### Error Tracking

Errors are tracked with detailed information:

```go
type SyncError struct {
    Type      string    `json:"type"`
    EntityID  string    `json:"entity_id,omitempty"`
    Operation string    `json:"operation"`
    Error     string    `json:"error"`
    Timestamp time.Time `json:"timestamp"`
    Retryable bool      `json:"retryable"`
}
```

## WebSocket Events

The service broadcasts sync events to connected clients:

```json
{
  "type": "sync_event",
  "event": "full_sync_completed",
  "data": {
    "entities_synced": 150,
    "duration": "2.5s"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

### Event Types

- `full_sync_started` - Full synchronization initiated
- `full_sync_completed` - Full synchronization finished
- `entity_synced` - Individual entity synchronized
- `sync_error` - Synchronization error occurred
- `service_called` - HA service called successfully

## Testing

### Unit Tests

Tests are located in `sync_service_test.go` and cover:

- Service lifecycle (start/stop)
- Full synchronization logic
- Entity mapping and filtering
- Error handling and recovery
- Event processing
- WebSocket integration

### Integration Testing

For integration testing with a live Home Assistant instance:

1. Set up test HA instance with known entities
2. Configure test tokens and URLs
3. Run integration test suite
4. Verify bidirectional synchronization

### Performance Testing

- Test with large entity sets (1000+ entities)
- Measure sync performance and memory usage
- Test WebSocket event processing throughput
- Verify graceful degradation under load

## Implementation Notes

### Entity ID Normalization

Entity IDs are normalized to ensure consistency:
- Converted to lowercase
- Whitespace trimmed
- Domain validation performed

### Attribute Mapping

HA attributes are mapped and normalized for PMA:
- Internal HA attributes filtered out
- Nested objects flattened where appropriate
- Data types normalized

### Area/Room Synchronization

HA Areas are synchronized with PMA Rooms:
- Automatic room creation for new areas
- Room name updates when area names change
- Entity assignment to rooms based on area membership

## Troubleshooting

### Common Issues

1. **Sync Service Won't Start**
   - Check HA client configuration
   - Verify network connectivity
   - Check authentication tokens

2. **Entities Not Syncing**
   - Verify domain is in supported list
   - Check entity filtering logic
   - Review sync error logs

3. **WebSocket Connection Issues**
   - Check HA WebSocket endpoint
   - Verify authentication
   - Review connection state logs

4. **Performance Issues**
   - Adjust batch size configuration
   - Increase event buffer size
   - Check for memory leaks

### Debug Logging

Enable debug logging for detailed operation traces:

```yaml
logging:
  level: "debug"
```

### Health Checks

Monitor service health via:
- Sync statistics endpoint
- Connection state monitoring
- Error rate tracking
- Performance metrics

## Future Enhancements

### Planned Features

1. **Advanced Filtering**: Entity filtering by attributes, state values
2. **Custom Mappings**: User-defined entity attribute mappings
3. **Backup/Restore**: Configuration backup and restore capabilities
4. **Rate Limiting**: Configurable rate limiting for API calls
5. **Metrics Export**: Prometheus metrics for monitoring
6. **Device Discovery**: Automatic device discovery and classification

### Extensibility

The service is designed for extensibility:

- Plugin system for custom entity types
- Configurable mapping strategies
- Custom event processors
- Integration with other home automation systems

## Contributing

When contributing to the sync service:

1. Follow existing code patterns and structure
2. Add comprehensive tests for new features
3. Update documentation for API changes
4. Consider backward compatibility
5. Ensure error handling and logging

## License

This component is part of the PMA backend system and follows the same licensing terms. 