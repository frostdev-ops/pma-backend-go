# Home Assistant Client Implementation

This package provides a robust Home Assistant client that supports both REST API and WebSocket connectivity for real-time communication with Home Assistant instances.

## Features

### REST API Client
- Configuration retrieval (`GetConfig`)
- Entity state management (`GetStates`, `GetState`, `SetState`)
- Service calls (`CallService`)
- Area/Room management (`GetAreas`, `GetArea`)
- Device management (`GetDevices`)
- Automatic retry logic with exponential backoff
- Request timeout handling
- Rate limiting compliance
- Comprehensive error handling

### WebSocket Client (Stub Implementation)
- Connection management
- Event subscriptions
- State change subscriptions
- Automatic reconnection (when fully implemented)
- Ping/pong for connection health

### Main Client
- Unified interface combining REST and WebSocket functionality
- Configuration management from database and config files
- Token retrieval from database with config file fallback
- Health checking
- Connection status monitoring

## Architecture

```
┌─────────────────┐
│   Main Client   │
├─────────────────┤
│  REST Client    │
│ WebSocket Client│
└─────────────────┘
        │
        ├── Models (HAConfig, EntityState, Area, etc.)
        ├── Error Types (HAError, custom errors)
        └── Configuration (Database + Config File)
```

## Usage

### Basic Setup

```go
import (
    "context"
    "github.com/frostdev-ops/pma-backend-go/internal/adapters/homeassistant"
    "github.com/frostdev-ops/pma-backend-go/internal/config"
    "github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
)

// Create client
client, err := homeassistant.NewClient(cfg, configRepo, logger)
if err != nil {
    log.Fatal(err)
}

// Initialize
ctx := context.Background()
if err := client.Initialize(ctx); err != nil {
    log.Fatal(err)
}

// Use the client
config, err := client.GetConfig(ctx)
states, err := client.GetStates(ctx)
```

### Configuration

The client supports multiple configuration sources:

1. **Database (Priority 1)**: Token stored in `system_config` table with key `home_assistant_token`
2. **Config File (Priority 2)**: Token in `config.yaml` under `home_assistant.token`
3. **Environment Variable (Priority 3)**: `HOME_ASSISTANT_TOKEN` environment variable

Base URL is configured in `config.yaml`:
```yaml
home_assistant:
  url: "http://192.168.100.2:8123"
  token: "optional-fallback-token"
```

### REST API Examples

```go
// Get all entity states
states, err := client.GetStates(ctx)

// Get specific entity state
state, err := client.GetState(ctx, "light.living_room")

// Call a service
err = client.CallService(ctx, "light", "turn_on", map[string]interface{}{
    "entity_id": "light.living_room",
    "brightness": 255,
})

// Get all areas
areas, err := client.GetAreas(ctx)
```

### WebSocket Examples (Stub Implementation)

```go
// Subscribe to all events
subID, err := client.SubscribeToEvents("", func(event homeassistant.Event) {
    log.Printf("Received event: %s", event.EventType)
})

// Subscribe to state changes for specific entity
subID, err := client.SubscribeToStateChanges("light.living_room", 
    func(entityID string, oldState, newState *homeassistant.EntityState) {
        log.Printf("Entity %s changed from %s to %s", 
            entityID, oldState.State, newState.State)
    })

// Unsubscribe
err = client.Unsubscribe(subID)
```

## Error Handling

The client provides comprehensive error handling with custom error types:

```go
if err != nil {
    if homeassistant.IsAuthError(err) {
        // Handle authentication errors
        log.Error("Authentication failed - check token")
    } else if homeassistant.IsConnectionError(err) {
        // Handle connection errors
        log.Error("Connection failed - check URL and network")
    } else {
        // Handle other errors
        log.Errorf("Other error: %v", err)
    }
}
```

## Current Implementation Status

### ✅ Completed
- **REST Client**: Full implementation with retry logic, error handling, and all major API endpoints
- **Models**: Complete data structures for HA API responses
- **Error Types**: Custom error types with helper functions
- **Main Client**: Configuration management, token handling, unified interface
- **Tests**: Unit tests with mock dependencies
- **Documentation**: This README

### 🚧 WebSocket Client (Stub Implementation)
The WebSocket client currently provides a stub implementation that:
- ✅ Implements the complete interface
- ✅ Provides proper method signatures
- ✅ Logs all operations for debugging
- ❌ Does not establish real WebSocket connections
- ❌ Does not handle real-time events
- ❌ Does not implement automatic reconnection

### 🔄 Next Steps for Full WebSocket Implementation
When ready to implement full WebSocket functionality:

1. Add `github.com/gorilla/websocket` dependency usage
2. Implement WebSocket connection establishment
3. Add authentication flow per HA WebSocket API
4. Implement message ID tracking and response correlation
5. Add event dispatching to registered handlers
6. Implement automatic reconnection with exponential backoff
7. Add ping/pong for connection health monitoring

## Integration Points

The client is designed to integrate with:

1. **Entity Service**: For synchronizing HA entities with local database
2. **Room Service**: For mapping HA areas to local rooms
3. **WebSocket Hub**: For forwarding HA events to connected clients
4. **Configuration Service**: For managing HA connection settings

## Configuration Management

### Database Token Storage
```sql
INSERT INTO system_config (key, value, encrypted, description, updated_at) 
VALUES ('home_assistant_token', 'your-long-lived-access-token', 0, 'Home Assistant Long-Lived Access Token', datetime('now'));
```

### Config File Example
```yaml
home_assistant:
  url: "http://192.168.100.2:8123"
  token: "fallback-token-if-not-in-database"
```

## Testing

Run tests with:
```bash
go test ./internal/adapters/homeassistant/ -v
```

Tests cover:
- Client creation and initialization
- Configuration management
- Token retrieval from database vs config
- Error type functionality
- Connection status reporting

## Security Considerations

1. **Token Storage**: Consider encrypting tokens in the database for production
2. **HTTPS**: Use HTTPS URLs for Home Assistant when possible
3. **Token Rotation**: Implement token refresh mechanisms for enhanced security
4. **Access Control**: Ensure proper access control for HA client configuration

## Performance Considerations

1. **Connection Pooling**: REST client uses HTTP connection pooling
2. **Retry Logic**: Exponential backoff prevents overwhelming HA instance
3. **Rate Limiting**: Built-in rate limiting compliance
4. **Timeouts**: Configurable request timeouts prevent hanging requests 