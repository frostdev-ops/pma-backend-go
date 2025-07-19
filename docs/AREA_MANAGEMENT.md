# Area Management System

## Overview

The Area Management System provides comprehensive area organization capabilities that extend beyond basic Home Assistant room management. It offers enhanced mapping, analytics, synchronization, and hierarchical organization features.

## Architecture

### Components

1. **Database Layer**: Complete schema with tables for areas, mappings, settings, analytics, and sync tracking
2. **Repository Layer**: Full CRUD operations with complex queries and analytics support
3. **Service Layer**: Business logic with validation, analytics engine, and synchronization
4. **API Layer**: RESTful endpoints for all area management operations
5. **Integration Layer**: Synchronization with external systems like Home Assistant

### Data Models

#### Core Models

- **Area**: Enhanced area with hierarchy, metadata, and type classification
- **AreaMapping**: Relationships between PMA areas and external systems
- **AreaSetting**: Configuration and preferences (global and area-specific)
- **AreaAnalytic**: Metrics and statistics for areas
- **AreaSyncLog**: Synchronization history and status tracking
- **RoomAreaAssignment**: Room-to-area relationships with confidence scoring

#### Area Types

- `room`: Standard room areas
- `zone`: Logical zones spanning multiple rooms
- `building`: Building-level areas
- `floor`: Floor-level organization
- `outdoor`: Outdoor areas
- `utility`: Utility spaces

## API Endpoints

### Area Management

#### Areas
- `GET /api/v1/area-management/areas` - List all areas
  - Query params: `include_inactive`, `hierarchy`
- `POST /api/v1/area-management/areas` - Create new area
- `GET /api/v1/area-management/areas/:id` - Get specific area
  - Query params: `include_children`
- `PUT /api/v1/area-management/areas/:id` - Update area
- `DELETE /api/v1/area-management/areas/:id` - Delete area

#### Area Mappings
- `GET /api/v1/area-management/mappings` - List area mappings
  - Query params: `external_system`
- `POST /api/v1/area-management/mappings` - Create area mapping
- `PUT /api/v1/area-management/mappings/:id` - Update mapping
- `DELETE /api/v1/area-management/mappings/:id` - Delete mapping

#### Enhanced Room Management
- `GET /api/v1/area-management/rooms` - Enhanced room listing
  - Query params: `include_entities`, `area_id`

#### Room-Area Assignments
- `POST /api/v1/area-management/rooms/:room_id/assign` - Assign room to area
- `GET /api/v1/area-management/rooms/:room_id/assignments` - Get room assignments
- `GET /api/v1/area-management/areas/:area_id/assignments` - Get area assignments

#### Synchronization
- `POST /api/v1/area-management/sync` - Trigger synchronization
- `GET /api/v1/area-management/sync/status` - Get sync status
- `GET /api/v1/area-management/sync/history` - Get sync history

#### Analytics and Status
- `GET /api/v1/area-management/status` - Overall system status
- `GET /api/v1/area-management/analytics` - Area analytics
- `GET /api/v1/area-management/analytics/summary` - Analytics summary

#### Settings
- `GET /api/v1/area-management/settings` - Get settings
  - Query params: `area_id`
- `PUT /api/v1/area-management/settings` - Update settings

## Features

### Hierarchical Organization

The system supports multi-level area hierarchies:

```
Building
├── Floor 1
│   ├── Living Room
│   ├── Kitchen
│   └── Bathroom
└── Floor 2
    ├── Master Bedroom
    ├── Guest Bedroom
    └── Office
```

### External System Integration

#### Home Assistant Integration
- Bidirectional synchronization with HA areas
- Automatic entity assignment based on area mappings
- Conflict resolution with priority settings
- Incremental and full sync modes

### Analytics Engine

The system provides comprehensive analytics:

#### Area Metrics
- Entity count per area
- Device distribution
- Activity levels
- Energy usage (when available)
- Health scoring

#### System Health
- Total areas vs. mapped areas
- Room assignment coverage
- Entity organization status
- Sync status and history

### Configuration Management

#### Global Settings
- `sync_enabled`: Enable/disable synchronization
- `sync_interval_minutes`: Automatic sync interval
- `auto_create_areas`: Automatically create areas from external systems
- `default_area_type`: Default type for new areas
- `max_hierarchy_depth`: Maximum nesting level
- `analytics_retention_days`: How long to keep analytics data
- `conflict_resolution`: How to handle sync conflicts

#### Area-Specific Settings
Per-area configuration overrides for:
- Sync behavior
- Analytics collection
- Display preferences
- Automation triggers

## Database Schema

### Areas Table
```sql
CREATE TABLE areas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    area_id TEXT UNIQUE,
    description TEXT,
    icon TEXT,
    floor_level INTEGER DEFAULT 0,
    parent_area_id INTEGER REFERENCES areas(id),
    color TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    area_type TEXT DEFAULT 'room',
    metadata TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Area Mappings Table
```sql
CREATE TABLE area_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pma_area_id INTEGER NOT NULL REFERENCES areas(id),
    external_area_id TEXT NOT NULL,
    external_system TEXT NOT NULL DEFAULT 'homeassistant',
    mapping_type TEXT DEFAULT 'direct',
    auto_sync BOOLEAN DEFAULT TRUE,
    sync_priority INTEGER DEFAULT 1,
    last_synced DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Additional Tables
- `area_settings`: Configuration storage
- `area_analytics`: Metrics and statistics
- `area_sync_log`: Synchronization history
- `room_area_assignments`: Room-to-area relationships

## Usage Examples

### Creating an Area Hierarchy

```bash
# Create a building
curl -X POST /api/v1/area-management/areas \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Main House",
    "area_type": "building",
    "description": "Primary residence"
  }'

# Create a floor
curl -X POST /api/v1/area-management/areas \
  -H "Content-Type: application/json" \
  -d '{
    "name": "First Floor",
    "area_type": "floor",
    "parent_area_id": 1,
    "floor_level": 1
  }'

# Create rooms
curl -X POST /api/v1/area-management/areas \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Living Room",
    "area_type": "room",
    "parent_area_id": 2,
    "icon": "sofa",
    "color": "#4CAF50"
  }'
```

### Setting Up Home Assistant Mapping

```bash
# Create mapping to HA area
curl -X POST /api/v1/area-management/mappings \
  -H "Content-Type: application/json" \
  -d '{
    "pma_area_id": 3,
    "external_area_id": "living_room",
    "external_system": "homeassistant",
    "auto_sync": true,
    "sync_priority": 1
  }'
```

### Triggering Synchronization

```bash
# Manual sync with Home Assistant
curl -X POST /api/v1/area-management/sync \
  -H "Content-Type: application/json" \
  -d '{
    "sync_type": "manual",
    "external_system": "homeassistant",
    "force_sync": true
  }'
```

### Getting Analytics

```bash
# Get area analytics summary
curl /api/v1/area-management/analytics/summary

# Get specific area analytics
curl "/api/v1/area-management/analytics?area_ids=1,2,3&time_period=daily"
```

## Error Handling

The system provides comprehensive error handling:

- **Validation Errors**: Invalid input data
- **Conflict Errors**: Circular references, duplicate mappings
- **Sync Errors**: External system communication issues
- **Not Found Errors**: Missing areas, mappings, or assignments

## Performance Considerations

### Indexing
All tables include appropriate indexes for:
- Primary key lookups
- Foreign key relationships
- Common query patterns
- Analytics queries

### Caching
- Area hierarchy is cached for performance
- Analytics summaries are pre-computed
- External system data is cached with TTL

### Batch Operations
- Bulk area creation/updates
- Batch synchronization
- Mass assignment operations

## Migration

The system includes migration 015 which:
1. Creates all necessary tables
2. Sets up indexes and constraints
3. Inserts default global settings
4. Creates database triggers for timestamps

To apply the migration:
```sql
-- Migration will be applied automatically on startup
-- Manual application: Run 015_area_management.up.sql
```

## Integration Points

### Frontend Integration
The frontend can use these endpoints to:
- Display area hierarchies in tree views
- Show analytics dashboards
- Manage area-to-room assignments
- Configure synchronization settings

### Automation Integration
Areas can be used in automation rules:
- Area-based triggers
- Zone-level automation
- Floor-wide actions
- Building-level policies

### Home Assistant Integration
- Automatic entity organization
- Bidirectional area sync
- Device assignment propagation
- State synchronization

## Security

### Authentication
All endpoints require valid JWT authentication.

### Authorization
- Admin users can manage all areas
- Regular users can view areas
- Kiosk tokens have limited access

### Data Protection
- Sensitive sync data is encrypted
- Analytics data is anonymized where possible
- Audit trails for all changes

## Monitoring

### Health Checks
- Database connectivity
- External system connectivity
- Sync status monitoring
- Performance metrics

### Logging
- All operations are logged
- Sync activities are tracked
- Error conditions are recorded
- Performance metrics are collected

## Future Enhancements

### Planned Features
1. **Visual Area Editor**: Drag-and-drop area management
2. **Floor Plan Integration**: Visual area representation
3. **Advanced Analytics**: Machine learning insights
4. **Multi-System Sync**: Support for OpenHAB, Domoticz
5. **Area Templates**: Pre-configured area setups
6. **Geofencing**: Location-based area triggers 