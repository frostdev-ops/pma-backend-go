# PMA Backend Go - Configuration Reference

This document provides comprehensive documentation for configuring the PMA Backend Go application.

## Table of Contents

- [Configuration Overview](#configuration-overview)
- [Configuration Sources](#configuration-sources)
- [Server Configuration](#server-configuration)
- [Database Configuration](#database-configuration)
- [Authentication & Security](#authentication--security)
- [Home Assistant Integration](#home-assistant-integration)
- [AI Services](#ai-services)
- [Performance Configuration](#performance-configuration)
- [Logging Configuration](#logging-configuration)
- [WebSocket Configuration](#websocket-configuration)
- [External Services](#external-services)
- [Monitoring & Analytics](#monitoring--analytics)
- [Environment Variables](#environment-variables)
- [Production Configuration](#production-configuration)
- [Configuration Validation](#configuration-validation)

## Configuration Overview

PMA Backend Go uses a hierarchical configuration system that supports multiple sources and formats. The configuration is designed to be flexible, secure, and suitable for both development and production environments.

### Configuration Hierarchy

1. **Default Values**: Built-in defaults for all options
2. **Configuration Files**: YAML configuration files
3. **Environment Variables**: Override any configuration value
4. **Command Line Flags**: Runtime parameter overrides

Higher priority sources override lower priority ones.

## Configuration Sources

### Primary Configuration File

**Location**: `configs/config.yaml`

This is the main configuration file with default values suitable for development.

### Local Configuration Override

**Location**: `configs/config.local.yaml` (optional)

Create this file to override default values without modifying the main configuration file. This file is typically gitignored.

### Environment-Specific Configuration

**Location**: `configs/config.{environment}.yaml`

Environment-specific configurations:
- `configs/config.development.yaml`
- `configs/config.production.yaml`
- `configs/config.staging.yaml`

Set the `APP_ENV` environment variable to load the appropriate configuration.

### Configuration Loading Order

```bash
# 1. Load defaults from config.yaml
# 2. Load environment-specific config (if APP_ENV is set)
# 3. Load local config.local.yaml (if exists)
# 4. Apply environment variable overrides
# 5. Apply command line flag overrides
```

## Server Configuration

Configure the HTTP server and basic application settings.

```yaml
server:
  # Server binding configuration
  port: 3001                    # Port to listen on
  host: "0.0.0.0"              # Host address to bind to
  mode: "development"           # Application mode: development|production|staging
  
  # Request handling
  read_timeout: "30s"           # Request read timeout
  write_timeout: "30s"          # Response write timeout
  idle_timeout: "120s"          # Keep-alive timeout
  
  # Request limits
  max_header_bytes: 1048576     # Maximum header size (1MB)
  max_request_size: 10485760    # Maximum request body size (10MB)
  
  # Development options
  hot_reload: true              # Enable hot reload in development
  debug_routes: true            # Log all registered routes
```

### Environment Variables

```bash
PORT=3001
HOST=0.0.0.0
SERVER_MODE=production
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s
```

## Database Configuration

Configure SQLite database settings and performance options.

```yaml
database:
  # Basic database settings
  path: "./data/pma.db"         # Database file path
  migrations_path: "./migrations" # Migration files directory
  
  # Connection settings
  max_connections: 25           # Maximum concurrent connections
  max_idle_conns: 10           # Maximum idle connections
  conn_max_lifetime: "1h"       # Connection maximum lifetime
  
  # Performance settings
  busy_timeout: "30s"           # SQLite busy timeout
  journal_mode: "WAL"           # Journal mode: DELETE|WAL|MEMORY
  synchronous: "NORMAL"         # Synchronous mode: OFF|NORMAL|FULL|EXTRA
  cache_size: 2000             # Page cache size
  
  # Backup configuration
  backup_enabled: true          # Enable automatic backups
  backup_path: "./data/backups" # Backup directory
  backup_interval: "24h"        # Backup frequency
  backup_retention: "7d"        # Keep backups for 7 days
  
  # Migration settings
  migration:
    enabled: true               # Enable automatic migrations
    auto_migrate: true          # Run migrations on startup
    timeout: "5m"               # Migration timeout
```

### Environment Variables

```bash
DATABASE_PATH=/data/pma.db
DATABASE_MAX_CONNECTIONS=25
DATABASE_BACKUP_ENABLED=true
DATABASE_BACKUP_PATH=/data/backups
```

## Authentication & Security

Configure authentication methods and security settings.

```yaml
auth:
  # Basic authentication
  enabled: true                 # Enable authentication
  jwt_secret: "your-secure-secret" # JWT signing secret (256+ bits)
  token_expiry: 1800           # Token expiry in seconds (30 minutes)
  
  # PIN authentication
  pin_enabled: true            # Enable PIN authentication
  pin_length: 4               # PIN length (4-8 digits)
  pin_expiry: 300             # PIN session expiry (5 minutes)
  
  # Session management
  max_sessions: 10            # Maximum concurrent sessions per user
  session_timeout: "24h"      # Session timeout
  refresh_enabled: true       # Enable token refresh
  
  # Password policies (if using password auth)
  password_min_length: 8      # Minimum password length
  password_require_special: true # Require special characters
  password_require_numbers: true # Require numbers

security:
  # Rate limiting
  rate_limiting:
    enabled: true             # Enable rate limiting
    requests_per_minute: 100  # Requests per minute per IP
    burst_size: 200          # Maximum burst size
    cleanup_interval: "5m"   # Cleanup interval for rate limit data
    
  # CORS settings
  cors:
    enabled: true            # Enable CORS
    allowed_origins:         # Allowed origins (use ["*"] for all)
      - "http://localhost:3000"
      - "http://localhost:3001"
    allowed_methods:         # Allowed HTTP methods
      - "GET"
      - "POST"
      - "PUT"
      - "DELETE"
      - "OPTIONS"
    allowed_headers:         # Allowed headers
      - "Content-Type"
      - "Authorization"
      - "X-Requested-With"
    allow_credentials: true  # Allow credentials
    max_age: 86400          # Preflight cache time
    
  # Security headers
  headers:
    enable_hsts: true        # Enable HTTP Strict Transport Security
    hsts_max_age: 31536000  # HSTS max age (1 year)
    enable_xss_protection: true # Enable XSS protection
    enable_content_type_nosniff: true # Prevent MIME sniffing
    enable_frame_deny: true  # Deny embedding in frames
```

### Environment Variables

```bash
JWT_SECRET=your-256-bit-secret
AUTH_ENABLED=true
AUTH_TOKEN_EXPIRY=1800
PIN_ENABLED=true
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_MINUTE=100
```

## Home Assistant Integration

Configure Home Assistant connection and synchronization.

```yaml
home_assistant:
  # Connection settings
  url: "http://homeassistant:8123"  # Home Assistant URL
  token: "your-long-lived-token"    # Long-lived access token
  verify_ssl: true                  # Verify SSL certificates
  timeout: "30s"                    # Request timeout
  
  # Synchronization settings
  sync:
    enabled: true                   # Enable sync service
    full_sync_interval: "1h"        # Full sync frequency
    incremental_sync_interval: "5m" # Incremental sync frequency
    
    # Entity filtering
    supported_domains:              # Supported entity domains
      - "light"
      - "switch"
      - "sensor"
      - "binary_sensor"
      - "climate"
      - "cover"
      - "lock"
      - "alarm_control_panel"
      - "camera"
      - "media_player"
      
    excluded_entities:              # Entities to exclude
      - "sensor.uptime"
      - "sensor.last_boot"
      
    # Conflict resolution
    conflict_resolution: "homeassistant_wins" # homeassistant_wins|pma_wins|prompt
    
    # Performance settings
    batch_size: 100                # Entities per batch
    retry_attempts: 3              # Retry failed requests
    retry_delay: "5s"              # Delay between retries
    concurrent_requests: 5         # Concurrent API requests
    
    # Event processing
    event_buffer_size: 1000        # Event buffer size
    event_processing_delay: "100ms" # Delay before processing events
    event_batch_size: 50           # Events per batch
    
  # WebSocket settings
  websocket:
    enabled: true                  # Enable HA WebSocket
    reconnect_interval: "30s"      # Reconnection interval
    ping_interval: "30s"           # Ping interval
    max_reconnect_attempts: 10     # Maximum reconnection attempts
```

### Environment Variables

```bash
HOME_ASSISTANT_URL=http://homeassistant:8123
HOME_ASSISTANT_TOKEN=your-token
HA_SYNC_ENABLED=true
HA_SYNC_INTERVAL=1h
HA_CONFLICT_RESOLUTION=homeassistant_wins
```

## AI Services

Configure AI/LLM providers and services.

```yaml
ai:
  # Global AI settings
  enabled: true                    # Enable AI services
  default_provider: "openai"      # Default provider
  max_tokens: 4000                # Default max tokens
  timeout: "30s"                  # Request timeout
  
  # Provider configurations
  providers:
    openai:
      enabled: true
      api_key: "your-openai-key"
      model: "gpt-4"
      base_url: "https://api.openai.com/v1" # Custom base URL
      organization: ""             # Organization ID
      temperature: 0.7            # Creativity (0.0-2.0)
      max_tokens: 4000           # Maximum tokens per request
      
    claude:
      enabled: true
      api_key: "your-claude-key"
      model: "claude-3-sonnet-20240229"
      base_url: "https://api.anthropic.com"
      temperature: 0.7
      max_tokens: 4000
      
    gemini:
      enabled: false
      api_key: "your-gemini-key"
      model: "gemini-pro"
      temperature: 0.7
      max_tokens: 4000
      
    ollama:
      enabled: false
      base_url: "http://localhost:11434"
      model: "llama2"
      temperature: 0.7
      max_tokens: 4000
      timeout: "60s"
      
  # MCP (Model Context Protocol) settings
  mcp:
    enabled: true                  # Enable MCP tools
    timeout: "30s"                 # Tool execution timeout
    max_tools_per_request: 5       # Maximum tools per request
    
  # Conversation settings
  conversations:
    max_history: 50               # Maximum messages per conversation
    cleanup_interval: "24h"       # Cleanup old conversations
    retention_days: 30           # Keep conversations for 30 days
```

### Environment Variables

```bash
AI_ENABLED=true
AI_DEFAULT_PROVIDER=openai
OPENAI_API_KEY=your-openai-key
CLAUDE_API_KEY=your-claude-key
GEMINI_API_KEY=your-gemini-key
OLLAMA_BASE_URL=http://localhost:11434
```

## Performance Configuration

Configure performance optimization settings.

```yaml
performance:
  # Database optimization
  database:
    enable_query_cache: true       # Enable query result caching
    cache_ttl: "30m"              # Cache TTL
    max_cache_size: "100MB"       # Maximum cache size
    slow_query_threshold: "1s"    # Log slow queries
    
  # Memory management
  memory:
    gc_target: 70                 # GC target percentage
    heap_limit: 1073741824       # Heap limit (1GB)
    enable_pooling: true         # Enable object pooling
    pool_size: 1000             # Pool size for common objects
    
  # API performance
  api:
    enable_compression: true      # Enable response compression
    compression_level: 6         # Compression level (1-9)
    max_request_size: 10485760   # Max request body size (10MB)
    request_timeout: "30s"       # Request timeout
    
  # Concurrency settings
  workers:
    automation_workers: 4        # Automation engine workers
    sync_workers: 2             # Sync service workers
    analytics_workers: 2        # Analytics workers
    
  # Caching configuration
  cache:
    enabled: true               # Enable caching
    default_ttl: "15m"         # Default cache TTL
    max_size: "500MB"          # Maximum cache size
    cleanup_interval: "5m"     # Cache cleanup interval
    
    # Specific cache configurations
    entity_cache:
      ttl: "5m"
      max_size: "100MB"
      
    room_cache:
      ttl: "30m"
      max_size: "50MB"
      
    analytics_cache:
      ttl: "1h"
      max_size: "200MB"
```

### Environment Variables

```bash
PERFORMANCE_CACHE_ENABLED=true
PERFORMANCE_COMPRESSION_ENABLED=true
PERFORMANCE_WORKERS=4
PERFORMANCE_HEAP_LIMIT=1073741824
```

## Logging Configuration

Configure application logging.

```yaml
logging:
  # Basic logging settings
  level: "info"                   # Log level: debug|info|warn|error
  format: "json"                  # Log format: json|text
  output: "stdout"                # Output: stdout|stderr|file
  
  # File logging (when output is "file")
  file:
    path: "./logs/pma.log"        # Log file path
    max_size: "100MB"             # Maximum log file size
    max_age: "30d"                # Maximum log file age
    max_backups: 10               # Maximum backup files
    compress: true                # Compress old log files
    
  # Structured logging
  structured:
    enabled: true                 # Enable structured logging
    include_caller: true          # Include caller information
    include_stack_trace: true     # Include stack traces for errors
    
  # Request logging
  requests:
    enabled: true                 # Log HTTP requests
    include_headers: false        # Include request headers
    include_body: false           # Include request body (development only)
    include_response: false       # Include response body
    
  # Component-specific logging
  components:
    database: "info"              # Database logging level
    websocket: "info"             # WebSocket logging level
    automation: "info"            # Automation logging level
    sync: "info"                  # Sync service logging level
    ai: "info"                    # AI service logging level
```

### Environment Variables

```bash
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT=stdout
LOG_FILE_PATH=./logs/pma.log
```

## WebSocket Configuration

Configure WebSocket server settings.

```yaml
websocket:
  # Connection settings
  max_connections: 1000           # Maximum concurrent connections
  ping_interval: "30s"            # Ping interval for keepalive
  pong_timeout: "60s"            # Pong timeout
  write_timeout: "10s"           # Write timeout
  read_timeout: "60s"            # Read timeout
  
  # Message settings
  max_message_size: 1024         # Maximum message size in bytes
  message_buffer_size: 256       # Message buffer per client
  enable_compression: true       # Enable message compression
  compression_threshold: 1024    # Compress messages larger than this
  
  # Home Assistant event forwarding
  homeassistant:
    enabled: true                # Enable HA event forwarding
    max_events_per_second: 100   # Rate limit HA events
    batch_events: true          # Batch multiple events
    batch_window: "100ms"       # Batch collection window
    
    # Default subscriptions for new clients
    default_subscriptions:
      - "state_changed"
      - "automation_triggered"
      
    forward_all_entities: false  # Forward all entity events
    max_errors_retained: 100    # Keep last N errors for debugging
    
  # Performance settings
  performance:
    enable_metrics: true        # Collect WebSocket metrics
    metrics_interval: "30s"     # Metrics collection interval
    cleanup_interval: "5m"     # Cleanup disconnected clients
```

### Environment Variables

```bash
WEBSOCKET_MAX_CONNECTIONS=1000
WEBSOCKET_PING_INTERVAL=30s
WEBSOCKET_COMPRESSION_ENABLED=true
WEBSOCKET_HA_ENABLED=true
```

## External Services

Configure external service integrations.

```yaml
external_services:
  # Ring security system
  ring:
    enabled: false               # Enable Ring integration
    username: "your-email"       # Ring account email
    password: "your-password"    # Ring account password
    two_factor_auth: false      # Enable 2FA support
    poll_interval: "30s"        # Polling interval
    
  # Shelly devices
  shelly:
    enabled: false              # Enable Shelly integration
    discovery_enabled: true     # Enable device discovery
    discovery_interval: "5m"    # Discovery interval
    device_timeout: "10s"       # Device communication timeout
    
  # UPS monitoring
  ups:
    enabled: false              # Enable UPS monitoring
    nut_host: "localhost"       # NUT server host
    nut_port: 3493             # NUT server port
    ups_name: "ups"            # UPS device name
    poll_interval: "30s"       # Polling interval
    history_retention_days: 30 # Keep history for 30 days
    
  # Network monitoring
  network:
    enabled: true              # Enable network monitoring
    router_url: "http://router" # Router management URL
    router_auth_token: ""      # Router authentication token
    scan_interval: "1h"        # Network scan interval
    ping_timeout: "5s"         # Ping timeout
    
  # Bluetooth LE
  bluetooth:
    enabled: false             # Enable Bluetooth LE
    scan_duration: "30s"       # Scan duration
    scan_interval: "5m"        # Scan interval
    device_timeout: "1m"       # Device timeout
```

### Environment Variables

```bash
RING_ENABLED=false
RING_USERNAME=your-email
RING_PASSWORD=your-password
SHELLY_ENABLED=false
UPS_ENABLED=false
UPS_NUT_HOST=localhost
NETWORK_ENABLED=true
BLUETOOTH_ENABLED=false
```

## Monitoring & Analytics

Configure monitoring and analytics features.

```yaml
monitoring:
  # Basic monitoring
  enabled: true                  # Enable monitoring
  metrics_interval: "30s"        # Metrics collection interval
  health_check_interval: "1m"    # Health check interval
  
  # Prometheus metrics
  prometheus:
    enabled: true               # Enable Prometheus metrics
    path: "/metrics"           # Metrics endpoint path
    listen_address: ":9090"    # Metrics server address
    
  # System monitoring
  system:
    cpu_threshold: 80.0        # CPU usage alert threshold
    memory_threshold: 85.0     # Memory usage alert threshold
    disk_threshold: 90.0       # Disk usage alert threshold
    
  # Application monitoring
  application:
    response_time_threshold: "1s" # Response time alert threshold
    error_rate_threshold: 5.0    # Error rate alert threshold (%)
    
analytics:
  # Basic analytics
  enabled: true                 # Enable analytics
  data_retention_days: 90      # Keep analytics data for 90 days
  aggregation_interval: "1h"   # Data aggregation interval
  
  # Performance analytics
  performance:
    enabled: true              # Enable performance analytics
    sample_rate: 0.1          # Sample 10% of requests
    slow_request_threshold: "1s" # Threshold for slow requests
    
  # Usage analytics
  usage:
    enabled: true             # Enable usage analytics
    track_entities: true      # Track entity usage
    track_automations: true   # Track automation usage
    track_api_usage: true     # Track API usage
    
  # Predictive analytics
  predictions:
    enabled: false            # Enable predictive analytics
    model_update_interval: "24h" # Update ML models every 24h
    prediction_horizon: "7d"  # Predict 7 days ahead
```

### Environment Variables

```bash
MONITORING_ENABLED=true
PROMETHEUS_ENABLED=true
ANALYTICS_ENABLED=true
ANALYTICS_RETENTION_DAYS=90
```

## Environment Variables

### Complete Environment Variable Reference

```bash
# Server Configuration
PORT=3001
HOST=0.0.0.0
SERVER_MODE=production

# Database Configuration
DATABASE_PATH=/data/pma.db
DATABASE_MAX_CONNECTIONS=25
DATABASE_BACKUP_ENABLED=true

# Authentication
JWT_SECRET=your-256-bit-secret
AUTH_ENABLED=true
PIN_ENABLED=true

# Home Assistant
HOME_ASSISTANT_URL=http://homeassistant:8123
HOME_ASSISTANT_TOKEN=your-long-lived-token
HA_SYNC_ENABLED=true

# AI Services
AI_ENABLED=true
OPENAI_API_KEY=your-openai-key
CLAUDE_API_KEY=your-claude-key

# Security
RATE_LIMIT_ENABLED=true
CORS_ENABLED=true

# Performance
PERFORMANCE_CACHE_ENABLED=true
PERFORMANCE_COMPRESSION_ENABLED=true

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# WebSocket
WEBSOCKET_MAX_CONNECTIONS=1000
WEBSOCKET_HA_ENABLED=true

# External Services
RING_ENABLED=false
UPS_ENABLED=false
NETWORK_ENABLED=true

# Monitoring
MONITORING_ENABLED=true
ANALYTICS_ENABLED=true
```

### Environment Variable Override Format

Environment variables use uppercase with underscores and follow this pattern:

```bash
# Nested configuration: section.subsection.option
# Becomes: SECTION_SUBSECTION_OPTION

# Examples:
home_assistant.url → HOME_ASSISTANT_URL
auth.jwt_secret → AUTH_JWT_SECRET
websocket.max_connections → WEBSOCKET_MAX_CONNECTIONS
performance.cache.enabled → PERFORMANCE_CACHE_ENABLED
```

## Production Configuration

### Recommended Production Settings

```yaml
# configs/config.production.yaml
server:
  mode: "production"
  port: 3001
  host: "0.0.0.0"

database:
  path: "/data/pma.db"
  max_connections: 50
  backup_enabled: true
  backup_path: "/data/backups"

auth:
  enabled: true
  jwt_secret: "${JWT_SECRET}"  # Read from environment
  token_expiry: 3600          # 1 hour

security:
  rate_limiting:
    enabled: true
    requests_per_minute: 200
  cors:
    enabled: true
    allowed_origins:
      - "https://your-domain.com"

performance:
  database:
    enable_query_cache: true
    cache_ttl: "1h"
  memory:
    gc_target: 80
    heap_limit: 2147483648     # 2GB
  api:
    enable_compression: true

logging:
  level: "info"
  format: "json"
  file:
    path: "/var/log/pma/pma.log"
    max_size: "500MB"
    max_backups: 5

monitoring:
  enabled: true
  prometheus:
    enabled: true
```

### Production Environment Variables

```bash
# Essential production environment variables
export APP_ENV=production
export JWT_SECRET="your-secure-256-bit-secret"
export HOME_ASSISTANT_URL="http://homeassistant:8123"
export HOME_ASSISTANT_TOKEN="your-long-lived-token"
export DATABASE_PATH="/data/pma.db"
export LOG_LEVEL="info"
export MONITORING_ENABLED=true
```

## Configuration Validation

The application validates configuration on startup and provides detailed error messages for invalid settings.

### Validation Rules

1. **Required Fields**: JWT secret, database path
2. **Format Validation**: URLs, durations, file paths
3. **Range Validation**: Numeric limits, percentages
4. **Dependency Validation**: Related settings consistency

### Configuration Testing

Test your configuration:

```bash
# Validate configuration
./pma-backend --validate-config

# Check specific configuration section
./pma-backend --validate-config --section=database

# Dry run with configuration
./pma-backend --dry-run
```

### Common Configuration Errors

1. **Invalid JWT Secret**: Must be at least 32 characters
2. **Database Path**: Must be writable directory
3. **Home Assistant URL**: Must be valid HTTP/HTTPS URL
4. **Port Conflicts**: Ensure ports are available
5. **File Permissions**: Log and data directories must be writable

---

For more information, see the [PMA Backend Go Documentation](../README.md) and [Deployment Guide](DEPLOYMENT.md).