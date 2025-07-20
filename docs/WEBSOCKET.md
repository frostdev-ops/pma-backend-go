# PMA Backend Go - WebSocket Guide

This document provides comprehensive documentation for the PMA Backend Go WebSocket implementation, covering real-time communication, message types, and subscription management.

## Table of Contents

- [Overview](#overview)
- [Connection](#connection)
- [Authentication](#authentication)
- [Message Format](#message-format)
- [Message Types](#message-types)
- [Subscription Management](#subscription-management)
- [Event Types](#event-types)
- [Client Examples](#client-examples)
- [Error Handling](#error-handling)
- [Performance & Scaling](#performance--scaling)
- [Troubleshooting](#troubleshooting)

## Overview

The PMA Backend Go WebSocket system provides real-time, bidirectional communication for smart home automation. It enables clients to:

- Receive real-time entity state changes
- Subscribe to specific events and entities
- Get system status updates
- Receive automation triggers
- Monitor adapter health

### Key Features

- **Real-time Updates**: Instant notification of state changes
- **Selective Subscriptions**: Subscribe to specific entities, rooms, or event types
- **Message Batching**: Efficient handling of high-frequency updates
- **Connection Management**: Automatic reconnection and heartbeat
- **Scalable Architecture**: Support for thousands of concurrent connections

## Connection

### WebSocket Endpoint

```
ws://localhost:3001/ws
```

For HTTPS environments:
```
wss://your-domain.com/ws
```

### Connection Parameters

WebSocket connections support the following query parameters:

- `client_id` (optional): Custom client identifier
- `user_agent` (optional): Client user agent string

Example:
```
ws://localhost:3001/ws?client_id=mobile_app&user_agent=PMA-Mobile/1.0
```

### Connection Lifecycle

1. **Handshake**: Client initiates WebSocket connection
2. **Welcome**: Server sends welcome message with client ID
3. **Subscription**: Client subscribes to desired event types
4. **Active Communication**: Real-time message exchange
5. **Heartbeat**: Periodic ping/pong for connection health
6. **Cleanup**: Graceful connection termination

## Authentication

WebSocket connections can be authenticated using query parameters or during the handshake:

### Query Parameter Authentication
```
ws://localhost:3001/ws?token=your_jwt_token
```

### Message-based Authentication
Send authentication message after connection:

```json
{
  "type": "authenticate",
  "data": {
    "token": "your_jwt_token"
  }
}
```

### Unauthenticated Access

Basic WebSocket access is available without authentication, but certain events may require authentication:
- System administration events
- Sensitive entity data
- User-specific notifications

## Message Format

All WebSocket messages use a consistent JSON format:

### Outbound Messages (Server → Client)

```json
{
  "type": "message_type",
  "data": {
    "key": "value"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Inbound Messages (Client → Server)

```json
{
  "type": "action_type",
  "data": {
    "parameter": "value"
  }
}
```

## Message Types

### Core PMA Message Types

#### Entity State Changes
```json
{
  "type": "pma_entity_state_changed",
  "data": {
    "entity_id": "light.living_room",
    "entity_type": "light",
    "source": "homeassistant",
    "old_state": "off",
    "new_state": "on",
    "attributes": {
      "brightness": 255,
      "color_temp": 3000
    },
    "room_id": "living_room",
    "area_id": "main_floor"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

#### Entity Added
```json
{
  "type": "pma_entity_added",
  "data": {
    "entity": {
      "id": "switch.new_device",
      "name": "New Device",
      "type": "switch",
      "source": "homeassistant",
      "state": "off"
    },
    "room_id": "bedroom",
    "area_id": "upstairs"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

#### Entity Removed
```json
{
  "type": "pma_entity_removed",
  "data": {
    "entity_id": "sensor.old_device",
    "source": "homeassistant",
    "room_id": "garage"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

#### Room Updates
```json
{
  "type": "pma_room_updated",
  "data": {
    "room": {
      "id": "living_room",
      "name": "Living Room",
      "area": "main_floor",
      "entity_count": 8
    },
    "action": "updated",
    "source": "homeassistant"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

#### Automation Triggered
```json
{
  "type": "pma_automation_triggered",
  "data": {
    "rule_id": "morning_routine",
    "rule_name": "Morning Routine",
    "trigger": {
      "type": "time",
      "value": "07:00:00"
    },
    "actions_executed": 3,
    "success": true
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### System Messages

#### Sync Status Updates
```json
{
  "type": "sync_status",
  "data": {
    "source": "homeassistant",
    "status": "syncing",
    "message": "Synchronizing entities",
    "entity_count": 150,
    "room_count": 12,
    "progress": 75
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

#### Adapter Status
```json
{
  "type": "adapter_status",
  "data": {
    "adapter_id": "ha_adapter_001",
    "adapter_name": "Home Assistant",
    "source": "homeassistant",
    "status": "connected",
    "health": {
      "cpu_usage": 5.2,
      "memory_usage": 45.8,
      "last_sync": "2024-01-01T11:55:00Z"
    }
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

#### System Status
```json
{
  "type": "system_status",
  "data": {
    "status": "healthy",
    "details": {
      "connected_clients": 15,
      "active_automations": 8,
      "system_load": 0.45
    }
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Control Messages

#### Welcome Message
```json
{
  "type": "welcome",
  "data": {
    "client_id": "client_001",
    "server_time": "2024-01-01T12:00:00Z",
    "message": "Connected to PMA WebSocket server",
    "features": ["subscriptions", "heartbeat", "batching"]
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

#### Heartbeat
```json
{
  "type": "ping",
  "data": {
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

Response:
```json
{
  "type": "pong",
  "data": {
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

## Subscription Management

Clients can subscribe to specific types of events to reduce message volume and improve performance.

### Subscribe to Home Assistant Events

```json
{
  "type": "subscribe_ha_events",
  "data": {
    "event_types": ["state_changed", "automation_triggered", "service_call"]
  }
}
```

### Subscribe to Specific Entities

```json
{
  "type": "subscribe_ha_entities",
  "data": {
    "entity_ids": ["light.living_room", "sensor.temperature", "switch.fan"]
  }
}
```

### Subscribe to Room Updates

```json
{
  "type": "subscribe_ha_rooms",
  "data": {
    "room_ids": ["living_room", "bedroom", "kitchen"]
  }
}
```

### Subscribe to PMA Rooms (Legacy)

```json
{
  "type": "subscribe_room",
  "data": {
    "room_id": 1
  }
}
```

### Unsubscribe Examples

```json
{
  "type": "unsubscribe_ha_events",
  "data": {
    "event_types": ["service_call"]
  }
}
```

```json
{
  "type": "unsubscribe_ha_entities",
  "data": {
    "entity_ids": ["sensor.temperature"]
  }
}
```

### Subscription Confirmation

Server confirms subscriptions:

```json
{
  "type": "subscription_update",
  "data": {
    "action": "subscribed",
    "type": "ha_events",
    "items": ["state_changed", "automation_triggered"],
    "total_subscriptions": 2
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Event Types

### Core Event Categories

1. **Entity Events**
   - `pma_entity_state_changed`
   - `pma_entity_added`
   - `pma_entity_removed`
   - `pma_entity_updated`

2. **Room/Area Events**
   - `pma_room_updated`
   - `pma_room_added`
   - `pma_room_removed`
   - `pma_area_updated`

3. **Automation Events**
   - `pma_automation_triggered`
   - `pma_scene_activated`

4. **System Events**
   - `system_status`
   - `sync_status`
   - `adapter_status`
   - `connection_status`

5. **Debugging Events**
   - `source_event` (for development)

### Event Filtering

Clients can filter events by:
- **Entity Type**: Only light entities
- **Source**: Only Home Assistant events
- **Room**: Only specific rooms
- **Domain**: Only specific domains (lights, switches, etc.)

Example filtered subscription:
```json
{
  "type": "subscribe_ha_events",
  "data": {
    "event_types": ["state_changed"],
    "filters": {
      "domains": ["light", "switch"],
      "rooms": ["living_room", "bedroom"]
    }
  }
}
```

## Client Examples

### JavaScript/Browser

```javascript
class PMAWebSocket {
  constructor(url) {
    this.url = url;
    this.ws = null;
    this.reconnectInterval = 5000;
    this.subscriptions = new Set();
  }

  connect() {
    this.ws = new WebSocket(this.url);
    
    this.ws.onopen = () => {
      console.log('Connected to PMA WebSocket');
      this.restoreSubscriptions();
    };
    
    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };
    
    this.ws.onclose = () => {
      console.log('WebSocket connection closed, reconnecting...');
      setTimeout(() => this.connect(), this.reconnectInterval);
    };
    
    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };
  }

  send(message) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  subscribeToEntities(entityIds) {
    const message = {
      type: 'subscribe_ha_entities',
      data: { entity_ids: entityIds }
    };
    this.send(message);
    entityIds.forEach(id => this.subscriptions.add(id));
  }

  handleMessage(message) {
    switch (message.type) {
      case 'pma_entity_state_changed':
        this.onEntityStateChanged(message.data);
        break;
      case 'welcome':
        console.log('Received welcome:', message.data);
        break;
      case 'ping':
        this.send({ type: 'pong', data: { timestamp: new Date().toISOString() } });
        break;
      default:
        console.log('Received message:', message);
    }
  }

  onEntityStateChanged(data) {
    console.log(`Entity ${data.entity_id} changed from ${data.old_state} to ${data.new_state}`);
    // Update UI
  }

  restoreSubscriptions() {
    if (this.subscriptions.size > 0) {
      this.subscribeToEntities(Array.from(this.subscriptions));
    }
  }
}

// Usage
const pmaWS = new PMAWebSocket('ws://localhost:3001/ws');
pmaWS.connect();

// Subscribe to specific entities
pmaWS.subscribeToEntities(['light.living_room', 'sensor.temperature']);
```

### Node.js

```javascript
const WebSocket = require('ws');

class PMAWebSocketClient {
  constructor(url, options = {}) {
    this.url = url;
    this.options = options;
    this.ws = null;
    this.subscriptions = new Map();
    this.eventHandlers = new Map();
  }

  connect() {
    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(this.url);
      
      this.ws.on('open', () => {
        console.log('Connected to PMA WebSocket server');
        this.setupHeartbeat();
        resolve();
      });
      
      this.ws.on('message', (data) => {
        try {
          const message = JSON.parse(data.toString());
          this.handleMessage(message);
        } catch (error) {
          console.error('Failed to parse message:', error);
        }
      });
      
      this.ws.on('close', () => {
        console.log('WebSocket connection closed');
        this.reconnect();
      });
      
      this.ws.on('error', reject);
    });
  }

  send(message) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message));
    }
  }

  on(eventType, handler) {
    if (!this.eventHandlers.has(eventType)) {
      this.eventHandlers.set(eventType, []);
    }
    this.eventHandlers.get(eventType).push(handler);
  }

  emit(eventType, data) {
    const handlers = this.eventHandlers.get(eventType);
    if (handlers) {
      handlers.forEach(handler => handler(data));
    }
  }

  subscribeToEvents(eventTypes) {
    const message = {
      type: 'subscribe_ha_events',
      data: { event_types: eventTypes }
    };
    this.send(message);
  }

  handleMessage(message) {
    switch (message.type) {
      case 'welcome':
        console.log('Welcome message:', message.data);
        break;
      case 'pma_entity_state_changed':
        this.emit('entityStateChanged', message.data);
        break;
      case 'pma_automation_triggered':
        this.emit('automationTriggered', message.data);
        break;
      case 'ping':
        this.send({ type: 'pong', data: { timestamp: new Date().toISOString() } });
        break;
    }
  }

  setupHeartbeat() {
    setInterval(() => {
      this.send({ type: 'ping', data: { timestamp: new Date().toISOString() } });
    }, 30000);
  }

  reconnect() {
    setTimeout(() => {
      console.log('Attempting to reconnect...');
      this.connect();
    }, 5000);
  }
}

// Usage
const client = new PMAWebSocketClient('ws://localhost:3001/ws');

client.on('entityStateChanged', (data) => {
  console.log('Entity state changed:', data);
});

client.on('automationTriggered', (data) => {
  console.log('Automation triggered:', data);
});

client.connect().then(() => {
  client.subscribeToEvents(['state_changed', 'automation_triggered']);
});
```

### Python

```python
import asyncio
import json
import websockets
from typing import Dict, List, Callable

class PMAWebSocketClient:
    def __init__(self, url: str):
        self.url = url
        self.websocket = None
        self.event_handlers = {}
        self.subscriptions = set()
        
    async def connect(self):
        self.websocket = await websockets.connect(self.url)
        asyncio.create_task(self.listen())
        print("Connected to PMA WebSocket server")
        
    async def listen(self):
        try:
            async for message in self.websocket:
                data = json.loads(message)
                await self.handle_message(data)
        except websockets.exceptions.ConnectionClosed:
            print("WebSocket connection closed")
            await self.reconnect()
            
    async def send(self, message: Dict):
        if self.websocket:
            await self.websocket.send(json.dumps(message))
            
    async def handle_message(self, message: Dict):
        msg_type = message.get('type')
        data = message.get('data', {})
        
        if msg_type == 'welcome':
            print(f"Welcome: {data}")
        elif msg_type == 'pma_entity_state_changed':
            await self.emit('entity_state_changed', data)
        elif msg_type == 'ping':
            await self.send({'type': 'pong', 'data': {'timestamp': '2024-01-01T12:00:00Z'}})
            
    async def emit(self, event: str, data: Dict):
        if event in self.event_handlers:
            for handler in self.event_handlers[event]:
                await handler(data)
                
    def on(self, event: str, handler: Callable):
        if event not in self.event_handlers:
            self.event_handlers[event] = []
        self.event_handlers[event].append(handler)
        
    async def subscribe_to_entities(self, entity_ids: List[str]):
        message = {
            'type': 'subscribe_ha_entities',
            'data': {'entity_ids': entity_ids}
        }
        await self.send(message)
        self.subscriptions.update(entity_ids)
        
    async def reconnect(self):
        await asyncio.sleep(5)
        await self.connect()

# Usage
async def main():
    client = PMAWebSocketClient('ws://localhost:3001/ws')
    
    async def on_entity_changed(data):
        print(f"Entity {data['entity_id']} changed: {data['old_state']} -> {data['new_state']}")
    
    client.on('entity_state_changed', on_entity_changed)
    
    await client.connect()
    await client.subscribe_to_entities(['light.living_room', 'sensor.temperature'])
    
    # Keep the connection alive
    await asyncio.Future()  # Run forever

if __name__ == "__main__":
    asyncio.run(main())
```

## Error Handling

### Connection Errors

- **Connection Refused**: Server not running or port blocked
- **Authentication Failed**: Invalid or expired JWT token
- **Rate Limited**: Too many connection attempts

### Message Errors

```json
{
  "type": "error",
  "data": {
    "code": "INVALID_MESSAGE_FORMAT",
    "message": "Invalid JSON format",
    "details": "Unexpected token at position 15"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Subscription Errors

```json
{
  "type": "subscription_error",
  "data": {
    "action": "subscribe",
    "error": "Invalid entity ID format",
    "entity_id": "invalid.entity",
    "code": "INVALID_ENTITY_ID"
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Performance & Scaling

### Connection Limits

- **Maximum Connections**: 1000 (configurable)
- **Message Buffer Size**: 256 messages per client
- **Compression**: Automatic for large messages

### Optimization Strategies

1. **Selective Subscriptions**: Only subscribe to needed events
2. **Message Batching**: Group related updates
3. **Connection Pooling**: Reuse connections when possible
4. **Heartbeat Tuning**: Adjust ping/pong intervals

### Monitoring

Monitor WebSocket performance:

```bash
# Get WebSocket metrics
curl http://localhost:3001/api/v1/websocket/metrics

# Get connected clients
curl http://localhost:3001/api/v1/websocket/clients
```

## Troubleshooting

### Common Issues

#### Connection Drops
- **Cause**: Network instability, server restart
- **Solution**: Implement automatic reconnection

#### Message Loss
- **Cause**: Buffer overflow, connection issues
- **Solution**: Implement message acknowledgment

#### High Memory Usage
- **Cause**: Too many subscriptions, message buildup
- **Solution**: Optimize subscriptions, increase buffer limits

### Debug Mode

Enable debug logging in configuration:

```yaml
websocket:
  debug: true
  log_messages: true
  log_subscriptions: true
```

### Testing WebSocket Connection

Use `wscat` tool for testing:

```bash
# Install wscat
npm install -g wscat

# Connect and test
wscat -c ws://localhost:3001/ws

# Send test message
{"type": "ping", "data": {}}
```

---

For more information, see the [PMA Backend Go Documentation](../README.md) and [API Reference](API_REFERENCE.md).