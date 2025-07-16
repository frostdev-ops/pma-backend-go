# Entity Management System

## Overview

The Entity Management System is the core component of the PMA Backend responsible for handling Home Assistant entities with full CRUD operations, state management, and real-time updates. This system provides a robust foundation for home automation management.

## Architecture

### Components

1. **Entity Service** (`internal/core/entities/service.go`)
   - Business logic layer for entity operations
   - Handles state management and room assignment
   - Provides domain filtering and relationship management

2. **Entity Handlers** (`internal/api/handlers/entities.go`)
   - HTTP request handlers for RESTful API endpoints
   - Input validation and error handling
   - Request/response transformation

3. **Entity Repository** (`internal/database/sqlite/entity_repository.go`)
   - Data access layer for entity operations
   - SQLite-specific implementation

4. **Entity Model** (`internal/database/models/models.go`)
   - Entity data structure definition
   - JSON serialization support

## API Endpoints

All entity endpoints require authentication and are prefixed with `/api/v1/entities`.

### Get All Entities
```
GET /api/v1/entities
```

**Query Parameters:**
- `include_room` (boolean): Include room information for each entity
- `domain` (string): Filter entities by domain (e.g., "light", "sensor")

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "entity_id": "light.living_room",
      "friendly_name": "Living Room Light",
      "domain": "light",
      "state": "on",
      "attributes": {"brightness": 255},
      "last_updated": "2024-01-15T10:30:00Z",
      "room_id": 1,
      "room": {
        "id": 1,
        "name": "Living Room"
      }
    }
  ],
  "meta": {
    "count": 1,
    "include_room": true
  }
}
```

### Get Specific Entity
```
GET /api/v1/entities/:id
```

**Query Parameters:**
- `include_room` (boolean): Include room information

**Response:**
```json
{
  "success": true,
  "data": {
    "entity_id": "light.living_room",
    "friendly_name": "Living Room Light",
    "domain": "light",
    "state": "on",
    "attributes": {"brightness": 255},
    "last_updated": "2024-01-15T10:30:00Z",
    "room_id": 1
  }
}
```

### Create or Update Entity
```
POST /api/v1/entities
```

**Request Body:**
```json
{
  "entity_id": "light.kitchen",
  "friendly_name": "Kitchen Light",
  "domain": "light",
  "state": "off",
  "attributes": {
    "brightness": 0,
    "color_mode": "onoff"
  },
  "room_id": 2
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Entity saved successfully",
    "entity_id": "light.kitchen"
  }
}
```

### Update Entity State
```
PUT /api/v1/entities/:id/state
```

**Request Body:**
```json
{
  "state": "on",
  "attributes": {
    "brightness": 255,
    "color_mode": "brightness"
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Entity state updated successfully",
    "entity_id": "light.kitchen",
    "state": "on"
  }
}
```

### Assign Entity to Room
```
PUT /api/v1/entities/:id/room
```

**Request Body:**
```json
{
  "room_id": 3
}
```

To remove room assignment, send `null`:
```json
{
  "room_id": null
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Entity assigned to room successfully",
    "entity_id": "light.kitchen",
    "room_id": 3
  }
}
```

### Delete Entity
```
DELETE /api/v1/entities/:id
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Entity deleted successfully",
    "entity_id": "light.kitchen"
  }
}
```

## Features

### 1. State Management
- Real-time entity state updates
- Attribute management with JSON storage
- Automatic timestamp tracking

### 2. Room Assignment
- Assign entities to rooms for organization
- Support for unassigned entities
- Foreign key relationships with proper cascading

### 3. Domain Filtering
- Filter entities by Home Assistant domain
- Useful for grouping lights, sensors, switches, etc.

### 4. Entity Relationships
- Optional room information inclusion
- Efficient relationship loading

### 5. Comprehensive Logging
- Structured logging with logrus
- Operation tracking and error reporting
- Performance monitoring

## Usage Examples

### 1. Get All Light Entities
```bash
curl -H "Authorization: Bearer your-token" \
  "http://localhost:3001/api/v1/entities?domain=light"
```

### 2. Create a New Sensor
```bash
curl -X POST -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "entity_id": "sensor.temperature",
    "friendly_name": "Temperature Sensor",
    "domain": "sensor",
    "state": "22.5",
    "attributes": {"unit_of_measurement": "°C"}
  }' \
  "http://localhost:3001/api/v1/entities"
```

### 3. Update Light State
```bash
curl -X PUT -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "on",
    "attributes": {"brightness": 128}
  }' \
  "http://localhost:3001/api/v1/entities/light.living_room/state"
```

## Testing

### Automated Testing
Run the comprehensive test suite:
```bash
python3 test_entity_endpoints.py
```

This script tests all endpoints and provides detailed feedback on functionality.

### Manual Testing
1. Start the server: `make run` or `./bin/pma-server`
2. Use curl or Postman to test endpoints
3. Check logs for operation details

## Error Handling

The system provides comprehensive error handling:

- **400 Bad Request**: Invalid request body or parameters
- **401 Unauthorized**: Missing or invalid authentication
- **404 Not Found**: Entity or room not found
- **500 Internal Server Error**: Database or server errors

All errors include descriptive messages and are logged for debugging.

## Database Schema

Entities are stored in the `entities` table:

```sql
CREATE TABLE entities (
    entity_id TEXT PRIMARY KEY,
    friendly_name TEXT,
    domain TEXT NOT NULL,
    state TEXT,
    attributes TEXT, -- JSON
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
    room_id INTEGER,
    FOREIGN KEY (room_id) REFERENCES rooms(id) ON DELETE SET NULL
);
```

## Performance Considerations

- Database indexes on `room_id` and `domain` for efficient filtering
- Connection pooling with configurable limits
- Timeout handling for all operations
- Structured logging for performance monitoring

## Security

- JWT-based authentication required for all endpoints
- Input validation on all request bodies
- SQL injection protection through parameterized queries
- CORS middleware for web client support

## Future Enhancements

1. **WebSocket Support**: Real-time entity state notifications
2. **Bulk Operations**: Batch entity updates
3. **Entity History**: Track state changes over time
4. **Advanced Filtering**: More sophisticated query capabilities
5. **Entity Templates**: Predefined entity configurations
6. **Validation Rules**: Custom validation for entity attributes 