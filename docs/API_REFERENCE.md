# PMA Backend Go API Reference

## Overview

The PMA Backend Go provides a comprehensive REST API for managing smart home automation, AI interactions, system monitoring, and device control. This document provides detailed information about all available endpoints, request/response formats, and usage examples.

## Base URL

```
http://localhost:3001/api/v1
```

## Authentication

### PIN Authentication

The system supports PIN-based authentication for simple access:

```http
POST /api/v1/auth/verify-pin
Content-Type: application/json

{
  "pin": "1234"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2024-01-01T12:30:00Z"
  }
}
```

### JWT Token Authentication

For API access, include the JWT token in the Authorization header:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### PIN Management

#### Set PIN
```http
POST /api/v1/auth/set-pin
Content-Type: application/json

{
  "pin": "1234"
}
```

#### Change PIN
```http
POST /api/v1/auth/change-pin
Content-Type: application/json

{
  "current_pin": "1234",
  "new_pin": "5678"
}
```

#### Validate Token
```http
POST /api/v1/auth/validate
Content-Type: application/json

{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

## Entity Management

### List Entities

```http
GET /api/v1/entities?domain=light&room=living_room&state=on
```

**Query Parameters:**
- `domain`: Filter by entity domain (light, switch, sensor, etc.)
- `room`: Filter by room ID
- `state`: Filter by current state
- `capability`: Filter by capability
- `limit`: Maximum number of entities to return
- `offset`: Number of entities to skip

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "light.living_room",
      "name": "Living Room Light",
      "domain": "light",
      "state": "on",
      "attributes": {
        "brightness": 255,
        "color_temp": 4000
      },
      "room_id": "living_room",
      "capabilities": ["brightness", "color_temp"],
      "last_updated": "2024-01-01T12:00:00Z"
    }
  ],
  "pagination": {
    "total": 50,
    "limit": 20,
    "offset": 0,
    "has_more": true
  }
}
```

### Get Entity Details

```http
GET /api/v1/entities/light.living_room
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "light.living_room",
    "name": "Living Room Light",
    "domain": "light",
    "state": "on",
    "attributes": {
      "brightness": 255,
      "color_temp": 4000,
      "supported_features": 1,
      "friendly_name": "Living Room Light"
    },
    "room_id": "living_room",
    "capabilities": ["brightness", "color_temp", "on_off"],
    "source": "home_assistant",
    "last_updated": "2024-01-01T12:00:00Z",
    "history": [
      {
        "state": "off",
        "timestamp": "2024-01-01T11:55:00Z"
      },
      {
        "state": "on",
        "timestamp": "2024-01-01T12:00:00Z"
      }
    ]
  }
}
```

### Execute Entity Action

```http
POST /api/v1/entities/light.living_room/action
Content-Type: application/json

{
  "action": "turn_on",
  "parameters": {
    "brightness": 128,
    "color_temp": 3000
  }
}
```

**Available Actions:**
- `turn_on`: Turn on the entity
- `turn_off`: Turn off the entity
- `toggle`: Toggle the entity state
- `set_brightness`: Set brightness (0-255)
- `set_color_temp`: Set color temperature
- `set_color`: Set RGB color
- `set_effect`: Set light effect

**Response:**
```json
{
  "success": true,
  "data": {
    "action": "turn_on",
    "entity_id": "light.living_room",
    "result": "success",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

### Entity Synchronization

#### Trigger Full Sync
```http
POST /api/v1/entities/sync
```

#### Get Sync Status
```http
GET /api/v1/entities/sync/status
```

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "syncing",
    "progress": 75,
    "total_entities": 100,
    "synced_entities": 75,
    "last_sync": "2024-01-01T11:00:00Z",
    "next_sync": "2024-01-01T12:00:00Z"
  }
}
```

## Room and Area Management

### List Rooms

```http
GET /api/v1/rooms
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "living_room",
      "name": "Living Room",
      "area_id": "main_floor",
      "entity_count": 5,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T12:00:00Z"
    }
  ]
}
```

### Create Room

```http
POST /api/v1/rooms
Content-Type: application/json

{
  "name": "Home Office",
  "area_id": "main_floor",
  "description": "Home office workspace"
}
```

### Update Room

```http
PUT /api/v1/rooms/living_room
Content-Type: application/json

{
  "name": "Living Room Updated",
  "description": "Updated description"
}
```

### Delete Room

```http
DELETE /api/v1/rooms/living_room
```

### List Areas

```http
GET /api/v1/areas
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "main_floor",
      "name": "Main Floor",
      "room_count": 3,
      "entity_count": 15,
      "created_at": "2024-01-01T00:00:00Z"
    }
  ]
}
```

### Create Area

```http
POST /api/v1/areas
Content-Type: application/json

{
  "name": "Basement",
  "description": "Basement area"
}
```

## Automation Engine

### List Automation Rules

```http
GET /api/v1/automation/rules?enabled=true&category=lighting
```

**Query Parameters:**
- `enabled`: Filter by enabled status
- `category`: Filter by category
- `limit`: Maximum number of rules
- `offset`: Number of rules to skip

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "rule_123",
      "name": "Turn on lights at sunset",
      "description": "Automatically turn on lights when sun sets",
      "enabled": true,
      "category": "lighting",
      "trigger": {
        "type": "sunset",
        "offset": "-30m"
      },
      "conditions": [
        {
          "type": "entity_state",
          "entity_id": "binary_sensor.motion",
          "state": "on"
        }
      ],
      "actions": [
        {
          "type": "turn_on",
          "entity_id": "light.living_room"
        }
      ],
      "last_executed": "2024-01-01T18:30:00Z",
      "execution_count": 15
    }
  ]
}
```

### Create Automation Rule

```http
POST /api/v1/automation/rules
Content-Type: application/json

{
  "name": "Motion-activated lights",
  "description": "Turn on lights when motion is detected",
  "enabled": true,
  "category": "lighting",
  "trigger": {
    "type": "state_changed",
    "entity_id": "binary_sensor.motion",
    "to": "on"
  },
  "conditions": [
    {
      "type": "time",
      "after": "18:00:00",
      "before": "06:00:00"
    }
  ],
  "actions": [
    {
      "type": "turn_on",
      "entity_id": "light.living_room",
      "parameters": {
        "brightness": 128
      }
    }
  ]
}
```

### Update Automation Rule

```http
PUT /api/v1/automation/rules/rule_123
Content-Type: application/json

{
  "name": "Updated Motion-activated lights",
  "enabled": false
}
```

### Delete Automation Rule

```http
DELETE /api/v1/automation/rules/rule_123
```

### Test Automation Rule

```http
POST /api/v1/automation/rules/rule_123/test
```

**Response:**
```json
{
  "success": true,
  "data": {
    "triggered": true,
    "conditions_met": true,
    "actions_executed": 1,
    "execution_time": "150ms",
    "results": [
      {
        "action": "turn_on",
        "entity_id": "light.living_room",
        "success": true
      }
    ]
  }
}
```

### Get Automation Statistics

```http
GET /api/v1/automation/statistics
```

**Response:**
```json
{
  "success": true,
  "data": {
    "total_rules": 25,
    "enabled_rules": 20,
    "disabled_rules": 5,
    "total_executions": 150,
    "successful_executions": 145,
    "failed_executions": 5,
    "average_execution_time": "120ms",
    "most_active_rules": [
      {
        "rule_id": "rule_123",
        "name": "Motion-activated lights",
        "execution_count": 45
      }
    ]
  }
}
```

## AI and Conversation Management

### Send Chat Message

```http
POST /api/v1/ai/chat
Content-Type: application/json

{
  "message": "Turn on the living room lights",
  "conversation_id": "conv_123",
  "model": "LFM2-1.2B",
  "provider": "llamacpp"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "response": "I've turned on the living room lights for you.",
    "conversation_id": "conv_123",
    "message_id": "msg_456",
    "actions_executed": [
      {
        "type": "turn_on",
        "entity_id": "light.living_room",
        "success": true
      }
    ],
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

### List Conversations

```http
GET /api/v1/ai/conversations?limit=10&offset=0
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "conv_123",
      "title": "Lighting control",
      "message_count": 5,
      "created_at": "2024-01-01T10:00:00Z",
      "updated_at": "2024-01-01T12:00:00Z",
      "last_message": "Turn on the living room lights"
    }
  ]
}
```

### Create Conversation

```http
POST /api/v1/ai/conversations
Content-Type: application/json

{
  "title": "New conversation",
  "initial_message": "Hello, how can I help you today?"
}
```

### Get Conversation

```http
GET /api/v1/ai/conversations/conv_123
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "conv_123",
    "title": "Lighting control",
    "messages": [
      {
        "id": "msg_123",
        "role": "user",
        "content": "Turn on the living room lights",
        "timestamp": "2024-01-01T12:00:00Z"
      },
      {
        "id": "msg_124",
        "role": "assistant",
        "content": "I've turned on the living room lights for you.",
        "actions": [
          {
            "type": "turn_on",
            "entity_id": "light.living_room",
            "success": true
          }
        ],
        "timestamp": "2024-01-01T12:00:01Z"
      }
    ],
    "created_at": "2024-01-01T10:00:00Z",
    "updated_at": "2024-01-01T12:00:01Z"
  }
}
```

### Add Message to Conversation

```http
POST /api/v1/ai/conversations/conv_123/messages
Content-Type: application/json

{
  "content": "What's the current temperature?",
  "role": "user"
}
```

### Get AI Models

```http
GET /api/v1/ai/models
```

**Response:**
```json
{
  "success": true,
  "data": {
    "available_models": [
      {
        "name": "LFM2-1.2B",
        "provider": "llamacpp",
        "enabled": true,
        "capabilities": ["text_generation", "conversation"]
      },
      {
        "name": "llama2-7b",
        "provider": "ollama",
        "enabled": true,
        "capabilities": ["text_generation", "conversation"]
      }
    ],
    "default_model": "LFM2-1.2B",
    "default_provider": "llamacpp"
  }
}
```

## System Management

### Get System Status

```http
GET /api/v1/system/status
```

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "online",
    "uptime": 3600,
    "version": "1.0.0",
    "build_date": "2024-01-01T00:00:00Z",
    "git_commit": "abc123",
    "services": {
      "database": "healthy",
      "home_assistant": "connected",
      "websocket": "running",
      "automation": "active"
    }
  }
}
```

### Health Check

```http
GET /api/v1/system/health
```

**Response:**
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "checks": {
      "database": {
        "status": "healthy",
        "response_time": "5ms"
      },
      "home_assistant": {
        "status": "connected",
        "response_time": "150ms"
      },
      "memory": {
        "status": "healthy",
        "usage_percent": 45
      },
      "disk": {
        "status": "healthy",
        "usage_percent": 30
      }
    }
  }
}
```

### Get System Metrics

```http
GET /api/v1/system/metrics
```

**Response:**
```json
{
  "success": true,
  "data": {
    "cpu": {
      "usage_percent": 12.5,
      "cores": 4
    },
    "memory": {
      "total_bytes": 8589934592,
      "used_bytes": 3865470566,
      "usage_percent": 45.0
    },
    "disk": {
      "total_bytes": 107374182400,
      "used_bytes": 32212254720,
      "usage_percent": 30.0
    },
    "network": {
      "bytes_sent": 1048576,
      "bytes_received": 2097152
    },
    "goroutines": 150,
    "heap_objects": 50000
  }
}
```

### Reboot System

```http
POST /api/v1/system/reboot
```

### Shutdown System

```http
POST /api/v1/system/shutdown
```

### Get System Configuration

```http
GET /api/v1/system/config
```

**Response:**
```json
{
  "success": true,
  "data": {
    "server": {
      "port": 3001,
      "host": "0.0.0.0",
      "mode": "production"
    },
    "database": {
      "path": "./data/pma.db",
      "max_connections": 25
    },
    "auth": {
      "enabled": true,
      "token_expiry": 1800
    },
    "home_assistant": {
      "url": "http://192.168.100.2:8123",
      "sync_enabled": true
    }
  }
}
```

## Monitoring and Analytics

### Get Monitoring Metrics

```http
GET /api/v1/monitoring/metrics?timeframe=24h
```

**Query Parameters:**
- `timeframe`: Time range (1h, 24h, 7d, 30d)
- `metric`: Specific metric name
- `aggregation`: Aggregation type (avg, sum, max, min)

**Response:**
```json
{
  "success": true,
  "data": {
    "metrics": [
      {
        "name": "api_requests_total",
        "value": 1500,
        "unit": "requests",
        "timestamp": "2024-01-01T12:00:00Z"
      },
      {
        "name": "response_time_avg",
        "value": 45.2,
        "unit": "ms",
        "timestamp": "2024-01-01T12:00:00Z"
      }
    ],
    "timeframe": "24h",
    "aggregation": "avg"
  }
}
```

### Get Active Alerts

```http
GET /api/v1/monitoring/alerts
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "alert_123",
      "name": "High CPU Usage",
      "severity": "warning",
      "message": "CPU usage is above 80%",
      "timestamp": "2024-01-01T12:00:00Z",
      "acknowledged": false
    }
  ]
}
```

### Get Analytics Events

```http
GET /api/v1/analytics/events?type=entity_action&limit=50
```

**Query Parameters:**
- `type`: Event type filter
- `entity_id`: Filter by entity
- `start_date`: Start date filter
- `end_date`: End date filter
- `limit`: Maximum number of events

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "event_123",
      "type": "entity_action",
      "entity_id": "light.living_room",
      "action": "turn_on",
      "user_id": "user_123",
      "timestamp": "2024-01-01T12:00:00Z",
      "metadata": {
        "source": "api",
        "ip_address": "192.168.1.100"
      }
    }
  ]
}
```

### Generate Analytics Report

```http
POST /api/v1/analytics/reports
Content-Type: application/json

{
  "type": "usage_summary",
  "timeframe": "7d",
  "filters": {
    "entity_domain": "light"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "report_id": "report_123",
    "type": "usage_summary",
    "timeframe": "7d",
    "generated_at": "2024-01-01T12:00:00Z",
    "data": {
      "total_actions": 150,
      "most_used_entities": [
        {
          "entity_id": "light.living_room",
          "action_count": 45
        }
      ],
      "usage_by_hour": [
        {
          "hour": 18,
          "action_count": 25
        }
      ]
    }
  }
}
```

## File Management

### List Files

```http
GET /api/v1/files?category=media&limit=20
```

**Query Parameters:**
- `category`: File category (media, backup, log, config)
- `type`: File type filter
- `limit`: Maximum number of files
- `offset`: Number of files to skip

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "file_123",
      "name": "screenshot.png",
      "path": "/media/screenshots/screenshot.png",
      "size": 1048576,
      "type": "image/png",
      "category": "media",
      "uploaded_at": "2024-01-01T12:00:00Z",
      "metadata": {
        "width": 1920,
        "height": 1080
      }
    }
  ]
}
```

### Upload File

```http
POST /api/v1/files/upload
Content-Type: multipart/form-data

file: [binary data]
category: media
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "file_123",
    "name": "uploaded_file.png",
    "path": "/media/uploads/uploaded_file.png",
    "size": 1048576,
    "type": "image/png",
    "category": "media",
    "uploaded_at": "2024-01-01T12:00:00Z"
  }
}
```

### Download File

```http
GET /api/v1/files/file_123
```

### Delete File

```http
DELETE /api/v1/files/file_123
```

## Kiosk Management

### List Kiosk Devices

```http
GET /api/v1/kiosk/devices
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "kiosk_123",
      "name": "Living Room Kiosk",
      "room_id": "living_room",
      "status": "online",
      "last_seen": "2024-01-01T12:00:00Z",
      "version": "1.0.0",
      "capabilities": ["display", "touch", "audio"]
    }
  ]
}
```

### Pair New Kiosk Device

```http
POST /api/v1/kiosk/pair
Content-Type: application/json

{
  "device_id": "kiosk_456",
  "room_id": "living_room",
  "name": "New Kiosk Device"
}
```

### Get Kiosk Device Status

```http
GET /api/v1/kiosk/devices/kiosk_123/status
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "kiosk_123",
    "status": "online",
    "uptime": 3600,
    "memory_usage": 45.2,
    "disk_usage": 30.1,
    "last_heartbeat": "2024-01-01T12:00:00Z",
    "active_sessions": 1
  }
}
```

### Send Command to Kiosk Device

```http
POST /api/v1/kiosk/devices/kiosk_123/command
Content-Type: application/json

{
  "command": "display_message",
  "parameters": {
    "message": "Hello from PMA!",
    "duration": 5000
  }
}
```

## Energy Management

### Get Current Energy Usage

```http
GET /api/v1/energy/current
```

**Response:**
```json
{
  "success": true,
  "data": {
    "total_power": 1250.5,
    "unit": "W",
    "devices": [
      {
        "entity_id": "light.living_room",
        "name": "Living Room Light",
        "power": 15.2,
        "unit": "W"
      }
    ],
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

### Get Energy History

```http
GET /api/v1/energy/history?period=24h&interval=1h
```

**Query Parameters:**
- `period`: Time period (1h, 24h, 7d, 30d)
- `interval`: Data interval (1m, 5m, 15m, 1h, 1d)
- `entity_id`: Filter by specific entity

**Response:**
```json
{
  "success": true,
  "data": {
    "period": "24h",
    "interval": "1h",
    "data": [
      {
        "timestamp": "2024-01-01T00:00:00Z",
        "total_power": 1200.5,
        "total_energy": 1.2,
        "unit": "kWh"
      }
    ]
  }
}
```

### Get Energy Settings

```http
GET /api/v1/energy/settings
```

**Response:**
```json
{
  "success": true,
  "data": {
    "enabled": true,
    "currency": "USD",
    "rate_per_kwh": 0.12,
    "billing_cycle": "monthly",
    "billing_start_date": "2024-01-01",
    "alerts": {
      "high_usage_threshold": 100,
      "cost_threshold": 50.0
    }
  }
}
```

### Update Energy Settings

```http
PUT /api/v1/energy/settings
Content-Type: application/json

{
  "rate_per_kwh": 0.15,
  "alerts": {
    "high_usage_threshold": 120,
    "cost_threshold": 60.0
  }
}
```

## WebSocket Communication

### Connection

Connect to the WebSocket endpoint:

```javascript
const ws = new WebSocket('ws://localhost:3001/ws');
```

### Authentication

Send authentication message:

```javascript
ws.send(JSON.stringify({
  type: 'authenticate',
  token: 'your-jwt-token'
}));
```

### Subscribe to Events

```javascript
ws.send(JSON.stringify({
  type: 'subscribe',
  topic: 'entities',
  filter: {
    domain: 'light',
    room_id: 'living_room'
  }
}));
```

### Event Types

#### Entity Updates
```json
{
  "type": "entity_updated",
  "data": {
    "entity_id": "light.living_room",
    "state": "on",
    "attributes": {
      "brightness": 255
    },
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

#### System Status
```json
{
  "type": "system_status",
  "data": {
    "status": "online",
    "uptime": 3600,
    "memory_usage": 45.2,
    "cpu_usage": 12.5
  }
}
```

#### Automation Events
```json
{
  "type": "automation_triggered",
  "data": {
    "rule_id": "rule_123",
    "rule_name": "Motion-activated lights",
    "trigger": "state_changed",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

#### AI Events
```json
{
  "type": "ai_message",
  "data": {
    "conversation_id": "conv_123",
    "message_id": "msg_456",
    "role": "assistant",
    "content": "I've turned on the lights for you.",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

## Error Handling

### Error Response Format

All API endpoints return consistent error responses:

```json
{
  "success": false,
  "error": {
    "code": "ENTITY_NOT_FOUND",
    "message": "Entity 'light.nonexistent' not found",
    "details": {
      "entity_id": "light.nonexistent",
      "available_entities": ["light.living_room", "light.kitchen"]
    }
  },
  "timestamp": "2024-01-01T12:00:00Z",
  "request_id": "req_123"
}
```

### Common Error Codes

- `AUTHENTICATION_FAILED`: Invalid credentials
- `AUTHORIZATION_FAILED`: Insufficient permissions
- `ENTITY_NOT_FOUND`: Entity does not exist
- `INVALID_REQUEST`: Malformed request data
- `RATE_LIMIT_EXCEEDED`: Too many requests
- `SERVICE_UNAVAILABLE`: Service temporarily unavailable
- `INTERNAL_ERROR`: Internal server error

### Rate Limiting

The API implements rate limiting to prevent abuse:

- **Default**: 1000 requests per minute per IP
- **Authentication endpoints**: 10 requests per minute per IP
- **File uploads**: 50 requests per minute per IP

Rate limit headers are included in responses:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 950
X-RateLimit-Reset: 1640995200
```

## Pagination

List endpoints support pagination:

### Request
```http
GET /api/v1/entities?limit=20&offset=40
```

### Response
```json
{
  "success": true,
  "data": [...],
  "pagination": {
    "total": 150,
    "limit": 20,
    "offset": 40,
    "has_more": true,
    "next_offset": 60,
    "prev_offset": 20
  }
}
```

## Filtering and Sorting

Many endpoints support filtering and sorting:

### Filtering
```http
GET /api/v1/entities?domain=light&state=on&room_id=living_room
```

### Sorting
```http
GET /api/v1/entities?sort=name&order=asc
```

### Search
```http
GET /api/v1/entities?search=living room
```

## Response Headers

All API responses include standard headers:

```
Content-Type: application/json
X-Request-ID: req_123
X-Response-Time: 45ms
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 950
X-RateLimit-Reset: 1640995200
```

## SDK and Client Libraries

### JavaScript/TypeScript

```javascript
import { PMAClient } from '@pma/client';

const client = new PMAClient({
  baseURL: 'http://localhost:3001/api/v1',
  token: 'your-jwt-token'
});

// Get entities
const entities = await client.entities.list({
  domain: 'light',
  room_id: 'living_room'
});

// Execute action
await client.entities.executeAction('light.living_room', {
  action: 'turn_on',
  parameters: { brightness: 128 }
});
```

### Python

```python
from pma_client import PMAClient

client = PMAClient(
    base_url='http://localhost:3001/api/v1',
    token='your-jwt-token'
)

# Get entities
entities = client.entities.list(
    domain='light',
    room_id='living_room'
)

# Execute action
client.entities.execute_action(
    'light.living_room',
    action='turn_on',
    parameters={'brightness': 128}
)
```

## WebSocket Client Libraries

### JavaScript

```javascript
import { PMAWebSocket } from '@pma/client';

const ws = new PMAWebSocket('ws://localhost:3001/ws', {
  token: 'your-jwt-token'
});

// Subscribe to entity updates
ws.subscribe('entities', {
  domain: 'light',
  room_id: 'living_room'
});

// Handle events
ws.on('entity_updated', (data) => {
  console.log('Entity updated:', data);
});
```

## Testing

### API Testing

Use the provided test endpoints for development:

```http
GET /api/v1/test/entities
GET /api/v1/test/automation
GET /api/v1/test/ai
```

### Load Testing

The API supports load testing with realistic data:

```http
POST /api/v1/test/load
Content-Type: application/json

{
  "duration": "5m",
  "requests_per_second": 100,
  "endpoints": ["/api/v1/entities", "/api/v1/system/status"]
}
```

## Versioning

The API uses semantic versioning:

- **Current Version**: v1
- **Base URL**: `/api/v1`
- **Deprecation Policy**: 6 months notice for breaking changes
- **Backward Compatibility**: Maintained within major versions

## Support

For API support and questions:

- **Documentation**: This reference guide
- **Examples**: See the examples directory
- **Issues**: Create an issue on GitHub
- **Discussions**: Use GitHub Discussions

---

**PMA Backend Go API** - Comprehensive REST API for smart home automation. 