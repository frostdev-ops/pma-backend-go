# PMA Backend Go - API Reference

This document provides comprehensive documentation for the PMA Backend Go REST API, including authentication, endpoints, request/response formats, and error handling.

## Table of Contents

- [Authentication](#authentication)
  - [JWT Authentication](#jwt-authentication)
  - [PIN Authentication](#pin-authentication)
- [Base URL & Versioning](#base-url--versioning)
- [Response Format](#response-format)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [Core Endpoints](#core-endpoints)
  - [System](#system)
  - [Authentication](#authentication-endpoints)
- [Entity Management](#entity-management)
- [Room & Area Management](#room--area-management)
- [Automation](#automation)
- [AI Services](#ai-services)
- [Analytics](#analytics)
- [Monitoring](#monitoring)
- [Configuration](#configuration)
- [File Management](#file-management)
- [WebSocket](#websocket)
- [SDK Examples](#sdk-examples)
  - [JavaScript Client](#javascript-client)
  - [Python Client](#python-client)

## Authentication

The PMA Backend uses a multi-layered authentication system with JWT for session management and PIN for sensitive operations.

### JWT Authentication

All API endpoints (except `/health` and auth endpoints) require a valid JWT token.

#### Authentication Flow

1. **Login**: Authenticate with username/password to receive a JWT.
2. **Authorize**: Include `Authorization: Bearer YOUR_JWT_TOKEN` in API requests.
3. **Refresh**: Use the refresh token to get a new JWT when the current one expires.

#### Endpoints

##### `POST /api/v1/auth/login`
Authenticate and receive JWT and refresh tokens.

**Request:**
```json
{
  "username": "admin",
  "password": "yourpassword"
}
```

**Response (Success):**
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_at": "2024-01-01T12:30:00Z",
    "refresh_token": "your-refresh-token",
    "user": {
      "id": "user123",
      "username": "admin",
      "roles": ["admin"]
    }
  }
}
```

##### `POST /api/v1/auth/refresh`
Refresh an expired JWT token.

**Request:**
```json
{
  "refresh_token": "your-refresh-token"
}
```

**Response (Success):**
```json
{
  "success": true,
  "data": {
    "token": "new-jwt-token...",
    "expires_at": "2024-01-01T13:00:00Z"
  }
}
```

##### `POST /api/v1/auth/logout`
Log out and invalidate the session.

**Response (Success):**
```json
{
  "success": true,
  "message": "Logged out successfully"
}
```

### PIN Authentication

PIN authentication is an additional layer of security for sensitive operations.

#### Endpoints

##### `POST /api/v1/auth/set-pin`
Set or change the user's PIN (requires JWT).

**Request:**
```json
{
  "pin": "1234"
}
```

##### `POST /api/v1/auth/verify-pin`
Verify the PIN for a sensitive operation.

**Request:**
```json
{
  "pin": "1234"
}
```

**Response (Success):**
```json
{
  "success": true,
  "data": {
    "verified": true,
    "expires_at": "2024-01-01T12:05:00Z"
  }
}
```

## Base URL & Versioning

- **Base URL**: `http://localhost:3001`
- **API Base**: `/api/v1`
- **Health Check**: `/health`
- **WebSocket**: `/ws`

Example: `http://localhost:3001/api/v1/entities`

## Response Format

All API responses follow a consistent JSON format.

### Success Response

```json
{
  "success": true,
  "data": { /* Response data */ },
  "message": "Optional success message",
  "timestamp": "2024-01-01T12:00:00Z",
  "request_id": "uuid-for-request"
}
```

### Error Response

```json
{
  "success": false,
  "error": "Detailed error description",
  "code": 404,
  "timestamp": "2024-01-01T12:00:00Z",
  "path": "/api/v1/invalid-endpoint",
  "method": "GET",
  "request_id": "uuid-for-request",
  "details": { /* Additional error details */ }
}
```

## Error Handling

| Status Code | Description |
|-------------|-------------|
| 400 | **Bad Request**: Invalid input or missing parameters |
| 401 | **Unauthorized**: Missing or invalid authentication token |
| 403 | **Forbidden**: Insufficient permissions for the operation |
| 404 | **Not Found**: The requested resource does not exist |
| 429 | **Too Many Requests**: Rate limit exceeded |
| 500 | **Internal Server Error**: An unexpected server error occurred |

## Rate Limiting

The API includes rate limiting to prevent abuse. The default is 100 requests per minute. Custom limits can be configured.

## Core Endpoints

### System

#### `GET /health`
Check the health and status of the system.

**Response:**
```json
{
  "status": "UP",
  "version": "1.0.0",
  "timestamp": "2024-01-01T12:00:00Z",
  "services": {
    "database": "connected",
    "home_assistant": "connected"
  }
}
```

#### `GET /api/v1/system/info`
Get detailed system information (requires auth).

### Authentication Endpoints

See [Authentication](#authentication) section for details.

## Entity Management

Manage unified smart home entities.

#### `GET /api/v1/entities`
List all entities with filtering and pagination.

**Query Parameters:**
- `type`: Filter by entity type (e.g., `light`, `sensor`)
- `source`: Filter by source (e.g., `homeassistant`)
- `room_id`: Filter by room ID
- `limit`: Number of entities to return (default: 100)
- `offset`: Offset for pagination

#### `GET /api/v1/entities/:id`
Get a single entity by its ID.

#### `POST /api/v1/entities`
Create a new virtual entity.

#### `PUT /api/v1/entities/:id`
Update an entity's properties.

#### `POST /api/v1/entities/:id/actions`
Execute an action on an entity (e.g., `turn_on`, `set_brightness`).

**Request:**
```json
{
  "action": "turn_on",
  "parameters": {
    "brightness": 200,
    "color_temp": 3000
  }
}
```

## Room & Area Management

Manage rooms and areas in your smart home.

#### `GET /api/v1/rooms`
List all rooms and their entities.

#### `POST /api/v1/rooms`
Create a new room.

#### `GET /api/v1/rooms/:id`
Get details for a specific room.

## Automation

Manage automation rules and triggers.

#### `GET /api/v1/automation/rules`
List all automation rules.

#### `POST /api/v1/automation/rules`
Create a new automation rule.

**Request:**
```json
{
  "name": "Evening Lights",
  "triggers": [{"type": "time", "at": "sunset"}],
  "conditions": [{"type": "state", "entity_id": "sun.sun", "state": "below_horizon"}],
  "actions": [{"type": "turn_on", "entity_id": "light.living_room"}]
}
```

#### `GET /api/v1/automation/rules/:id`
Get a specific automation rule.

## AI Services

Interact with the integrated AI assistant.

#### `POST /api/v1/ai/chat`
Send a message to the AI assistant for command execution or questions.

**Request:**
```json
{
  "message": "Turn on the living room lights",
  "context": {
    "current_room": "living_room"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "response": "I've turned on the living room lights for you.",
    "actions_executed": 1,
    "entities_changed": ["light.living_room"]
  }
}
```

## Analytics

Access analytics and historical data.

#### `GET /api/v1/analytics/summary`
Get an analytics summary for a given period.

#### `GET /api/v1/analytics/history/:entity_id`
Get historical data for a specific entity.

## Monitoring

Access system monitoring and performance metrics.

#### `GET /api/v1/monitoring/status`
Get the current status of all monitored services.

#### `GET /api/v1/performance/metrics`
Get detailed performance metrics in Prometheus format.

## Configuration

Manage system configuration.

#### `GET /api/v1/config`
Get the current system configuration (admin only).

#### `PUT /api/v1/config`
Update the system configuration (admin only).

## File Management

Manage files, including uploads for mobile clients.

#### `POST /api/v1/files/upload`
Upload a file.

## WebSocket

For real-time communication details, see [WebSocket Guide](docs/WEBSOCKET.md).

## SDK Examples

### JavaScript Client

```javascript
class PMA_API {
    constructor(baseUrl, token) {
        this.baseUrl = baseUrl;
        this.token = token;
    }

    async request(endpoint, method = 'GET', body = null) {
        const headers = {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${this.token}`,
        };

        const config = {
            method,
            headers,
        };

        if (body) {
            config.body = JSON.stringify(body);
        }

        const response = await fetch(`${this.baseUrl}/api/v1${endpoint}`, config);
        return response.json();
    }

    async getEntities() {
        return this.request('/entities');
    }

    async getEntity(id) {
        return this.request(`/entities/${id}`);
    }

    async executeAction(id, action, params = {}) {
        return this.request(`/entities/${id}/actions`, 'POST', {
            action,
            parameters: params,
        });
    }
}
```

### Python Client

```python
import requests
import json

class PMA_API:
    def __init__(self, base_url, token):
        self.base_url = base_url
        self.headers = {
            'Content-Type': 'application/json',
            'Authorization': f'Bearer {token}'
        }

    def request(self, endpoint, method='GET', data=None):
        url = f"{self.base_url}/api/v1{endpoint}"
        
        if method == 'GET':
            response = requests.get(url, headers=self.headers)
        elif method == 'POST':
            response = requests.post(url, headers=self.headers, data=json.dumps(data))
        
        return response.json()

    def get_entities(self):
        return self.request('/entities')

    def get_entity(self, entity_id):
        return self.request(f'/entities/{entity_id}')

    def execute_action(self, entity_id, action, params={}):
        payload = {
            'action': action,
            'parameters': params
        }
        return self.request(f'/entities/{entity_id}/actions', 'POST', data=payload)
```