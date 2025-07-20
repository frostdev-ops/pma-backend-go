# Enhanced Error Handling and Recovery System

This document describes the comprehensive error handling and recovery system implemented in the PMA backend, providing resilient operation through intelligent error classification, automatic recovery mechanisms, and detailed monitoring.

## Overview

The enhanced error handling system provides:
- **Comprehensive Error Classification** with severity levels and categories
- **Automatic Recovery Mechanisms** including circuit breakers and retry policies
- **Intelligent Error Reporting** with aggregation and trend analysis
- **Real-time Monitoring** with health status and alerting
- **Contextual Error Information** for improved debugging and resolution

## Core Components

### 1. Enhanced Error Types

#### EnhancedError Structure
```go
type EnhancedError struct {
    Code             int                    `json:"code"`
    Message          string                 `json:"message"`
    Details          string                 `json:"details,omitempty"`
    Category         ErrorCategory          `json:"category"`
    Severity         ErrorSeverity          `json:"severity"`
    Retryable        bool                   `json:"retryable"`
    RetryAfter       *time.Duration         `json:"retry_after,omitempty"`
    RetryStrategy    RecoveryStrategy       `json:"retry_strategy"`
    MaxRetries       int                    `json:"max_retries"`
    Permanent        bool                   `json:"permanent"`
    UserFacing       bool                   `json:"user_facing"`
    Context          *ErrorContext          `json:"context,omitempty"`
    Underlying       error                  `json:"-"`
    RelatedErrors    []*EnhancedError       `json:"related_errors,omitempty"`
    SuggestedActions []string               `json:"suggested_actions,omitempty"`
    DocumentationURL string                 `json:"documentation_url,omitempty"`
    ErrorID          string                 `json:"error_id,omitempty"`
}
```

#### Error Categories
- **Validation**: Input validation failures
- **Authentication**: Authentication-related errors
- **Authorization**: Permission and access control errors
- **NotFound**: Resource not found errors
- **Conflict**: Resource conflict and concurrency errors
- **RateLimit**: Rate limiting and throttling errors
- **Network**: Network connectivity and communication errors
- **Database**: Database operation and connectivity errors
- **Service**: External service integration errors
- **Adapter**: Device adapter and integration errors
- **Internal**: Internal system errors
- **External**: External dependency errors
- **Timeout**: Operation timeout errors
- **Unavailable**: Service unavailability errors

#### Severity Levels
- **Critical**: System-threatening errors requiring immediate attention
- **High**: Significant errors affecting functionality
- **Medium**: Moderate errors with workarounds available
- **Low**: Minor errors with minimal impact
- **Info**: Informational errors for logging purposes

### 2. Recovery Mechanisms

#### Circuit Breaker
Prevents cascade failures by temporarily blocking operations to failing services:

```go
type CircuitBreaker struct {
    name               string
    maxFailures        int
    timeout            time.Duration
    resetTimeout       time.Duration
    halfOpenMaxCalls   int
    state              CircuitBreakerState
    // ... additional fields
}

// States: Closed, Open, Half-Open
```

**Features:**
- Automatic state transitions based on failure rates
- Configurable failure thresholds and timeouts
- Health monitoring and metrics collection
- Manual reset capabilities

#### Retry Policy
Intelligent retry logic with exponential backoff:

```go
type RetryPolicy struct {
    MaxAttempts     int           `yaml:"max_attempts"`
    InitialDelay    time.Duration `yaml:"initial_delay"`
    MaxDelay        time.Duration `yaml:"max_delay"`
    BackoffFactor   float64       `yaml:"backoff_factor"`
    Jitter          bool          `yaml:"jitter"`
    RetryableErrors []ErrorCategory `yaml:"retryable_errors"`
}
```

**Features:**
- Exponential backoff with optional jitter
- Category-based retry decisions
- Context-aware cancellation
- Configurable retry limits

#### Error Reporter
Aggregates and analyzes error patterns:

```go
type ErrorReporter struct {
    errors     []ErrorReport
    maxErrors  int
    logger     *logrus.Logger
    onError    func(ErrorReport)
    onCritical func(ErrorReport)
}
```

**Features:**
- Error aggregation and deduplication
- Trend analysis and pattern detection
- Automatic resolution tracking
- Configurable alerting callbacks

### 3. Recovery Manager

Coordinates all recovery mechanisms:

```go
type RecoveryManager struct {
    circuitBreakers map[string]*CircuitBreaker
    retryExecutor   *RetryExecutor
    errorReporter   *ErrorReporter
    logger          *logrus.Logger
}
```

## Configuration

### Basic Configuration

```yaml
# Error handling configuration
error_handling:
  # Recovery settings
  recovery:
    # Retry policy
    retry:
      max_attempts: 3
      initial_delay: "100ms"
      max_delay: "30s"
      backoff_factor: 2.0
      jitter: true
      retryable_errors: ["network", "timeout", "unavailable", "rate_limit"]
    
    # Circuit breaker defaults
    circuit_breaker:
      max_failures: 5
      timeout: "60s"
      reset_timeout: "30s"
      half_open_max_calls: 3
  
  # Error reporting
  reporting:
    max_errors: 1000
    cleanup_interval: "24h"
    max_age_days: 30
    
  # Alerting
  alerting:
    critical_errors: true
    error_rate_threshold: 10
    circuit_breaker_alerts: true
```

### Advanced Configuration

```yaml
error_handling:
  recovery:
    # Service-specific circuit breakers
    circuit_breakers:
      homeassistant:
        max_failures: 3
        timeout: "30s"
        reset_timeout: "15s"
      
      database:
        max_failures: 10
        timeout: "120s"
        reset_timeout: "60s"
      
      external_api:
        max_failures: 5
        timeout: "60s"
        reset_timeout: "30s"
    
    # Custom retry policies
    retry_policies:
      network_operations:
        max_attempts: 5
        initial_delay: "500ms"
        max_delay: "60s"
        backoff_factor: 1.5
      
      database_operations:
        max_attempts: 3
        initial_delay: "100ms"
        max_delay: "10s"
        backoff_factor: 2.0
```

## Usage Examples

### Basic Error Handling

```go
// Creating enhanced errors
func validateInput(input string) error {
    if input == "" {
        return errors.NewValidationError("input", "cannot be empty")
    }
    return nil
}

// Wrapping existing errors
func processData(data []byte) error {
    if err := json.Unmarshal(data, &result); err != nil {
        return errors.Wrap(err, "Failed to parse JSON", errors.CategoryValidation)
    }
    return nil
}

// Adding context to errors
func handleRequest(c *gin.Context) {
    err := someOperation()
    if err != nil {
        enhancedErr := errors.Wrap(err, "Request processing failed", errors.CategoryInternal)
        enhancedErr = enhancedErr.WithContext(&errors.ErrorContext{
            RequestID: c.GetString("request_id"),
            UserID:    c.GetString("user_id"),
            Operation: "handle_request",
            Component: "api_handler",
        })
        c.Error(enhancedErr)
        return
    }
}
```

### Recovery Manager Usage

```go
// Initialize recovery manager
recoveryManager := errors.NewRecoveryManager(logger)

// Add circuit breakers for critical services
recoveryManager.AddCircuitBreaker("homeassistant", errors.CircuitBreakerConfig{
    MaxFailures:      3,
    Timeout:          time.Minute,
    ResetTimeout:     time.Second * 30,
    HalfOpenMaxCalls: 2,
})

// Execute operations with full recovery
err := recoveryManager.ExecuteWithRecovery(ctx, "homeassistant_sync", func() error {
    return homeAssistantAdapter.SyncEntities(ctx)
})

if err != nil {
    // Error has been automatically retried and circuit breaker applied
    logger.WithError(err).Error("Operation failed after recovery attempts")
}
```

### Middleware Integration

```go
// Enhanced error middleware
router.Use(middleware.RequestIDMiddleware())
router.Use(middleware.ErrorHandlingMiddleware(logger, recoveryManager))
router.Use(middleware.ErrorResponseMiddleware(logger, recoveryManager))
router.Use(middleware.ValidationErrorMiddleware())
router.Use(middleware.RateLimitErrorMiddleware())
```

## API Endpoints

### Error Monitoring Endpoints

```bash
# Get paginated error reports with filtering
GET /api/v1/errors/reports?page=1&limit=50&category=network&severity=high&resolved=false

# Get specific error report
GET /api/v1/errors/reports/{error_id}

# Resolve an error report
POST /api/v1/errors/reports/{error_id}/resolve
{
  "resolved_by": "admin@example.com",
  "notes": "Fixed network configuration"
}

# Get error statistics and trends
GET /api/v1/errors/stats

# Get recovery system metrics
GET /api/v1/errors/recovery/metrics

# Get circuit breaker status
GET /api/v1/errors/recovery/circuit-breakers

# Reset a circuit breaker
POST /api/v1/errors/recovery/circuit-breakers/{name}/reset

# Get overall error system health
GET /api/v1/errors/health

# Cleanup old error reports
POST /api/v1/errors/cleanup
{
  "max_age_days": 30
}

# Test error recovery system
POST /api/v1/errors/test
{
  "error_type": "network",
  "component": "test_service",
  "simulate": true
}
```

### Response Examples

#### Error Reports Response
```json
{
  "success": true,
  "data": {
    "reports": [
      {
        "error": {
          "code": 503,
          "message": "Service unavailable",
          "category": "service",
          "severity": "high",
          "retryable": true,
          "context": {
            "component": "homeassistant",
            "operation": "sync_entities",
            "request_id": "req-123",
            "timestamp": "2024-01-01T12:00:00Z"
          }
        },
        "timestamp": "2024-01-01T12:00:00Z",
        "count": 5,
        "last_seen": "2024-01-01T12:05:00Z",
        "first_seen": "2024-01-01T12:00:00Z",
        "resolved": false
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 50,
      "total": 125,
      "total_pages": 3
    }
  }
}
```

#### Error Health Status Response
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "timestamp": "2024-01-01T12:00:00Z",
    "recent_errors": 2,
    "critical_errors": 0,
    "unresolved_errors": 3,
    "open_circuit_breakers": 0,
    "total_circuit_breakers": 4,
    "recommendations": [
      "System error handling is operating normally",
      "Continue regular monitoring"
    ]
  }
}
```

#### Circuit Breaker Status Response
```json
{
  "success": true,
  "data": {
    "homeassistant": {
      "name": "homeassistant",
      "state": "closed",
      "failures": 1,
      "max_failures": 5,
      "health_status": "healthy",
      "failure_rate": 0.2,
      "last_fail_time": "2024-01-01T11:30:00Z",
      "timeout": "60s",
      "reset_timeout": "30s"
    },
    "database": {
      "name": "database",
      "state": "half-open",
      "failures": 3,
      "max_failures": 5,
      "health_status": "recovering",
      "failure_rate": 0.6,
      "half_open_calls": 2,
      "half_open_successes": 1
    }
  }
}
```

## Error Response Format

### Standard Error Response
```json
{
  "success": false,
  "error": "Validation failed for field 'email'",
  "code": 400,
  "timestamp": "2024-01-01T12:00:00Z",
  "path": "/api/v1/users",
  "method": "POST",
  "error_id": "err-123456",
  "request_id": "req-789012",
  "retryable": false,
  "suggestions": [
    "Check email format",
    "Ensure email is not empty"
  ]
}
```

### Development Mode Response
```json
{
  "success": false,
  "error": "Database operation failed",
  "code": 500,
  "category": "database",
  "severity": "high",
  "details": "Connection timeout after 30 seconds",
  "retryable": true,
  "retry_after": 5.0,
  "max_retries": 3,
  "stack_trace": [
    "main.go:123 main.handleRequest",
    "database.go:456 database.Query",
    "connection.go:789 connection.Execute"
  ]
}
```

## Monitoring and Alerting

### Health Indicators

1. **Error Rate**: Errors per time period
2. **Critical Error Count**: Unresolved critical errors
3. **Circuit Breaker Status**: Open/closed state of circuit breakers
4. **Recovery Success Rate**: Percentage of successfully recovered operations
5. **Average Resolution Time**: Time to resolve error reports

### Alert Conditions

- **Critical Errors**: Immediate alerts for critical severity errors
- **High Error Rate**: Alerts when error rate exceeds threshold
- **Circuit Breaker Opens**: Alerts when circuit breakers open
- **Unresolved Error Accumulation**: Alerts for increasing unresolved errors
- **Recovery Failure**: Alerts when recovery mechanisms fail

### Dashboard Metrics

```go
// Example metrics collection
type ErrorMetrics struct {
    TotalErrors        int64     `json:"total_errors"`
    ErrorRate          float64   `json:"error_rate"`
    CriticalErrors     int       `json:"critical_errors"`
    UnresolvedErrors   int       `json:"unresolved_errors"`
    CircuitBreakerStatus map[string]string `json:"circuit_breaker_status"`
    RecoverySuccessRate float64  `json:"recovery_success_rate"`
    AvgResolutionTime  float64   `json:"avg_resolution_time"`
    TopErrorSources    map[string]int `json:"top_error_sources"`
    ErrorTrends        []int     `json:"error_trends"`
}
```

## Best Practices

### Error Creation
1. **Use Appropriate Categories**: Choose the most specific category
2. **Set Correct Severity**: Align severity with business impact
3. **Include Context**: Add relevant request/operation context
4. **Provide Suggestions**: Include actionable resolution steps
5. **Mark User-Facing Appropriately**: Distinguish internal vs. user errors

### Recovery Configuration
1. **Set Realistic Thresholds**: Circuit breaker thresholds based on SLA
2. **Configure Appropriate Timeouts**: Balance responsiveness vs. resilience
3. **Use Jitter**: Avoid thundering herd problems in retries
4. **Monitor and Tune**: Regularly review and adjust configurations

### Error Handling in Code
1. **Fail Fast**: Use enhanced errors consistently
2. **Provide Context**: Always add relevant context information
3. **Handle Gracefully**: Implement appropriate fallback mechanisms
4. **Log Appropriately**: Use structured logging with error context
5. **Monitor Continuously**: Regular health checks and alerting

### Monitoring and Maintenance
1. **Regular Review**: Weekly review of error reports and trends
2. **Proactive Resolution**: Address patterns before they become critical
3. **Documentation Updates**: Keep error handling docs current
4. **Performance Impact**: Monitor overhead of error handling system
5. **Capacity Planning**: Plan for error report storage growth

## Troubleshooting

### Common Issues

**High Error Rates**:
- Check external service health
- Review recent code deployments
- Verify configuration changes
- Monitor resource utilization

**Circuit Breaker Stuck Open**:
- Check underlying service health
- Review failure thresholds
- Manually reset if service recovered
- Adjust timeout configurations

**Error Report Accumulation**:
- Review resolution processes
- Check automated cleanup settings
- Implement error triaging
- Monitor storage capacity

**Recovery Mechanism Failures**:
- Verify retry policy configurations
- Check circuit breaker settings
- Review error categorization
- Monitor recovery success rates

### Debug Information

Enable debug logging for detailed error handling information:

```go
logger.SetLevel(logrus.DebugLevel)
```

This provides:
- Detailed error context and stack traces
- Circuit breaker state transitions
- Retry attempt details
- Recovery mechanism decisions
- Error aggregation and reporting

## Integration Examples

### Service Integration
```go
// Integrating with existing services
type UserService struct {
    repo            UserRepository
    recoveryManager *errors.RecoveryManager
    logger          *logrus.Logger
}

func (s *UserService) CreateUser(ctx context.Context, user *User) error {
    return s.recoveryManager.ExecuteWithRecovery(ctx, "create_user", func() error {
        if err := s.validateUser(user); err != nil {
            return errors.NewValidationError("user", err.Error()).
                WithContext(errors.FromContext(ctx))
        }
        
        if err := s.repo.Create(ctx, user); err != nil {
            return errors.NewDatabaseError("create_user", err).
                WithContext(errors.FromContext(ctx))
        }
        
        return nil
    })
}
```

### Adapter Integration
```go
// Integrating with device adapters
type HomeAssistantAdapter struct {
    client          *http.Client
    recoveryManager *errors.RecoveryManager
    logger          *logrus.Logger
}

func (a *HomeAssistantAdapter) SyncEntities(ctx context.Context) error {
    return a.recoveryManager.ExecuteWithRecovery(ctx, "ha_sync", func() error {
        entities, err := a.fetchEntities(ctx)
        if err != nil {
            return errors.NewNetworkError("fetch_entities", err).
                WithContext(&errors.ErrorContext{
                    Component: "homeassistant",
                    Operation: "sync_entities",
                })
        }
        
        return a.processEntities(ctx, entities)
    })
}
```

## Migration Guide

### From Basic to Enhanced Error Handling

1. **Update Error Creation**:
   ```go
   // Old way
   return fmt.Errorf("user not found")
   
   // New way
   return errors.NewNotFoundError("user", userID)
   ```

2. **Add Recovery Manager**:
   ```go
   // Initialize in main.go
   recoveryManager := errors.NewRecoveryManager(logger)
   
   // Configure circuit breakers
   recoveryManager.AddCircuitBreaker("database", errors.CircuitBreakerConfig{
       MaxFailures: 5,
       Timeout:     time.Minute,
   })
   ```

3. **Update Middleware**:
   ```go
   // Update router setup
   router.Use(middleware.ErrorHandlingMiddleware(logger, recoveryManager))
   ```

4. **Integrate API Endpoints**:
   ```go
   // Add error monitoring routes
   errorHandler := handlers.NewErrorHandler(recoveryManager, logger)
   errorHandler.RegisterRoutes(apiGroup)
   ```

The enhanced error handling system provides comprehensive visibility and automatic recovery capabilities, significantly improving system reliability and operational efficiency. 