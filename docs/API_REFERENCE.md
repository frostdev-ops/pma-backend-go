# PMA Backend Go - API Reference

This document provides comprehensive documentation for the PMA Backend Go REST API.

## Table of Contents

- [Authentication](#authentication)
- [Base URL & Versioning](#base-url--versioning)
- [Response Format](#response-format)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [Core Endpoints](#core-endpoints)
- [Entity Management](#entity-management)
- [Room & Area Management](#room--area-management)
- [Automation](#automation)
- [AI Services](#ai-services)
- [Analytics](#analytics)
- [Monitoring](#monitoring)
- [System Management](#system-management)
- [WebSocket](#websocket)

## Authentication

The PMA Backend uses JWT (JSON Web Token) authentication for API access.

### Authentication Endpoints

#### POST `/api/v1/auth/login`
Authenticate with username/password and receive a JWT token.

**Request:**
```json
{
  "username": "admin",
  "password": "password"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2024-01-01T12:00:00Z",
    "user": {
      "id": "user123",
      "username": "admin",
      "roles": ["admin"]
    }
  }
}
```

#### POST `/api/v1/auth/verify-pin`
Verify PIN for additional security (if enabled).

**Request:**
```json
{
  "pin": "1234"
}
```

#### POST `/api/v1/auth/validate`
Validate an existing JWT token.

**Headers:**
```
Authorization: Bearer <jwt_token>
```

#### GET `/api/v1/auth/session`
Get current session information.

### Using Authentication

Include the JWT token in the Authorization header for all authenticated requests:

```bash
curl -H "Authorization: Bearer <your_jwt_token>" \
  http://localhost:3001/api/v1/entities
```

## Base URL & Versioning

- **Base URL**: `http://localhost:3001`
- **API Version**: `v1`
- **Full API Base**: `http://localhost:3001/api/v1`

All API endpoints are versioned and prefixed with `/api/v1/`.

## Response Format

All API responses follow a consistent format:

### Success Response
```json
{
  "success": true,
  "data": { /* response data */ },
  "meta": { /* optional metadata */ },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### Error Response
```json
{
  "success": false,
  "error": "Error message",
  "code": 400,
  "details": "Detailed error information",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Error Handling

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request |
| 401 | Unauthorized |
| 403 | Forbidden |
| 404 | Not Found |
| 409 | Conflict |
| 429 | Too Many Requests |
| 500 | Internal Server Error |

### Error Response Details

```json
{
  "success": false,
  "error": "Validation failed",
  "code": 400,
  "details": {
    "field": "entity_id",
    "message": "Entity ID is required"
  },
  "path": "/api/v1/entities",
  "method": "POST",
  "timestamp": "2024-01-01T12:00:00Z"
}
```

## Rate Limiting

The API implements rate limiting to prevent abuse:

- **Default Limit**: 100 requests per minute
- **Burst Limit**: 200 requests
- **Headers**: Rate limit info included in response headers

**Rate Limit Headers:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 85
X-RateLimit-Reset: 1640995200
```

## Core Endpoints

### Health Check

#### GET `/health`
Get system health status.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T12:00:00Z",
  "service": "pma-backend-go",
  "version": "1.0.0",
  "adapters": {
    "homeassistant": {
      "connected": true,
      "type": "homeassistant",
      "version": "2024.1.0"
    }
  }
}
```

## Entity Management

Entity management endpoints for smart home devices and services.

### GET `/api/v1/entities`
Retrieve all entities with optional filtering.

**Query Parameters:**
- `include_room` (boolean): Include room information
- `include_area` (boolean): Include area information
- `domain` (string): Filter by entity domain
- `available_only` (boolean): Only available entities
- `capabilities` (array): Filter by capabilities

**Example:**
```bash
GET /api/v1/entities?include_room=true&domain=light&available_only=true
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "entity": {
        "id": "light.living_room",
        "name": "Living Room Light",
        "type": "light",
        "source": "homeassistant",
        "state": "on",
        "attributes": {
          "brightness": 255,
          "color_temp": 3000
        },
        "last_updated": "2024-01-01T12:00:00Z"
      },
      "room": {
        "id": "living_room",
        "name": "Living Room",
        "area": "main_floor"
      }
    }
  ],
  "meta": {
    "count": 1,
    "by_source": {
      "homeassistant": 1
    }
  }
}
```

### GET `/api/v1/entities/{id}`
Get a specific entity by ID.

**Response:**
```json
{
  "success": true,
  "data": {
    "entity": {
      "id": "light.living_room",
      "name": "Living Room Light",
      "type": "light",
      "source": "homeassistant",
      "state": "on",
      "attributes": {
        "brightness": 255
      }
    }
  }
}
```

### GET `/api/v1/entities/type/{type}`
Get entities by type.

**Path Parameters:**
- `type`: Entity type (e.g., "light", "switch", "sensor")

### POST `/api/v1/entities/{id}/action`
Execute an action on an entity.

**Request:**
```json
{
  "action": "turn_on",
  "parameters": {
    "brightness": 128,
    "color_temp": 2700
  }
}
```

## Room & Area Management

### GET `/api/v1/rooms`
Get all rooms with optional hierarchy.

**Query Parameters:**
- `include_entities` (boolean): Include entity list
- `include_area` (boolean): Include area information

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "living_room",
      "name": "Living Room",
      "area": "main_floor",
      "entity_count": 5,
      "entities": [
        "light.living_room",
        "switch.fan"
      ]
    }
  ]
}
```

### GET `/api/v1/areas`
Get all areas with optional hierarchy.

**Query Parameters:**
- `include_rooms` (boolean): Include room list
- `hierarchy` (boolean): Build hierarchical structure

### POST `/api/v1/areas`
Create a new area.

**Request:**
```json
{
  "name": "Main Floor",
  "description": "Primary living area",
  "parent_id": null,
  "metadata": {
    "floor": 1
  }
}
```

## Automation

Automation rule management endpoints.

### GET `/api/v1/automation/rules`
Get all automation rules.

**Query Parameters:**
- `enabled` (boolean): Filter by enabled status
- `category` (string): Filter by category
- `tag` (string): Filter by tag

**Response:**
```json
{
  "success": true,
  "data": {
    "rules": [
      {
        "id": "rule_001",
        "name": "Morning Lights",
        "description": "Turn on lights in the morning",
        "enabled": true,
        "category": "lighting",
        "triggers": [
          {
            "platform": "time",
            "at": "07:00:00"
          }
        ],
        "conditions": [
          {
            "condition": "state",
            "entity_id": "sun.sun",
            "state": "below_horizon"
          }
        ],
        "actions": [
          {
            "service": "light.turn_on",
            "target": {
              "area_id": "living_area"
            }
          }
        ],
        "created_at": "2024-01-01T00:00:00Z",
        "last_triggered": "2024-01-01T07:00:00Z"
      }
    ],
    "count": 1
  }
}
```

### POST `/api/v1/automation/rules`
Create a new automation rule.

**Request:**
```json
{
  "name": "Evening Lights",
  "description": "Turn off lights at night",
  "enabled": true,
  "category": "lighting",
  "triggers": [
    {
      "platform": "time",
      "at": "23:00:00"
    }
  ],
  "actions": [
    {
      "service": "light.turn_off",
      "target": {
        "area_id": "all"
      }
    }
  ]
}
```

### PUT `/api/v1/automation/rules/{id}`
Update an automation rule.

### DELETE `/api/v1/automation/rules/{id}`
Delete an automation rule.

### POST `/api/v1/automation/rules/{id}/trigger`
Manually trigger an automation rule.

### GET `/api/v1/automation/templates`
Get automation rule templates.

## AI Services

AI and LLM integration endpoints.

### POST `/api/v1/ai/chat`
Chat with AI assistant.

**Request:**
```json
{
  "message": "Turn on the living room lights",
  "conversation_id": "conv_123",
  "provider": "openai",
  "context": {
    "user_id": "user123",
    "location": "home"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "response": "I'll turn on the living room lights for you.",
    "actions": [
      {
        "type": "entity_action",
        "entity_id": "light.living_room",
        "action": "turn_on"
      }
    ],
    "conversation_id": "conv_123",
    "provider": "openai"
  }
}
```

### POST `/api/v1/ai/complete`
Text completion endpoint.

**Request:**
```json
{
  "prompt": "Create an automation rule that",
  "max_tokens": 100,
  "provider": "openai"
}
```

### GET `/api/v1/ai/providers`
Get available AI providers.

### GET `/api/v1/conversations`
Get conversation history.

### POST `/api/v1/conversations`
Create a new conversation.

## Analytics

Analytics and reporting endpoints.

### GET `/api/v1/analytics/data`
Get historical analytics data.

**Query Parameters:**
- `entity_id` (string): Specific entity
- `start_date` (ISO date): Start date
- `end_date` (ISO date): End date
- `aggregation` (string): hour, day, week, month

**Response:**
```json
{
  "success": true,
  "data": {
    "entity_id": "sensor.temperature",
    "start_date": "2024-01-01T00:00:00Z",
    "end_date": "2024-01-02T00:00:00Z",
    "aggregation": "hour",
    "data_points": [
      {
        "timestamp": "2024-01-01T00:00:00Z",
        "value": 21.5,
        "attributes": {
          "unit": "°C"
        }
      }
    ]
  }
}
```

### GET `/api/v1/analytics/reports`
Get available reports.

### POST `/api/v1/analytics/reports/generate`
Generate a new report.

**Request:**
```json
{
  "type": "energy_usage",
  "period": "week",
  "entities": ["sensor.power_meter"],
  "format": "pdf"
}
```

### GET `/api/v1/analytics/insights/{entityType}`
Get insights for entity type.

## Monitoring

System monitoring and alerting endpoints.

### GET `/api/v1/monitoring/metrics`
Get system metrics.

**Response:**
```json
{
  "success": true,
  "data": {
    "system": {
      "cpu_usage": 15.2,
      "memory_usage": 45.8,
      "disk_usage": 67.3,
      "uptime": "5d 12h 30m"
    },
    "application": {
      "connected_clients": 12,
      "active_automations": 8,
      "entities_count": 150,
      "response_time": 8.5
    }
  }
}
```

### GET `/api/v1/monitoring/alerts`
Get active alerts.

### POST `/api/v1/monitoring/alerts/rules`
Create alert rule.

**Request:**
```json
{
  "name": "High CPU Usage",
  "condition": "cpu_usage > 80",
  "severity": "warning",
  "actions": [
    {
      "type": "notification",
      "message": "CPU usage is high: {{value}}%"
    }
  ]
}
```

## System Management

System administration endpoints.

### GET `/api/v1/system/info`
Get system information.

**Response:**
```json
{
  "success": true,
  "data": {
    "hostname": "pma-server",
    "platform": "linux",
    "architecture": "amd64",
    "go_version": "go1.23.0",
    "version": "1.0.0",
    "build_time": "2024-01-01T00:00:00Z",
    "cpu_cores": 4,
    "memory_total": "8GB"
  }
}
```

### GET `/api/v1/system/status`
Get detailed system status.

### POST `/api/v1/system/restart`
Restart the application.

### GET `/api/v1/cache/stats`
Get cache statistics.

### POST `/api/v1/cache/clear`
Clear system caches.

### GET `/api/v1/performance/status`
Get performance metrics.

## WebSocket

### Connection

Connect to WebSocket endpoint:
```
ws://localhost:3001/ws
```

### Subscription Messages

Subscribe to real-time updates:

```json
{
  "type": "subscribe_ha_events",
  "data": {
    "event_types": ["state_changed", "automation_triggered"]
  }
}
```

### Real-time Events

Receive real-time updates:

```json
{
  "type": "pma_entity_state_changed",
  "data": {
    "entity_id": "light.living_room",
    "old_state": "off",
    "new_state": "on",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

For detailed WebSocket documentation, see [WebSocket Guide](WEBSOCKET.md).

## SDK Examples

### JavaScript/Node.js

```javascript
const axios = require('axios');

class PMAClient {
  constructor(baseURL, token) {
    this.baseURL = baseURL;
    this.token = token;
    this.client = axios.create({
      baseURL,
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }
    });
  }

  async getEntities(options = {}) {
    const response = await this.client.get('/api/v1/entities', {
      params: options
    });
    return response.data;
  }

  async executeAction(entityId, action, parameters = {}) {
    const response = await this.client.post(`/api/v1/entities/${entityId}/action`, {
      action,
      parameters
    });
    return response.data;
  }
}

// Usage
const client = new PMAClient('http://localhost:3001', 'your-jwt-token');
const entities = await client.getEntities({ domain: 'light' });
```

### Python

```python
import requests
from typing import Dict, List, Optional

class PMAClient:
    def __init__(self, base_url: str, token: str):
        self.base_url = base_url
        self.session = requests.Session()
        self.session.headers.update({
            'Authorization': f'Bearer {token}',
            'Content-Type': 'application/json'
        })
    
    def get_entities(self, **params) -> Dict:
        response = self.session.get(f'{self.base_url}/api/v1/entities', params=params)
        response.raise_for_status()
        return response.json()
    
    def execute_action(self, entity_id: str, action: str, parameters: Dict = None) -> Dict:
        data = {'action': action, 'parameters': parameters or {}}
        response = self.session.post(f'{self.base_url}/api/v1/entities/{entity_id}/action', json=data)
        response.raise_for_status()
        return response.json()

# Usage
client = PMAClient('http://localhost:3001', 'your-jwt-token')
entities = client.get_entities(domain='light')
```

## Error Codes Reference

### Common Error Codes

| Code | Message | Description |
|------|---------|-------------|
| `ENTITY_NOT_FOUND` | Entity not found | Requested entity doesn't exist |
| `INVALID_ACTION` | Invalid action | Action not supported for entity |
| `AUTH_REQUIRED` | Authentication required | JWT token missing or invalid |
| `INSUFFICIENT_PERMISSIONS` | Insufficient permissions | User lacks required permissions |
| `RATE_LIMIT_EXCEEDED` | Rate limit exceeded | Too many requests |
| `VALIDATION_FAILED` | Validation failed | Request validation error |
| `SERVICE_UNAVAILABLE` | Service unavailable | External service unavailable |

## Changelog

### v1.0.0
- Initial API release
- Core entity management
- Automation engine
- AI integration
- WebSocket support
- Analytics and monitoring

---

For more information, see the [PMA Backend Go Documentation](../README.md).