# Test Endpoints Documentation

This document describes the test and mock endpoints available in the PMA backend system for development, testing, and diagnostics.

## Overview

The test endpoints provide comprehensive testing and development capabilities including:
- Mock entity generation and management
- Connection testing for all integrations
- System health diagnostics
- Performance testing
- Development data management

**Important**: Test endpoints are automatically disabled in production mode and only available when `server.mode` is set to `development`.

## Authentication

All test endpoints require authentication and are located under `/api/v1/test/`. You must include a valid JWT token in the Authorization header:

```bash
Authorization: Bearer <your-jwt-token>
```

## Endpoint Reference

### Test Configuration and Status

#### GET /api/v1/test/endpoint-status
Get the current status of test endpoints.

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Test endpoint status",
    "status": "enabled",
    "server_mode": "development",
    "endpoints_enabled": true,
    "checked_at": "2024-01-15T10:30:00Z"
  }
}
```

#### GET /api/v1/test/config
Get test configuration settings.

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Test configuration retrieved",
    "config": {
      "mock_entities_enabled": true,
      "mock_data_persistence": false,
      "test_endpoints_enabled": true,
      "diagnostics_enabled": true,
      "performance_tests_enabled": true
    },
    "server_mode": "development",
    "retrieved_at": "2024-01-15T10:30:00Z"
  }
}
```

### Mock Entity Management

#### POST /api/v1/test/mock-entities
Generate mock entities for testing.

**Request Body:**
```json
{
  "count": 25,
  "entity_types": ["light", "switch", "sensor", "climate"],
  "reset": true
}
```

**Parameters:**
- `count` (required): Number of entities to generate (1-100)
- `entity_types` (optional): Array of entity types to generate
- `reset` (optional): Whether to clear existing mock data first

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Mock entities generated successfully",
    "count": 25,
    "entities": [
      {
        "id": "light.test_light_0",
        "name": "Living Room Ceiling Light",
        "domain": "light",
        "state": "on",
        "attributes": {
          "brightness": 200,
          "color_mode": "brightness",
          "color_temp": 350,
          "friendly_name": "Living Room Ceiling Light",
          "supported_features": 3
        },
        "last_changed": "2024-01-15T09:45:00Z",
        "last_updated": "2024-01-15T10:30:00Z",
        "entity_type": "light"
      }
    ],
    "generated_at": "2024-01-15T10:30:00Z",
    "entity_types": ["light", "switch", "sensor", "climate"]
  }
}
```

#### GET /api/v1/test/mock-entities
Retrieve all existing mock entities.

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Mock entities retrieved successfully",
    "entities": {
      "light.test_light_0": { ... },
      "switch.test_switch_1": { ... }
    },
    "count": 25,
    "retrieved_at": "2024-01-15T10:30:00Z"
  }
}
```

#### PUT /api/v1/test/mock-entities/:id
Update a specific mock entity's state.

**Request Body:**
```json
{
  "state": "off",
  "attributes": {
    "brightness": 0
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Mock entity updated successfully",
    "entity_id": "light.test_light_0",
    "state": "off",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

#### DELETE /api/v1/test/mock-entities/:id
Delete a specific mock entity.

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Mock entity deleted successfully",
    "entity_id": "light.test_light_0",
    "deleted_at": "2024-01-15T10:30:00Z"
  }
}
```

### Connection Testing

#### POST /api/v1/test/connections
Test all system connections and integrations.

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Connection tests completed",
    "results": {
      "database": {
        "service": "database",
        "status": "healthy",
        "response_time": 5000000,
        "details": {
          "version": "3.46.0",
          "driver": "sqlite3"
        }
      },
      "home_assistant": {
        "service": "home_assistant",
        "status": "healthy",
        "response_time": 150000000,
        "details": {
          "version": "2024.1.0",
          "components": ["light", "switch", "sensor"]
        }
      },
      "pma_router": {
        "service": "pma_router",
        "status": "healthy",
        "response_time": 80000000,
        "details": {
          "version": "1.0.0",
          "uptime": "24h30m"
        }
      }
    },
    "summary": {
      "healthy": 6,
      "unhealthy": 1,
      "disabled": 0,
      "not_configured": 0
    },
    "tested_at": "2024-01-15T10:30:00Z",
    "total_tests": 7
  }
}
```

#### POST /api/v1/test/router
Test PMA Router connectivity specifically.

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Router connectivity test completed",
    "router_status": {
      "service": "pma_router",
      "status": "healthy",
      "response_time": 80000000,
      "details": {
        "version": "1.0.0",
        "interfaces": ["eth0", "eth1"],
        "routes": 12
      }
    },
    "network_status": {
      "service": "network",
      "status": "healthy",
      "response_time": 10000000,
      "details": {
        "interfaces": [
          {
            "name": "eth0",
            "addresses": ["192.168.10.247/24"],
            "mtu": 1500
          }
        ],
        "count": 2
      }
    },
    "tested_at": "2024-01-15T10:30:00Z"
  }
}
```

### System Diagnostics

#### GET /api/v1/test/system-health
Perform comprehensive system health checks.

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "System health check completed",
    "overall_status": "healthy",
    "results": {
      "system_resources": {
        "service": "system_resources",
        "status": "healthy",
        "timestamp": "2024-01-15T10:30:00Z",
        "details": {
          "memory": {
            "alloc_mb": 45.2,
            "total_alloc_mb": 123.8,
            "sys_mb": 67.1,
            "gc_cycles": 15
          },
          "goroutines": 42,
          "cpu_cores": 4
        }
      },
      "database": {
        "service": "database",
        "status": "healthy",
        "timestamp": "2024-01-15T10:30:00Z",
        "details": {
          "entity_count": 79,
          "query_time_ms": 2,
          "database_size_mb": 8.5
        }
      },
      "configuration": {
        "service": "configuration",
        "status": "warning",
        "timestamp": "2024-01-15T10:30:00Z",
        "details": {
          "issues": ["Default JWT secret in use"],
          "server_mode": "development",
          "server_port": 3001,
          "log_level": "info"
        }
      }
    },
    "summary": {
      "warnings": 1,
      "errors": 0,
      "checks": 4
    },
    "checked_at": "2024-01-15T10:30:00Z"
  }
}
```

### Integration Testing

#### POST /api/v1/test/integrations
Test specific integrations.

**Request Body:**
```json
{
  "integrations": ["home_assistant", "ring", "ollama"],
  "timeout": 30
}
```

**Parameters:**
- `integrations` (optional): Array of specific integrations to test. If empty, tests all.
- `timeout` (optional): Timeout in seconds (default: 30)

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Integration tests completed",
    "results": {
      "home_assistant": {
        "service": "home_assistant",
        "status": "healthy",
        "response_time": 150000000
      },
      "ring": {
        "service": "ring",
        "status": "not_configured",
        "details": {
          "message": "Ring integration test not implemented"
        }
      },
      "ollama": {
        "service": "ollama",
        "status": "healthy",
        "response_time": 200000000,
        "details": {
          "models": ["llama2", "codellama"]
        }
      }
    },
    "requested": ["home_assistant", "ring", "ollama"],
    "tested_at": "2024-01-15T10:30:00Z",
    "timeout_sec": 30
  }
}
```

### Performance Testing

#### POST /api/v1/test/performance
Run performance tests on system components.

**Request Body:**
```json
{
  "test_type": "database",
  "duration": 30,
  "iterations": 100
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Performance test completed",
    "test_type": "database",
    "results": {
      "test_type": "database",
      "query_time_ms": 2,
      "entity_count": 79,
      "queries_tested": 1
    },
    "total_time_ms": 5,
    "tested_at": "2024-01-15T10:30:00Z"
  }
}
```

### Development Helpers

#### POST /api/v1/test/reset
Reset test data and mock entities.

**Request Body:**
```json
{
  "confirm_reset": true,
  "reset_types": ["mock_entities", "test_data"]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Test data reset completed",
    "reset_types": ["mock_entities", "test_data"],
    "actions": ["mock entities and test data"],
    "reset_at": "2024-01-15T10:30:00Z"
  }
}
```

#### POST /api/v1/test/generate-data
Generate test data for development.

**Request Body:**
```json
{
  "data_type": "entities",
  "count": 15,
  "options": {
    "entity_types": ["light", "sensor"]
  }
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Test entities generated",
    "data_type": "entities",
    "count": 15,
    "entities": [...],
    "generated_at": "2024-01-15T10:30:00Z"
  }
}
```

### WebSocket Testing

#### GET /api/v1/test/websocket
Test WebSocket endpoint availability.

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "WebSocket test completed",
    "status": "available",
    "endpoint": "/ws",
    "statistics": {
      "hub_initialized": true,
      "endpoint_available": true
    },
    "tested_at": "2024-01-15T10:30:00Z"
  }
}
```

## Usage Examples

### Basic Mock Entity Setup

1. **Check test endpoint status:**
```bash
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:3001/api/v1/test/endpoint-status
```

2. **Generate mock entities:**
```bash
curl -X POST -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"count": 20, "entity_types": ["light", "switch", "sensor"], "reset": true}' \
  http://localhost:3001/api/v1/test/mock-entities
```

3. **Get generated entities:**
```bash
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:3001/api/v1/test/mock-entities
```

### System Health Check

```bash
curl -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:3001/api/v1/test/system-health
```

### Test All Connections

```bash
curl -X POST -H "Authorization: Bearer $JWT_TOKEN" \
  http://localhost:3001/api/v1/test/connections
```

### Reset Test Environment

```bash
curl -X POST -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"confirm_reset": true, "reset_types": ["mock_entities"]}' \
  http://localhost:3001/api/v1/test/reset
```

## Configuration

Test endpoints can be configured in `config.yaml`:

```yaml
test:
  endpoints_enabled: true  # Automatically disabled in production
  mock_data_persistence: false
  default_entity_count: 20
  supported_entity_types: ["light", "switch", "sensor", "binary_sensor", "climate", "cover", "lock"]
  performance_tests_enabled: true
  max_performance_test_duration: "60s"
  connection_timeout: "30s"
  health_check_interval: "5m"
  auto_generate_test_data: false
  reset_data_on_startup: false
```

## Security Notes

- Test endpoints are automatically disabled when `server.mode` is set to `production`
- All test endpoints require valid JWT authentication
- Mock data operations are isolated and don't affect production entities
- Test endpoints include rate limiting to prevent abuse
- All test operations are logged for audit purposes

## Troubleshooting

### Test Endpoints Disabled

If you see "Test endpoints are disabled in production mode", check:
1. `server.mode` is set to `development` in config.yaml
2. Server has been restarted after configuration changes

### Mock Entities Not Persisting

Mock entities are stored in memory by default. To enable persistence:
1. Set `test.mock_data_persistence: true` in config.yaml
2. Restart the server

### Connection Tests Failing

For failing connection tests:
1. Check service configuration in config.yaml
2. Verify services are running and accessible
3. Check network connectivity and firewall settings
4. Review application logs for detailed error messages 