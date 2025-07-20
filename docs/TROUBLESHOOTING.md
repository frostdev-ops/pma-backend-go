# PMA Backend Go - Troubleshooting Guide

This document provides a comprehensive guide to troubleshooting common issues with PMA Backend Go, from startup problems to performance degradation.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Startup Issues](#startup-issues)
- [Connection Problems](#connection-problems)
- [Database Issues](#database-issues)
- [WebSocket Issues](#websocket-issues)
- [Authentication and Authorization](#authentication-and-authorization)
- [Home Assistant Integration](#home-assistant-integration)
- [Performance Problems](#performance-problems)
- [Memory Issues](#memory-issues)
- [AI Service Issues](#ai-service-issues)
- [Logging and Debugging](#logging-and-debugging)
- [Network Connectivity](#network-connectivity)
- [Common Error Codes](#common-error-codes)
- [Diagnostic Tools](#diagnostic-tools)

## Quick Diagnostics

```bash
# 1. Check service status
sudo systemctl status pma-backend

# 2. Check health endpoint
curl http://localhost:3001/health

# 3. Check logs for recent errors
sudo journalctl -u pma-backend -n 100 --no-pager | grep ERROR

# 4. Validate configuration
./bin/pma-server -config configs/config.local.yaml -validate
```

## Startup Issues

#### Problem: Service fails to start

1. **Check Logs**: `sudo journalctl -u pma-backend -f` for detailed error messages.
2. **Configuration**:
   - Validate syntax: `./bin/pma-server -config ... -validate`
   - Check for missing required values (e.g., `jwt_secret`).
3. **Permissions**:
   - Ensure the `pma` user has read/write access to `data/` and `logs/`.
   - `sudo chown -R pma:pma /opt/pma-backend`
4. **Port Conflicts**:
   - Check if port 3001 is in use: `sudo netstat -tlnp | grep 3001`

#### Problem: Database migration fails

1. **Check Logs**: Look for migration-specific errors.
2. **Database Permissions**: Ensure the `pma` user can write to `data/pma.db`.
3. **Corrupt Migration**: If a migration is corrupt, you may need to manually fix it or restore from backup.
4. **Manual Migration**: Run migrations manually for more detailed output:
   `make migrate`

## Connection Problems

#### Problem: Cannot connect to the API

1. **Service Status**: Check if the service is running (`systemctl status`).
2. **Firewall**: Ensure port 3001 (or your configured port) is open.
   - `sudo ufw status`
3. **Reverse Proxy**: If using Nginx/Apache, check its logs for errors.
   - `sudo tail -f /var/log/nginx/error.log`
4. **Network**: Use `ping` and `curl` to test network connectivity.

## Database Issues

#### Problem: Slow queries or high database load

1. **Enable Slow Query Logging**: Set `performance.database.slow_query_threshold` in your config.
2. **Analyze Queries**: Use `EXPLAIN QUERY PLAN` in SQLite to analyze slow queries.
3. **Indexing**: Ensure your database has appropriate indexes for common queries.
4. **Optimize Database**: Run `VACUUM; ANALYZE;` periodically.
5. **Hardware**: Check for disk I/O bottlenecks.

#### Problem: Database is locked

1. **Check for Long-Running Transactions**: Review logs for queries that may be holding locks.
2. **WAL Mode**: Ensure WAL mode is enabled for better concurrency.
3. **Increase Busy Timeout**: Adjust `database.busy_timeout`.

## WebSocket Issues

#### Problem: WebSocket connection fails

1. **Check Logs**: Look for WebSocket-related errors.
2. **Reverse Proxy**: Ensure your reverse proxy is configured for WebSocket proxying (`Upgrade` and `Connection` headers).
3. **Client-Side Issues**: Check the browser console for connection errors.

#### Problem: Messages are not being received

1. **Subscription**: Verify that the client is subscribed to the correct topics.
2. **Server Logs**: Check for errors in message broadcasting.
3. **Connection State**: Ensure the WebSocket connection is stable.

## Authentication and Authorization

#### Problem: Login fails

1. **Check Credentials**: Verify username and password.
2. **JWT Secret**: Ensure `jwt_secret` is correctly configured and consistent across all instances.
3. **Logs**: Look for authentication errors in the logs.

#### Problem: API requests return 401 Unauthorized

1. **Token**: Ensure the `Authorization: Bearer <token>` header is present and correct.
2. **Token Expiration**: Check if the JWT has expired.
3. **Token Validity**: Verify that the token was signed with the correct secret.

## Home Assistant Integration

#### Problem: Home Assistant entities are not syncing

1. **Check HA Configuration**: Verify URL and long-lived access token.
2. **Test HA API**: `curl -H "Authorization: Bearer YOUR_HA_TOKEN" http://your-ha/api/`
3. **Logs**: Look for synchronization errors in the PMA Backend logs.
4. **Network**: Ensure PMA Backend can reach the Home Assistant instance.

## Performance Problems

#### Problem: High CPU usage

1. **Profiling**: Use `go tool pprof` to identify CPU-intensive functions.
2. **Check for Infinite Loops**: Review recent code changes for potential loops.
3. **Automation Rules**: A misconfigured automation rule can cause high CPU usage.
4. **Resource Limits**: Ensure the server has adequate CPU resources.

## Memory Issues

#### Problem: High memory usage or memory leaks

1. **Profiling**: Use `go tool pprof` to analyze heap allocations.
2. **Goroutine Leaks**: Check for a constantly increasing number of goroutines.
3. **Cache Configuration**: Review cache sizes and eviction policies.
4. **GC Tuning**: Adjust `GOGC` and `GOMEMLIMIT` environment variables.

## AI Service Issues

#### Problem: AI chat or completion fails

1. **Check API Keys**: Ensure API keys for AI providers are correct.
2. **Provider Status**: Check the status of the AI provider (e.g., OpenAI, Ollama).
3. **Network**: Ensure the PMA Backend can reach the AI provider's API.
4. **Logs**: Look for detailed error messages from the AI provider.

## Logging and Debugging

#### Enabling Debug Mode

Set `logging.level: "debug"` in your configuration for more detailed logs.

#### Analyzing Logs

```bash
# Monitor real-time logs with JSON formatting
sudo journalctl -u pma-backend -f -o json-pretty

# Search for errors
grep "level=error" logs/pma.log
```

## Network Connectivity

Use standard networking tools to diagnose issues:

- `ping <host>`: Test basic connectivity.
- `traceroute <host>`: Trace the network path.
- `netstat -tlnp`: Check for listening ports.
- `ss -tulwn`: Socket statistics.

## Common Error Codes

| Code | Description | Solution |
|---|---|---|
| 1001 | Database connection failed | Check path and permissions |
| 1002 | Configuration invalid | Validate configuration syntax |
| 1003 | Home Assistant unreachable | Check URL and token |
| 1004 | Authentication failed | Verify JWT secret and token |
| 1005 | WebSocket connection limit | Increase `max_connections` |

## Diagnostic Tools

- **pprof**: For performance profiling (CPU, memory).
- **curl**: For testing API endpoints.
- **websocat**: For testing WebSocket connections.
- **sqlite3**: For direct database inspection.