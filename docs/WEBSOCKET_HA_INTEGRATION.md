# WebSocket Real-Time Updates Integration

This document explains how the PMA backend integrates WebSocket real-time updates with the unified entity system.

## Overview

The WebSocket system provides real-time updates for:
- Entity state changes (when devices change state)
- New entity discovery (when new devices are found)  
- Entity removal (when devices are disconnected)
- Synchronization status (sync progress from adapters)
- Adapter health and status updates

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Unified Entity  │───▶│ EventEmitter     │───▶│ WebSocket Hub   │
│ Service         │    │ Interface        │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                          │
                                                          ▼
                                                ┌─────────────────┐
                                                │ WebSocket       │
                                                │ Clients         │
                                                │ (Frontend)      │
                                                └─────────────────┘
```

## Setup Example

```go
package main

import (
    "github.com/frostdev-ops/pma-backend-go/internal/websocket"
    "github.com/frostdev-ops/pma-backend-go/internal/core/unified"
    "github.com/sirupsen/logrus"
)

func main() {
    logger := logrus.New()
    
    // Create WebSocket hub
    wsHub := websocket.NewHub(logger)
    go wsHub.Run()
    
    // Create event emitter wrapper
    eventEmitter := websocket.NewWebSocketEventEmitter(wsHub)
    
    // Create unified entity service
    unifiedService := unified.NewUnifiedEntityService(typeRegistry, config, logger)
    
    // Connect the services for real-time updates
    unifiedService.SetEventEmitter(eventEmitter)
    
    // Now entity changes will automatically broadcast to WebSocket clients
}
```

## WebSocket Message Types

### Entity State Changes
```json
{
  "type": "pma_entity_state_changed",
  "data": {
    "entity_id": "light.living_room",
    "old_state": "off",
    "new_state": "on", 
    "entity": { /* full entity object */ },
    "timestamp": "2024-01-01T12:00:00Z"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### New Entity Added
```json
{
  "type": "pma_entity_added",
  "data": {
    "entity": { /* full entity object */ },
    "timestamp": "2024-01-01T12:00:00Z"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Sync Status Updates
```json
{
  "type": "pma_sync_status", 
  "data": {
    "source": "home_assistant",
    "status": "syncing", // "syncing", "completed", "error"
    "details": {
      "entities_found": 25,
      "entities_registered": 3,
      "entities_updated": 8,
      "duration": "2.5s"
    },
    "timestamp": "2024-01-01T12:00:00Z"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Adapter Status Updates
```json
{
  "type": "pma_adapter_status",
  "data": {
    "adapter_id": "ha_adapter_001",
    "adapter_name": "Home Assistant",
    "source": "home_assistant", 
    "status": "connected", // "connected", "disconnected", "error"
    "health": { /* health metrics */ },
    "metrics": { /* performance metrics */ },
    "timestamp": "2024-01-01T12:00:00Z"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Client Subscription

WebSocket clients can subscribe to specific topics for filtered updates:

### Subscribe to Entity Updates
```javascript
// Subscribe to all entity updates
ws.send(JSON.stringify({
  type: "subscribe_topic",
  data: { topic: "entity:*" }
}));

// Subscribe to specific entity
ws.send(JSON.stringify({
  type: "subscribe_topic", 
  data: { topic: "entity:light.living_room" }
}));
```

### Subscribe to Source Updates
```javascript
// Subscribe to Home Assistant updates
ws.send(JSON.stringify({
  type: "subscribe_topic",
  data: { topic: "source:home_assistant" }
}));
```

### Subscribe to Adapter Status
```javascript
// Subscribe to specific adapter
ws.send(JSON.stringify({
  type: "subscribe_topic",
  data: { topic: "adapter:ha_adapter_001" }
}));
```

## Implementation Details

### EventEmitter Interface
The `EventEmitter` interface enables loose coupling between the unified entity service and WebSocket system:

```go
type EventEmitter interface {
    BroadcastPMAEntityStateChange(entityID string, oldState, newState interface{}, entity interface{})
    BroadcastPMAEntityAdded(entity interface{}) 
    BroadcastPMAEntityRemoved(entityID string, source interface{})
    BroadcastPMASyncStatus(source string, status string, details map[string]interface{})
    BroadcastPMAAdapterStatus(adapterID, adapterName, source, status string, health interface{}, metrics interface{})
}
```

### Real-time Triggers

1. **Entity Actions**: When `ExecuteAction()` is called and succeeds
2. **Entity Refresh**: When `refreshEntity()` detects state changes  
3. **Sync Operations**: During `SyncFromSource()` for new/updated entities
4. **Adapter Events**: When adapters connect/disconnect or report health changes

### Performance Considerations

- WebSocket broadcasting is non-blocking (uses goroutines)
- State change detection compares old vs new state to avoid unnecessary broadcasts
- Topic-based subscriptions reduce unnecessary client traffic
- Message queuing for offline clients (configurable)

## Frontend Integration

Use the WebSocket API to receive real-time updates in your frontend:

```javascript
const ws = new WebSocket('ws://localhost:3001/ws');

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  switch(message.type) {
    case 'pma_entity_state_changed':
      updateEntityInUI(message.data.entity_id, message.data.new_state);
      break;
      
    case 'pma_entity_added':
      addEntityToUI(message.data.entity);
      break;
      
    case 'pma_sync_status':
      updateSyncStatus(message.data.source, message.data.status);
      break;
  }
};
```

## Benefits

✅ **Real-time Updates**: UI immediately reflects device state changes  
✅ **Efficient**: Only broadcasts when state actually changes  
✅ **Scalable**: Topic-based subscriptions reduce bandwidth  
✅ **Reliable**: Non-blocking, fault-tolerant broadcasting  
✅ **Flexible**: Easy to add new event types  
✅ **Testable**: EventEmitter interface enables easy mocking 