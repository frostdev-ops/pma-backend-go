# PMA Backend Go - Comprehensive API Reference

This document provides complete documentation for the PMA Backend Go REST API, covering all 500+ endpoints with authentication, request/response formats, and examples.

## Table of Contents

1. [Overview](#overview)
2. [Authentication](#authentication)
3. [Response Format](#response-format)
4. [Error Handling](#error-handling)
5. [Core System](#core-system)
6. [Entity Management](#entity-management)
7. [Room & Area Management](#room--area-management)
8. [Automation Engine](#automation-engine)
9. [AI Services](#ai-services)
10. [Analytics & Reporting](#analytics--reporting)
11. [Monitoring & Alerting](#monitoring--alerting)
12. [WebSocket Optimization](#websocket-optimization)
13. [Performance & Memory](#performance--memory)
14. [Cache Management](#cache-management)
15. [Backup & Media](#backup--media)
16. [Network Management](#network-management)
17. [Security & Safety](#security--safety)
18. [User Preferences](#user-preferences)
19. [Hardware Integration](#hardware-integration)
20. [Configuration Management](#configuration-management)
21. [SDK Examples](#sdk-examples)

## Overview

**Base URL:** `http://localhost:3001` (configurable via environment)
**API Version:** v1
**Content-Type:** `application/json`
**WebSocket:** `ws://localhost:3001/ws`

### Quick Start
1. Authenticate: `POST /api/v1/auth/login`
2. Get entities: `GET /api/v1/entities`
3. Control devices: `POST /api/v1/entities/{id}/action`
4. Monitor system: `GET /health`

## Authentication

### JWT Authentication
All endpoints (except `/health` and auth endpoints) require JWT authentication.

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

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
    "expires_at": "2024-01-15T12:00:00Z",
    "refresh_token": "refresh_token_here",
    "user": {
      "id": "1",
      "username": "admin",
      "roles": ["admin"]
    }
  }
}
```

#### Authentication Headers
```http
Authorization: Bearer YOUR_JWT_TOKEN
```

### PIN Authentication
Additional security layer for sensitive operations.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/set-pin` | POST | Set user PIN |
| `/api/v1/auth/verify-pin` | POST | Verify PIN for operations |
| `/api/v1/auth/change-pin` | POST | Change existing PIN |
| `/api/v1/auth/disable-pin` | POST | Disable PIN authentication |
| `/api/v1/auth/pin-status` | GET | Get PIN configuration status |

### Session Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/register` | POST | Register new user |
| `/api/v1/auth/validate` | POST | Validate JWT token |
| `/api/v1/auth/session` | GET | Get current session info |
| `/api/v1/auth/logout` | POST | Logout and invalidate session |

## Response Format

### Success Response
```json
{
  "success": true,
  "data": { /* Response data */ },
  "message": "Operation completed successfully",
  "timestamp": "2024-01-15T10:30:00Z",
  "request_id": "uuid-12345"
}
```

### Error Response
```json
{
  "success": false,
  "error": "Detailed error message",
  "code": 400,
  "timestamp": "2024-01-15T10:30:00Z",
  "path": "/api/v1/endpoint",
  "method": "POST",
  "request_id": "uuid-12345",
  "details": { /* Additional context */ }
}
```

## Error Handling

| Status Code | Description | Common Causes |
|-------------|-------------|---------------|
| 400 | Bad Request | Invalid JSON, missing parameters |
| 401 | Unauthorized | Missing/expired JWT token |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Resource already exists |
| 422 | Unprocessable Entity | Validation errors |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server-side error |

## Core System

### Health & Status

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Basic health check |
| `/api/v1/status` | GET | System status |
| `/api/v1/system/health` | GET | Basic system health |
| `/api/v1/system/health/detailed` | GET | Detailed health report |
| `/api/v1/system/info` | GET | System information |
| `/api/v1/system/status` | GET | Current system status |
| `/api/v1/system/metrics` | GET | System metrics |

**Example:**
```http
GET /health
```

**Response:**
```json
{
  "status": "UP",
  "version": "1.0.0",
  "timestamp": "2024-01-15T10:30:00Z",
  "uptime": "24h 5m 32s",
  "services": {
    "database": "connected",
    "homeassistant": "connected",
    "websocket": "active"
  }
}
```

### System Control

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/system/reboot` | POST | Reboot system |
| `/api/v1/system/shutdown` | POST | Shutdown system |
| `/api/v1/system/config` | GET | Get system configuration |
| `/api/v1/system/config` | POST | Update system configuration |
| `/api/v1/system/logs` | GET | Get system logs |
| `/api/v1/system/errors` | GET | Get error history |
| `/api/v1/system/errors` | DELETE | Clear error history |

## Entity Management

The entity system provides unified control over all smart home devices and virtual entities.

### Core Entity Operations

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/entities` | GET | List all entities |
| `/api/v1/entities` | POST | Create entity |
| `/api/v1/entities/{id}` | GET | Get specific entity |
| `/api/v1/entities/{id}` | DELETE | Delete entity |
| `/api/v1/entities/{id}/action` | POST | **Execute entity action** |
| `/api/v1/entities/{id}/state` | PUT | Update entity state |
| `/api/v1/entities/{id}/room` | PUT | Assign to room |

### Entity Discovery & Filtering

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/entities/search` | GET | Search entities |
| `/api/v1/entities/types` | GET | Get entity types |
| `/api/v1/entities/capabilities` | GET | Get entity capabilities |
| `/api/v1/entities/type/{type}` | GET | Filter by type |
| `/api/v1/entities/source/{source}` | GET | Filter by source |
| `/api/v1/entities/room/{roomId}` | GET | Filter by room |

### Entity Synchronization

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/entities/sync` | POST | Sync all entities |
| `/api/v1/entities/sync/status` | GET | Get sync status |

### Entity Control Examples

#### Turn on a Light
```http
POST /api/v1/entities/light.living_room/action
Content-Type: application/json
Authorization: Bearer YOUR_TOKEN

{
  "action": "turn_on",
  "parameters": {
    "brightness": 255,
    "color_temp": 3000
  }
}
```

#### Set Cover Position
```http
POST /api/v1/entities/cover.bedroom_blinds/action
Content-Type: application/json

{
  "action": "set_position",
  "parameters": {
    "position": 75
  }
}
```

#### Toggle Switch
```http
POST /api/v1/entities/switch.garden_lights/action
Content-Type: application/json

{
  "action": "toggle"
}
```

### Supported Actions by Entity Type

#### Lights
- `turn_on`, `turn_off`, `toggle`
- `set_brightness` (0-255)
- `set_color` (RGB values)
- `set_color_temp` (Kelvin)
- `flash`, `effect`

#### Switches/Outlets
- `turn_on`, `turn_off`, `toggle`

#### Covers/Blinds
- `open`, `close`, `stop`
- `set_position` (0-100%)
- `set_tilt` (0-100%)

#### Climate
- `set_temperature`
- `set_hvac_mode` (heat, cool, auto, off)
- `set_fan_mode`
- `set_humidity`

#### Media Players
- `play`, `pause`, `stop`
- `volume_up`, `volume_down`, `volume_set`
- `next_track`, `previous_track`

## Room & Area Management

### Rooms

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/rooms` | GET | List all rooms |
| `/api/v1/rooms` | POST | Create room |
| `/api/v1/rooms/{id}` | GET | Get room details |
| `/api/v1/rooms/{id}` | PUT | Update room |
| `/api/v1/rooms/{id}` | DELETE | Delete room |
| `/api/v1/rooms/stats` | GET | Room statistics |
| `/api/v1/rooms/sync-ha` | POST | Sync with Home Assistant |

### Areas

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/areas` | GET | List all areas |
| `/api/v1/areas` | POST | Create area |
| `/api/v1/areas/{id}` | GET | Get area details |
| `/api/v1/areas/{id}` | PUT | Update area |
| `/api/v1/areas/{id}` | DELETE | Delete area |
| `/api/v1/areas/{id}/entities` | GET | Get area entities |
| `/api/v1/areas/{id}/entities` | POST | Assign entities to area |
| `/api/v1/areas/{id}/entities/{entity_id}` | DELETE | Remove entity from area |

### Scenes

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/scenes` | GET | List all scenes |
| `/api/v1/scenes/{id}` | GET | Get scene details |
| `/api/v1/scenes/{id}/activate` | POST | Activate scene |

**Example - Create Room:**
```http
POST /api/v1/rooms
Content-Type: application/json

{
  "name": "Living Room",
  "description": "Main living area",
  "floor": "Ground Floor",
  "icon": "mdi:sofa"
}
```

## Automation Engine

### Automation Rules

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/automation/rules` | GET | List automation rules |
| `/api/v1/automation/rules` | POST | Create automation rule |
| `/api/v1/automation/rules/{id}` | GET | Get rule details |
| `/api/v1/automation/rules/{id}` | PUT | Update rule |
| `/api/v1/automation/rules/{id}` | DELETE | Delete rule |
| `/api/v1/automation/rules/{id}/enable` | POST | Enable rule |
| `/api/v1/automation/rules/{id}/disable` | POST | Disable rule |
| `/api/v1/automation/rules/{id}/test` | POST | Test rule |
| `/api/v1/automation/rules/{id}/trigger` | POST | Manually trigger rule |

### Automation Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/automation/rules/import` | POST | Import rules |
| `/api/v1/automation/rules/export` | GET | Export rules |
| `/api/v1/automation/rules/validate` | POST | Validate rule syntax |
| `/api/v1/automation/statistics` | GET | Automation statistics |
| `/api/v1/automation/templates` | GET | Rule templates |
| `/api/v1/automation/history` | GET | Execution history |
| `/api/v1/automation/stats` | GET | Performance stats |

**Example - Create Automation:**
```http
POST /api/v1/automation/rules
Content-Type: application/json

{
  "name": "Evening Lights",
  "description": "Turn on lights at sunset",
  "triggers": [
    {
      "type": "sun",
      "event": "sunset",
      "offset": "-00:30:00"
    }
  ],
  "conditions": [
    {
      "type": "state",
      "entity_id": "binary_sensor.someone_home",
      "state": "on"
    }
  ],
  "actions": [
    {
      "type": "service",
      "service": "light.turn_on",
      "entity_id": "light.living_room",
      "data": {
        "brightness": 180
      }
    }
  ]
}
```

## AI Services

### Chat & Conversations

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/ai/chat` | POST | Chat with AI |
| `/api/v1/ai/complete` | POST | Text completion |
| `/api/v1/ai/chat/context` | POST | Chat with context |
| `/api/v1/conversations` | GET | List conversations |
| `/api/v1/conversations` | POST | Create conversation |
| `/api/v1/conversations/{id}` | GET | Get conversation |
| `/api/v1/conversations/{id}` | PUT | Update conversation |
| `/api/v1/conversations/{id}` | DELETE | Delete conversation |
| `/api/v1/conversations/{id}/messages` | GET | Get messages |
| `/api/v1/conversations/{id}/messages` | POST | Send message |

### AI Configuration

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/ai/providers` | GET | List AI providers |
| `/api/v1/ai/models` | GET | Available models |
| `/api/v1/ai/settings` | GET | AI settings |
| `/api/v1/ai/settings` | POST | Update settings |
| `/api/v1/ai/test-connection` | POST | Test AI connection |
| `/api/v1/ai/statistics` | GET | AI usage statistics |

### AI Analysis

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/ai/summary` | GET | System summary |
| `/api/v1/ai/analyze/entity/{id}` | POST | Analyze entity |
| `/api/v1/ai/generate/automation` | POST | Generate automation |
| `/api/v1/ai/test/{provider}` | POST | Test AI provider |

### Ollama Integration

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/ai/ollama/status` | GET | Ollama status |
| `/api/v1/ai/ollama/metrics` | GET | Ollama metrics |
| `/api/v1/ai/ollama/health` | GET | Ollama health |
| `/api/v1/ai/ollama/start` | POST | Start Ollama |
| `/api/v1/ai/ollama/stop` | POST | Stop Ollama |
| `/api/v1/ai/ollama/restart` | POST | Restart Ollama |
| `/api/v1/ai/ollama/monitoring` | GET | Ollama monitoring |

### MCP (Model Context Protocol)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/ai/mcp/servers` | GET | List MCP servers |
| `/api/v1/ai/mcp/servers` | POST | Add MCP server |
| `/api/v1/ai/mcp/servers/{id}` | DELETE | Remove MCP server |
| `/api/v1/ai/mcp/servers/{id}/restart` | POST | Restart MCP server |
| `/api/v1/mcp/status` | GET | MCP status |
| `/api/v1/mcp/tools` | GET | Available MCP tools |
| `/api/v1/mcp/tools/execute` | POST | Execute MCP tool |

**Example - AI Chat:**
```http
POST /api/v1/ai/chat
Content-Type: application/json

{
  "message": "Turn on the living room lights and set them to 50% brightness",
  "context": {
    "room": "living_room",
    "user_preferences": {
      "default_brightness": 180
    }
  }
}
```

## Analytics & Reporting

### Historical Data

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/analytics/data` | GET | Get historical data |
| `/api/v1/analytics/events` | POST | Submit event |
| `/api/v1/analytics/metrics` | GET | Custom metrics |
| `/api/v1/analytics/metrics` | POST | Create metric |
| `/api/v1/analytics/insights/{entityType}` | GET | Get insights |

### Reports

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/analytics/reports` | GET | List reports |
| `/api/v1/analytics/reports/generate` | POST | Generate report |
| `/api/v1/analytics/reports/{id}` | GET | Get report |
| `/api/v1/analytics/reports/templates` | GET | Report templates |
| `/api/v1/analytics/reports/templates` | POST | Create template |
| `/api/v1/analytics/reports/schedule` | POST | Schedule report |
| `/api/v1/analytics/reports/schedules` | GET | List scheduled reports |
| `/api/v1/analytics/reports/schedules/{id}` | DELETE | Delete schedule |

### Visualizations

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/analytics/visualizations` | GET | List visualizations |
| `/api/v1/analytics/visualizations` | POST | Create visualization |
| `/api/v1/analytics/visualizations/{id}/data` | GET | Get visualization data |
| `/api/v1/analytics/visualizations/{id}` | PUT | Update visualization |
| `/api/v1/analytics/visualizations/{id}` | DELETE | Delete visualization |

### Dashboards

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/analytics/dashboards` | GET | List dashboards |
| `/api/v1/analytics/dashboards` | POST | Create dashboard |
| `/api/v1/analytics/dashboards/{id}` | GET | Get dashboard |
| `/api/v1/analytics/dashboards/{id}` | PUT | Update dashboard |
| `/api/v1/analytics/dashboards/{id}` | DELETE | Delete dashboard |

### Export

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/analytics/export/csv` | POST | Export to CSV |
| `/api/v1/analytics/export/json` | POST | Export to JSON |
| `/api/v1/analytics/export/excel` | POST | Export to Excel |
| `/api/v1/analytics/export/pdf` | POST | Export to PDF |
| `/api/v1/analytics/export/schedules` | GET | Export schedules |
| `/api/v1/analytics/export/schedules` | POST | Create export schedule |

### Predictions

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/analytics/predictions/train` | POST | Train model |
| `/api/v1/analytics/predictions/predict` | POST | Make prediction |
| `/api/v1/analytics/predictions/models` | GET | List models |
| `/api/v1/analytics/predictions/models/{id}/accuracy` | GET | Model accuracy |
| `/api/v1/analytics/predictions/models/{id}` | DELETE | Delete model |

## Monitoring & Alerting

### Alerts & Rules

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/monitoring/alerts` | GET | List alerts |
| `/api/v1/monitoring/alerts/rules` | GET | Alert rules |
| `/api/v1/monitoring/alerts/rules` | POST | Create alert rule |
| `/api/v1/monitoring/alerts/rules/{id}` | PUT | Update rule |
| `/api/v1/monitoring/alerts/rules/{id}` | DELETE | Delete rule |
| `/api/v1/monitoring/alerts/rules/{id}/test` | POST | Test rule |
| `/api/v1/monitoring/alerts/active` | GET | Active alerts |
| `/api/v1/monitoring/alerts/history` | GET | Alert history |
| `/api/v1/monitoring/alerts/{id}/acknowledge` | POST | Acknowledge alert |
| `/api/v1/monitoring/alerts/{id}/resolve` | POST | Resolve alert |

### Dashboards

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/monitoring/dashboards` | GET | Monitoring dashboards |
| `/api/v1/monitoring/dashboards` | POST | Create dashboard |
| `/api/v1/monitoring/dashboards/{id}` | GET | Get dashboard |
| `/api/v1/monitoring/dashboards/{id}` | PUT | Update dashboard |
| `/api/v1/monitoring/dashboards/{id}` | DELETE | Delete dashboard |
| `/api/v1/monitoring/dashboards/{id}/data` | GET | Dashboard data |
| `/api/v1/monitoring/dashboards/{id}/export` | GET | Export dashboard |

### Prediction & Anomaly Detection

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/monitoring/prediction/models` | GET | Prediction models |
| `/api/v1/monitoring/prediction/models` | POST | Create model |
| `/api/v1/monitoring/prediction/models/{id}/train` | POST | Train model |
| `/api/v1/monitoring/prediction/models/{id}/predict` | POST | Generate prediction |
| `/api/v1/monitoring/anomalies/detectors` | GET | Anomaly detectors |
| `/api/v1/monitoring/anomalies/detectors` | POST | Create detector |
| `/api/v1/monitoring/anomalies` | GET | List anomalies |
| `/api/v1/monitoring/anomalies/{id}/feedback` | POST | Provide feedback |

### Forecasting

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/monitoring/forecasting/forecasters` | GET | List forecasters |
| `/api/v1/monitoring/forecasting/forecasters` | POST | Create forecaster |
| `/api/v1/monitoring/forecasting/forecasters/{id}/forecast` | POST | Generate forecast |
| `/api/v1/monitoring/forecasting/forecasts` | GET | List forecasts |
| `/api/v1/monitoring/forecasting/forecasts/{id}/accuracy` | GET | Forecast accuracy |

### System Monitoring

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/monitoring/overview` | GET | Monitoring overview |
| `/api/v1/monitoring/health` | GET | Monitoring health |
| `/api/v1/monitoring/metrics/summary` | GET | Metrics summary |
| `/api/v1/monitoring/system/performance` | GET | System performance |

### Reports

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/monitoring/reports/daily` | GET | Daily report |
| `/api/v1/monitoring/reports/weekly` | GET | Weekly report |
| `/api/v1/monitoring/reports/monthly` | GET | Monthly report |
| `/api/v1/monitoring/reports/custom` | POST | Custom report |

## WebSocket Optimization

### Connection Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/websocket/connections` | GET | List connections |
| `/api/v1/websocket/connections/stats` | GET | Connection statistics |
| `/api/v1/websocket/connections/{id}` | GET | Get connection |
| `/api/v1/websocket/connections/{id}` | DELETE | Disconnect client |
| `/api/v1/websocket/connections/{id}/ping` | POST | Ping client |
| `/api/v1/websocket/connections/{id}/health` | GET | Client health |
| `/api/v1/websocket/connections/{id}/metrics` | GET | Client metrics |

### Connection Pools

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/websocket/pools` | GET | List connection pools |
| `/api/v1/websocket/pools` | POST | Create pool |
| `/api/v1/websocket/pools/{name}` | GET | Get pool |
| `/api/v1/websocket/pools/{name}` | PUT | Update pool |
| `/api/v1/websocket/pools/{name}` | DELETE | Delete pool |
| `/api/v1/websocket/pools/{name}/stats` | GET | Pool statistics |
| `/api/v1/websocket/pools/{name}/resize` | POST | Resize pool |
| `/api/v1/websocket/pools/{name}/cleanup` | POST | Cleanup pool |

### Compression

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/websocket/compression/stats` | GET | Compression stats |
| `/api/v1/websocket/compression/config` | GET | Compression config |
| `/api/v1/websocket/compression/config` | PUT | Update config |
| `/api/v1/websocket/compression/test` | POST | Test compression |
| `/api/v1/websocket/compression/algorithms` | GET | Supported algorithms |
| `/api/v1/websocket/compression/performance` | GET | Performance metrics |

### Load Balancing

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/websocket/loadbalancer/stats` | GET | Load balancer stats |
| `/api/v1/websocket/loadbalancer/config` | GET | Load balancer config |
| `/api/v1/websocket/loadbalancer/config` | PUT | Update config |
| `/api/v1/websocket/loadbalancer/workers` | GET | Worker pools |
| `/api/v1/websocket/loadbalancer/workers/{id}` | GET | Get worker pool |
| `/api/v1/websocket/loadbalancer/workers/{id}/scale` | POST | Scale worker pool |
| `/api/v1/websocket/loadbalancer/distribution` | GET | Load distribution |

### Message Batching

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/websocket/batching/stats` | GET | Batching statistics |
| `/api/v1/websocket/batching/config` | GET | Batching config |
| `/api/v1/websocket/batching/config` | PUT | Update config |
| `/api/v1/websocket/batching/flush` | POST | Flush batches |
| `/api/v1/websocket/batching/performance` | GET | Performance metrics |

### Performance Monitoring

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/websocket/performance/overview` | GET | Performance overview |
| `/api/v1/websocket/performance/metrics` | GET | Performance metrics |
| `/api/v1/websocket/performance/history` | GET | Performance history |
| `/api/v1/websocket/performance/latency` | GET | Latency metrics |
| `/api/v1/websocket/performance/throughput` | GET | Throughput metrics |
| `/api/v1/websocket/performance/resources` | GET | Resource usage |
| `/api/v1/websocket/performance/benchmark` | POST | Run benchmark |

### Circuit Breaker

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/websocket/circuitbreaker/status` | GET | Circuit breaker status |
| `/api/v1/websocket/circuitbreaker/stats` | GET | Circuit breaker stats |
| `/api/v1/websocket/circuitbreaker/reset` | POST | Reset circuit breaker |
| `/api/v1/websocket/circuitbreaker/config` | PUT | Update config |
| `/api/v1/websocket/circuitbreaker/history` | GET | Circuit breaker history |

### Health & Diagnostics

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/websocket/health` | GET | Overall health |
| `/api/v1/websocket/health/components` | GET | Component health |
| `/api/v1/websocket/health/alerts` | GET | Health alerts |
| `/api/v1/websocket/health/check` | POST | Trigger health check |
| `/api/v1/websocket/health/history` | GET | Health history |

### Configuration

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/websocket/config` | GET | Optimization config |
| `/api/v1/websocket/config` | PUT | Update config |
| `/api/v1/websocket/config/reset` | POST | Reset to defaults |
| `/api/v1/websocket/config/presets` | GET | Config presets |
| `/api/v1/websocket/config/presets/{name}/apply` | POST | Apply preset |
| `/api/v1/websocket/config/export` | POST | Export config |
| `/api/v1/websocket/config/import` | POST | Import config |

### Advanced Diagnostics

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/websocket/diagnostics` | GET | Get diagnostics |
| `/api/v1/websocket/diagnostics/trace` | POST | Start tracing |
| `/api/v1/websocket/diagnostics/trace` | DELETE | Stop tracing |
| `/api/v1/websocket/diagnostics/trace/results` | GET | Trace results |
| `/api/v1/websocket/diagnostics/profile` | POST | Profile performance |
| `/api/v1/websocket/diagnostics/logs` | GET | Optimization logs |

### Real-time Monitoring

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/websocket/realtime/stream` | GET | Stream metrics |
| `/api/v1/websocket/realtime/dashboard` | GET | Real-time dashboard |
| `/api/v1/websocket/realtime/alerts/subscribe` | POST | Subscribe to alerts |
| `/api/v1/websocket/realtime/alerts/unsubscribe` | DELETE | Unsubscribe from alerts |

### Administration

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/websocket/admin/optimize` | POST | Trigger optimization |
| `/api/v1/websocket/admin/maintenance` | POST | Start maintenance |
| `/api/v1/websocket/admin/maintenance` | DELETE | Stop maintenance |
| `/api/v1/websocket/admin/backup` | POST | Backup configuration |
| `/api/v1/websocket/admin/restore` | POST | Restore configuration |
| `/api/v1/websocket/admin/system` | GET | System information |

### WebSocket Events

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/websocket/stats` | GET | WebSocket statistics |
| `/api/v1/websocket/broadcast` | POST | Broadcast message |
| `/api/v1/websocket/ha/subscribe` | POST | Subscribe to HA events |
| `/api/v1/websocket/ha/unsubscribe` | POST | Unsubscribe from HA |
| `/api/v1/websocket/ha/subscriptions` | GET | HA subscriptions |

## Performance & Memory

### Performance Monitoring

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/performance/status` | GET | Performance status |
| `/api/v1/performance/profile` | GET | Start profiling |
| `/api/v1/performance/optimize` | POST | Trigger optimization |
| `/api/v1/performance/report` | GET | Performance report |
| `/api/v1/performance/queries/slow` | GET | Slow queries |
| `/api/v1/performance/benchmark` | POST | Run benchmarks |

### Memory Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/memory/status` | GET | Memory status |
| `/api/v1/memory/stats` | GET | Memory statistics |
| `/api/v1/memory/gc` | POST | Force garbage collection |
| `/api/v1/memory/optimize` | POST | Optimize memory |
| `/api/v1/memory/leaks` | GET | Detect memory leaks |
| `/api/v1/memory/leaks/scan` | GET | Scan for leaks |

### Memory Pools

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/memory/pools` | GET | Pool statistics |
| `/api/v1/memory/pools/{name}` | GET | Pool details |
| `/api/v1/memory/pools/{name}/resize` | POST | Resize pool |
| `/api/v1/memory/pools/optimize` | POST | Optimize pools |

### Memory Pressure

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/memory/pressure` | GET | Memory pressure |
| `/api/v1/memory/pressure/handle` | POST | Handle pressure |
| `/api/v1/memory/pressure/config` | GET | Pressure config |
| `/api/v1/memory/pressure/config` | PUT | Update config |

### Preallocation

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/memory/preallocation` | GET | Preallocation stats |
| `/api/v1/memory/preallocation/analyze` | POST | Analyze patterns |
| `/api/v1/memory/preallocation/optimize` | POST | Optimize preallocation |

### Optimization Engine

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/memory/optimization/status` | GET | Optimization status |
| `/api/v1/memory/optimization/start` | POST | Start optimization |
| `/api/v1/memory/optimization/stop` | POST | Stop optimization |
| `/api/v1/memory/optimization/history` | GET | Optimization history |
| `/api/v1/memory/optimization/report` | GET | Optimization report |

### Memory Monitoring

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/memory/monitor` | GET | Memory monitoring |
| `/api/v1/memory/monitor/start` | POST | Start monitoring |
| `/api/v1/memory/monitor/stop` | POST | Stop monitoring |

### Database Performance

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/performance/memory` | GET | Memory stats |
| `/api/v1/performance/memory/gc` | POST | Force GC |
| `/api/v1/performance/database/pool` | GET | DB pool stats |
| `/api/v1/performance/database/optimize` | POST | Optimize database |

## Cache Management

### Cache Operations

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/cache/clear` | POST | Clear all caches |
| `/api/v1/cache/refresh` | POST | Refresh all caches |
| `/api/v1/cache/status` | GET | Cache status |
| `/api/v1/cache/warm` | POST | Warm caches |
| `/api/v1/cache/stats` | GET | Cache statistics |
| `/api/v1/cache/invalidate` | POST | Invalidate keys |
| `/api/v1/cache/health` | GET | Cache health |
| `/api/v1/cache/optimize` | POST | Optimize caches |

### Cache Types

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/cache/list` | GET | List all caches |
| `/api/v1/cache/types` | GET | Cache types |
| `/api/v1/cache/clear/{type}` | POST | Clear by type |
| `/api/v1/cache/refresh/{type}` | POST | Refresh by type |

### Individual Cache Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/cache/{name}/stats` | GET | Individual cache stats |
| `/api/v1/cache/{name}/clear` | POST | Clear individual cache |
| `/api/v1/cache/{name}/refresh` | POST | Refresh individual cache |
| `/api/v1/cache/{name}/keys` | GET | Cache keys |

### Memory Usage

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/cache/memory` | GET | Memory usage |
| `/api/v1/cache/memory/free` | POST | Free memory |

## Backup & Media

### Backup Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/backup` | POST | Create backup |
| `/api/v1/backup` | GET | List backups |
| `/api/v1/backup/{id}` | GET | Get backup |
| `/api/v1/backup/{id}` | DELETE | Delete backup |
| `/api/v1/backup/{id}/restore` | POST | Restore backup |
| `/api/v1/backup/{id}/validate` | POST | Validate backup |
| `/api/v1/backup/{id}/export` | GET | Export backup |
| `/api/v1/backup/import` | POST | Import backup |
| `/api/v1/backup/schedule` | POST | Schedule backup |
| `/api/v1/backup/statistics` | GET | Backup statistics |
| `/api/v1/backup/cleanup` | POST | Cleanup old backups |

### Media Processing

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/media/process` | POST | Process media |
| `/api/v1/media/validate` | POST | Validate media |
| `/api/v1/media/info/{id}` | GET | Get media info |
| `/api/v1/media/thumbnail` | POST | Generate thumbnail |
| `/api/v1/media/thumbnail/{id}` | GET | Get thumbnail |
| `/api/v1/media/thumbnails/multiple` | POST | Generate multiple |

### Media Streaming

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/media/stream/video/{id}` | GET | Stream video |
| `/api/v1/media/stream/audio/{id}` | GET | Stream audio |
| `/api/v1/media/stream/url/{id}` | GET | Get streaming URL |
| `/api/v1/media/transcode/{id}` | POST | Transcode video |
| `/api/v1/media/formats` | GET | Supported formats |
| `/api/v1/media/stats` | GET | Media statistics |

### Camera Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/cameras` | GET | List cameras |
| `/api/v1/cameras/enabled` | GET | Enabled cameras |
| `/api/v1/cameras` | POST | Create camera |
| `/api/v1/cameras/{id}` | GET | Get camera |
| `/api/v1/cameras/{id}` | PUT | Update camera |
| `/api/v1/cameras/{id}` | DELETE | Delete camera |
| `/api/v1/cameras/entity/{entityId}` | GET | Get by entity ID |
| `/api/v1/cameras/type/{type}` | GET | Get by type |
| `/api/v1/cameras/search` | GET | Search cameras |
| `/api/v1/cameras/{id}/status` | PUT | Update status |
| `/api/v1/cameras/{id}/stream` | GET | Camera stream |
| `/api/v1/cameras/{id}/snapshot` | GET | Camera snapshot |
| `/api/v1/cameras/stats` | GET | Camera statistics |

## Network Management

### Network Status

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/network/status` | GET | Network status |
| `/api/v1/network/interfaces` | GET | Network interfaces |
| `/api/v1/network/traffic` | GET | Traffic statistics |
| `/api/v1/network/metrics` | GET | Network metrics |
| `/api/v1/network/config` | GET | Network configuration |
| `/api/v1/network/test-connection` | POST | Test connection |

### Device Discovery

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/network/devices` | GET | Discovered devices |
| `/api/v1/network/devices/scan` | POST | Scan network |
| `/api/v1/network/devices/suggestions` | GET | Port suggestions |

### Port Forwarding

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/network/port-forwarding` | GET | Port forwarding rules |
| `/api/v1/network/port-forwarding` | POST | Create rule |
| `/api/v1/network/port-forwarding/{ruleId}` | PUT | Update rule |
| `/api/v1/network/port-forwarding/{ruleId}` | DELETE | Delete rule |

## Security & Safety

### Security Status

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/security/status` | GET | Security status |
| `/api/v1/security/metrics` | GET | Security metrics |
| `/api/v1/security/events` | GET | Security events |

### Rate Limiting

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/security/ratelimit/status` | GET | Rate limit status |
| `/api/v1/security/ratelimit/metrics` | GET | Rate limit metrics |
| `/api/v1/security/ratelimit/violators` | GET | Top violators |
| `/api/v1/security/ratelimit/block` | POST | Block IP |
| `/api/v1/security/ratelimit/unblock` | POST | Unblock IP |

### IP Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/security/ips/blocked` | GET | Blocked IPs |
| `/api/v1/security/ips/block` | POST | Block IP address |
| `/api/v1/security/ips/unblock` | POST | Unblock IP address |
| `/api/v1/security/ips/whitelist` | GET | Whitelisted IPs |
| `/api/v1/security/ips/whitelist` | POST | Add to whitelist |
| `/api/v1/security/ips/whitelist` | DELETE | Remove from whitelist |

### Threat Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/security/threats` | GET | List threats |
| `/api/v1/security/threats` | POST | Add threat |
| `/api/v1/security/threats/{ip}` | DELETE | Remove threat |
| `/api/v1/security/threats/analysis` | GET | Threat analysis |

### Attack Monitoring

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/security/attacks` | GET | Attack data |
| `/api/v1/security/attacks/patterns` | GET | Attack patterns |
| `/api/v1/security/attacks/summary` | GET | Attack summary |

### Security Configuration

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/security/config` | GET | Security config |
| `/api/v1/security/config` | PUT | Update config |
| `/api/v1/security/config/reset` | POST | Reset config |

### Security Reports

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/security/reports/summary` | GET | Security summary |
| `/api/v1/security/reports/detailed` | GET | Detailed report |
| `/api/v1/security/reports/export` | POST | Export report |

### Live Monitoring

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/security/monitor/live` | GET | Live security data |
| `/api/v1/security/monitor/alerts` | GET | Security alerts |

### Error Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/errors/reports` | GET | Error reports |
| `/api/v1/errors/reports/{error_id}` | GET | Error report |
| `/api/v1/errors/reports/{error_id}/resolve` | POST | Resolve error |
| `/api/v1/errors/stats` | GET | Error statistics |
| `/api/v1/errors/recovery/metrics` | GET | Recovery metrics |
| `/api/v1/errors/recovery/circuit-breakers` | GET | Circuit breakers |
| `/api/v1/errors/recovery/circuit-breakers/{name}/reset` | POST | Reset breaker |
| `/api/v1/errors/health` | GET | Error health |
| `/api/v1/errors/cleanup` | POST | Cleanup old errors |
| `/api/v1/errors/test` | POST | Test error recovery |

## User Preferences

### User Preferences

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/preferences` | GET | Get user preferences |
| `/api/v1/preferences` | PUT | Update preferences |
| `/api/v1/preferences/reset` | POST | Reset to defaults |
| `/api/v1/preferences/section/{section}` | GET | Get section |
| `/api/v1/preferences/section/{section}` | PUT | Update section |

### Theme Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/preferences/themes` | GET | Available themes |
| `/api/v1/preferences/themes/{id}` | GET | Get theme |
| `/api/v1/preferences/themes` | POST | Create custom theme |
| `/api/v1/preferences/themes/{id}` | DELETE | Delete theme |
| `/api/v1/preferences/themes/{id}/apply` | POST | Apply theme |

### Dashboard Customization

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/preferences/dashboard` | GET | User dashboard |
| `/api/v1/preferences/dashboard` | POST | Save dashboard |
| `/api/v1/preferences/dashboard/widgets` | POST | Add widget |
| `/api/v1/preferences/dashboard/widgets/{id}` | PUT | Update widget |
| `/api/v1/preferences/dashboard/widgets/{id}` | DELETE | Remove widget |
| `/api/v1/preferences/dashboard/available-widgets` | GET | Available widgets |
| `/api/v1/preferences/dashboard/widgets/{id}/data` | GET | Widget data |
| `/api/v1/preferences/dashboard/widgets/{id}/refresh` | POST | Refresh widget |

### Localization

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/preferences/locale` | GET | User locale |
| `/api/v1/preferences/locale` | PUT | Set locale |
| `/api/v1/preferences/locale/supported` | GET | Supported locales |
| `/api/v1/preferences/locale/translations/{locale}` | GET | Translations |

### Preference Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/preferences/export` | GET | Export preferences |
| `/api/v1/preferences/import` | POST | Import preferences |
| `/api/v1/preferences/statistics` | GET | Preference statistics |

## Hardware Integration

### Display Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/display-settings` | GET | Display settings |
| `/api/v1/display-settings` | POST | Update settings |
| `/api/v1/display-settings` | PUT | Put settings |
| `/api/v1/display-settings/capabilities` | GET | Display capabilities |
| `/api/v1/display-settings/hardware` | GET | Hardware info |
| `/api/v1/display-settings/wake` | POST | Wake screen |

### Bluetooth Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/bluetooth/status` | GET | Bluetooth status |
| `/api/v1/bluetooth/capabilities` | GET | BT capabilities |
| `/api/v1/bluetooth/power` | POST | Set power |
| `/api/v1/bluetooth/discoverable` | POST | Set discoverable |
| `/api/v1/bluetooth/scan` | POST | Scan for devices |
| `/api/v1/bluetooth/devices` | GET | All devices |
| `/api/v1/bluetooth/devices/paired` | GET | Paired devices |
| `/api/v1/bluetooth/devices/connected` | GET | Connected devices |
| `/api/v1/bluetooth/devices/{address}` | GET | Get device |
| `/api/v1/bluetooth/devices/{address}/pair` | POST | Pair device |
| `/api/v1/bluetooth/devices/{address}/connect` | POST | Connect device |
| `/api/v1/bluetooth/devices/{address}/disconnect` | POST | Disconnect device |
| `/api/v1/bluetooth/devices/{address}` | DELETE | Remove device |
| `/api/v1/bluetooth/stats` | GET | BT statistics |

### UPS Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/ups/status` | GET | UPS status |
| `/api/v1/ups/history` | GET | UPS history |
| `/api/v1/ups/battery-trends` | GET | Battery trends |
| `/api/v1/ups/metrics` | GET | UPS metrics |
| `/api/v1/ups/config` | GET | UPS configuration |
| `/api/v1/ups/info` | GET | UPS information |
| `/api/v1/ups/variables` | GET | UPS variables |
| `/api/v1/ups/connection` | GET | Connection info |
| `/api/v1/ups/test-connection` | POST | Test connection |
| `/api/v1/ups/monitoring/start` | POST | Start monitoring |
| `/api/v1/ups/monitoring/stop` | POST | Stop monitoring |
| `/api/v1/ups/alerts/thresholds` | PUT | Update thresholds |

### Energy Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/energy/settings` | GET | Energy settings |
| `/api/v1/energy/settings` | PUT | Update settings |
| `/api/v1/energy/data` | GET | Energy data |
| `/api/v1/energy/metrics` | GET | Energy metrics |
| `/api/v1/energy/history` | GET | Energy history |
| `/api/v1/energy/statistics` | GET | Energy statistics |
| `/api/v1/energy/devices/breakdown` | GET | Device breakdown |
| `/api/v1/energy/devices/{entityId}/history` | GET | Device history |
| `/api/v1/energy/devices/{entityId}/data` | GET | Device data |
| `/api/v1/energy/tracking/start` | POST | Start tracking |
| `/api/v1/energy/tracking/stop` | POST | Stop tracking |
| `/api/v1/energy/service/status` | GET | Service status |
| `/api/v1/energy/cleanup` | POST | Cleanup old data |

### Ring Integration

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/ring/config/status` | GET | Ring config status |
| `/api/v1/ring/config/setup` | POST | Setup Ring config |
| `/api/v1/ring/config/test` | POST | Test connection |
| `/api/v1/ring/config` | DELETE | Delete config |
| `/api/v1/ring/config/restart` | POST | Restart service |
| `/api/v1/ring/auth/start` | POST | Start authentication |
| `/api/v1/ring/auth/verify` | POST | Complete 2FA |
| `/api/v1/ring/cameras` | GET | Ring cameras |
| `/api/v1/ring/cameras/{cameraId}` | GET | Get camera |
| `/api/v1/ring/cameras/{cameraId}/snapshot` | GET | Camera snapshot |
| `/api/v1/ring/cameras/{cameraId}/light` | POST | Control light |
| `/api/v1/ring/cameras/{cameraId}/siren` | POST | Control siren |
| `/api/v1/ring/cameras/{cameraId}/events` | GET | Camera events |
| `/api/v1/ring/status` | GET | Ring status |

### Shelly Integration

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/shelly/devices` | POST | Add Shelly device |
| `/api/v1/shelly/devices/{id}` | DELETE | Remove device |
| `/api/v1/shelly/devices` | GET | Get devices |
| `/api/v1/shelly/devices/{id}/status` | GET | Device status |
| `/api/v1/shelly/devices/{id}/control` | POST | Control device |
| `/api/v1/shelly/devices/{id}/energy` | GET | Device energy |
| `/api/v1/shelly/discovery/devices` | GET | Discovered devices |
| `/api/v1/shelly/discovery/start` | POST | Start discovery |
| `/api/v1/shelly/discovery/stop` | POST | Stop discovery |

## Configuration Management

### System Configuration

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/config` | GET | All configuration |
| `/api/v1/config/{key}` | GET | Get config value |
| `/api/v1/config/{key}` | PUT | Set config value |

### Profile Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/profile` | GET | Get profile |
| `/api/v1/profile/password` | PUT | Update password |

### User Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/users` | GET | Get all users |
| `/api/v1/users/{id}` | DELETE | Delete user |

### File Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/screensaver/images` | GET | Screensaver images |
| `/api/v1/screensaver/storage` | GET | Storage info |
| `/api/v1/screensaver/images/upload` | POST | Upload images |
| `/api/v1/screensaver/images/{id}` | DELETE | Delete image |
| `/api/v1/screensaver/images/{filename}` | GET | Get image |

### Event Streaming

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/events/stream` | GET | Event stream (SSE) |
| `/api/v1/events/status` | GET | Event status |

### Mobile Upload

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/upload` | GET | Mobile upload page |

### Kiosk Mode

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/kiosk/status` | GET | Kiosk status |
| `/api/kiosk/screenshot` | POST | Take screenshot |
| `/api/kiosk/restart` | POST | Restart kiosk |
| `/api/kiosk/logs` | GET | Kiosk logs |
| `/api/kiosk/display/status` | GET | Display status |
| `/api/kiosk/display/brightness` | POST | Control brightness |
| `/api/kiosk/display/sleep` | POST | Put to sleep |
| `/api/kiosk/display/wake` | POST | Wake display |
| `/api/kiosk/config` | GET | Kiosk configuration |
| `/api/kiosk/config` | PUT | Update configuration |

### Legacy Endpoints

Many legacy endpoints are maintained for backward compatibility:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/auth/*` | * | Legacy auth endpoints |
| `/api/status` | GET | Legacy status |
| `/api/health` | GET | Legacy health |
| `/api/entities` | GET | Legacy entities |
| `/api/scenes` | GET | Legacy scenes |
| `/api/rooms` | GET | Legacy rooms |
| `/api/config/*` | * | Legacy config |

## SDK Examples

### JavaScript/TypeScript Client

```typescript
interface PMAConfig {
  baseUrl: string;
  token?: string;
  timeout?: number;
}

class PMAClient {
  private baseUrl: string;
  private token?: string;
  private timeout: number;

  constructor(config: PMAConfig) {
    this.baseUrl = config.baseUrl.replace(/\/$/, '');
    this.token = config.token;
    this.timeout = config.timeout || 30000;
  }

  async login(username: string, password: string): Promise<any> {
    const response = await this.request('/auth/login', 'POST', {
      username,
      password
    });
    
    if (response.success) {
      this.token = response.data.token;
    }
    
    return response;
  }

  async getEntities(filters?: {
    type?: string;
    source?: string;
    room_id?: string;
    limit?: number;
    offset?: number;
  }): Promise<any> {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) {
          params.append(key, value.toString());
        }
      });
    }
    
    const url = `/entities${params.toString() ? '?' + params.toString() : ''}`;
    return this.request(url);
  }

  async controlEntity(entityId: string, action: string, parameters?: any): Promise<any> {
    return this.request(`/entities/${entityId}/action`, 'POST', {
      action,
      parameters: parameters || {}
    });
  }

  async getSystemHealth(): Promise<any> {
    return this.request('/system/health/detailed');
  }

  async createAutomation(rule: any): Promise<any> {
    return this.request('/automation/rules', 'POST', rule);
  }

  async chatWithAI(message: string, context?: any): Promise<any> {
    return this.request('/ai/chat', 'POST', {
      message,
      context: context || {}
    });
  }

  private async request(
    endpoint: string, 
    method: string = 'GET', 
    body?: any
  ): Promise<any> {
    const url = `${this.baseUrl}/api/v1${endpoint}`;
    const headers: HeadersInit = {
      'Content-Type': 'application/json',
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const config: RequestInit = {
      method,
      headers,
      signal: AbortSignal.timeout(this.timeout),
    };

    if (body && method !== 'GET') {
      config.body = JSON.stringify(body);
    }

    try {
      const response = await fetch(url, config);
      const data = await response.json();
      
      if (!response.ok) {
        throw new Error(data.error || `HTTP ${response.status}`);
      }
      
      return data;
    } catch (error) {
      console.error('PMA API Error:', error);
      throw error;
    }
  }

  // WebSocket connection
  connectWebSocket(): WebSocket {
    const wsUrl = this.baseUrl.replace(/^http/, 'ws') + '/ws';
    const ws = new WebSocket(wsUrl);
    
    ws.onopen = () => {
      console.log('Connected to PMA WebSocket');
      if (this.token) {
        ws.send(JSON.stringify({
          type: 'auth',
          token: this.token
        }));
      }
    };
    
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      console.log('WebSocket message:', data);
    };
    
    return ws;
  }
}

// Usage example
const pma = new PMAClient({
  baseUrl: 'http://localhost:3001',
  timeout: 30000
});

// Login
await pma.login('admin', 'password');

// Control devices
await pma.controlEntity('light.living_room', 'turn_on', {
  brightness: 255,
  color_temp: 3000
});

// Get system health
const health = await pma.getSystemHealth();
console.log('System health:', health);
```

### Python Client

```python
import requests
import json
import websocket
from typing import Optional, Dict, Any
from datetime import datetime

class PMAClient:
    def __init__(self, base_url: str, token: Optional[str] = None, timeout: int = 30):
        self.base_url = base_url.rstrip('/')
        self.token = token
        self.timeout = timeout
        self.session = requests.Session()
        
    def login(self, username: str, password: str) -> Dict[str, Any]:
        """Login and store the token"""
        response = self.request('/auth/login', 'POST', {
            'username': username,
            'password': password
        })
        
        if response.get('success'):
            self.token = response['data']['token']
            self.session.headers.update({
                'Authorization': f'Bearer {self.token}'
            })
            
        return response
    
    def get_entities(self, **filters) -> Dict[str, Any]:
        """Get entities with optional filters"""
        params = {k: v for k, v in filters.items() if v is not None}
        return self.request('/entities', params=params)
    
    def control_entity(self, entity_id: str, action: str, parameters: Optional[Dict] = None) -> Dict[str, Any]:
        """Control an entity"""
        return self.request(f'/entities/{entity_id}/action', 'POST', {
            'action': action,
            'parameters': parameters or {}
        })
    
    def get_system_health(self) -> Dict[str, Any]:
        """Get detailed system health"""
        return self.request('/system/health/detailed')
    
    def create_automation(self, rule: Dict[str, Any]) -> Dict[str, Any]:
        """Create automation rule"""
        return self.request('/automation/rules', 'POST', rule)
    
    def chat_with_ai(self, message: str, context: Optional[Dict] = None) -> Dict[str, Any]:
        """Chat with AI assistant"""
        return self.request('/ai/chat', 'POST', {
            'message': message,
            'context': context or {}
        })
    
    def get_rooms(self) -> Dict[str, Any]:
        """Get all rooms"""
        return self.request('/rooms')
    
    def create_room(self, name: str, description: str = '', floor: str = '') -> Dict[str, Any]:
        """Create a new room"""
        return self.request('/rooms', 'POST', {
            'name': name,
            'description': description,
            'floor': floor
        })
    
    def request(self, endpoint: str, method: str = 'GET', data: Optional[Dict] = None, params: Optional[Dict] = None) -> Dict[str, Any]:
        """Make API request"""
        url = f"{self.base_url}/api/v1{endpoint}"
        
        headers = {'Content-Type': 'application/json'}
        if self.token:
            headers['Authorization'] = f'Bearer {self.token}'
        
        try:
            if method == 'GET':
                response = requests.get(url, headers=headers, params=params, timeout=self.timeout)
            elif method == 'POST':
                response = requests.post(url, headers=headers, json=data, timeout=self.timeout)
            elif method == 'PUT':
                response = requests.put(url, headers=headers, json=data, timeout=self.timeout)
            elif method == 'DELETE':
                response = requests.delete(url, headers=headers, timeout=self.timeout)
            else:
                raise ValueError(f"Unsupported method: {method}")
            
            response.raise_for_status()
            return response.json()
            
        except requests.RequestException as e:
            print(f"API Request failed: {e}")
            raise
    
    def connect_websocket(self, on_message=None, on_error=None, on_close=None):
        """Connect to WebSocket"""
        ws_url = self.base_url.replace('http', 'ws') + '/ws'
        
        def on_open(ws):
            print("Connected to PMA WebSocket")
            if self.token:
                ws.send(json.dumps({
                    'type': 'auth',
                    'token': self.token
                }))
        
        ws = websocket.WebSocketApp(
            ws_url,
            on_open=on_open,
            on_message=on_message or self._default_on_message,
            on_error=on_error or self._default_on_error,
            on_close=on_close or self._default_on_close
        )
        
        return ws
    
    def _default_on_message(self, ws, message):
        data = json.loads(message)
        print(f"WebSocket message: {data}")
    
    def _default_on_error(self, ws, error):
        print(f"WebSocket error: {error}")
    
    def _default_on_close(self, ws, close_status_code, close_msg):
        print("WebSocket connection closed")

# Usage example
if __name__ == "__main__":
    # Initialize client
    pma = PMAClient('http://localhost:3001')
    
    # Login
    login_result = pma.login('admin', 'password')
    if login_result.get('success'):
        print("Logged in successfully")
        
        # Get all lights
        lights = pma.get_entities(type='light')
        print(f"Found {len(lights.get('data', []))} lights")
        
        # Turn on living room light
        result = pma.control_entity('light.living_room', 'turn_on', {
            'brightness': 255,
            'color_temp': 3000
        })
        print(f"Light control result: {result}")
        
        # Get system health
        health = pma.get_system_health()
        print(f"System health: {health.get('data', {}).get('status', 'Unknown')}")
        
        # Chat with AI
        ai_response = pma.chat_with_ai("What's the current status of my home?")
        print(f"AI Response: {ai_response.get('data', {}).get('response', 'No response')}")
    
    else:
        print("Login failed")
```

### Go Client

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "time"
)

type PMAClient struct {
    BaseURL    string
    Token      string
    HTTPClient *http.Client
}

type APIResponse struct {
    Success   bool                   `json:"success"`
    Data      interface{}           `json:"data"`
    Error     string                `json:"error,omitempty"`
    Message   string                `json:"message,omitempty"`
    Timestamp time.Time             `json:"timestamp"`
    RequestID string                `json:"request_id,omitempty"`
}

func NewPMAClient(baseURL string) *PMAClient {
    return &PMAClient{
        BaseURL: baseURL,
        HTTPClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (c *PMAClient) Login(username, password string) error {
    loginData := map[string]string{
        "username": username,
        "password": password,
    }
    
    var response APIResponse
    err := c.request("POST", "/auth/login", loginData, &response)
    if err != nil {
        return err
    }
    
    if !response.Success {
        return fmt.Errorf("login failed: %s", response.Error)
    }
    
    if data, ok := response.Data.(map[string]interface{}); ok {
        if token, ok := data["token"].(string); ok {
            c.Token = token
        }
    }
    
    return nil
}

func (c *PMAClient) GetEntities(filters map[string]string) (interface{}, error) {
    var response APIResponse
    err := c.requestWithParams("GET", "/entities", nil, filters, &response)
    if err != nil {
        return nil, err
    }
    
    if !response.Success {
        return nil, fmt.Errorf("failed to get entities: %s", response.Error)
    }
    
    return response.Data, nil
}

func (c *PMAClient) ControlEntity(entityID, action string, parameters map[string]interface{}) (interface{}, error) {
    controlData := map[string]interface{}{
        "action":     action,
        "parameters": parameters,
    }
    
    var response APIResponse
    err := c.request("POST", fmt.Sprintf("/entities/%s/action", entityID), controlData, &response)
    if err != nil {
        return nil, err
    }
    
    if !response.Success {
        return nil, fmt.Errorf("failed to control entity: %s", response.Error)
    }
    
    return response.Data, nil
}

func (c *PMAClient) GetSystemHealth() (interface{}, error) {
    var response APIResponse
    err := c.request("GET", "/system/health/detailed", nil, &response)
    if err != nil {
        return nil, err
    }
    
    if !response.Success {
        return nil, fmt.Errorf("failed to get system health: %s", response.Error)
    }
    
    return response.Data, nil
}

func (c *PMAClient) ChatWithAI(message string, context map[string]interface{}) (interface{}, error) {
    chatData := map[string]interface{}{
        "message": message,
        "context": context,
    }
    
    var response APIResponse
    err := c.request("POST", "/ai/chat", chatData, &response)
    if err != nil {
        return nil, err
    }
    
    if !response.Success {
        return nil, fmt.Errorf("AI chat failed: %s", response.Error)
    }
    
    return response.Data, nil
}

func (c *PMAClient) request(method, endpoint string, body interface{}, result *APIResponse) error {
    return c.requestWithParams(method, endpoint, body, nil, result)
}

func (c *PMAClient) requestWithParams(method, endpoint string, body interface{}, params map[string]string, result *APIResponse) error {
    fullURL := c.BaseURL + "/api/v1" + endpoint
    
    // Add query parameters
    if params != nil && len(params) > 0 {
        u, err := url.Parse(fullURL)
        if err != nil {
            return err
        }
        q := u.Query()
        for k, v := range params {
            q.Set(k, v)
        }
        u.RawQuery = q.Encode()
        fullURL = u.String()
    }
    
    var req *http.Request
    var err error
    
    if body != nil {
        jsonData, err := json.Marshal(body)
        if err != nil {
            return err
        }
        req, err = http.NewRequest(method, fullURL, bytes.NewBuffer(jsonData))
        if err != nil {
            return err
        }
        req.Header.Set("Content-Type", "application/json")
    } else {
        req, err = http.NewRequest(method, fullURL, nil)
        if err != nil {
            return err
        }
    }
    
    if c.Token != "" {
        req.Header.Set("Authorization", "Bearer "+c.Token)
    }
    
    resp, err := c.HTTPClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return json.NewDecoder(resp.Body).Decode(result)
}

func main() {
    // Example usage
    client := NewPMAClient("http://localhost:3001")
    
    // Login
    err := client.Login("admin", "password")
    if err != nil {
        fmt.Printf("Login failed: %v\n", err)
        return
    }
    fmt.Println("Logged in successfully")
    
    // Get entities
    entities, err := client.GetEntities(map[string]string{
        "type": "light",
    })
    if err != nil {
        fmt.Printf("Failed to get entities: %v\n", err)
    } else {
        fmt.Printf("Entities: %+v\n", entities)
    }
    
    // Control entity
    result, err := client.ControlEntity("light.living_room", "turn_on", map[string]interface{}{
        "brightness": 255,
        "color_temp": 3000,
    })
    if err != nil {
        fmt.Printf("Failed to control entity: %v\n", err)
    } else {
        fmt.Printf("Control result: %+v\n", result)
    }
    
    // Get system health
    health, err := client.GetSystemHealth()
    if err != nil {
        fmt.Printf("Failed to get health: %v\n", err)
    } else {
        fmt.Printf("System health: %+v\n", health)
    }
    
    // Chat with AI
    aiResponse, err := client.ChatWithAI("Turn on the bedroom lights", map[string]interface{}{
        "room": "bedroom",
    })
    if err != nil {
        fmt.Printf("AI chat failed: %v\n", err)
    } else {
        fmt.Printf("AI response: %+v\n", aiResponse)
    }
}
```

## Rate Limiting & Best Practices

### Rate Limits
- **Default**: 100 requests per minute per IP
- **Authenticated**: 1000 requests per minute per user
- **WebSocket**: No rate limit on messages
- **File Upload**: 10 uploads per minute

### Best Practices

1. **Authentication**: Always include JWT token for authenticated endpoints
2. **Error Handling**: Check the `success` field in responses
3. **Pagination**: Use `limit` and `offset` for large datasets
4. **WebSocket**: Use WebSocket for real-time updates instead of polling
5. **Caching**: Implement client-side caching for frequently accessed data
6. **Timeouts**: Set appropriate timeouts for your HTTP client
7. **Retry Logic**: Implement exponential backoff for failed requests

### Common Patterns

#### Polling vs WebSocket
```javascript
// ❌ Don't poll frequently
setInterval(() => {
    pma.getEntities();
}, 1000); // Every second - bad!

// ✅ Use WebSocket for real-time updates
const ws = pma.connectWebSocket();
ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === 'entity_state_changed') {
        updateUI(data.entity);
    }
};
```

#### Batch Operations
```javascript
// ❌ Don't make many individual requests
for (const light of lights) {
    await pma.controlEntity(light.id, 'turn_on');
}

// ✅ Use scene activation for multiple entities
await pma.request('/scenes/evening_lights/activate', 'POST');
```

This comprehensive API reference covers all 500+ endpoints in the PMA Backend Go system. For the most up-to-date information, always refer to the `/health` endpoint and the OpenAPI documentation available at `/docs` when the server is running.