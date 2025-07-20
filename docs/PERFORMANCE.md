# PMA Backend Go - Performance Guide

This document provides comprehensive guidance for optimizing and monitoring the performance of PMA Backend Go.

## Table of Contents

- [Performance Overview](#performance-overview)
- [Benchmarking and Profiling](#benchmarking-and-profiling)
- [Database Performance](#database-performance)
- [Memory Optimization](#memory-optimization)
- [CPU Optimization](#cpu-optimization)
- [Network Performance](#network-performance)
- [WebSocket Optimization](#websocket-optimization)
- [Caching Strategies](#caching-strategies)
- [Concurrent Processing](#concurrent-processing)
- [Monitoring and Metrics](#monitoring-and-metrics)
- [Scaling Strategies](#scaling-strategies)
- [Performance Testing](#performance-testing)
- [Troubleshooting Performance Issues](#troubleshooting-performance-issues)

## Performance Overview

PMA Backend Go is designed for high performance with the following characteristics:

- **Concurrent Architecture**: Goroutine-based concurrency for handling multiple requests
- **Efficient Database Layer**: Optimized SQLite with connection pooling and caching
- **Memory Management**: Garbage collection tuning and memory pooling
- **Network Optimization**: HTTP/2 support, compression, and keep-alive connections
- **Real-time Communication**: Optimized WebSocket handling for live updates

### Performance Targets

| Metric | Target | Production |
|--------|--------|------------|
| API Response Time | < 100ms (95th percentile) | < 50ms average |
| WebSocket Latency | < 10ms | < 5ms average |
| Memory Usage | < 512MB baseline | < 100MB idle |
| CPU Usage | < 50% average | < 20% idle |
| Concurrent Connections | 1000+ WebSocket | 10000+ HTTP |
| Database Queries | < 10ms average | < 5ms for simple queries |

## Benchmarking and Profiling

### Built-in Profiling

PMA Backend includes built-in profiling endpoints:

```go
// Enable profiling in development
import _ "net/http/pprof"

func setupProfiling() {
    if config.IsDevelopment() {
        go func() {
            log.Println("Profiling server starting on :6060")
            log.Println(http.ListenAndServe("localhost:6060", nil))
        }()
    }
}
```

### Profiling Tools

```bash
# CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap

# Goroutine profiling
go tool pprof http://localhost:6060/debug/pprof/goroutine

# Mutex profiling
go tool pprof http://localhost:6060/debug/pprof/mutex

# Block profiling
go tool pprof http://localhost:6060/debug/pprof/block
```

### Performance Monitoring API

```bash
# Get performance metrics
curl http://localhost:3001/api/v1/performance/status

# Get memory statistics
curl http://localhost:3001/api/v1/performance/memory

# Get database performance
curl http://localhost:3001/api/v1/performance/database

# Start CPU profiling
curl -X POST http://localhost:3001/api/v1/performance/profile/cpu

# Get trace information
curl http://localhost:3001/api/v1/performance/trace
```

### Benchmarking Framework

```go
// benchmark_test.go
package main

import (
    "context"
    "testing"
    "time"
)

func BenchmarkEntityGet(b *testing.B) {
    service := setupTestService()
    ctx := context.Background()
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := service.GetEntity(ctx, "test-entity")
            if err != nil {
                b.Fatalf("GetEntity failed: %v", err)
            }
        }
    })
}

func BenchmarkDatabaseOperations(b *testing.B) {
    db := setupTestDB()
    defer db.Close()
    
    b.Run("Insert", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            entity := generateTestEntity(i)
            if err := insertEntity(db, entity); err != nil {
                b.Fatalf("Insert failed: %v", err)
            }
        }
    })
    
    b.Run("Select", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            if _, err := selectEntity(db, "test-id"); err != nil {
                b.Fatalf("Select failed: %v", err)
            }
        }
    })
}

func BenchmarkWebSocketBroadcast(b *testing.B) {
    hub := NewHub()
    clients := createTestClients(100)
    
    message := createTestMessage()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        hub.BroadcastToAll("test_message", message)
    }
}
```

## Database Performance

### SQLite Optimization

```yaml
# Configuration for optimal SQLite performance
database:
  path: "./data/pma.db"
  max_connections: 50
  max_idle_conns: 10
  conn_max_lifetime: "1h"
  
  # SQLite-specific optimizations
  busy_timeout: "30s"
  journal_mode: "WAL"           # Write-Ahead Logging
  synchronous: "NORMAL"         # Balance safety vs performance
  cache_size: 2000             # 2MB page cache
  temp_store: "MEMORY"         # Store temp tables in memory
  mmap_size: 268435456         # 256MB memory map
```

### Query Optimization

```sql
-- Create proper indexes
CREATE INDEX IF NOT EXISTS idx_entities_type ON entities(type);
CREATE INDEX IF NOT EXISTS idx_entities_state ON entities(state);
CREATE INDEX IF NOT EXISTS idx_entities_source ON entities(source);
CREATE INDEX IF NOT EXISTS idx_entities_room ON entities(room_id);
CREATE INDEX IF NOT EXISTS idx_entities_updated ON entities(updated_at);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_entities_type_state ON entities(type, state);
CREATE INDEX IF NOT EXISTS idx_entities_source_type ON entities(source, type);

-- Partial indexes for specific use cases
CREATE INDEX IF NOT EXISTS idx_entities_active ON entities(id) WHERE state != 'unavailable';
```

### Connection Pooling

```go
type DatabasePool struct {
    db          *sql.DB
    maxOpen     int
    maxIdle     int
    maxLifetime time.Duration
    statsCache  *sync.Map
}

func NewDatabasePool(config DatabaseConfig) (*DatabasePool, error) {
    db, err := sql.Open("sqlite3", config.Path+"?_journal_mode=WAL&_synchronous=NORMAL&cache=shared")
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    
    // Configure connection pool
    db.SetMaxOpenConns(config.MaxConnections)
    db.SetMaxIdleConns(config.MaxIdleConns)
    db.SetConnMaxLifetime(config.ConnMaxLifetime)
    
    return &DatabasePool{
        db:          db,
        maxOpen:     config.MaxConnections,
        maxIdle:     config.MaxIdleConns,
        maxLifetime: config.ConnMaxLifetime,
        statsCache:  &sync.Map{},
    }, nil
}

func (p *DatabasePool) GetStats() *DatabaseStats {
    stats := p.db.Stats()
    return &DatabaseStats{
        OpenConnections: stats.OpenConnections,
        InUse:          stats.InUse,
        Idle:           stats.Idle,
        WaitCount:      stats.WaitCount,
        WaitDuration:   stats.WaitDuration,
        MaxIdleClosed:  stats.MaxIdleClosed,
        MaxLifetimeClosed: stats.MaxLifetimeClosed,
    }
}
```

### Query Caching

```go
type QueryCache struct {
    cache    *sync.Map
    ttl      time.Duration
    maxSize  int
    metrics  *CacheMetrics
}

type CacheEntry struct {
    data      interface{}
    createdAt time.Time
    hits      int64
}

func NewQueryCache(ttl time.Duration, maxSize int) *QueryCache {
    cache := &QueryCache{
        cache:   &sync.Map{},
        ttl:     ttl,
        maxSize: maxSize,
        metrics: &CacheMetrics{},
    }
    
    // Start cleanup goroutine
    go cache.cleanup()
    return cache
}

func (c *QueryCache) Get(key string) (interface{}, bool) {
    if entry, ok := c.cache.Load(key); ok {
        cacheEntry := entry.(*CacheEntry)
        
        // Check TTL
        if time.Since(cacheEntry.createdAt) > c.ttl {
            c.cache.Delete(key)
            c.metrics.Misses++
            return nil, false
        }
        
        atomic.AddInt64(&cacheEntry.hits, 1)
        c.metrics.Hits++
        return cacheEntry.data, true
    }
    
    c.metrics.Misses++
    return nil, false
}

func (c *QueryCache) Set(key string, value interface{}) {
    entry := &CacheEntry{
        data:      value,
        createdAt: time.Now(),
        hits:      0,
    }
    
    c.cache.Store(key, entry)
    c.metrics.Sets++
}
```

### Database Performance Monitoring

```go
type DatabaseMonitor struct {
    db      *sql.DB
    logger  *logrus.Logger
    metrics *prometheus.Registry
}

func (m *DatabaseMonitor) LogSlowQueries(threshold time.Duration) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        stats := m.db.Stats()
        
        if stats.WaitDuration > threshold {
            m.logger.WithFields(logrus.Fields{
                "wait_count":    stats.WaitCount,
                "wait_duration": stats.WaitDuration,
                "in_use":        stats.InUse,
                "idle":          stats.Idle,
            }).Warn("Database performance degradation detected")
        }
    }
}

func (m *DatabaseMonitor) TrackQuery(query string, duration time.Duration) {
    // Log slow queries
    if duration > 100*time.Millisecond {
        m.logger.WithFields(logrus.Fields{
            "query":    query,
            "duration": duration,
        }).Warn("Slow query detected")
    }
    
    // Update metrics
    m.metrics.WithLabelValues("query_duration").Observe(duration.Seconds())
}
```

## Memory Optimization

### Garbage Collection Tuning

```go
import (
    "runtime"
    "runtime/debug"
)

func optimizeGC(config PerformanceConfig) {
    // Set GC target percentage
    debug.SetGCPercent(config.Memory.GCTarget)
    
    // Set memory limit
    if config.Memory.HeapLimit > 0 {
        debug.SetMemoryLimit(config.Memory.HeapLimit)
    }
    
    // Configure runtime
    runtime.GOMAXPROCS(config.Workers.MaxProcs)
}

func monitorMemoryUsage(logger *logrus.Logger) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        
        logger.WithFields(logrus.Fields{
            "alloc":         bToMb(m.Alloc),
            "total_alloc":   bToMb(m.TotalAlloc),
            "sys":           bToMb(m.Sys),
            "heap_alloc":    bToMb(m.HeapAlloc),
            "heap_sys":      bToMb(m.HeapSys),
            "heap_idle":     bToMb(m.HeapIdle),
            "heap_released": bToMb(m.HeapReleased),
            "gc_cycles":     m.NumGC,
            "gc_pause":      time.Duration(m.PauseNs[(m.NumGC+255)%256]),
        }).Debug("Memory statistics")
        
        // Alert on high memory usage
        if m.Alloc > 500*1024*1024 { // 500MB
            logger.WithField("alloc_mb", bToMb(m.Alloc)).Warn("High memory usage detected")
        }
    }
}

func bToMb(b uint64) uint64 {
    return b / 1024 / 1024
}
```

### Object Pooling

```go
import "sync"

// Pool for frequently allocated objects
type EntityPool struct {
    pool sync.Pool
}

func NewEntityPool() *EntityPool {
    return &EntityPool{
        pool: sync.Pool{
            New: func() interface{} {
                return &Entity{
                    Attributes: make(map[string]interface{}),
                }
            },
        },
    }
}

func (p *EntityPool) Get() *Entity {
    return p.pool.Get().(*Entity)
}

func (p *EntityPool) Put(entity *Entity) {
    // Reset entity state
    entity.ID = ""
    entity.Name = ""
    entity.Type = ""
    entity.State = ""
    
    // Clear map but keep capacity
    for k := range entity.Attributes {
        delete(entity.Attributes, k)
    }
    
    p.pool.Put(entity)
}

// Buffer pooling for WebSocket messages
var messageBufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 0, 1024) // 1KB initial capacity
    },
}

func getMessageBuffer() []byte {
    return messageBufferPool.Get().([]byte)
}

func putMessageBuffer(buf []byte) {
    if cap(buf) < 64*1024 { // Don't pool buffers larger than 64KB
        messageBufferPool.Put(buf[:0])
    }
}
```

### Memory Leak Detection

```go
type MemoryLeakDetector struct {
    baseline    runtime.MemStats
    logger      *logrus.Logger
    alertThreshold uint64
}

func NewMemoryLeakDetector(logger *logrus.Logger) *MemoryLeakDetector {
    var baseline runtime.MemStats
    runtime.ReadMemStats(&baseline)
    
    return &MemoryLeakDetector{
        baseline:       baseline,
        logger:         logger,
        alertThreshold: 100 * 1024 * 1024, // 100MB
    }
}

func (d *MemoryLeakDetector) Check() {
    var current runtime.MemStats
    runtime.ReadMemStats(&current)
    
    growth := current.Alloc - d.baseline.Alloc
    
    if growth > d.alertThreshold {
        d.logger.WithFields(logrus.Fields{
            "baseline_mb": bToMb(d.baseline.Alloc),
            "current_mb":  bToMb(current.Alloc),
            "growth_mb":   bToMb(growth),
            "goroutines":  runtime.NumGoroutine(),
        }).Warn("Potential memory leak detected")
        
        // Force GC and check again
        runtime.GC()
        runtime.ReadMemStats(&current)
        
        if current.Alloc > d.baseline.Alloc+d.alertThreshold/2 {
            d.logger.Error("Memory leak confirmed after forced GC")
        }
    }
}
```

## CPU Optimization

### Goroutine Management

```go
type WorkerPool struct {
    workers    int
    jobs       chan Job
    results    chan Result
    wg         sync.WaitGroup
    ctx        context.Context
    cancel     context.CancelFunc
}

type Job interface {
    Execute() Result
}

type Result interface {
    Error() error
}

func NewWorkerPool(workers int) *WorkerPool {
    ctx, cancel := context.WithCancel(context.Background())
    
    return &WorkerPool{
        workers: workers,
        jobs:    make(chan Job, workers*2), // Buffer jobs
        results: make(chan Result, workers*2),
        ctx:     ctx,
        cancel:  cancel,
    }
}

func (p *WorkerPool) Start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker()
    }
}

func (p *WorkerPool) worker() {
    defer p.wg.Done()
    
    for {
        select {
        case job := <-p.jobs:
            result := job.Execute()
            
            select {
            case p.results <- result:
            case <-p.ctx.Done():
                return
            }
            
        case <-p.ctx.Done():
            return
        }
    }
}

func (p *WorkerPool) Submit(job Job) error {
    select {
    case p.jobs <- job:
        return nil
    case <-p.ctx.Done():
        return ErrPoolClosed
    default:
        return ErrPoolFull
    }
}

func (p *WorkerPool) Stop() {
    p.cancel()
    close(p.jobs)
    p.wg.Wait()
    close(p.results)
}
```

### CPU Profiling and Optimization

```go
func optimizeCPUUsage() {
    // Limit the number of OS threads
    runtime.GOMAXPROCS(runtime.NumCPU())
    
    // Monitor CPU usage
    go monitorCPUUsage()
}

func monitorCPUUsage() {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for range ticker.C {
        var rusage syscall.Rusage
        if err := syscall.Getrusage(syscall.RUSAGE_SELF, &rusage); err == nil {
            userTime := time.Duration(rusage.Utime.Nano())
            sysTime := time.Duration(rusage.Stime.Nano())
            
            logger.WithFields(logrus.Fields{
                "user_time_ms": userTime.Milliseconds(),
                "sys_time_ms":  sysTime.Milliseconds(),
                "goroutines":   runtime.NumGoroutine(),
            }).Debug("CPU usage statistics")
        }
    }
}
```

### Efficient Data Processing

```go
// Use efficient data structures and algorithms
type EntityIndex struct {
    byID     map[string]*Entity
    byType   map[EntityType][]*Entity
    bySource map[SourceType][]*Entity
    mutex    sync.RWMutex
}

func NewEntityIndex() *EntityIndex {
    return &EntityIndex{
        byID:     make(map[string]*Entity),
        byType:   make(map[EntityType][]*Entity),
        bySource: make(map[SourceType][]*Entity),
    }
}

func (idx *EntityIndex) Add(entity *Entity) {
    idx.mutex.Lock()
    defer idx.mutex.Unlock()
    
    idx.byID[entity.ID] = entity
    idx.byType[entity.Type] = append(idx.byType[entity.Type], entity)
    idx.bySource[entity.Source] = append(idx.bySource[entity.Source], entity)
}

func (idx *EntityIndex) GetByType(entityType EntityType) []*Entity {
    idx.mutex.RLock()
    defer idx.mutex.RUnlock()
    
    entities := idx.byType[entityType]
    result := make([]*Entity, len(entities))
    copy(result, entities)
    return result
}

// Use byte pools for JSON processing
var jsonBufferPool = sync.Pool{
    New: func() interface{} {
        return bytes.NewBuffer(make([]byte, 0, 1024))
    },
}

func marshalEntityOptimized(entity *Entity) ([]byte, error) {
    buf := jsonBufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        jsonBufferPool.Put(buf)
    }()
    
    encoder := json.NewEncoder(buf)
    if err := encoder.Encode(entity); err != nil {
        return nil, err
    }
    
    result := make([]byte, buf.Len())
    copy(result, buf.Bytes())
    return result, nil
}
```

## Network Performance

### HTTP Optimization

```yaml
# Server configuration for optimal performance
server:
  port: 3001
  host: "0.0.0.0"
  read_timeout: "30s"
  write_timeout: "30s"
  idle_timeout: "120s"
  max_header_bytes: 1048576
  
  # Enable HTTP/2
  enable_h2: true
  
  # Connection optimization
  keep_alive: true
  keep_alive_timeout: "30s"
  max_connections: 10000
```

### Response Compression

```go
import "github.com/gin-contrib/gzip"

func setupCompression(router *gin.Engine) {
    // Configure gzip compression
    router.Use(gzip.Gzip(gzip.BestSpeed, gzip.WithExcludedExtensions([]string{".pdf", ".mp4"})))
}

// Custom compression middleware for better control
func CompressionMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Check if client accepts compression
        if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
            c.Next()
            return
        }
        
        // Skip compression for small responses
        c.Header("Content-Encoding", "gzip")
        
        writer := &gzipWriter{
            ResponseWriter: c.Writer,
            writer:         gzip.NewWriter(c.Writer),
        }
        defer writer.Close()
        
        c.Writer = writer
        c.Next()
    }
}

type gzipWriter struct {
    gin.ResponseWriter
    writer *gzip.Writer
}

func (g *gzipWriter) Write(data []byte) (int, error) {
    return g.writer.Write(data)
}

func (g *gzipWriter) Close() {
    g.writer.Close()
}
```

### Connection Pooling

```go
import "net/http"

func setupHTTPClient() *http.Client {
    transport := &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        DisableCompression:  false,
        ForceAttemptHTTP2:   true,
        
        // TCP settings
        DialContext: (&net.Dialer{
            Timeout:   30 * time.Second,
            KeepAlive: 30 * time.Second,
        }).DialContext,
        
        // TLS settings
        TLSHandshakeTimeout: 10 * time.Second,
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: false,
        },
    }
    
    return &http.Client{
        Transport: transport,
        Timeout:   60 * time.Second,
    }
}
```

## WebSocket Optimization

### Connection Management

```go
type OptimizedHub struct {
    clients    sync.Map // map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    
    // Message batching
    batchBuffer [][]byte
    batchTimer  *time.Timer
    batchMutex  sync.Mutex
    
    // Metrics
    metrics *HubMetrics
    logger  *logrus.Logger
}

func NewOptimizedHub(logger *logrus.Logger) *OptimizedHub {
    hub := &OptimizedHub{
        broadcast:   make(chan []byte, 256),
        register:    make(chan *Client, 64),
        unregister:  make(chan *Client, 64),
        batchBuffer: make([][]byte, 0, 100),
        metrics:     &HubMetrics{},
        logger:      logger,
    }
    
    hub.batchTimer = time.AfterFunc(50*time.Millisecond, hub.flushBatch)
    return hub
}

func (h *OptimizedHub) Run() {
    defer h.batchTimer.Stop()
    
    for {
        select {
        case client := <-h.register:
            h.clients.Store(client, true)
            h.metrics.ConnectedClients++
            
        case client := <-h.unregister:
            if _, ok := h.clients.LoadAndDelete(client); ok {
                close(client.send)
                h.metrics.ConnectedClients--
            }
            
        case message := <-h.broadcast:
            h.batchMessage(message)
        }
    }
}

func (h *OptimizedHub) batchMessage(message []byte) {
    h.batchMutex.Lock()
    defer h.batchMutex.Unlock()
    
    h.batchBuffer = append(h.batchBuffer, message)
    
    // Flush immediately if buffer is full
    if len(h.batchBuffer) >= 50 {
        h.flushBatchUnlocked()
    } else {
        // Reset timer for next batch
        h.batchTimer.Reset(50 * time.Millisecond)
    }
}

func (h *OptimizedHub) flushBatch() {
    h.batchMutex.Lock()
    defer h.batchMutex.Unlock()
    h.flushBatchUnlocked()
}

func (h *OptimizedHub) flushBatchUnlocked() {
    if len(h.batchBuffer) == 0 {
        return
    }
    
    // Create batch message
    var batchedMessage []byte
    if len(h.batchBuffer) == 1 {
        batchedMessage = h.batchBuffer[0]
    } else {
        batchedMessage = h.createBatchMessage(h.batchBuffer)
    }
    
    // Send to all clients
    h.clients.Range(func(key, value interface{}) bool {
        client := key.(*Client)
        select {
        case client.send <- batchedMessage:
            h.metrics.MessagesSent++
        default:
            h.unregister <- client
        }
        return true
    })
    
    // Clear buffer
    h.batchBuffer = h.batchBuffer[:0]
}

func (h *OptimizedHub) createBatchMessage(messages [][]byte) []byte {
    var totalSize int
    for _, msg := range messages {
        totalSize += len(msg) + 1 // +1 for newline
    }
    
    result := make([]byte, 0, totalSize)
    for i, msg := range messages {
        result = append(result, msg...)
        if i < len(messages)-1 {
            result = append(result, '\n')
        }
    }
    
    return result
}
```

### Message Compression

```go
import "github.com/gorilla/websocket"

func setupWebSocketCompression() websocket.Upgrader {
    return websocket.Upgrader{
        ReadBufferSize:    1024,
        WriteBufferSize:   1024,
        EnableCompression: true,
        CompressionLevel:  6, // Balance between compression ratio and speed
        CheckOrigin: func(r *http.Request) bool {
            return true
        },
    }
}

// Compress large messages
func (c *Client) sendCompressed(message []byte) error {
    if len(message) > 1024 { // Compress messages larger than 1KB
        compressed, err := compressMessage(message)
        if err != nil {
            return err
        }
        
        // Only use compressed if it's actually smaller
        if len(compressed) < len(message) {
            return c.conn.WriteMessage(websocket.BinaryMessage, compressed)
        }
    }
    
    return c.conn.WriteMessage(websocket.TextMessage, message)
}

func compressMessage(data []byte) ([]byte, error) {
    var buf bytes.Buffer
    writer := gzip.NewWriter(&buf)
    
    if _, err := writer.Write(data); err != nil {
        return nil, err
    }
    
    if err := writer.Close(); err != nil {
        return nil, err
    }
    
    return buf.Bytes(), nil
}
```

## Caching Strategies

### Multi-Level Caching

```go
type CacheManager struct {
    l1Cache    *FastCache    // In-memory, small, fast
    l2Cache    *LargeCache   // In-memory, large, slower
    l3Cache    *PersistentCache // Disk-based, persistent
    
    stats      *CacheStats
    logger     *logrus.Logger
}

type CacheStats struct {
    L1Hits   int64
    L1Misses int64
    L2Hits   int64
    L2Misses int64
    L3Hits   int64
    L3Misses int64
}

func NewCacheManager(config CacheConfig, logger *logrus.Logger) *CacheManager {
    return &CacheManager{
        l1Cache: NewFastCache(config.L1Size),
        l2Cache: NewLargeCache(config.L2Size),
        l3Cache: NewPersistentCache(config.L3Path),
        stats:   &CacheStats{},
        logger:  logger,
    }
}

func (cm *CacheManager) Get(key string) (interface{}, bool) {
    // Try L1 cache first (fastest)
    if value, ok := cm.l1Cache.Get(key); ok {
        atomic.AddInt64(&cm.stats.L1Hits, 1)
        return value, true
    }
    atomic.AddInt64(&cm.stats.L1Misses, 1)
    
    // Try L2 cache
    if value, ok := cm.l2Cache.Get(key); ok {
        atomic.AddInt64(&cm.stats.L2Hits, 1)
        // Promote to L1
        cm.l1Cache.Set(key, value)
        return value, true
    }
    atomic.AddInt64(&cm.stats.L2Misses, 1)
    
    // Try L3 cache (persistent)
    if value, ok := cm.l3Cache.Get(key); ok {
        atomic.AddInt64(&cm.stats.L3Hits, 1)
        // Promote to L2 and L1
        cm.l2Cache.Set(key, value)
        cm.l1Cache.Set(key, value)
        return value, true
    }
    atomic.AddInt64(&cm.stats.L3Misses, 1)
    
    return nil, false
}

func (cm *CacheManager) Set(key string, value interface{}) {
    // Write to all cache levels
    cm.l1Cache.Set(key, value)
    cm.l2Cache.Set(key, value)
    cm.l3Cache.Set(key, value)
}
```

### Cache Warming

```go
type CacheWarmer struct {
    cacheManager *CacheManager
    entityRepo   EntityRepository
    logger       *logrus.Logger
}

func NewCacheWarmer(cm *CacheManager, repo EntityRepository, logger *logrus.Logger) *CacheWarmer {
    return &CacheWarmer{
        cacheManager: cm,
        entityRepo:   repo,
        logger:       logger,
    }
}

func (cw *CacheWarmer) WarmCache(ctx context.Context) error {
    cw.logger.Info("Starting cache warming")
    start := time.Now()
    
    // Warm entity cache
    if err := cw.warmEntityCache(ctx); err != nil {
        return fmt.Errorf("failed to warm entity cache: %w", err)
    }
    
    // Warm room cache
    if err := cw.warmRoomCache(ctx); err != nil {
        return fmt.Errorf("failed to warm room cache: %w", err)
    }
    
    duration := time.Since(start)
    cw.logger.WithField("duration", duration).Info("Cache warming completed")
    
    return nil
}

func (cw *CacheWarmer) warmEntityCache(ctx context.Context) error {
    entities, err := cw.entityRepo.GetAll(ctx)
    if err != nil {
        return err
    }
    
    for _, entity := range entities {
        key := fmt.Sprintf("entity:%s", entity.ID)
        cw.cacheManager.Set(key, entity)
        
        // Cache by type as well
        typeKey := fmt.Sprintf("entities:type:%s", entity.Type)
        if existing, ok := cw.cacheManager.Get(typeKey); ok {
            entityList := existing.([]*Entity)
            entityList = append(entityList, entity)
            cw.cacheManager.Set(typeKey, entityList)
        } else {
            cw.cacheManager.Set(typeKey, []*Entity{entity})
        }
    }
    
    cw.logger.WithField("count", len(entities)).Info("Entity cache warmed")
    return nil
}
```

## Concurrent Processing

### Goroutine Patterns

```go
// Fan-out pattern for parallel processing
func processEntitiesParallel(entities []*Entity, workers int) error {
    jobs := make(chan *Entity, len(entities))
    results := make(chan error, len(entities))
    
    // Start workers
    for w := 0; w < workers; w++ {
        go func() {
            for entity := range jobs {
                results <- processEntity(entity)
            }
        }()
    }
    
    // Send jobs
    for _, entity := range entities {
        jobs <- entity
    }
    close(jobs)
    
    // Collect results
    var errors []error
    for i := 0; i < len(entities); i++ {
        if err := <-results; err != nil {
            errors = append(errors, err)
        }
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("failed to process %d entities", len(errors))
    }
    
    return nil
}

// Pipeline pattern for streaming processing
func createProcessingPipeline(input <-chan *Entity) <-chan *ProcessedEntity {
    // Stage 1: Validation
    validated := make(chan *Entity, 10)
    go func() {
        defer close(validated)
        for entity := range input {
            if validateEntity(entity) {
                validated <- entity
            }
        }
    }()
    
    // Stage 2: Enrichment
    enriched := make(chan *Entity, 10)
    go func() {
        defer close(enriched)
        for entity := range validated {
            enrichEntity(entity)
            enriched <- entity
        }
    }()
    
    // Stage 3: Processing
    output := make(chan *ProcessedEntity, 10)
    go func() {
        defer close(output)
        for entity := range enriched {
            processed := &ProcessedEntity{
                Entity:    entity,
                Timestamp: time.Now(),
            }
            output <- processed
        }
    }()
    
    return output
}
```

### Rate Limiting

```go
import "golang.org/x/time/rate"

type RateLimitedProcessor struct {
    limiter   *rate.Limiter
    processor func(interface{}) error
    logger    *logrus.Logger
}

func NewRateLimitedProcessor(rps int, burst int, processor func(interface{}) error) *RateLimitedProcessor {
    return &RateLimitedProcessor{
        limiter:   rate.NewLimiter(rate.Limit(rps), burst),
        processor: processor,
        logger:    logrus.New(),
    }
}

func (rlp *RateLimitedProcessor) Process(ctx context.Context, item interface{}) error {
    // Wait for rate limiter
    if err := rlp.limiter.Wait(ctx); err != nil {
        return fmt.Errorf("rate limiter error: %w", err)
    }
    
    return rlp.processor(item)
}

// Batch processing with rate limiting
func (rlp *RateLimitedProcessor) ProcessBatch(ctx context.Context, items []interface{}) error {
    for _, item := range items {
        if err := rlp.Process(ctx, item); err != nil {
            rlp.logger.WithError(err).Error("Failed to process item")
            continue
        }
    }
    return nil
}
```

## Monitoring and Metrics

### Prometheus Integration

```go
import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
    RequestDuration   *prometheus.HistogramVec
    RequestCount      *prometheus.CounterVec
    ActiveConnections prometheus.Gauge
    DatabaseQueries   *prometheus.CounterVec
    CacheHits         *prometheus.CounterVec
    MemoryUsage       prometheus.Gauge
    GoroutineCount    prometheus.Gauge
}

func NewMetrics() *Metrics {
    metrics := &Metrics{
        RequestDuration: prometheus.NewHistogramVec(
            prometheus.HistogramOpts{
                Name:    "http_request_duration_seconds",
                Help:    "HTTP request duration in seconds",
                Buckets: prometheus.DefBuckets,
            },
            []string{"method", "endpoint", "status"},
        ),
        
        RequestCount: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "http_requests_total",
                Help: "Total number of HTTP requests",
            },
            []string{"method", "endpoint", "status"},
        ),
        
        ActiveConnections: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Name: "websocket_connections_active",
                Help: "Number of active WebSocket connections",
            },
        ),
        
        DatabaseQueries: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "database_queries_total",
                Help: "Total number of database queries",
            },
            []string{"operation", "table"},
        ),
        
        CacheHits: prometheus.NewCounterVec(
            prometheus.CounterOpts{
                Name: "cache_operations_total",
                Help: "Total number of cache operations",
            },
            []string{"type", "level", "result"},
        ),
        
        MemoryUsage: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Name: "memory_usage_bytes",
                Help: "Current memory usage in bytes",
            },
        ),
        
        GoroutineCount: prometheus.NewGauge(
            prometheus.GaugeOpts{
                Name: "goroutines_count",
                Help: "Number of goroutines",
            },
        ),
    }
    
    // Register metrics
    prometheus.MustRegister(
        metrics.RequestDuration,
        metrics.RequestCount,
        metrics.ActiveConnections,
        metrics.DatabaseQueries,
        metrics.CacheHits,
        metrics.MemoryUsage,
        metrics.GoroutineCount,
    )
    
    return metrics
}

func (m *Metrics) UpdateSystemMetrics() {
    var memStats runtime.MemStats
    runtime.ReadMemStats(&memStats)
    
    m.MemoryUsage.Set(float64(memStats.Alloc))
    m.GoroutineCount.Set(float64(runtime.NumGoroutine()))
}
```

### Performance Dashboard

```json
{
  "dashboard": {
    "title": "PMA Backend Performance",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{method}} {{endpoint}}"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "50th percentile"
          }
        ]
      },
      {
        "title": "Memory Usage",
        "type": "graph",
        "targets": [
          {
            "expr": "memory_usage_bytes / 1024 / 1024",
            "legendFormat": "Memory (MB)"
          }
        ]
      },
      {
        "title": "Database Performance",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(database_queries_total[5m])",
            "legendFormat": "{{operation}} {{table}}"
          }
        ]
      }
    ]
  }
}
```

## Scaling Strategies

### Horizontal Scaling

```yaml
# Load balancer configuration (Nginx)
upstream pma_backend {
    least_conn;
    server pma-backend-1:3001 max_fails=3 fail_timeout=30s;
    server pma-backend-2:3001 max_fails=3 fail_timeout=30s;
    server pma-backend-3:3001 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    location / {
        proxy_pass http://pma_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        
        # Load balancing for WebSocket
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### Database Scaling

```go
// Read/Write splitting for database operations
type DatabaseCluster struct {
    writeDB *sql.DB
    readDBs []*sql.DB
    current int32
    logger  *logrus.Logger
}

func NewDatabaseCluster(writeDB *sql.DB, readDBs []*sql.DB, logger *logrus.Logger) *DatabaseCluster {
    return &DatabaseCluster{
        writeDB: writeDB,
        readDBs: readDBs,
        logger:  logger,
    }
}

func (dc *DatabaseCluster) GetReadDB() *sql.DB {
    if len(dc.readDBs) == 0 {
        return dc.writeDB
    }
    
    // Round-robin selection
    index := atomic.AddInt32(&dc.current, 1) % int32(len(dc.readDBs))
    return dc.readDBs[index]
}

func (dc *DatabaseCluster) GetWriteDB() *sql.DB {
    return dc.writeDB
}

func (dc *DatabaseCluster) ExecuteRead(query string, args ...interface{}) (*sql.Rows, error) {
    db := dc.GetReadDB()
    return db.Query(query, args...)
}

func (dc *DatabaseCluster) ExecuteWrite(query string, args ...interface{}) (sql.Result, error) {
    db := dc.GetWriteDB()
    return db.Exec(query, args...)
}
```

## Performance Testing

### Load Testing

```go
// load_test.go
package main

import (
    "context"
    "fmt"
    "net/http"
    "sync"
    "testing"
    "time"
)

func TestAPILoad(t *testing.T) {
    const (
        concurrency = 100
        requests    = 10000
        timeout     = 30 * time.Second
    )
    
    client := &http.Client{
        Timeout: 5 * time.Second,
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
        },
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), timeout)
    defer cancel()
    
    requestChan := make(chan int, requests)
    results := make(chan time.Duration, requests)
    
    // Generate requests
    for i := 0; i < requests; i++ {
        requestChan <- i
    }
    close(requestChan)
    
    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < concurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for range requestChan {
                start := time.Now()
                resp, err := client.Get("http://localhost:3001/api/v1/health")
                duration := time.Since(start)
                
                if err != nil || resp.StatusCode != 200 {
                    t.Errorf("Request failed: %v", err)
                    continue
                }
                resp.Body.Close()
                
                results <- duration
            }
        }()
    }
    
    go func() {
        wg.Wait()
        close(results)
    }()
    
    // Collect results
    var durations []time.Duration
    for duration := range results {
        durations = append(durations, duration)
    }
    
    // Calculate statistics
    stats := calculateStats(durations)
    fmt.Printf("Load test results:\n")
    fmt.Printf("Total requests: %d\n", len(durations))
    fmt.Printf("Average: %v\n", stats.Average)
    fmt.Printf("95th percentile: %v\n", stats.P95)
    fmt.Printf("99th percentile: %v\n", stats.P99)
    fmt.Printf("Max: %v\n", stats.Max)
}

type Stats struct {
    Average time.Duration
    P50     time.Duration
    P95     time.Duration
    P99     time.Duration
    Max     time.Duration
}

func calculateStats(durations []time.Duration) Stats {
    sort.Slice(durations, func(i, j int) bool {
        return durations[i] < durations[j]
    })
    
    var total time.Duration
    for _, d := range durations {
        total += d
    }
    
    return Stats{
        Average: total / time.Duration(len(durations)),
        P50:     durations[len(durations)*50/100],
        P95:     durations[len(durations)*95/100],
        P99:     durations[len(durations)*99/100],
        Max:     durations[len(durations)-1],
    }
}
```

### WebSocket Load Testing

```bash
# Use artillery for WebSocket load testing
npm install -g artillery

# artillery_websocket.yml
config:
  target: 'ws://localhost:3001'
  phases:
    - duration: 60
      arrivalRate: 10
scenarios:
  - name: "WebSocket connections"
    engine: ws
    
# Run load test
artillery run artillery_websocket.yml
```

## Troubleshooting Performance Issues

### Common Performance Problems

1. **High Memory Usage**
   - Check for memory leaks with pprof
   - Review goroutine count
   - Analyze heap allocations

2. **Slow Database Queries**
   - Enable query logging
   - Analyze query execution plans
   - Add missing indexes

3. **High CPU Usage**
   - Profile CPU usage with pprof
   - Check for inefficient algorithms
   - Review goroutine usage

4. **WebSocket Performance**
   - Monitor connection count
   - Check message buffer sizes
   - Review subscription patterns

### Performance Debugging Commands

```bash
# Get current performance metrics
curl http://localhost:3001/api/v1/performance/status

# Start CPU profiling
curl -X POST http://localhost:3001/api/v1/performance/profile/cpu

# Get memory profile
curl http://localhost:3001/api/v1/performance/profile/memory

# Get goroutine information
curl http://localhost:3001/api/v1/performance/goroutines

# Database performance
curl http://localhost:3001/api/v1/performance/database

# WebSocket metrics
curl http://localhost:3001/api/v1/websocket/metrics
```

For more information, see the [PMA Backend Go Documentation](../README.md) and [Troubleshooting Guide](TROUBLESHOOTING.md).