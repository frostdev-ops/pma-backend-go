# Cache Management System

The PMA Backend includes a comprehensive cache management system that provides centralized control over all system caches, enabling administrators to monitor, optimize, and troubleshoot cache performance.

## Overview

The cache management system consists of several key components:

- **Cache Interface**: Standardized interface that all caches must implement
- **Cache Registry**: Central registry for managing cache instances
- **Cache Manager**: High-level operations for cache management
- **Cache Handler**: REST API endpoints for cache operations
- **Base Cache**: Common functionality for cache implementations
- **Cache Adapters**: Wrappers for existing cache systems

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   REST API      │    │  Cache Manager  │    │ Cache Registry  │
│   Endpoints     │────│                 │────│                 │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │                        │
                                │                        │
                       ┌─────────────────┐    ┌─────────────────┐
                       │  Cache Adapters │    │   Individual    │
                       │                 │    │     Caches      │
                       │                 │────│                 │
                       └─────────────────┘    └─────────────────┘
```

## Cache Types

The system supports the following cache types:

- `entity`: Entity data caches
- `display`: Display settings and capabilities caches
- `network`: Network device and routing caches
- `energy`: Energy management and UPS caches
- `configuration`: System configuration caches
- `query`: Database query result caches
- `response`: API response caches
- `session`: User session caches
- `websocket`: WebSocket connection caches
- `analytics`: Analytics and metrics caches
- `system`: System metrics and status caches
- `other`: Miscellaneous caches

## API Endpoints

### Core Operations

#### Clear Caches
```http
POST /api/v1/cache/clear
Content-Type: application/json

{
  "caches": ["cache_name_1", "cache_name_2"],
  "force": false,
  "async": false
}
```

#### Refresh Caches
```http
POST /api/v1/cache/refresh
Content-Type: application/json

{
  "caches": ["cache_name_1", "cache_name_2"],
  "force": false,
  "async": false
}
```

#### Get Cache Status
```http
GET /api/v1/cache/status
```

#### Warm Caches
```http
POST /api/v1/cache/warm
Content-Type: application/json

{
  "caches": ["cache_name_1", "cache_name_2"],
  "async": false
}
```

### Statistics and Monitoring

#### Get Cache Statistics
```http
GET /api/v1/cache/stats?type=entity&cache=cache_name
```

#### Get Cache Health
```http
GET /api/v1/cache/health
```

#### Get Memory Usage
```http
GET /api/v1/cache/memory
```

### Management Operations

#### Optimize Caches
```http
POST /api/v1/cache/optimize
```

#### Free Memory
```http
POST /api/v1/cache/memory/free
Content-Type: application/json

{
  "target_mb": 100
}
```

#### List Caches
```http
GET /api/v1/cache/list?type=entity
```

#### Get Cache Types
```http
GET /api/v1/cache/types
```

### Type-Based Operations

#### Clear Caches by Type
```http
POST /api/v1/cache/clear/{type}
```

#### Refresh Caches by Type
```http
POST /api/v1/cache/refresh/{type}
```

### Individual Cache Operations

#### Get Individual Cache Stats
```http
GET /api/v1/cache/{name}/stats
```

#### Clear Individual Cache
```http
POST /api/v1/cache/{name}/clear
```

#### Refresh Individual Cache
```http
POST /api/v1/cache/{name}/refresh
```

#### Get Cache Keys
```http
GET /api/v1/cache/{name}/keys
```

#### Invalidate Specific Keys
```http
POST /api/v1/cache/invalidate
Content-Type: application/json

{
  "cache_name": "cache_name",
  "keys": ["key1", "key2", "key3"]
}
```

## Response Formats

### Cache Statistics Response
```json
{
  "status": "success",
  "data": {
    "name": "cache_name",
    "type": "entity",
    "size": 150,
    "memory_usage_bytes": 1048576,
    "hit_count": 1250,
    "miss_count": 200,
    "hit_rate": 0.862,
    "last_accessed": "2024-01-15T10:30:00Z",
    "created_at": "2024-01-15T09:00:00Z",
    "ttl": "5m",
    "is_healthy": true,
    "error_count": 0
  }
}
```

### Cache Operation Result
```json
{
  "status": "success",
  "data": {
    "operations": [
      {
        "cache_name": "entity_cache",
        "operation": "clear",
        "success": true,
        "duration": "12ms",
        "entries_affected": 150,
        "memory_freed_bytes": 1048576,
        "timestamp": "2024-01-15T10:30:00Z"
      }
    ],
    "success_count": 1,
    "error_count": 0,
    "total_duration": "12ms",
    "total_memory_freed_bytes": 1048576,
    "total_entries_affected": 150
  }
}
```

### Health Report Response
```json
{
  "status": "success",
  "data": {
    "overall_health": true,
    "healthy_caches": ["cache1", "cache2"],
    "unhealthy_caches": [],
    "cache_health": {
      "cache1": true,
      "cache2": true
    },
    "issues": [],
    "last_check": "2024-01-15T10:30:00Z"
  }
}
```

### Memory Usage Response
```json
{
  "status": "success",
  "data": {
    "total_memory_usage_bytes": 10485760,
    "memory_by_cache": {
      "entity_cache": 5242880,
      "display_cache": 2097152,
      "system_cache": 1048576
    },
    "memory_by_type": {
      "entity": 5242880,
      "display": 2097152,
      "system": 1048576
    },
    "largest_caches": [
      {
        "name": "entity_cache",
        "type": "entity",
        "memory_usage_bytes": 5242880,
        "percentage": 50.0
      }
    ],
    "memory_pressure": false,
    "recommended_actions": []
  }
}
```

## Implementation Guide

### Creating a Custom Cache

```go
package myservice

import (
    "time"
    "github.com/frostdev-ops/pma-backend-go/internal/core/cache"
)

type MyServiceCache struct {
    *cache.BaseCache
    service *MyService
}

func NewMyServiceCache(service *MyService) *MyServiceCache {
    return &MyServiceCache{
        BaseCache: cache.NewBaseCache("my_service", cache.CacheTypeOther, 10*time.Minute),
        service:   service,
    }
}

func (c *MyServiceCache) Refresh() error {
    // Clear existing cache
    if err := c.Clear(); err != nil {
        return err
    }
    
    // Load fresh data from service
    data, err := c.service.GetFreshData()
    if err != nil {
        c.recordError(err)
        return err
    }
    
    // Populate cache with fresh data
    for key, value := range data {
        c.Set(key, value, 0) // Use default TTL
    }
    
    return nil
}
```

### Registering a Cache

```go
// In your service initialization
cacheRegistry := cache.NewRegistry(logger)
myCache := NewMyServiceCache(myService)

if err := cacheRegistry.Register(myCache); err != nil {
    logger.WithError(err).Error("Failed to register cache")
}
```

### Using the Cache Manager

```go
manager := cache.NewManager(cacheRegistry, logger)

// Clear specific caches
results := manager.ClearCaches(ctx, []string{"my_service"})

// Refresh all caches of a type
err := manager.RefreshByType(ctx, cache.CacheTypeEntity)

// Get overall statistics
stats := manager.GetStats()

// Perform optimization
err = manager.OptimizeCaches(ctx)
```

## Configuration

Cache behavior can be configured through the system configuration:

```yaml
performance:
  database:
    enable_query_cache: true
    cache_ttl: "5m"
  api:
    enable_response_cache: true
    cache_ttl: "5m"

storage:
  cache_path: "./data/cache"
```

## Monitoring and Alerting

The cache management system provides several metrics for monitoring:

- **Cache Hit Rates**: Monitor cache effectiveness
- **Memory Usage**: Track cache memory consumption
- **Error Rates**: Detect cache-related issues
- **Health Status**: Overall cache system health

### Recommended Monitoring

1. **Set up alerts for low hit rates** (< 50%)
2. **Monitor memory usage trends**
3. **Track cache error rates**
4. **Set up health check monitoring**

## Best Practices

### Cache Design
- Use appropriate TTL values for your data freshness requirements
- Implement proper error handling in refresh methods
- Consider memory usage when designing cache keys and values
- Use cache types to organize related caches

### Performance
- Use batch operations when possible
- Implement async refresh for large caches
- Monitor cache hit rates and adjust TTL accordingly
- Use cache warming for critical data

### Maintenance
- Regularly run cache optimization
- Monitor memory usage and free memory when needed
- Use health checks to detect issues early
- Implement proper logging for cache operations

### Security
- Ensure cache clearing operations are authenticated
- Validate input parameters for cache operations
- Log all administrative cache operations
- Use appropriate error messages that don't leak sensitive information

## Troubleshooting

### Common Issues

1. **Low Hit Rate**
   - Check TTL settings
   - Verify refresh logic
   - Consider cache warming

2. **High Memory Usage**
   - Use memory optimization
   - Check for memory leaks
   - Adjust cache sizes

3. **Cache Errors**
   - Check health status
   - Review error logs
   - Verify service dependencies

4. **Performance Issues**
   - Run cache optimization
   - Monitor cache statistics
   - Consider cache architecture changes

### Debugging

Use the cache management API to debug issues:

```bash
# Check overall health
curl -X GET "http://localhost:3001/api/v1/cache/health"

# Get detailed statistics
curl -X GET "http://localhost:3001/api/v1/cache/stats"

# Check memory usage
curl -X GET "http://localhost:3001/api/v1/cache/memory"

# List all caches
curl -X GET "http://localhost:3001/api/v1/cache/list"
```

## Future Enhancements

- **Distributed Caching**: Support for Redis and other distributed cache backends
- **Cache Replication**: Multi-instance cache synchronization
- **Advanced Analytics**: Detailed cache usage analytics and recommendations
- **Automatic Scaling**: Dynamic cache size adjustment based on usage patterns
- **Cache Policies**: Configurable eviction and refresh policies 