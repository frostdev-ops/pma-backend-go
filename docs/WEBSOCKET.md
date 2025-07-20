# PMA Backend Go - WebSocket Guide

This document provides a comprehensive guide to using the PMA Backend Go WebSocket for real-time communication, including connection, authentication, subscription management, and message formats.

## Table of Contents

- [Introduction](#introduction)
- [Connection Lifecycle](#connection-lifecycle)
- [Authentication](#authentication)
- [Message Format](#message-format)
- [Message Types](#message-types)
  - [Client to Server](#client-to-server)
  - [Server to Client](#server-to-client)
- [Subscription Management](#subscription-management)
- [Client Examples](#client-examples)
  - [JavaScript (Browser/Node.js)](#javascript-browsernodejs)
  - [Python](#python)
- [Error Handling](#error-handling)
- [Performance & Best Practices](#performance--best-practices)
- [Troubleshooting](#troubleshooting)

## Introduction

The WebSocket service provides a persistent, low-latency connection for real-time updates on entity states, system events, and automation triggers. It is the primary mechanism for frontends to stay in sync with the backend.

## Connection Lifecycle

1. **Connect**: Establish a WebSocket connection to `ws://your-pma-backend/ws`.
2. **Authenticate (Optional)**: Send an `authenticate` message with a valid JWT token.
3. **Subscribe**: Send subscription messages for desired event types.
4. **Receive Messages**: Handle real-time messages from the server.
5. **Keep-alive**: The server sends periodic ping frames; clients should respond with pong frames.
6. **Disconnect/Reconnect**: The client should implement auto-reconnect logic.

## Authentication

Authentication is optional but recommended for personalized features.

**Authentication Message:**
```json
{
  "type": "authenticate",
  "data": {
    "token": "your-jwt-token"
  }
}
```

**Server Response (Success):**
```json
{
  "type": "authentication_success",
  "data": {
    "user_id": "user123",
    "message": "Authentication successful"
  }
}
```

**Server Response (Failure):**
```json
{
  "type": "authentication_failure",
  "data": {
    "error": "Invalid token"
  }
}
```

## Message Format

All messages are in JSON format and follow a consistent structure:

```json
{
  "type": "message_type",
  "timestamp": "2024-01-01T12:00:00Z",
  "data": { /* Message payload */ },
  "request_id": "optional-request-id"
}
```

## Message Types

### Client to Server

#### `authenticate`
- **Description**: Authenticate the WebSocket client.
- **Payload**: `{ "token": "your-jwt-token" }`

#### `subscribe`
- **Description**: Subscribe to one or more topics.
- **Payload**: `{ "topics": ["ha_events", "system_status"] }`
- **Alternative**: Use specific subscription messages (e.g., `subscribe_entity_updates`).

#### `unsubscribe`
- **Description**: Unsubscribe from topics.
- **Payload**: `{ "topics": ["ha_events"] }`

#### `subscribe_entity_updates`
- **Description**: Subscribe to updates for specific entities.
- **Payload**: `{ "entity_ids": ["light.living_room"], "include_attributes": true }`

#### `subscribe_ha_events`
- **Description**: Subscribe to Home Assistant events.
- **Payload**: `{ "event_types": ["state_changed"], "domains": ["light", "sensor"] }`

#### `subscribe_system_events`
- **Description**: Subscribe to system-level events.
- **Payload**: `{ "events": ["adapter_status", "performance_alert"] }`

### Server to Client

#### `pma_entity_state_changed`
- **Description**: Sent when an entity's state changes.
- **Payload**: 
  ```json
  {
    "entity_id": "light.living_room",
    "old_state": "off",
    "new_state": "on",
    "attributes": { "brightness": 255 },
    "source": "homeassistant"
  }
  ```

#### `pma_entity_added` / `pma_entity_removed`
- **Description**: Sent when an entity is added or removed.

#### `pma_sync_status`
- **Description**: Reports the status of synchronization with external systems.

#### `pma_adapter_status`
- **Description**: Reports the health and status of integration adapters.

#### `system_status`
- **Description**: Provides an overview of the system's health.

#### `automation_triggered`
- **Description**: Sent when an automation rule is triggered.

#### `notification`
- **Description**: General-purpose notification from the backend.

## Subscription Management

Clients can subscribe to various topics to receive specific updates.

**Available Topics:**
- `ha_events`: Home Assistant events
- `system_status`: System health updates
- `automation_events`: Automation rule triggers
- `entity_updates`: Specific entity state changes

**Subscription Example:**
```json
{
  "type": "subscribe",
  "data": {
    "topics": [
      "ha_events:state_changed",
      "entity_updates:light.living_room",
      "system_status"
    ]
  }
}
```

## Client Examples

### JavaScript (Browser/Node.js)

```javascript
class PMAWebSocketClient {
  constructor(url, token) {
    this.url = url;
    this.token = token;
    this.ws = null;
    this.subscriptions = new Set();
  }

  connect() {
    this.ws = new WebSocket(this.url);
    
    this.ws.onopen = () => {
      console.log('Connected to PMA Backend');
      if (this.token) this.authenticate();
      this.resubscribe();
    };

    this.ws.onmessage = (event) => {
      this.handleMessage(JSON.parse(event.data));
    };

    this.ws.onclose = () => {
      console.log('Disconnected, reconnecting in 5s...');
      setTimeout(() => this.connect(), 5000);
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket Error:', error);
    };
  }

  send(type, data) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type, data }));
    }
  }

  authenticate() {
    this.send('authenticate', { token: this.token });
  }

  subscribeToEntity(entityId) {
    const topic = `entity_updates:${entityId}`;
    this.subscriptions.add(topic);
    this.send('subscribe', { topics: [topic] });
  }

  resubscribe() {
    if (this.subscriptions.size > 0) {
      this.send('subscribe', { topics: Array.from(this.subscriptions) });
    }
  }

  handleMessage(message) {
    switch (message.type) {
      case 'pma_entity_state_changed':
        console.log(`Entity ${message.data.entity_id} changed state.`);
        break;
      // Handle other types
    }
  }
}
```

### Python

```python
import asyncio
import websockets
import json

class PMAWebSocketClient:
    def __init__(self, url, token=None):
        self.url = url
        self.token = token
        self.ws = None
        self.subscriptions = set()

    async def connect(self):
        while True:
            try:
                async with websockets.connect(self.url) as ws:
                    self.ws = ws
                    print("Connected to PMA Backend")
                    
                    if self.token:
                        await self.authenticate()
                    
                    await self.resubscribe()

                    async for message in self.ws:
                        await self.handle_message(json.loads(message))
            except websockets.exceptions.ConnectionClosed:
                print("Connection closed, reconnecting in 5s...")
                await asyncio.sleep(5)

    async def send(self, msg_type, data):
        if self.ws:
            await self.ws.send(json.dumps({'type': msg_type, 'data': data}))

    async def authenticate(self):
        await self.send('authenticate', {'token': self.token})

    async def subscribe_to_entity(self, entity_id):
        topic = f"entity_updates:{entity_id}"
        self.subscriptions.add(topic)
        await self.send('subscribe', {'topics': [topic]})
        
    async def resubscribe(self):
        if self.subscriptions:
            await self.send('subscribe', {'topics': list(self.subscriptions)})

    async def handle_message(self, message):
        msg_type = message.get('type')
        if msg_type == 'pma_entity_state_changed':
            print(f"Entity {message['data']['entity_id']} changed state.")
```

## Error Handling

| Error Type | Description |
|------------|-------------|
| `authentication_failure` | Invalid JWT token |
| `subscription_failure` | Invalid topic or insufficient permissions |
| `invalid_message` | Malformed JSON or invalid message structure |

## Performance & Best Practices

- **Batch Subscriptions**: Subscribe to multiple topics in a single message.
- **Selective Subscriptions**: Only subscribe to events you need.
- **Auto-Reconnect**: Implement robust auto-reconnect logic.
- **Message Buffering**: Buffer messages on the client side if UI updates are frequent.
- **Compression**: The server supports WebSocket compression; ensure your client does too.

## Troubleshooting

- **Connection Issues**: Check network connectivity and server status (`/health`).
- **Authentication Problems**: Verify JWT token validity and expiration.
- **Missing Messages**: Check subscription topics and server logs for errors.
- **Performance**: Monitor WebSocket metrics via the API (`/api/v1/websocket/stats`).
- **Debugging**: Use browser developer tools or a WebSocket client like `websocat` for inspection.