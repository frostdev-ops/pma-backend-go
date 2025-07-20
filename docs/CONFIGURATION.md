# PMA Backend Go - Configuration Reference

This document provides a comprehensive reference for all configuration options in PMA Backend Go, including environment variables and production settings.

## Table of Contents

- [Configuration Layers](#configuration-layers)
- [Configuration Sections](#configuration-sections)
  - [Server](#server)
  - [Database](#database)
  - [Authentication](#authentication)
  - [Security](#security)
  - [Home Assistant](#home-assistant)
  - [AI Services](#ai-services)
  - [Device Integration](#device-integration)
  - [Logging](#logging)
  - [WebSocket](#websocket)
  - [Performance](#performance)
  - [System](#system)
  - [Storage](#storage)
  - [Test](#test)
- [Environment Variables](#environment-variables)
- [Production Configuration](#production-configuration)
- [Configuration Validation](#configuration-validation)

## Configuration Layers

Configuration is loaded in the following order of precedence:

1. **Command Line Flags**: Highest priority
2. **Environment Variables**: Override any config value
3. **Local Configuration**: `configs/config.local.yaml` (git-ignored)
4. **Default Configuration**: `configs/config.yaml` (version controlled)

## Configuration Sections

### Server

| Key | Type | Default | Description |
|---|---|---|---|
| `port` | int | 3001 | Server port |
| `host` | string | "0.0.0.0" | Bind address |
| `mode` | string | "development" | `development` or `production` |
| `shutdown_timeout` | string | "30s" | Graceful shutdown timeout |
| `read_timeout` | string | "15s" | Request read timeout |
| `write_timeout` | string | "15s" | Response write timeout |
| `max_header_bytes` | int | 1048576 | Max header size (1MB) |

### Database

| Key | Type | Default | Description |
|---|---|---|---|
| `path` | string | "./data/pma.db" | Database file path |
| `migrations_path` | string | "./migrations" | Migrations directory |
| `backup_enabled` | bool | true | Enable automatic backups |
| `backup_path` | string | "./data/backups" | Backup directory |
| `max_connections` | int | 25 | Max open connections |
| `max_idle_conns` | int | 10 | Max idle connections |
| `conn_max_lifetime` | string | "1h" | Max connection lifetime |
| `query_timeout` | string | "30s" | Default query timeout |
| `enable_query_cache` | bool | true | Enable query caching |
| `cache_ttl` | string | "30m" | Query cache TTL |

### Authentication

| Key | Type | Default | Description |
|---|---|---|---|
| `enabled` | bool | true | Enable authentication |
| `jwt_secret` | string | "secret" | JWT signing secret |
| `token_expiry` | int | 1800 | Token expiry in seconds |
| `refresh_enabled` | bool | true | Enable refresh tokens |
| `pin_required` | bool | false | Require PIN for sensitive ops |

### Security

| Key | Type | Default | Description |
|---|---|---|---|
| `rate_limiting.enabled` | bool | true | Enable rate limiting |
| `rate_limiting.requests_per_minute`| int | 100 | Requests per minute |
| `rate_limiting.burst_size` | int | 200 | Burst request size |
| `rate_limiting.whitelist_ips` | []string | [] | Whitelisted IPs |
| `cors.enabled` | bool | true | Enable CORS |
| `cors.allowed_origins` | []string | ["*"] | Allowed origins |
| `cors.allowed_methods` | []string | [...] | Allowed HTTP methods |

### Home Assistant

| Key | Type | Default | Description |
|---|---|---|---|
| `url` | string | "" | Home Assistant URL |
| `token` | string | "" | Long-lived access token |
| `sync.enabled` | bool | true | Enable synchronization |
| `sync.full_sync_interval` | string | "1h" | Full sync interval |
| `sync.supported_domains` | []string | [...] | Domains to sync |
| `sync.conflict_resolution` | string | "homeassistant_wins" | Conflict resolution strategy |
| `sync.batch_size` | int | 100 | Sync batch size |

### AI Services

| Key | Type | Default | Description |
|---|---|---|---|
| `fallback_enabled` | bool | true | Enable provider fallback |
| `default_provider` | string | "ollama" | Default AI provider |
| `max_retries` | int | 3 | Max retry attempts |
| `timeout` | string | "30s" | AI request timeout |
| `providers` | [] | | List of AI providers |

#### Provider Configuration

| Key | Type | Description |
|---|---|---|
| `type` | string | `ollama`, `openai`, `claude`, `gemini` |
| `enabled` | bool | Enable/disable provider |
| `url` | string | Provider URL (for Ollama) |
| `api_key` | string | API key for cloud providers |
| `default_model` | string | Default model to use |
| `priority` | int | Priority for fallback |

### Device Integration

| Key | Type | Default | Description |
|---|---|---|---|
| `health_check_interval` | string | "30s" | Device health check interval |
| `ring.enabled` | bool | false | Enable Ring integration |
| `ring.email` | string | "" | Ring account email |
| `ring.password` | string | "" | Ring account password |
| `shelly.enabled` | bool | false | Enable Shelly integration |
| `ups.enabled` | bool | false | Enable UPS/NUT integration |
| `network.enabled` | bool | true | Enable network discovery |
| `network.scan_subnets` | []string | [] | Subnets to scan |

### Logging

| Key | Type | Default | Description |
|---|---|---|---|
| `level` | string | "info" | `debug`, `info`, `warn`, `error` |
| `format` | string | "json" | `json` or `text` |
| `output` | string | "stdout" | `stdout`, `file`, `both` |
| `file_path` | string | "./logs/pma.log"| Log file path |
| `max_size` | int | 100 | Max size in MB |
| `max_backups` | int | 3 | Max backup files |
| `max_age` | int | 30 | Max age in days |
| `compress` | bool | true | Compress old log files |

### WebSocket

| Key | Type | Default | Description |
|---|---|---|---|
| `ping_interval` | int | 30 | Ping interval (seconds) |
| `pong_timeout` | int | 60 | Pong timeout (seconds) |
| `write_timeout` | int | 10 | Write timeout (seconds) |
| `read_buffer_size` | int | 1024 | Read buffer size |
| `write_buffer_size`| int | 1024 | Write buffer size |
| `max_message_size`| int | 512 | Max message size (KB) |
| `compression` | bool | true | Enable compression |

### Performance

| Key | Type | Default | Description |
|---|---|---|---|
| `memory.gc_target` | int | 70 | GC target percentage |
| `memory.heap_limit` | int | 1GB | Heap size limit |
| `api.enable_compression` | bool | true | Response compression |
| `api.rate_limit_requests` | int | 1000 | Requests per window |
| `api.rate_limit_window` | string | "1m" | Rate limit window |

### System

| Key | Type | Default | Description |
|---|---|---|---|
| `environment` | string | "development" | `development`, `production`, `testing` |
| `device_id_file` | string | "./data/device_id" | Device ID file path |
| `max_log_entries` | int | 1000 | Max log entries to keep |
| `performance_monitoring` | bool | true | Enable performance monitoring |

### Storage

| Key | Type | Default | Description |
|---|---|---|---|
| `base_path` | string | "./data" | Base data directory |
| `temp_path` | string | "./data/temp" | Temporary files path |
| `cache_path` | string | "./data/cache" | Cache storage path |
| `logs_path` | string | "./logs" | Logs directory |

### Test

| Key | Type | Default | Description |
|---|---|---|---|
| `endpoints_enabled` | bool | true | Enable test endpoints |
| `auto_generate_test_data`| bool | false | Generate test data on startup |
| `reset_data_on_startup`| bool | false | Reset database on startup |

## Environment Variables

All configuration values can be overridden with environment variables.

| Environment Variable | Configuration Key |
|---|---|
| `PORT` | `server.port` |
| `SERVER_MODE` | `server.mode` |
| `DATABASE_PATH` | `database.path` |
| `JWT_SECRET` | `auth.jwt_secret` |
| `HOME_ASSISTANT_URL` | `home_assistant.url` |
| `HOME_ASSISTANT_TOKEN` | `home_assistant.token` |
| `HA_TOKEN` | `home_assistant.token` |
| `OPENAI_API_KEY` | `ai.providers.openai.api_key` |
| `CLAUDE_API_KEY` | `ai.providers.claude.api_key` |
| `GEMINI_API_KEY` | `ai.providers.gemini.api_key` |
| `RING_EMAIL` | `devices.ring.email` |
| `RING_PASSWORD` | `devices.ring.password` |
| `SHELLY_PASSWORD` | `devices.shelly.password` |
| `LOG_LEVEL` | `logging.level` |
| `PMA_ALLOWED_ORIGINS` | `security.cors.allowed_origins` |

## Production Configuration

For production environments, it is recommended to create a `configs/config.production.yaml` file with optimized settings.

```yaml
# configs/config.production.yaml
server:
  mode: "production"
  shutdown_timeout: "60s"

database:
  max_connections: 50
  max_idle_conns: 25
  conn_max_lifetime: "2h"
  query_timeout: "10s"

performance:
  memory:
    gc_target: 60
    heap_limit: 2147483648 # 2GB
  api:
    rate_limit_requests: 5000
    rate_limit_window: "1m"
  websocket:
    max_connections: 2000

logging:
  level: "warn"
  format: "json"
  output: "file"
  file_path: "/var/log/pma/pma.log"

security:
  cors:
    allowed_origins: ["https://your-frontend-domain.com"]
```

## Configuration Validation

The application validates the configuration on startup. You can also validate it manually:

```bash
./bin/pma-server -config configs/config.local.yaml -validate
```