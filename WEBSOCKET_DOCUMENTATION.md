# WebSocket Infrastructure Documentation

## Overview

The PMA Backend now includes comprehensive WebSocket infrastructure for real-time communication, event broadcasting, client connection management, and live updates for home automation events.

## Features

- **Real-time Client Management**: Automatic client registration/unregistration with connection tracking
- **Message Broadcasting**: Send messages to all clients or specific room subscribers
- **Room Subscriptions**: Clients can subscribe to specific room updates
- **Connection Statistics**: Monitor WebSocket connection metrics and activity
- **Heartbeat System**: Automatic connection health monitoring
- **Event Broadcasting**: Automatic WebSocket notifications for room CRUD operations

## WebSocket Endpoints

### Connection Endpoint
- **URL**: `ws://localhost:3001/ws`
- **Auth**: No authentication required for connection
- **Purpose**: Establish WebSocket connection for real-time updates

### Management API Endpoints (Protected)
- **GET** `/api/v1/websocket/stats` - Get WebSocket hub statistics
- **POST** `/api/v1/websocket/broadcast` - Broadcast message to all clients

## Message Types

### Client to Server Messages

#### Ping/Pong
```json
{
  "type": "ping",
  "data": {}
}
```

#### Room Subscription
```json
{
  "type": "subscribe_room",
  "data": {
    "room_id": 1
  }
}
```

#### Room Unsubscription
```json
{
  "type": "unsubscribe_room",
  "data": {
    "room_id": 1
  }
}
```

### Server to Client Messages

#### Connection Welcome
```json
{
  "type": "connection",
  "data": {
    "status": "connected",
    "client_id": "uuid-string",
    "timestamp": "2025-07-16T21:30:00Z"
  },
  "timestamp": "2025-07-16T21:30:00Z"
}
```

#### Pong Response
```json
{
  "type": "pong",
  "data": {
    "timestamp": "2025-07-16T21:30:00Z"
  },
  "timestamp": "2025-07-16T21:30:00Z"
}
```

#### Heartbeat
```json
{
  "type": "heartbeat",
  "data": {
    "timestamp": "2025-07-16T21:30:00Z",
    "clients": 3
  },
  "timestamp": "2025-07-16T21:30:00Z"
}
```

#### Room Updates
```json
{
  "type": "room_updated",
  "data": {
    "room_id": 1,
    "room_name": "Living Room",
    "action": "created"  // "created", "updated", "deleted"
  },
  "timestamp": "2025-07-16T21:30:00Z"
}
```

#### Entity State Changes
```json
{
  "type": "entity_state_changed",
  "data": {
    "entity_id": "light.living_room",
    "old_state": "off",
    "new_state": "on",
    "attributes": {
      "brightness": 255,
      "color": "#ffffff"
    }
  },
  "timestamp": "2025-07-16T21:30:00Z"
}
```

#### System Status
```json
{
  "type": "system_status",
  "data": {
    "status": "healthy",
    "details": {
      "uptime": "2h 30m",
      "memory_usage": "45%"
    }
  },
  "timestamp": "2025-07-16T21:30:00Z"
}
```

## API Endpoints

### Get WebSocket Statistics

**GET** `/api/v1/websocket/stats`

**Headers**: `Authorization: Bearer <jwt-token>`

**Response**:
```json
{
  "success": true,
  "data": {
    "connected_clients": 3,
    "total_connections": 15,
    "messages_sent": 142,
    "messages_received": 67,
    "last_activity": "2025-07-16T21:30:00Z"
  },
  "timestamp": "2025-07-16T21:30:00Z"
}
```

### Broadcast Message

**POST** `/api/v1/websocket/broadcast`

**Headers**: 
- `Authorization: Bearer <jwt-token>`
- `Content-Type: application/json`

**Request Body**:
```json
{
  "type": "custom_notification",
  "data": {
    "message": "System maintenance scheduled",
    "level": "warning",
    "expires_at": "2025-07-16T22:00:00Z"
  }
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "message": "Message broadcasted successfully",
    "clients_count": 3,
    "message_type": "custom_notification"
  },
  "timestamp": "2025-07-16T21:30:00Z"
}
```

## Client Implementation Examples

### JavaScript/Browser
```javascript
const ws = new WebSocket('ws://localhost:3001/ws');

ws.onopen = function() {
    console.log('Connected to WebSocket');
    
    // Subscribe to room updates
    ws.send(JSON.stringify({
        type: 'subscribe_room',
        data: { room_id: 1 }
    }));
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Received:', message);
    
    if (message.type === 'room_updated') {
        updateRoomUI(message.data);
    }
};

ws.onclose = function() {
    console.log('WebSocket connection closed');
};
```

### Python
```python
import asyncio
import websockets
import json

async def client():
    uri = "ws://localhost:3001/ws"
    async with websockets.connect(uri) as websocket:
        # Send ping
        await websocket.send(json.dumps({
            "type": "ping",
            "data": {}
        }))
        
        # Listen for messages
        async for message in websocket:
            data = json.loads(message)
            print(f"Received: {data}")
```

## Testing

A test script is provided: `test_websocket.py`

**Run tests**:
```bash
# Install dependencies
pip install websockets requests

# Run tests (ensure server is running)
python3 test_websocket.py
```

## Connection Management

### Client Lifecycle
1. **Connection**: Client connects to `/ws` endpoint
2. **Registration**: Hub assigns unique ID and sends welcome message
3. **Subscription**: Client can subscribe to room-specific updates
4. **Communication**: Bidirectional message exchange
5. **Heartbeat**: Automatic connection health monitoring (30-second intervals)
6. **Disconnection**: Automatic cleanup on connection close

### Connection Limits
- **Write Timeout**: 10 seconds
- **Read Timeout**: 60 seconds (pong timeout)
- **Ping Interval**: 54 seconds
- **Max Message Size**: 512 bytes (for client messages)
- **Broadcast Buffer**: 256 messages

## Error Handling

### Common Error Scenarios
- **Connection Refused**: Server not running or port blocked
- **Authentication Failed**: Invalid JWT token for API endpoints
- **Message Too Large**: Client message exceeds 512 bytes
- **Timeout**: Client doesn't respond to ping within 60 seconds
- **Invalid JSON**: Malformed message format

### Error Responses
API endpoints return standard error format:
```json
{
  "success": false,
  "error": "Invalid request body",
  "timestamp": "2025-07-16T21:30:00Z"
}
```

## Integration with Existing Systems

### Automatic Room Events
The WebSocket system automatically broadcasts events when:
- **Room Created**: Triggers `room_updated` with action "created"
- **Room Updated**: Triggers `room_updated` with action "updated" 
- **Room Deleted**: Triggers `room_updated` with action "deleted"

### Entity State Changes
When entity states change (future implementation), the system will broadcast:
- Entity ID and state transition
- Updated attributes
- Timestamp of change

## Performance Considerations

### Scalability
- **Memory Usage**: ~1KB per active connection
- **CPU Impact**: Minimal with efficient goroutine-based design
- **Network Overhead**: Heartbeat every 30 seconds per client

### Monitoring
- Connection statistics available via API
- Detailed logging for debugging
- Graceful handling of connection failures

## Security

### Authentication
- WebSocket connection: No authentication required (configure as needed)
- Management API: JWT token required
- CORS: Configurable origin checking

### Rate Limiting
- Existing middleware applies to API endpoints
- WebSocket message rate limiting can be implemented per client

## Future Enhancements

- **Authentication for WebSocket connections**
- **Message rate limiting per client**
- **Client-specific message routing**
- **Persistent message queuing**
- **WebSocket over TLS (WSS)**
- **Custom event types and handlers** 