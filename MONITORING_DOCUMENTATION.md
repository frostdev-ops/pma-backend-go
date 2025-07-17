# PMA Backend Go - System Monitoring & Metrics

This document describes the comprehensive monitoring and metrics system implemented for the PMA Backend Go application.

## Overview

The monitoring system provides:
- **Prometheus-compatible metrics** for integration with Grafana and other monitoring tools
- **Comprehensive health checks** for all system components
- **Real-time performance analytics** with percentile calculations
- **System resource monitoring** (CPU, memory, disk, network)
- **Intelligent alerting** with configurable thresholds
- **Performance tracking** with request latency and error rate monitoring

## Architecture

The monitoring system consists of several key components:

### 1. Metrics Framework (`internal/core/metrics/`)

#### MetricsCollector Interface
```go
type MetricsCollector interface {
    RecordHTTPRequest(method, path string, status int, duration time.Duration)
    RecordWebSocketConnection(action string)
    RecordDatabaseQuery(operation string, duration time.Duration)
    RecordDeviceOperation(deviceType, operation string, success bool, duration time.Duration)
    RecordAutomationExecution(ruleID string, success bool, duration time.Duration)
    RecordLLMRequest(provider string, success bool, duration time.Duration, tokens int)
    RecordSystemResource(cpu, memory, disk float64)
    RecordAlert(severity, source, message string)
}
```

#### Prometheus Integration
- HTTP request metrics (counter, histogram)
- WebSocket connection tracking
- Database query performance
- Device operation metrics
- LLM usage and token tracking
- System resource gauges
- Custom business metrics

### 2. Health Checking (`internal/core/metrics/health.go`)

#### HealthChecker Interface
```go
type HealthChecker interface {
    CheckDatabase() HealthStatus
    CheckHomeAssistant() HealthStatus
    CheckDeviceAdapters() map[string]HealthStatus
    CheckLLMProviders() map[string]HealthStatus
    CheckSystemResources() HealthStatus
    GetOverallHealth() HealthReport
}
```

#### Health Status Types
- **healthy**: Component is functioning normally
- **degraded**: Component has issues but is still functional
- **unhealthy**: Component is not functioning
- **unknown**: Component status cannot be determined

### 3. System Monitoring (`internal/core/monitor/`)

#### Resource Monitor
Tracks system resources using `gopsutil`:
- CPU usage (per-core and total)
- Memory usage (total, available, used, cached)
- Disk usage (per-partition and total)
- Network I/O statistics
- Go runtime metrics (goroutines, GC, heap)
- Temperature monitoring (Raspberry Pi)

#### Alert Manager
Manages system alerts with:
- Configurable severity levels (info, warning, critical)
- Threshold-based alerting
- Alert lifecycle management
- Notification callbacks
- Alert history and retention

### 4. Performance Analytics (`internal/core/analytics/`)

#### Performance Tracker
Provides detailed performance analytics:
- Request latency percentiles (P50, P95, P99)
- Throughput metrics (requests per second)
- Error rates by endpoint
- Slow request detection
- Time-series data generation

### 5. API Endpoints

The monitoring system exposes the following REST endpoints:

#### Health Endpoints
- `GET /api/health` - Comprehensive health check
- `GET /api/health/live` - Kubernetes liveness probe
- `GET /api/health/ready` - Kubernetes readiness probe

#### Metrics Endpoints
- `GET /metrics` - Prometheus-compatible metrics
- `GET /api/monitor/system` - System resource statistics
- `GET /api/monitor/services` - Service status overview

#### Alert Endpoints
- `GET /api/alerts` - List alerts (with filtering)
- `POST /api/alerts/:id/resolve` - Resolve specific alert
- `GET /api/alerts/stats` - Alert statistics

#### Analytics Endpoints
- `GET /api/analytics/performance` - Performance metrics
- `GET /api/analytics/usage` - Usage statistics
- `GET /api/analytics/endpoints` - Per-endpoint metrics
- `GET /api/analytics/slow-requests` - Slow request analysis

## Configuration

### Monitoring Configuration
```yaml
monitoring:
  enabled: true
  metrics_retention: "24h"
  snapshot_interval: "30s"
  
  alerts:
    enabled: true
    thresholds:
      cpu_percent: 80.0
      memory_percent: 85.0
      disk_percent: 90.0
      error_rate: 0.05
  
  prometheus:
    enabled: true
    path: "/metrics"
  
  performance:
    enabled: true
    calculate_p99: true
    calculate_p95: true
    calculate_p50: true
    track_user_agents: false
```

## Database Schema

The monitoring system uses several database tables:

### Metrics Table
```sql
CREATE TABLE metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_name TEXT NOT NULL,
    metric_type TEXT NOT NULL,
    value REAL NOT NULL,
    labels JSON,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Alerts Table
```sql
CREATE TABLE alerts (
    id TEXT PRIMARY KEY,
    severity TEXT NOT NULL,
    source TEXT NOT NULL,
    message TEXT NOT NULL,
    details JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    resolved_by TEXT,
    resolved BOOLEAN DEFAULT FALSE
);
```

### System Snapshots Table
```sql
CREATE TABLE system_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cpu_usage REAL,
    memory_usage REAL,
    disk_usage REAL,
    snapshot_data JSON,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Usage Examples

### Basic Setup
```go
// Create monitoring service
config := &monitor.MonitoringServiceConfig{
    Monitoring: appConfig.Monitoring,
    MetricsRetention: 24 * time.Hour,
    SnapshotInterval: 30 * time.Second,
}

monitoringService := monitor.NewMonitoringService(config, logger)

// Set up health checks
monitoringService.SetDatabaseHealthCheck(func() metrics.HealthStatus {
    // Database health check logic
    return metrics.NewHealthStatus("healthy", "Database is responsive")
})

// Start the service
err := monitoringService.Start(context.Background())
if err != nil {
    log.Fatal("Failed to start monitoring service:", err)
}
```

### HTTP Middleware Integration
```go
// Add metrics middleware to Gin router
router.Use(middleware.MetricsMiddleware(
    monitoringService.GetMetricsCollector(),
    monitoringService.GetPerformanceTracker(),
))

// Add threshold alerting middleware
router.Use(middleware.ThresholdAlertMiddleware(
    monitoringService.GetMetricsCollector(),
    5*time.Second, // Alert on requests slower than 5 seconds
))
```

### Custom Metrics
```go
collector := monitoringService.GetMetricsCollector()

// Record custom metrics
collector.IncrementCounter("custom_events_total", map[string]string{
    "event_type": "user_login",
    "source": "web",
})

collector.RecordHistogram("custom_operation_duration", 
    operationTime.Seconds(), 
    map[string]string{"operation": "data_processing"})
```

### Alert Management
```go
alertManager := monitoringService.GetAlertManager()

// Create custom alert
alert := monitor.NewAlert(
    monitor.AlertSeverityWarning,
    "custom_service",
    "Service experiencing high latency",
).WithDetails(map[string]interface{}{
    "current_latency": 5.2,
    "threshold": 5.0,
})

err := alertManager.CreateAlert(alert)

// Set up alert callbacks
alertManager.OnAlertCreated(func(alert *monitor.Alert) {
    // Send notification (email, Slack, etc.)
    notificationService.SendAlert(alert)
})
```

## Grafana Integration

### Sample Prometheus Queries

#### HTTP Request Rate
```promql
rate(pma_http_requests_total[5m])
```

#### HTTP Request Latency P95
```promql
histogram_quantile(0.95, rate(pma_http_request_duration_seconds_bucket[5m]))
```

#### System Resource Usage
```promql
pma_system_cpu_usage_percent
pma_system_memory_usage_percent
pma_system_disk_usage_percent
```

#### Error Rate
```promql
rate(pma_http_requests_total{status=~"5.."}[5m]) / rate(pma_http_requests_total[5m])
```

### Dashboard Templates

The system includes Grafana dashboard templates for:
- System Overview (CPU, Memory, Disk, Network)
- HTTP Performance (Latency, Throughput, Error Rate)
- Application Metrics (Goroutines, GC, Heap)
- Alert Dashboard (Active Alerts, Alert History)
- Device Monitoring (Device Status, Operations)
- LLM Usage (Requests, Tokens, Response Time)

## Alerting Rules

### Default Alert Thresholds

| Metric | Warning | Critical | Description |
|--------|---------|----------|-------------|
| CPU Usage | 80% | 90% | System CPU utilization |
| Memory Usage | 85% | 95% | System memory utilization |
| Disk Usage | 90% | 95% | Disk space utilization |
| Error Rate | 5% | 10% | HTTP error rate |
| Response Time | 5s | 10s | HTTP response latency |

### Custom Alert Rules
```go
// Add custom alert rule
rule := monitor.AlertRule{
    Name: "High Database Latency",
    Metric: "database_query_duration",
    Operator: ">",
    Threshold: 1.0, // 1 second
    Duration: 5 * time.Minute,
    Severity: monitor.AlertSeverityWarning,
    Message: "Database queries are taking longer than expected",
}

alertManager.AddRule(rule)
```

## Performance Considerations

### Metrics Collection
- Metrics are collected in-memory with configurable retention
- Background workers handle periodic collection and cleanup
- Database writes are batched to minimize I/O

### Resource Usage
- System monitoring uses minimal CPU and memory
- Metrics storage is optimized with proper indexing
- Configurable data retention prevents unbounded growth

### Scalability
- Prometheus metrics scale horizontally
- Performance tracking uses sliding windows
- Alert processing is asynchronous

## Best Practices

### Monitoring Strategy
1. **Start with system metrics** (CPU, memory, disk)
2. **Add application metrics** (request rate, latency, errors)
3. **Implement business metrics** (user actions, feature usage)
4. **Set up meaningful alerts** (actionable, not noisy)
5. **Create dashboards** for different audiences

### Alert Configuration
1. **Use graduated thresholds** (warning before critical)
2. **Include context** in alert messages
3. **Set appropriate retention** periods
4. **Test alert conditions** before deployment
5. **Document escalation** procedures

### Performance Optimization
1. **Monitor the monitors** (check overhead)
2. **Use sampling** for high-volume metrics
3. **Optimize queries** for dashboard performance
4. **Regular cleanup** of old data
5. **Horizontal scaling** for high loads

## Troubleshooting

### Common Issues

#### High Memory Usage
- Check metrics retention period
- Verify cleanup workers are running
- Monitor performance tracker data size

#### Missing Metrics
- Verify monitoring service is started
- Check middleware is properly configured
- Ensure health check functions are set

#### Alert Fatigue
- Review alert thresholds
- Implement alert suppression rules
- Use alert grouping and severity levels

### Debug Endpoints
- `GET /api/monitor/system` - Current system status
- `GET /api/health` - Component health status
- `GET /api/alerts/stats` - Alert statistics

### Logs
Monitor service logs for:
- Metric collection errors
- Health check failures
- Alert generation and resolution
- Performance warnings

## Future Enhancements

### Planned Features
1. **Distributed tracing** integration
2. **Custom dashboard** builder
3. **Anomaly detection** using machine learning
4. **Integration** with external notification services
5. **Mobile app** for monitoring on-the-go

### Extension Points
- Custom metric collectors
- Additional health check providers
- Alert notification channels
- Performance analysis plugins
- Dashboard widgets

This monitoring system provides a solid foundation for observability and can be extended based on specific operational requirements. 