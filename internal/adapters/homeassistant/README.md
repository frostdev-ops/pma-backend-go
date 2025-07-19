# HomeAssistant Adapter

This package provides a complete HomeAssistant adapter implementation for the PMA (Personal Management Assistant) backend system. The adapter implements the `PMAAdapter` interface and handles all conversions between HomeAssistant entities and PMA unified types.

## Overview

The HomeAssistant adapter consists of several key components:

- **adapter.go**: Main adapter implementation with PMAAdapter interface
- **client_wrapper.go**: HTTP and WebSocket client for HomeAssistant API
- **converter.go**: Entity type conversion logic between HA and PMA formats
- **mapper.go**: State and attribute mapping with action routing
- **example_integration.go**: Usage examples and integration patterns

## Features

### Core Functionality
- ✅ Complete PMAAdapter interface implementation
- ✅ Bidirectional entity conversion (HA ↔ PMA)
- ✅ Support for 10+ entity types (lights, switches, sensors, etc.)
- ✅ Real-time updates via WebSocket
- ✅ Action execution with comprehensive error handling
- ✅ Health monitoring and metrics tracking
- ✅ Room/Area synchronization
- ✅ Quality scoring for entity reliability

### Supported Entity Types
- **Light**: Full support including brightness, color, and color temperature
- **Switch**: Basic on/off/toggle functionality
- **Sensor**: Numeric values with unit conversion and device class detection
- **Binary Sensor**: Motion, connectivity, and other binary sensors
- **Climate**: Temperature and humidity control
- **Cover**: Position control with open/close/stop operations
- **Camera**: Snapshot and recording capabilities
- **Lock**: Lock/unlock operations
- **Fan**: Speed control and on/off operations
- **Media Player**: Playback control and volume management

### Capabilities Detection
- ✅ Automatic capability detection from HA attributes
- ✅ Brightness and dimming support
- ✅ Color control (RGB, color temperature)
- ✅ Temperature and humidity monitoring
- ✅ Position control for covers
- ✅ Battery level monitoring
- ✅ Motion detection
- ✅ Connectivity status

## Configuration

Add HomeAssistant configuration to your `config.yaml`:

```yaml
home_assistant:
  url: "http://homeassistant.local:8123"
  token: "your_long_lived_access_token"
  sync:
    enabled: true
    full_sync_interval: "5m"
    supported_domains:
      - "light"
      - "switch"
      - "sensor"
      - "binary_sensor"
      - "climate"
      - "cover"
      - "camera"
      - "lock"
      - "fan"
      - "media_player"
    conflict_resolution: "homeassistant_priority"
    batch_size: 100
    retry_attempts: 3
    retry_delay: "1s"
    event_buffer_size: 1000
    event_processing_delay: "100ms"
```

## Usage Examples

### Basic Integration

```go
package main

import (
    "github.com/frostdev-ops/pma-backend-go/internal/adapters/homeassistant"
    "github.com/frostdev-ops/pma-backend-go/internal/config"
    "github.com/sirupsen/logrus"
)

func main() {
    logger := logrus.New()
    cfg := &config.Config{ /* your config */ }
    
    // Create adapter
    adapter := homeassistant.NewHomeAssistantAdapter(cfg, logger)
    
    // Register with adapter registry
    registry.RegisterAdapter(adapter)
}
```

### Entity Synchronization

```go
ctx := context.Background()

// Connect to HomeAssistant
err := adapter.Connect(ctx)
if err != nil {
    log.Fatal("Failed to connect:", err)
}
defer adapter.Disconnect(ctx)

// Sync all entities
entities, err := adapter.SyncEntities(ctx)
if err != nil {
    log.Fatal("Failed to sync entities:", err)
}

log.Printf("Synced %d entities", len(entities))

// Sync rooms/areas
rooms, err := adapter.SyncRooms(ctx)
if err != nil {
    log.Fatal("Failed to sync rooms:", err)
}

log.Printf("Synced %d rooms", len(rooms))
```

### Action Execution

```go
// Turn on a light with specific settings
action := types.PMAControlAction{
    EntityID: "ha_light.living_room",
    Action:   "turn_on",
    Parameters: map[string]interface{}{
        "brightness": 0.8,  // 80% brightness
        "color": map[string]interface{}{
            "r": 255.0,
            "g": 200.0,
            "b": 100.0,
        },
    },
}

result, err := adapter.ExecuteAction(ctx, action)
if err != nil {
    log.Fatal("Action failed:", err)
}

if result.Success {
    log.Println("Light turned on successfully")
} else {
    log.Printf("Action failed: %s", result.Error.Message)
}
```

### Health and Metrics Monitoring

```go
// Get adapter health
health := adapter.GetHealth()
if !health.IsHealthy {
    log.Printf("Adapter unhealthy: %v", health.Issues)
}

// Get performance metrics
metrics := adapter.GetMetrics()
log.Printf("Managed entities: %d", metrics.EntitiesManaged)
log.Printf("Success rate: %.2f%%", 
    float64(metrics.SuccessfulActions)/float64(metrics.ActionsExecuted)*100)
```

## Entity ID Format

The adapter uses a consistent entity ID format:

- **PMA Format**: `ha_{domain}.{entity_name}` (e.g., `ha_light.living_room`)
- **HA Format**: `{domain}.{entity_name}` (e.g., `light.living_room`)

The adapter automatically handles conversion between these formats.

## Error Handling

The adapter provides comprehensive error handling:

- **Connection Errors**: Detailed connection failure information
- **Action Errors**: Categorized error codes with retry indicators
- **Conversion Errors**: Validation errors for invalid entity data
- **API Errors**: HTTP status codes and error messages from HA

Error types include:
- `INVALID_ACTION`: Missing required action parameters
- `MAPPING_ERROR`: Unable to map PMA action to HA service
- `EXECUTION_ERROR`: HA API call failed
- `CONNECTION_ERROR`: Unable to connect to HA instance

## Quality Scoring

The adapter calculates quality scores (0.0-1.0) for entities based on:

- **Availability**: Reduces score for unavailable entities
- **State Quality**: Penalizes unknown states
- **Metadata Completeness**: Rewards friendly names and area assignments
- **Device Integration**: Higher scores for properly configured devices

## WebSocket Support

The adapter supports real-time updates via HomeAssistant's WebSocket API:

- Automatic reconnection with exponential backoff
- Event filtering and batching
- State change propagation to PMA system
- Error recovery and logging

## Testing

Run the comprehensive test suite:

```bash
go test ./internal/adapters/homeassistant -v
```

Tests cover:
- Interface implementation compliance
- Entity conversion accuracy
- Action mapping correctness
- Error handling scenarios
- Health and metrics tracking

## Thread Safety

The adapter is fully thread-safe:
- All public methods use appropriate locking
- Concurrent entity conversions are supported
- Metrics updates are atomic
- WebSocket operations are synchronized

## Performance Considerations

- **Batch Processing**: Entities are processed in configurable batches
- **Connection Pooling**: HTTP client reuses connections
- **Caching**: Converted entities can be cached by the registry
- **Async Operations**: Non-blocking action execution
- **Memory Management**: Efficient attribute copying and filtering

## Integration with PMA System

The adapter integrates seamlessly with:

- **AdapterRegistry**: For registration and discovery
- **EntityRegistry**: For unified entity management
- **UnifiedEntityService**: For cross-adapter operations
- **WebSocket Hub**: For real-time event distribution
- **Metrics System**: For monitoring and alerting

## Troubleshooting

### Common Issues

1. **Connection Failed**
   - Verify HomeAssistant URL and token
   - Check network connectivity
   - Ensure HA is running and accessible

2. **Entity Not Found**
   - Verify entity exists in HomeAssistant
   - Check entity ID format
   - Ensure entity is not disabled

3. **Action Failed**
   - Verify entity supports the requested action
   - Check action parameters
   - Review HomeAssistant logs

4. **Sync Issues**
   - Check HomeAssistant API permissions
   - Verify supported domains configuration
   - Monitor memory usage during large syncs

### Debug Logging

Enable debug logging for detailed information:

```go
logger.SetLevel(logrus.DebugLevel)
```

This will log:
- Entity conversion details
- API request/response data
- WebSocket message flow
- Action mapping decisions

## Contributing

When contributing to the HomeAssistant adapter:

1. Follow the existing code structure
2. Add tests for new functionality
3. Update documentation for changes
4. Ensure thread safety for concurrent operations
5. Handle errors gracefully with appropriate error codes

## Dependencies

- **github.com/gorilla/websocket**: WebSocket client
- **github.com/sirupsen/logrus**: Structured logging
- **github.com/stretchr/testify**: Testing framework

All dependencies are included in the main project's `go.mod` file. 