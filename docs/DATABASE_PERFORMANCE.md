# Database Performance Optimizations

This document describes the comprehensive database performance optimizations implemented in the PMA backend system.

## Overview

The enhanced database system provides:
- **Advanced Connection Pooling** with health monitoring and auto-optimization
- **Intelligent Query Caching** with table-aware invalidation
- **Query Optimization** with automatic schema analysis and recommendations
- **Real-time Performance Monitoring** with detailed metrics and health reporting
- **SQLite-specific Optimizations** for maximum performance

## Features

### 1. Enhanced Connection Pooling

The connection pool manager provides:

```go
type PoolManager interface {
    GetOptimalPoolSize() int
    MonitorConnections() *PoolStats
    OptimizePool() error
    HandleLeaks() error
    GetConnectionHealth() map[string]HealthMetric
}
```

**Key Features:**
- ✅ Automatic pool sizing based on load patterns
- ✅ Connection leak detection and cleanup
- ✅ Health monitoring with configurable thresholds
- ✅ Real-time statistics and performance metrics
- ✅ Auto-optimization based on usage patterns

### 2. Query Result Caching

The query cache provides intelligent caching with:

```go
type QueryCache interface {
    Get(query string, params []interface{}) ([]byte, bool)
    Set(query string, params []interface{}, result []byte, ttl time.Duration) error
    Invalidate(pattern string) error
    InvalidateTable(tableName string) error
    GetStats() *CacheStats
}
```

**Key Features:**
- ✅ Table-aware invalidation (automatically invalidates when tables change)
- ✅ LRU eviction policy with configurable memory limits
- ✅ TTL-based expiration with per-query customization
- ✅ Pattern-based cache invalidation
- ✅ Comprehensive cache statistics and hit rate monitoring

### 3. Query Optimization

The query optimizer provides:

```go
type QueryOptimizer interface {
    OptimizeQuery(query string, params []interface{}) (string, []interface{}, error)
    AddIndex(table, column string) error
    AnalyzeQuery(query string) (*QueryAnalysis, error)
    GetSlowQueries() ([]*SlowQuery, error)
    OptimizeSchema() error
}
```

**Key Features:**
- ✅ Automatic query rewriting for better performance
- ✅ Slow query detection and analysis
- ✅ Index recommendation based on query patterns
- ✅ Schema optimization suggestions
- ✅ Query execution plan analysis

## Configuration

### Basic Configuration

```yaml
database:
  path: "./data/pma.db"
  max_connections: 25

performance:
  database:
    max_connections: 25
    max_idle_conns: 10
    conn_max_lifetime: "1h"
    query_timeout: "30s"
    enable_query_cache: true
    cache_ttl: "30m"
```

### Advanced Configuration

```yaml
performance:
  database:
    # Connection pool settings
    max_connections: 50        # Maximum open connections
    max_idle_conns: 20        # Maximum idle connections
    conn_max_lifetime: "2h"   # Maximum connection lifetime
    query_timeout: "45s"      # Query execution timeout
    
    # Cache settings
    enable_query_cache: true  # Enable query result caching
    cache_ttl: "45m"         # Default cache TTL
    cache_max_memory: 200    # Maximum cache memory (MB)
    cache_max_entries: 50000 # Maximum cache entries
    
    # Optimization settings
    enable_auto_optimize: true      # Enable automatic optimization
    optimize_interval: "15m"        # Optimization interval
    slow_query_threshold: "100ms"   # Slow query detection threshold
```

## Usage

### Basic Usage (Backward Compatible)

```go
// Basic database initialization (existing code works unchanged)
db, err := database.Initialize(cfg.Database)
if err != nil {
    log.Fatal("Failed to initialize database:", err)
}
```

### Enhanced Usage (New Features)

```go
// Enhanced database with performance features
enhancedDB, err := database.InitializeEnhanced(cfg.Database, logger)
if err != nil {
    log.Fatal("Failed to initialize enhanced database:", err)
}

// Get performance statistics
stats := enhancedDB.GetPerformanceStats()
health := enhancedDB.GetHealthStatus()

// Execute optimized queries
rows, err := enhancedDB.Query("SELECT * FROM entities WHERE room_id = ?", roomID)
```

### Performance API Endpoints

```bash
# Get overall performance status
GET /api/v1/performance/status

# Get database connection pool statistics
GET /api/v1/performance/database/pool

# Get query cache statistics
GET /api/v1/performance/cache/stats

# Trigger database optimization
POST /api/v1/performance/database/optimize
{
  "operations": ["analyze", "vacuum", "reindex"],
  "force": false
}

# Get slow query analysis
GET /api/v1/performance/queries/slow
```

## Performance Improvements

### SQLite Optimizations

Applied automatically during initialization:

```sql
PRAGMA foreign_keys = ON;          -- Enable foreign key constraints
PRAGMA journal_mode = WAL;         -- Write-Ahead Logging for better concurrency
PRAGMA synchronous = NORMAL;       -- Balanced safety/performance
PRAGMA cache_size = 10000;         -- 10MB page cache
PRAGMA temp_store = MEMORY;        -- Store temporary tables in memory
PRAGMA mmap_size = 268435456;      -- 256MB memory-mapped I/O
PRAGMA optimize;                   -- Optimize database structure
```

### Connection Pool Optimizations

- **Dynamic Sizing**: Pool size adjusts based on actual usage patterns
- **Leak Detection**: Automatically detects and cleans up leaked connections
- **Health Monitoring**: Continuous monitoring of connection health metrics
- **Load Balancing**: Distributes connections efficiently across workers

### Query Cache Optimizations

- **Intelligent Invalidation**: Automatically invalidates cache when underlying data changes
- **Memory Management**: LRU eviction with configurable memory limits
- **Pattern Matching**: Support for wildcard and regex-based cache invalidation
- **Statistics Tracking**: Detailed hit rate and performance monitoring

## Monitoring and Metrics

### Connection Pool Metrics

```json
{
  "active_connections": 8,
  "idle_connections": 2,
  "total_connections": 10,
  "wait_count": 0,
  "wait_duration": "0ms",
  "leaked_connections": 0,
  "max_lifetime": "1h",
  "utilization": 0.8,
  "health_metrics": {
    "connection_health": {
      "status": "healthy",
      "value": 0.95,
      "threshold": 0.8
    }
  }
}
```

### Query Cache Metrics

```json
{
  "hit_rate": 0.85,
  "total_hits": 12450,
  "total_misses": 2201,
  "entry_count": 156,
  "memory_usage": 8912384,
  "avg_ttl": "5m30s",
  "last_cleared": "2024-01-01T12:00:00Z"
}
```

### Health Status

```json
{
  "connection": "healthy",
  "pool": {
    "utilization": {
      "status": "healthy",
      "value": 0.4,
      "threshold": 0.8
    },
    "leak_rate": {
      "status": "healthy", 
      "value": 0.0,
      "threshold": 0.1
    }
  },
  "cache": {
    "hit_rate": 0.85,
    "memory_usage": 8912384,
    "status": "healthy"
  }
}
```

## Best Practices

### Configuration Guidelines

1. **Connection Pool Sizing**:
   - Start with `max_connections = 25` for most applications
   - Increase if you see high wait times
   - Monitor utilization and adjust accordingly

2. **Cache Settings**:
   - Enable caching for read-heavy workloads
   - Set `cache_ttl` based on data freshness requirements
   - Monitor hit rates and adjust cache size if needed

3. **Optimization Schedule**:
   - Run optimizations during low-traffic periods
   - Use `optimize_interval` of 15-30 minutes for most cases
   - Monitor slow queries and add indexes as needed

### Performance Tuning

1. **Monitor Key Metrics**:
   - Connection pool utilization (target: 60-80%)
   - Cache hit rate (target: >70%)
   - Query execution times (target: <100ms average)
   - Memory usage trends

2. **Optimize Queries**:
   - Use the slow query analysis to identify bottlenecks
   - Add indexes for frequently queried columns
   - Consider query rewriting for complex operations

3. **Cache Strategy**:
   - Cache frequently accessed, slowly changing data
   - Use table-based invalidation for data consistency
   - Monitor memory usage and eviction rates

## Troubleshooting

### Common Issues

**High Memory Usage**:
- Check cache memory settings
- Monitor for cache entry leaks
- Adjust `cache_max_memory` if needed

**Slow Query Performance**:
- Use slow query analysis to identify issues
- Check for missing indexes
- Consider query optimization

**Connection Pool Issues**:
- Monitor for connection leaks
- Check pool size vs. actual load
- Review connection lifetime settings

### Debug Information

Enable debug logging for detailed performance information:

```go
logger.SetLevel(logrus.DebugLevel)
```

This provides detailed logs for:
- Connection pool operations
- Cache hit/miss patterns
- Query optimization decisions
- Performance metrics updates

## Migration Guide

### From Basic to Enhanced Database

1. **Update Initialization**:
   ```go
   // Old way
   db, err := database.Initialize(cfg.Database)
   
   // New way
   enhancedDB, err := database.InitializeEnhanced(cfg.Database, logger)
   ```

2. **Update Configuration**:
   Add performance section to your config file

3. **Update Handlers** (optional):
   ```go
   // Update performance handler to use enhanced database
   perfHandler := handlers.NewPerformanceHandler(enhancedDB)
   ```

4. **Monitor and Tune**:
   Use the performance endpoints to monitor and optimize

The enhanced database is fully backward compatible - existing code continues to work unchanged while gaining automatic performance benefits. 