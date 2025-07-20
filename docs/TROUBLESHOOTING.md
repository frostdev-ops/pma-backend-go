# PMA Backend Go - Troubleshooting Guide

This document provides comprehensive troubleshooting guidance for common issues with PMA Backend Go.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Startup Issues](#startup-issues)
- [Connection Problems](#connection-problems)
- [Database Issues](#database-issues)
- [WebSocket Problems](#websocket-problems)
- [Authentication Issues](#authentication-issues)
- [Home Assistant Integration](#home-assistant-integration)
- [Performance Issues](#performance-issues)
- [Memory and Resource Problems](#memory-and-resource-problems)
- [AI Service Issues](#ai-service-issues)
- [Logging and Debugging](#logging-and-debugging)
- [Network and Connectivity](#network-and-connectivity)
- [Error Codes Reference](#error-codes-reference)
- [Diagnostic Tools](#diagnostic-tools)

## Quick Diagnostics

### Health Check

First, verify the application is running and healthy:

```bash
# Check if service is running
curl -s http://localhost:3001/health | jq .

# Check system status
curl -s http://localhost:3001/api/v1/system/status | jq .

# Check service status (systemd)
sudo systemctl status pma-backend

# Check process
ps aux | grep pma-backend
```

### Log Analysis

```bash
# Check recent logs
sudo journalctl -u pma-backend -n 50

# Follow logs in real-time
sudo journalctl -u pma-backend -f

# Check application logs
tail -f /opt/pma-backend/logs/pma.log | jq .

# Search for errors
grep -i error /opt/pma-backend/logs/pma.log | tail -10
```

### Configuration Validation

```bash
# Validate configuration
./pma-backend --validate-config

# Check specific configuration section
./pma-backend --validate-config --section=database

# Test configuration loading
./pma-backend --dry-run
```

## Startup Issues

### Service Fails to Start

**Symptoms:**
- Service shows "failed" status
- Application exits immediately
- No response on port 3001

**Diagnostic Steps:**

```bash
# Check service status
sudo systemctl status pma-backend

# Check recent logs
sudo journalctl -u pma-backend -n 20

# Try running manually
sudo -u pma /opt/pma-backend/pma-backend

# Check file permissions
ls -la /opt/pma-backend/
ls -la /opt/pma-backend/data/
```

**Common Solutions:**

1. **Permission Issues:**
```bash
sudo chown -R pma:pma /opt/pma-backend
sudo chmod 755 /opt/pma-backend/pma-backend
sudo chmod 700 /opt/pma-backend/data
```

2. **Missing Configuration:**
```bash
# Create missing directories
sudo mkdir -p /opt/pma-backend/{data,logs,configs}
sudo chown pma:pma /opt/pma-backend/{data,logs}

# Copy default configuration
sudo cp configs/config.yaml /opt/pma-backend/configs/
```

3. **Port Already in Use:**
```bash
# Check what's using port 3001
sudo netstat -tlnp | grep :3001
sudo lsof -i :3001

# Kill conflicting process or change port
```

### Configuration Errors

**Symptoms:**
- Application starts but exits with config error
- "Invalid configuration" messages

**Common Issues:**

1. **Invalid JWT Secret:**
```yaml
# Must be at least 32 characters
auth:
  jwt_secret: "your-secure-256-bit-secret-key-here"
```

2. **Database Path Issues:**
```bash
# Ensure directory exists and is writable
mkdir -p /opt/pma-backend/data
chown pma:pma /opt/pma-backend/data
chmod 700 /opt/pma-backend/data
```

3. **Environment Variable Override:**
```bash
# Check environment variables
sudo systemctl show pma-backend --property=Environment
```

### Binary Issues

**Symptoms:**
- "Permission denied" when executing
- "No such file or directory"

**Solutions:**

1. **Executable Permissions:**
```bash
chmod +x /opt/pma-backend/pma-backend
```

2. **Missing Dependencies:**
```bash
# Check required libraries
ldd /opt/pma-backend/pma-backend

# Install missing dependencies (Ubuntu/Debian)
sudo apt install libc6 libsqlite3-0
```

3. **Architecture Mismatch:**
```bash
# Check binary architecture
file /opt/pma-backend/pma-backend

# Check system architecture
uname -m
```

## Connection Problems

### Cannot Connect to API

**Symptoms:**
- "Connection refused" errors
- Timeouts when accessing API
- 502 Bad Gateway from reverse proxy

**Diagnostic Steps:**

```bash
# Test local connection
curl -v http://localhost:3001/health

# Check if port is open
sudo netstat -tlnp | grep :3001

# Test from different host
curl -v http://your-server-ip:3001/health

# Check firewall
sudo ufw status
sudo iptables -L
```

**Solutions:**

1. **Service Not Running:**
```bash
sudo systemctl start pma-backend
sudo systemctl enable pma-backend
```

2. **Firewall Blocking:**
```bash
# Open port 3001
sudo ufw allow 3001/tcp

# Or if using iptables
sudo iptables -A INPUT -p tcp --dport 3001 -j ACCEPT
```

3. **Binding Issues:**
```yaml
# Check server configuration
server:
  host: "0.0.0.0"  # Not 127.0.0.1 for external access
  port: 3001
```

### Reverse Proxy Issues

**Symptoms:**
- 502 Bad Gateway
- Proxy timeout errors
- SSL/TLS errors

**Nginx Troubleshooting:**

```bash
# Test nginx configuration
sudo nginx -t

# Check nginx error logs
sudo tail -f /var/log/nginx/error.log

# Check backend connectivity from nginx
sudo -u www-data curl http://127.0.0.1:3001/health
```

**Common Nginx Fixes:**

1. **Backend Connection:**
```nginx
upstream pma_backend {
    server 127.0.0.1:3001 max_fails=3 fail_timeout=30s;
}
```

2. **Timeout Settings:**
```nginx
proxy_connect_timeout 5s;
proxy_send_timeout 30s;
proxy_read_timeout 30s;
```

3. **SSL Issues:**
```bash
# Check SSL certificate
sudo certbot certificates

# Renew certificate
sudo certbot renew
```

## Database Issues

### Database Connection Errors

**Symptoms:**
- "Database locked" errors
- "No such table" errors
- Slow database queries

**Diagnostic Steps:**

```bash
# Check database file
ls -la /opt/pma-backend/data/pma.db

# Test database connectivity
sqlite3 /opt/pma-backend/data/pma.db "SELECT 1;"

# Check database integrity
sqlite3 /opt/pma-backend/data/pma.db "PRAGMA integrity_check;"

# Check database size
du -h /opt/pma-backend/data/pma.db*
```

**Common Solutions:**

1. **Database Locked:**
```bash
# Check for stale locks
ls -la /opt/pma-backend/data/pma.db*

# Kill WAL mode locks
sqlite3 /opt/pma-backend/data/pma.db "PRAGMA wal_checkpoint(TRUNCATE);"
```

2. **Corrupted Database:**
```bash
# Backup current database
cp /opt/pma-backend/data/pma.db /opt/pma-backend/data/pma.db.backup

# Try to repair
sqlite3 /opt/pma-backend/data/pma.db "VACUUM;"

# If severely corrupted, restore from backup
```

3. **Missing Tables (Migration Issues):**
```bash
# Check migration status
./pma-backend --migrate-status

# Run migrations manually
./pma-backend --migrate
```

### Migration Problems

**Symptoms:**
- Application fails to start after update
- "Migration failed" errors
- Schema version mismatch

**Solutions:**

1. **Manual Migration:**
```bash
# Check current schema version
sqlite3 /opt/pma-backend/data/pma.db "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;"

# Run specific migration
./pma-backend --migrate-to=015

# Force migration (use with caution)
./pma-backend --migrate --force
```

2. **Reset Database (Development Only):**
```bash
# Backup existing data
cp /opt/pma-backend/data/pma.db /opt/pma-backend/data/pma.db.backup

# Remove database and recreate
rm /opt/pma-backend/data/pma.db
./pma-backend --migrate
```

### Performance Issues

**Symptoms:**
- Slow API responses
- High CPU usage
- Large database file

**Solutions:**

1. **Database Optimization:**
```sql
-- Analyze tables
ANALYZE;

-- Vacuum database
VACUUM;

-- Check query plan for slow queries
EXPLAIN QUERY PLAN SELECT * FROM entities WHERE state = 'on';
```

2. **Enable Query Cache:**
```yaml
performance:
  database:
    enable_query_cache: true
    cache_ttl: "30m"
    max_cache_size: "100MB"
```

3. **Increase Connection Pool:**
```yaml
database:
  max_connections: 50
  max_idle_conns: 10
```

## WebSocket Problems

### WebSocket Connection Fails

**Symptoms:**
- Cannot establish WebSocket connection
- Connection drops frequently
- No real-time updates

**Diagnostic Steps:**

```bash
# Test WebSocket connection
wscat -c ws://localhost:3001/ws

# Check WebSocket metrics
curl http://localhost:3001/api/v1/websocket/metrics

# Check connected clients
curl http://localhost:3001/api/v1/websocket/clients
```

**Common Solutions:**

1. **Reverse Proxy Configuration:**
```nginx
# Nginx WebSocket configuration
location /ws {
    proxy_pass http://pma_backend;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 3600s;
    proxy_send_timeout 3600s;
}
```

2. **Firewall Issues:**
```bash
# Ensure WebSocket port is open
sudo ufw allow 3001/tcp
```

3. **Connection Limits:**
```yaml
websocket:
  max_connections: 1000
  message_buffer_size: 256
```

### WebSocket Message Issues

**Symptoms:**
- Messages not received
- Partial message delivery
- High memory usage

**Solutions:**

1. **Subscription Management:**
```javascript
// Subscribe to specific events only
ws.send(JSON.stringify({
  type: 'subscribe_ha_events',
  data: { event_types: ['state_changed'] }
}));
```

2. **Message Buffer Size:**
```yaml
websocket:
  message_buffer_size: 512  # Increase buffer
  max_message_size: 2048    # Increase max message size
```

3. **Connection Cleanup:**
```bash
# Check for stale connections
curl http://localhost:3001/api/v1/websocket/clients | jq length

# Restart service to clean up
sudo systemctl restart pma-backend
```

## Authentication Issues

### JWT Token Problems

**Symptoms:**
- "Invalid token" errors
- Token expiration issues
- Authentication fails

**Diagnostic Steps:**

```bash
# Test authentication
curl -X POST http://localhost:3001/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'

# Validate token
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://localhost:3001/api/v1/auth/validate
```

**Solutions:**

1. **JWT Secret Issues:**
```bash
# Generate new JWT secret
openssl rand -base64 32

# Update configuration
JWT_SECRET=your-new-secret
```

2. **Token Expiration:**
```yaml
auth:
  token_expiry: 3600  # Increase to 1 hour
```

3. **Clock Skew:**
```bash
# Synchronize system time
sudo ntpdate -s time.nist.gov
# Or use systemd-timesyncd
sudo systemctl enable systemd-timesyncd
sudo systemctl start systemd-timesyncd
```

### PIN Authentication Issues

**Symptoms:**
- PIN verification fails
- PIN not accepted
- PIN session expires quickly

**Solutions:**

1. **PIN Configuration:**
```yaml
auth:
  pin_enabled: true
  pin_length: 4
  pin_expiry: 300  # 5 minutes
```

2. **Reset PIN:**
```bash
# Reset PIN through API (if authenticated)
curl -X POST http://localhost:3001/api/v1/auth/set-pin \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"pin":"1234"}'
```

## Home Assistant Integration

### Connection Issues

**Symptoms:**
- "Home Assistant unreachable"
- Sync failures
- Entity data not updating

**Diagnostic Steps:**

```bash
# Test Home Assistant connection
curl -H "Authorization: Bearer YOUR_HA_TOKEN" \
  http://your-ha-instance:8123/api/

# Check Home Assistant health
curl -H "Authorization: Bearer YOUR_HA_TOKEN" \
  http://your-ha-instance:8123/api/config

# Check sync status
curl http://localhost:3001/api/v1/adapters/homeassistant/health
```

**Solutions:**

1. **URL Configuration:**
```yaml
home_assistant:
  url: "http://homeassistant.local:8123"  # Use proper hostname/IP
  token: "your-long-lived-access-token"
  verify_ssl: false  # If using self-signed certificates
```

2. **Network Connectivity:**
```bash
# Test network connectivity
ping homeassistant.local
telnet homeassistant.local 8123

# Check DNS resolution
nslookup homeassistant.local
```

3. **Token Issues:**
```bash
# Verify token in Home Assistant
# Go to Profile -> Long-Lived Access Tokens
# Create new token if needed
```

### Sync Issues

**Symptoms:**
- Entities not synchronizing
- Partial entity data
- Sync process stuck

**Solutions:**

1. **Restart Sync Service:**
```bash
# Force full sync via API
curl -X POST http://localhost:3001/api/v1/adapters/homeassistant/sync

# Restart application
sudo systemctl restart pma-backend
```

2. **Adjust Sync Settings:**
```yaml
home_assistant:
  sync:
    full_sync_interval: "30m"  # More frequent sync
    batch_size: 50            # Smaller batches
    retry_attempts: 5         # More retries
```

3. **Entity Filtering:**
```yaml
home_assistant:
  sync:
    supported_domains:
      - "light"
      - "switch"
      - "sensor"
    excluded_entities:
      - "sensor.uptime"  # Exclude problematic entities
```

## Performance Issues

### High CPU Usage

**Symptoms:**
- CPU usage above 80%
- Slow API responses
- System becomes unresponsive

**Diagnostic Steps:**

```bash
# Monitor CPU usage
top -p $(pgrep pma-backend)
htop

# Profile application
curl http://localhost:3001/api/v1/performance/profile
```

**Solutions:**

1. **Reduce Worker Count:**
```yaml
performance:
  workers:
    automation_workers: 2  # Reduce from default 4
    sync_workers: 1        # Reduce from default 2
```

2. **Optimize Sync Frequency:**
```yaml
home_assistant:
  sync:
    full_sync_interval: "2h"      # Less frequent
    incremental_sync_interval: "10m"  # Less frequent
```

3. **Enable Caching:**
```yaml
performance:
  cache:
    enabled: true
    default_ttl: "30m"
    max_size: "200MB"
```

### High Memory Usage

**Symptoms:**
- Memory usage growing over time
- Out of memory errors
- System swapping

**Solutions:**

1. **Set Memory Limits:**
```yaml
performance:
  memory:
    heap_limit: 1073741824  # 1GB limit
    gc_target: 70          # More aggressive GC
```

2. **Reduce Cache Size:**
```yaml
performance:
  cache:
    max_size: "100MB"  # Reduce cache size
    cleanup_interval: "2m"  # More frequent cleanup
```

3. **Monitor Memory Leaks:**
```bash
# Check memory usage over time
while true; do
  ps -p $(pgrep pma-backend) -o pid,vsz,rss,pmem
  sleep 60
done
```

### Slow API Responses

**Symptoms:**
- API requests taking > 1 second
- Timeouts from clients
- Poor user experience

**Solutions:**

1. **Database Optimization:**
```sql
-- Analyze and optimize
ANALYZE;
VACUUM;

-- Add missing indexes
CREATE INDEX IF NOT EXISTS idx_entities_state ON entities(state);
CREATE INDEX IF NOT EXISTS idx_entities_domain ON entities(domain);
```

2. **Enable Compression:**
```yaml
performance:
  api:
    enable_compression: true
    compression_level: 6
```

3. **Increase Timeouts:**
```yaml
server:
  read_timeout: "60s"
  write_timeout: "60s"
```

## Memory and Resource Problems

### Memory Leaks

**Symptoms:**
- Continuously increasing memory usage
- Eventually runs out of memory
- Performance degrades over time

**Diagnostic Steps:**

```bash
# Monitor memory over time
watch -n 5 'ps -p $(pgrep pma-backend) -o pid,vsz,rss,pmem'

# Check garbage collection stats
curl http://localhost:3001/api/v1/performance/memory

# Enable memory profiling
curl http://localhost:3001/api/v1/performance/profile?type=heap
```

**Solutions:**

1. **Adjust GC Settings:**
```yaml
performance:
  memory:
    gc_target: 60        # More aggressive GC
    enable_pooling: true # Enable object pooling
```

2. **Restart Service Periodically:**
```bash
# Add to crontab for weekly restart
0 2 * * 0 /bin/systemctl restart pma-backend
```

3. **Monitor and Alert:**
```bash
# Monitor memory usage script
#!/bin/bash
MEMORY_USAGE=$(ps -p $(pgrep pma-backend) -o pmem --no-headers | tr -d ' ')
if (( $(echo "$MEMORY_USAGE > 80" | bc -l) )); then
    echo "High memory usage: $MEMORY_USAGE%"
    # Send alert or restart service
fi
```

### File Descriptor Limits

**Symptoms:**
- "Too many open files" errors
- Connection refused errors
- WebSocket connection failures

**Solutions:**

1. **Increase System Limits:**
```bash
# Add to /etc/security/limits.conf
pma soft nofile 65536
pma hard nofile 65536

# Add to systemd service
echo "LimitNOFILE=65536" >> /etc/systemd/system/pma-backend.service
sudo systemctl daemon-reload
sudo systemctl restart pma-backend
```

2. **Check Current Usage:**
```bash
# Check file descriptor usage
lsof -p $(pgrep pma-backend) | wc -l

# Check limits
cat /proc/$(pgrep pma-backend)/limits | grep files
```

## AI Service Issues

### AI Provider Connection Issues

**Symptoms:**
- AI requests failing
- "Provider not available" errors
- Timeout errors

**Solutions:**

1. **Check API Keys:**
```bash
# Test OpenAI connection
curl -H "Authorization: Bearer $OPENAI_API_KEY" \
  https://api.openai.com/v1/models

# Test Claude connection
curl -H "x-api-key: $CLAUDE_API_KEY" \
  https://api.anthropic.com/v1/messages
```

2. **Provider Configuration:**
```yaml
ai:
  providers:
    openai:
      enabled: true
      api_key: "your-key"
      timeout: "60s"  # Increase timeout
      base_url: "https://api.openai.com/v1"  # Ensure correct URL
```

3. **Network Issues:**
```bash
# Test connectivity to AI providers
curl -I https://api.openai.com/v1/models
curl -I https://api.anthropic.com/v1/messages
```

### AI Request Failures

**Symptoms:**
- AI responses are empty
- Request timeout errors
- Rate limiting errors

**Solutions:**

1. **Adjust Request Parameters:**
```yaml
ai:
  max_tokens: 2000      # Reduce token count
  timeout: "120s"       # Increase timeout
  default_provider: "openai"  # Use reliable provider
```

2. **Handle Rate Limits:**
```yaml
ai:
  providers:
    openai:
      rate_limit: 60    # Requests per minute
      retry_attempts: 3
      retry_delay: "2s"
```

## Logging and Debugging

### Enable Debug Logging

```yaml
logging:
  level: "debug"
  components:
    websocket: "debug"
    automation: "debug"
    sync: "debug"
    ai: "debug"
```

### Structured Logging Analysis

```bash
# Search for specific errors
jq 'select(.level == "error")' /opt/pma-backend/logs/pma.log

# Filter by component
jq 'select(.component == "websocket")' /opt/pma-backend/logs/pma.log

# Find slow requests
jq 'select(.response_time > 1000)' /opt/pma-backend/logs/pma.log

# Monitor error patterns
grep '"level":"error"' /opt/pma-backend/logs/pma.log | jq .message | sort | uniq -c
```

### Debug Mode

```bash
# Run in debug mode
./pma-backend --debug --log-level=debug

# Enable profiling
./pma-backend --enable-profiling --profile-port=6060

# Access profiling data
go tool pprof http://localhost:6060/debug/pprof/profile
```

## Network and Connectivity

### DNS Resolution Issues

**Symptoms:**
- Cannot resolve hostnames
- "No such host" errors
- Intermittent connectivity

**Solutions:**

1. **Test DNS Resolution:**
```bash
nslookup homeassistant.local
dig homeassistant.local
host homeassistant.local
```

2. **Use IP Addresses:**
```yaml
home_assistant:
  url: "http://192.168.1.100:8123"  # Use IP instead of hostname
```

3. **Configure DNS:**
```bash
# Add to /etc/hosts
echo "192.168.1.100 homeassistant.local" >> /etc/hosts
```

### Network Timeout Issues

**Symptoms:**
- Random timeout errors
- Intermittent connection failures
- Slow external API calls

**Solutions:**

1. **Increase Timeouts:**
```yaml
home_assistant:
  timeout: "60s"
  
ai:
  timeout: "120s"
  
external_services:
  ring:
    timeout: "30s"
```

2. **Test Network Latency:**
```bash
# Test latency to Home Assistant
ping -c 10 homeassistant.local

# Test HTTP response time
curl -w "@curl-format.txt" -o /dev/null http://homeassistant.local:8123/api/

# curl-format.txt content:
#     time_namelookup:  %{time_namelookup}\n
#        time_connect:  %{time_connect}\n
#     time_appconnect:  %{time_appconnect}\n
#    time_pretransfer:  %{time_pretransfer}\n
#       time_redirect:  %{time_redirect}\n
#  time_starttransfer:  %{time_starttransfer}\n
#                     ----------\n
#          time_total:  %{time_total}\n
```

## Error Codes Reference

### HTTP Error Codes

| Code | Description | Common Causes |
|------|-------------|---------------|
| 400 | Bad Request | Invalid JSON, missing parameters |
| 401 | Unauthorized | Invalid/missing JWT token |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Entity/endpoint doesn't exist |
| 409 | Conflict | Resource already exists |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Application error, check logs |
| 502 | Bad Gateway | Reverse proxy can't connect |
| 503 | Service Unavailable | Service overloaded/down |

### Application Error Codes

| Code | Message | Solution |
|------|---------|----------|
| `ENTITY_NOT_FOUND` | Entity not found | Check entity ID, sync with HA |
| `INVALID_ACTION` | Invalid action | Check supported actions for entity type |
| `CONFIG_ERROR` | Configuration error | Validate configuration file |
| `DB_CONNECTION_FAILED` | Database connection failed | Check database file permissions |
| `HA_CONNECTION_FAILED` | Home Assistant connection failed | Check HA URL and token |
| `AUTH_FAILED` | Authentication failed | Check credentials/token |
| `RATE_LIMIT_EXCEEDED` | Rate limit exceeded | Reduce request frequency |

### WebSocket Error Codes

| Code | Description | Solution |
|------|-------------|----------|
| 1000 | Normal Closure | Expected closure |
| 1001 | Going Away | Server restart/client disconnect |
| 1002 | Protocol Error | Invalid message format |
| 1003 | Unsupported Data | Invalid message type |
| 1006 | Abnormal Closure | Network issues |
| 1011 | Internal Error | Server error, check logs |

## Diagnostic Tools

### Built-in Diagnostics

```bash
# System information
curl http://localhost:3001/api/v1/system/info

# Performance metrics
curl http://localhost:3001/api/v1/performance/status

# Cache statistics
curl http://localhost:3001/api/v1/cache/stats

# WebSocket metrics
curl http://localhost:3001/api/v1/websocket/metrics

# Adapter health
curl http://localhost:3001/api/v1/adapters/health
```

### External Tools

```bash
# Network connectivity
nc -zv localhost 3001

# HTTP testing
httpie http://localhost:3001/health

# WebSocket testing
wscat -c ws://localhost:3001/ws

# SSL/TLS testing
openssl s_client -connect your-domain.com:443

# Database testing
sqlite3 /opt/pma-backend/data/pma.db ".schema"
```

### Monitoring Scripts

Create monitoring script `/opt/pma-backend/scripts/monitor.sh`:

```bash
#!/bin/bash

echo "=== PMA Backend Health Check ==="
echo "Date: $(date)"
echo

# Check service status
echo "Service Status:"
systemctl is-active pma-backend
echo

# Check API health
echo "API Health:"
curl -s http://localhost:3001/health | jq . || echo "API not responding"
echo

# Check memory usage
echo "Memory Usage:"
ps -p $(pgrep pma-backend) -o pid,vsz,rss,pmem 2>/dev/null || echo "Process not found"
echo

# Check disk space
echo "Disk Usage:"
df -h /opt/pma-backend/data
echo

# Check recent errors
echo "Recent Errors:"
journalctl -u pma-backend --since "1 hour ago" | grep -i error | tail -5
echo

echo "=== End Health Check ==="
```

### Log Analysis Script

Create log analysis script `/opt/pma-backend/scripts/analyze_logs.sh`:

```bash
#!/bin/bash

LOG_FILE="/opt/pma-backend/logs/pma.log"
HOURS=${1:-24}

echo "=== Log Analysis (Last ${HOURS} hours) ==="

# Error summary
echo "Error Summary:"
jq -r 'select(.level == "error") | .message' "$LOG_FILE" | sort | uniq -c | sort -nr
echo

# Performance metrics
echo "Average Response Time:"
jq -r 'select(.response_time) | .response_time' "$LOG_FILE" | awk '{sum+=$1; count++} END {print sum/count " ms"}'
echo

# Top endpoints
echo "Top Endpoints:"
jq -r 'select(.path) | .path' "$LOG_FILE" | sort | uniq -c | sort -nr | head -10
echo

# WebSocket metrics
echo "WebSocket Connections:"
jq -r 'select(.component == "websocket") | .message' "$LOG_FILE" | grep -E "(connected|disconnected)" | wc -l
echo

echo "=== End Analysis ==="
```

For additional help, check the [PMA Backend Go Documentation](../README.md) or open an issue on GitHub.