# Cover Control API Documentation

## Overview

The Cover Control API provides comprehensive endpoints for managing window covers, blinds, shades, and similar motorized devices. This API integrates with Home Assistant to control various types of covers with position control, tilt adjustment, and automation capabilities.

## Base URL

All cover endpoints are available under the protected API routes (authentication required):

```
/api/v1/entities/:id/cover/*
/api/v1/covers/*
```

## Authentication

All cover control endpoints require JWT authentication. Include the token in the Authorization header:

```
Authorization: Bearer <jwt_token>
```

## Endpoints

### Individual Cover Control

#### Open Cover

**POST** `/api/v1/entities/:id/cover/open`

Opens a cover to 100% position.

**Parameters:**
- `id` (path): Entity ID of the cover

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Cover opened successfully",
    "entity_id": "cover.living_room_blinds",
    "state": "opening",
    "position": 100
  }
}
```

#### Close Cover

**POST** `/api/v1/entities/:id/cover/close`

Closes a cover to 0% position.

**Parameters:**
- `id` (path): Entity ID of the cover

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Cover closed successfully",
    "entity_id": "cover.living_room_blinds",
    "state": "closing",
    "position": 0
  }
}
```

#### Stop Cover

**POST** `/api/v1/entities/:id/cover/stop`

Stops the current cover movement.

**Parameters:**
- `id` (path): Entity ID of the cover

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Cover stopped successfully",
    "entity_id": "cover.living_room_blinds",
    "state": "stopped"
  }
}
```

**Error Responses:**
- `400 Bad Request`: Cover does not support stop operation

#### Set Cover Position

**PUT** `/api/v1/entities/:id/cover/position`

Sets a specific position for the cover.

**Parameters:**
- `id` (path): Entity ID of the cover

**Request Body:**
```json
{
  "position": 75
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Cover position set successfully",
    "entity_id": "cover.living_room_blinds",
    "state": "moving",
    "position": 75
  }
}
```

**Validation:**
- `position`: Integer between 0-100 (required)

**Error Responses:**
- `400 Bad Request`: Invalid position or cover doesn't support position control

#### Set Cover Tilt

**PUT** `/api/v1/entities/:id/cover/tilt`

Sets the tilt position for venetian blinds.

**Parameters:**
- `id` (path): Entity ID of the cover

**Request Body:**
```json
{
  "tilt_position": 45
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Cover tilt set successfully",
    "entity_id": "cover.venetian_blinds",
    "tilt_position": 45
  }
}
```

**Validation:**
- `tilt_position`: Integer between 0-100 (required)

**Error Responses:**
- `400 Bad Request`: Invalid tilt position or cover doesn't support tilt control

#### Get Cover Status

**GET** `/api/v1/entities/:id/cover/status`

Retrieves the current status of a cover.

**Parameters:**
- `id` (path): Entity ID of the cover

**Response:**
```json
{
  "success": true,
  "data": {
    "entity_id": "cover.living_room_blinds",
    "status": {
      "state": "open",
      "position": 100,
      "tilt_position": 50,
      "features": ["position", "tilt", "stop", "open_close"]
    }
  }
}
```

#### Set Cover Preset

**PUT** `/api/v1/entities/:id/cover/preset`

Sets a cover to a predefined preset position.

**Parameters:**
- `id` (path): Entity ID of the cover

**Request Body:**
```json
{
  "preset": "privacy"
}
```

**Available Presets:**
- `closed` (0%)
- `privacy` (25%)
- `half_open` (50%)
- `ventilate` (75%)
- `open` (100%)

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Cover preset set successfully",
    "entity_id": "cover.living_room_blinds",
    "preset": "privacy",
    "position": 25
  }
}
```

### Group Operations

#### Operate Covers in Group

**POST** `/api/v1/covers/group-operation`

Operates multiple covers simultaneously.

**Request Body:**
```json
{
  "entity_ids": [
    "cover.living_room_blinds",
    "cover.bedroom_blinds",
    "cover.kitchen_shades"
  ],
  "operation": "set_position",
  "position": 50
}
```

**Supported Operations:**
- `open`: Open all covers
- `close`: Close all covers
- `stop`: Stop all covers
- `set_position`: Set specific position (requires `position` field)
- `set_tilt`: Set tilt position (requires `tilt_position` field)

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Group operation completed successfully",
    "operation": "set_position",
    "entity_ids": [
      "cover.living_room_blinds",
      "cover.bedroom_blinds",
      "cover.kitchen_shades"
    ]
  }
}
```

## Cover Capabilities

Different covers support different features. The API automatically detects and validates capabilities:

### Position Control
- **Supported**: Covers with motor control for specific positioning
- **Feature**: `position`
- **Range**: 0-100 (0 = fully closed, 100 = fully open)

### Tilt Control
- **Supported**: Venetian blinds and similar covers with slat adjustment
- **Feature**: `tilt`
- **Range**: 0-100 (0 = horizontal, 100 = vertical)

### Stop Control
- **Supported**: Covers that can be stopped during movement
- **Feature**: `stop`
- **Usage**: Emergency stop or precise positioning

### Open/Close Control
- **Supported**: All covers (basic functionality)
- **Feature**: `open_close`
- **Usage**: Simple open/close without position control

## Error Handling

### Common Error Responses

#### 404 Not Found
```json
{
  "success": false,
  "error": "Cover not found"
}
```

#### 400 Bad Request
```json
{
  "success": false,
  "error": "Cover does not support position control"
}
```

#### 500 Internal Server Error
```json
{
  "success": false,
  "error": "Failed to connect to Home Assistant"
}
```

### Validation Errors

Position and tilt values must be between 0-100:

```json
{
  "success": false,
  "error": "Invalid request body"
}
```

## Integration Notes

### Home Assistant Services

The API maps to the following Home Assistant services:

- `cover.open_cover` - Open cover
- `cover.close_cover` - Close cover  
- `cover.stop_cover` - Stop cover
- `cover.set_cover_position` - Set position
- `cover.set_cover_tilt_position` - Set tilt

### WebSocket Updates

Cover state changes are automatically broadcasted via WebSocket when the Home Assistant sync service receives updates. Frontend applications can subscribe to entity updates to receive real-time status changes.

### Supported Cover Types

- **Window Blinds**: Venetian blinds with tilt control
- **Window Shades**: Roller shades with position control
- **Curtains**: Motorized curtains with position control
- **Garage Doors**: Large covers with open/close/stop
- **Awnings**: Outdoor covers with position control

## Usage Examples

### JavaScript/TypeScript Frontend

```typescript
// Open a cover
const response = await fetch('/api/v1/entities/cover.living_room_blinds/cover/open', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  }
});

// Set position
const response = await fetch('/api/v1/entities/cover.living_room_blinds/cover/position', {
  method: 'PUT',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({ position: 75 })
});

// Group operation
const response = await fetch('/api/v1/covers/group-operation', {
  method: 'POST',
  headers: {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    entity_ids: ['cover.blind1', 'cover.blind2'],
    operation: 'open'
  })
});
```

### Python Script

```python
import requests

# Configuration
API_BASE = "http://localhost:3001/api/v1"
TOKEN = "your_jwt_token"
HEADERS = {
    "Authorization": f"Bearer {TOKEN}",
    "Content-Type": "application/json"
}

# Open cover
response = requests.post(
    f"{API_BASE}/entities/cover.living_room_blinds/cover/open",
    headers=HEADERS
)

# Set preset
response = requests.put(
    f"{API_BASE}/entities/cover.living_room_blinds/cover/preset",
    headers=HEADERS,
    json={"preset": "privacy"}
)
```

## Security Considerations

- All endpoints require valid JWT authentication
- Cover operations are logged for audit purposes
- Rate limiting is applied to prevent abuse
- Input validation prevents invalid position/tilt values
- Entity validation ensures only covers can be controlled

## Performance Notes

- Cover operations may take several seconds to complete
- Group operations are performed simultaneously for efficiency
- WebSocket updates provide real-time status without polling
- Local state is updated optimistically with HA synchronization 