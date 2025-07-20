# PMA Backend Go - Deployment Guide

This document provides comprehensive guidance for deploying PMA Backend Go to production environments.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Production Build](#production-build)
- [Docker Deployment](#docker-deployment)
- [Systemd Service](#systemd-service)
- [Reverse Proxy Setup](#reverse-proxy-setup)
- [Database Setup](#database-setup)
- [Environment Configuration](#environment-configuration)
- [SSL/TLS Configuration](#ssltls-configuration)
- [Monitoring & Logging](#monitoring--logging)
- [Backup & Recovery](#backup--recovery)
- [Security Hardening](#security-hardening)
- [Performance Optimization](#performance-optimization)
- [Health Checks](#health-checks)
- [Troubleshooting](#troubleshooting)

## Overview

PMA Backend Go is designed for easy deployment with minimal dependencies. This guide covers multiple deployment scenarios from single-server deployments to containerized environments.

### Deployment Options

1. **Single Binary**: Direct deployment with systemd
2. **Docker Container**: Containerized deployment with Docker Compose
3. **Kubernetes**: Scalable container orchestration
4. **Cloud Platforms**: AWS, Google Cloud, Azure deployments

## Prerequisites

### System Requirements

- **OS**: Linux (Ubuntu 20.04+, CentOS 8+, RHEL 8+)
- **CPU**: 2+ cores (4+ recommended)
- **RAM**: 2GB minimum (4GB+ recommended)
- **Storage**: 10GB minimum (SSD recommended)
- **Network**: Internet access for external integrations

### Software Dependencies

- **Go**: 1.23.0+ (for building from source)
- **SQLite3**: Embedded (no external database required)
- **Reverse Proxy**: Nginx or Apache (recommended)
- **Process Manager**: systemd (recommended)

### External Services

- **Home Assistant**: Optional but recommended
- **AI Providers**: OpenAI, Claude, etc. (optional)
- **Monitoring**: Prometheus, Grafana (optional)

## Production Build

### Building from Source

```bash
# Clone the repository
git clone https://github.com/frostdev-ops/pma-backend-go.git
cd pma-backend-go

# Install dependencies
go mod download

# Build for production
make build-prod

# Alternatively, build manually with optimizations
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
  -ldflags="-w -s -X main.version=$(git describe --tags --always)" \
  -o pma-backend \
  ./cmd/server
```

### Cross-Platform Building

```bash
# Build for different architectures
make build-linux-amd64
make build-linux-arm64
make build-darwin-amd64

# Custom build with Docker
docker run --rm -v "$PWD":/usr/src/app -w /usr/src/app \
  golang:1.23-alpine \
  go build -o pma-backend-linux ./cmd/server
```

### Build Artifacts

After building, you'll have:
- `pma-backend`: Main executable
- `configs/`: Configuration files
- `migrations/`: Database migration files

## Docker Deployment

### Dockerfile

Create a production Dockerfile:

```dockerfile
# Multi-stage build for smaller image
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s" \
    -o pma-backend \
    ./cmd/server

# Production image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite tzdata

# Create non-root user
RUN addgroup -g 1001 pma && \
    adduser -u 1001 -G pma -s /bin/sh -D pma

# Create directories
RUN mkdir -p /app/data /app/logs /app/configs /app/migrations && \
    chown -R pma:pma /app

WORKDIR /app

# Copy built binary
COPY --from=builder /app/pma-backend .
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/migrations ./migrations

# Switch to non-root user
USER pma

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3001/health || exit 1

EXPOSE 3001

CMD ["./pma-backend"]
```

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'

services:
  pma-backend:
    build: .
    container_name: pma-backend
    restart: unless-stopped
    ports:
      - "3001:3001"
    volumes:
      - ./data:/app/data
      - ./logs:/app/logs
      - ./configs/config.production.yaml:/app/configs/config.production.yaml:ro
    environment:
      - APP_ENV=production
      - JWT_SECRET=${JWT_SECRET}
      - HOME_ASSISTANT_URL=${HOME_ASSISTANT_URL}
      - HOME_ASSISTANT_TOKEN=${HOME_ASSISTANT_TOKEN}
      - LOG_LEVEL=info
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:3001/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    networks:
      - pma-network

  # Optional: Nginx reverse proxy
  nginx:
    image: nginx:alpine
    container_name: pma-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/certs:/etc/nginx/certs:ro
    depends_on:
      - pma-backend
    networks:
      - pma-network

networks:
  pma-network:
    driver: bridge

volumes:
  pma-data:
  pma-logs:
```

### Docker Deployment Commands

```bash
# Create production environment file
cat > .env << EOF
JWT_SECRET=your-secure-256-bit-secret
HOME_ASSISTANT_URL=http://homeassistant:8123
HOME_ASSISTANT_TOKEN=your-long-lived-token
EOF

# Build and start services
docker-compose up -d

# View logs
docker-compose logs -f pma-backend

# Update application
docker-compose pull
docker-compose up -d

# Backup data
docker run --rm -v pma-backend_pma-data:/data -v $(pwd):/backup \
  alpine tar czf /backup/pma-backup-$(date +%Y%m%d_%H%M%S).tar.gz -C /data .
```

## Systemd Service

### Service File

Create `/etc/systemd/system/pma-backend.service`:

```ini
[Unit]
Description=PMA Backend Go Service
Documentation=https://github.com/frostdev-ops/pma-backend-go
After=network.target
Wants=network.target

[Service]
Type=simple
User=pma
Group=pma
WorkingDirectory=/opt/pma-backend
ExecStart=/opt/pma-backend/pma-backend
ExecReload=/bin/kill -HUP $MAINPID

# Restart policy
Restart=always
RestartSec=5
StartLimitIntervalSec=60
StartLimitBurst=3

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/pma-backend/data /opt/pma-backend/logs

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

# Environment
Environment=APP_ENV=production
Environment=JWT_SECRET=your-secure-secret
Environment=HOME_ASSISTANT_URL=http://homeassistant:8123
Environment=HOME_ASSISTANT_TOKEN=your-token
Environment=LOG_LEVEL=info
Environment=DATABASE_PATH=/opt/pma-backend/data/pma.db

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=pma-backend

[Install]
WantedBy=multi-user.target
```

### Service Management

```bash
# Create user and directories
sudo useradd -r -s /bin/false pma
sudo mkdir -p /opt/pma-backend/{data,logs,configs}
sudo chown -R pma:pma /opt/pma-backend

# Install binary and configuration
sudo cp pma-backend /opt/pma-backend/
sudo cp -r configs/* /opt/pma-backend/configs/
sudo cp -r migrations /opt/pma-backend/
sudo chown -R pma:pma /opt/pma-backend

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable pma-backend
sudo systemctl start pma-backend

# Check status
sudo systemctl status pma-backend

# View logs
sudo journalctl -u pma-backend -f

# Reload configuration
sudo systemctl reload pma-backend

# Restart service
sudo systemctl restart pma-backend
```

## Reverse Proxy Setup

### Nginx Configuration

Create `/etc/nginx/sites-available/pma-backend`:

```nginx
# Rate limiting
limit_req_zone $binary_remote_addr zone=pma_api:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=pma_ws:10m rate=5r/s;

# Upstream backend
upstream pma_backend {
    server 127.0.0.1:3001 max_fails=3 fail_timeout=30s;
    keepalive 32;
}

# HTTP redirect to HTTPS
server {
    listen 80;
    listen [::]:80;
    server_name your-domain.com;
    
    # Security headers
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;
    add_header X-XSS-Protection "1; mode=block";
    
    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

# HTTPS server
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name your-domain.com;

    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;
    ssl_stapling on;
    ssl_stapling_verify on;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options nosniff;
    add_header X-Frame-Options DENY;
    add_header X-XSS-Protection "1; mode=block";
    add_header Referrer-Policy "strict-origin-when-cross-origin";

    # Logging
    access_log /var/log/nginx/pma-backend.access.log;
    error_log /var/log/nginx/pma-backend.error.log;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_types
        text/plain
        text/css
        text/xml
        text/javascript
        application/json
        application/javascript
        application/xml+rss
        application/atom+xml
        image/svg+xml;

    # API endpoints
    location /api/ {
        limit_req zone=pma_api burst=20 nodelay;
        
        proxy_pass http://pma_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        
        # Timeouts
        proxy_connect_timeout 5s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    # WebSocket endpoint
    location /ws {
        limit_req zone=pma_ws burst=10 nodelay;
        
        proxy_pass http://pma_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket specific timeouts
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }

    # Health check endpoint
    location /health {
        proxy_pass http://pma_backend;
        access_log off;
    }

    # Static files (if serving frontend)
    location / {
        root /var/www/pma-frontend;
        try_files $uri $uri/ /index.html;
        expires 1d;
        add_header Cache-Control "public, immutable";
    }
}
```

### Apache Configuration

Create `/etc/apache2/sites-available/pma-backend.conf`:

```apache
<VirtualHost *:80>
    ServerName your-domain.com
    DocumentRoot /var/www/html
    
    # Redirect to HTTPS
    Redirect permanent / https://your-domain.com/
</VirtualHost>

<VirtualHost *:443>
    ServerName your-domain.com
    DocumentRoot /var/www/pma-frontend

    # SSL Configuration
    SSLEngine on
    SSLCertificateFile /etc/letsencrypt/live/your-domain.com/fullchain.pem
    SSLCertificateKeyFile /etc/letsencrypt/live/your-domain.com/privkey.pem
    
    # Security headers
    Header always set Strict-Transport-Security "max-age=31536000; includeSubDomains"
    Header always set X-Content-Type-Options nosniff
    Header always set X-Frame-Options DENY
    Header always set X-XSS-Protection "1; mode=block"

    # Proxy API requests
    ProxyPreserveHost On
    ProxyPass /api/ http://127.0.0.1:3001/api/
    ProxyPassReverse /api/ http://127.0.0.1:3001/api/

    # WebSocket proxy
    ProxyPass /ws ws://127.0.0.1:3001/ws
    ProxyPassReverse /ws ws://127.0.0.1:3001/ws

    # Health check
    ProxyPass /health http://127.0.0.1:3001/health
    ProxyPassReverse /health http://127.0.0.1:3001/health

    # Logging
    ErrorLog ${APACHE_LOG_DIR}/pma-backend_error.log
    CustomLog ${APACHE_LOG_DIR}/pma-backend_access.log combined
</VirtualHost>
```

## Database Setup

### SQLite Configuration

```bash
# Create database directory
sudo mkdir -p /opt/pma-backend/data
sudo chown pma:pma /opt/pma-backend/data
sudo chmod 755 /opt/pma-backend/data

# Set SQLite performance settings
echo "PRAGMA journal_mode=WAL;" | sqlite3 /opt/pma-backend/data/pma.db
echo "PRAGMA synchronous=NORMAL;" | sqlite3 /opt/pma-backend/data/pma.db
echo "PRAGMA cache_size=2000;" | sqlite3 /opt/pma-backend/data/pma.db
```

### Database Backup Script

Create `/opt/pma-backend/scripts/backup.sh`:

```bash
#!/bin/bash

# Configuration
DB_PATH="/opt/pma-backend/data/pma.db"
BACKUP_DIR="/opt/pma-backend/data/backups"
RETENTION_DAYS=7

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Create backup
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="$BACKUP_DIR/pma_backup_$TIMESTAMP.db"

# Backup database with WAL checkpoint
sqlite3 "$DB_PATH" "PRAGMA wal_checkpoint(FULL);"
cp "$DB_PATH" "$BACKUP_FILE"
gzip "$BACKUP_FILE"

# Clean old backups
find "$BACKUP_DIR" -name "pma_backup_*.db.gz" -mtime +$RETENTION_DAYS -delete

echo "Backup completed: $BACKUP_FILE.gz"
```

### Automated Backups

Add to crontab:

```bash
# Edit crontab for pma user
sudo crontab -u pma -e

# Add backup job (daily at 2 AM)
0 2 * * * /opt/pma-backend/scripts/backup.sh

# Add weekly database optimization
0 3 * * 0 sqlite3 /opt/pma-backend/data/pma.db "VACUUM; ANALYZE;"
```

## Environment Configuration

### Production Environment File

Create `/opt/pma-backend/.env`:

```bash
# Application
APP_ENV=production
LOG_LEVEL=info
LOG_FORMAT=json

# Server
PORT=3001
HOST=127.0.0.1
SERVER_MODE=production

# Database
DATABASE_PATH=/opt/pma-backend/data/pma.db
DATABASE_MAX_CONNECTIONS=50
DATABASE_BACKUP_ENABLED=true

# Security
JWT_SECRET=your-secure-256-bit-secret
AUTH_ENABLED=true
PIN_ENABLED=true
RATE_LIMIT_ENABLED=true

# Home Assistant
HOME_ASSISTANT_URL=http://homeassistant:8123
HOME_ASSISTANT_TOKEN=your-long-lived-token
HA_SYNC_ENABLED=true

# AI Services (optional)
AI_ENABLED=true
OPENAI_API_KEY=your-openai-key
CLAUDE_API_KEY=your-claude-key

# External Services
RING_ENABLED=false
UPS_ENABLED=false
NETWORK_ENABLED=true

# Performance
PERFORMANCE_CACHE_ENABLED=true
PERFORMANCE_COMPRESSION_ENABLED=true

# Monitoring
MONITORING_ENABLED=true
PROMETHEUS_ENABLED=true
```

### Security Considerations

```bash
# Secure environment file
sudo chmod 600 /opt/pma-backend/.env
sudo chown pma:pma /opt/pma-backend/.env

# Generate secure JWT secret
openssl rand -base64 32
```

## SSL/TLS Configuration

### Let's Encrypt with Certbot

```bash
# Install Certbot
sudo apt update
sudo apt install certbot python3-certbot-nginx

# Obtain certificate
sudo certbot --nginx -d your-domain.com

# Test renewal
sudo certbot renew --dry-run

# Auto-renewal (add to crontab)
0 12 * * * /usr/bin/certbot renew --quiet
```

### Self-Signed Certificate (Development)

```bash
# Create private key
openssl genrsa -out pma-backend.key 2048

# Create certificate
openssl req -new -x509 -key pma-backend.key -out pma-backend.crt -days 365 \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=your-domain.com"

# Install certificates
sudo mkdir -p /etc/ssl/pma-backend
sudo cp pma-backend.crt /etc/ssl/pma-backend/
sudo cp pma-backend.key /etc/ssl/pma-backend/
sudo chmod 644 /etc/ssl/pma-backend/pma-backend.crt
sudo chmod 600 /etc/ssl/pma-backend/pma-backend.key
```

## Monitoring & Logging

### Log Configuration

```yaml
# configs/config.production.yaml
logging:
  level: "info"
  format: "json"
  output: "file"
  file:
    path: "/opt/pma-backend/logs/pma.log"
    max_size: "100MB"
    max_age: "30d"
    max_backups: 10
    compress: true
```

### Log Rotation

Create `/etc/logrotate.d/pma-backend`:

```
/opt/pma-backend/logs/*.log {
    daily
    missingok
    rotate 30
    compress
    delaycompress
    notifempty
    create 644 pma pma
    postrotate
        systemctl reload pma-backend
    endscript
}
```

### Prometheus Monitoring

Configure Prometheus scraping:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'pma-backend'
    static_configs:
      - targets: ['localhost:3001']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

### Grafana Dashboard

Import PMA Backend dashboard:

```json
{
  "dashboard": {
    "title": "PMA Backend Monitoring",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])"
          }
        ]
      }
    ]
  }
}
```

## Health Checks

### Application Health Check

The application provides a built-in health endpoint:

```bash
# Basic health check
curl http://localhost:3001/health

# Detailed health check
curl http://localhost:3001/api/v1/system/status
```

### External Health Monitoring

Create `/opt/pma-backend/scripts/health_check.sh`:

```bash
#!/bin/bash

HEALTH_URL="http://localhost:3001/health"
TIMEOUT=10

response=$(curl -s -w "%{http_code}" --max-time $TIMEOUT "$HEALTH_URL" -o /dev/null)

if [ "$response" = "200" ]; then
    echo "Health check passed"
    exit 0
else
    echo "Health check failed with status: $response"
    exit 1
fi
```

### Service Monitoring with Monit

Install and configure Monit:

```bash
sudo apt install monit

# Configure monitoring
cat > /etc/monit/conf.d/pma-backend << EOF
check process pma-backend with pidfile /var/run/pma-backend.pid
    start program = "/bin/systemctl start pma-backend"
    stop program = "/bin/systemctl stop pma-backend"
    if failed host 127.0.0.1 port 3001 protocol http
        request /health
        with timeout 10 seconds
    then restart
    if 3 restarts within 5 cycles then timeout
EOF

sudo systemctl restart monit
```

## Backup & Recovery

### Automated Backup Strategy

```bash
#!/bin/bash
# /opt/pma-backend/scripts/full_backup.sh

BACKUP_ROOT="/backup/pma-backend"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_DIR="$BACKUP_ROOT/$TIMESTAMP"

mkdir -p "$BACKUP_DIR"

# Backup database
cp /opt/pma-backend/data/pma.db "$BACKUP_DIR/"

# Backup configuration
cp -r /opt/pma-backend/configs "$BACKUP_DIR/"

# Backup logs (last 7 days)
find /opt/pma-backend/logs -name "*.log" -mtime -7 -exec cp {} "$BACKUP_DIR/" \;

# Create archive
tar czf "$BACKUP_ROOT/pma_full_backup_$TIMESTAMP.tar.gz" -C "$BACKUP_ROOT" "$TIMESTAMP"
rm -rf "$BACKUP_DIR"

# Clean old backups (keep 30 days)
find "$BACKUP_ROOT" -name "pma_full_backup_*.tar.gz" -mtime +30 -delete

echo "Full backup completed: pma_full_backup_$TIMESTAMP.tar.gz"
```

### Recovery Procedures

```bash
#!/bin/bash
# Recovery script

BACKUP_FILE="$1"
RECOVERY_DIR="/opt/pma-backend-recovery"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file.tar.gz>"
    exit 1
fi

# Stop service
sudo systemctl stop pma-backend

# Extract backup
mkdir -p "$RECOVERY_DIR"
tar xzf "$BACKUP_FILE" -C "$RECOVERY_DIR"

# Restore database
cp "$RECOVERY_DIR"/*/pma.db /opt/pma-backend/data/

# Restore configuration if needed
# cp -r "$RECOVERY_DIR"/*/configs/* /opt/pma-backend/configs/

# Fix permissions
sudo chown -R pma:pma /opt/pma-backend/data

# Start service
sudo systemctl start pma-backend

# Cleanup
rm -rf "$RECOVERY_DIR"

echo "Recovery completed from $BACKUP_FILE"
```

## Security Hardening

### Firewall Configuration

```bash
# UFW (Ubuntu)
sudo ufw allow ssh
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# iptables
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
sudo iptables -A INPUT -i lo -j ACCEPT
sudo iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
sudo iptables -P INPUT DROP
```

### File Permissions

```bash
# Secure application files
sudo chown -R pma:pma /opt/pma-backend
sudo chmod 755 /opt/pma-backend
sudo chmod 700 /opt/pma-backend/data
sudo chmod 640 /opt/pma-backend/configs/*.yaml
sudo chmod 600 /opt/pma-backend/.env
```

### System Hardening

```bash
# Disable unused services
sudo systemctl disable bluetooth
sudo systemctl disable cups

# Update system
sudo apt update && sudo apt upgrade -y

# Install security updates automatically
sudo apt install unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades
```

## Performance Optimization

### System Optimization

```bash
# Increase file descriptor limits
echo "pma soft nofile 65536" >> /etc/security/limits.conf
echo "pma hard nofile 65536" >> /etc/security/limits.conf

# Optimize network settings
echo "net.core.rmem_max = 16777216" >> /etc/sysctl.conf
echo "net.core.wmem_max = 16777216" >> /etc/sysctl.conf
echo "net.ipv4.tcp_rmem = 4096 12582912 16777216" >> /etc/sysctl.conf
echo "net.ipv4.tcp_wmem = 4096 12582912 16777216" >> /etc/sysctl.conf

sudo sysctl -p
```

### Application Tuning

```yaml
# configs/config.production.yaml
performance:
  database:
    max_connections: 50
    enable_query_cache: true
    cache_ttl: "1h"
  memory:
    gc_target: 80
    heap_limit: 2147483648  # 2GB
  api:
    enable_compression: true
    max_request_size: 10485760
  workers:
    automation_workers: 8
    sync_workers: 4
```

## Troubleshooting

### Common Issues

#### Service Won't Start

```bash
# Check service status
sudo systemctl status pma-backend

# Check logs
sudo journalctl -u pma-backend -n 50

# Check configuration
/opt/pma-backend/pma-backend --validate-config
```

#### High Memory Usage

```bash
# Monitor memory usage
top -p $(pgrep pma-backend)

# Check for memory leaks
sudo systemctl status pma-backend
curl http://localhost:3001/api/v1/system/status
```

#### Database Issues

```bash
# Check database integrity
sqlite3 /opt/pma-backend/data/pma.db "PRAGMA integrity_check;"

# Optimize database
sqlite3 /opt/pma-backend/data/pma.db "VACUUM; ANALYZE;"

# Check database size
du -h /opt/pma-backend/data/pma.db
```

#### Connection Issues

```bash
# Test connectivity
curl -v http://localhost:3001/health

# Check port binding
sudo netstat -tlnp | grep :3001

# Test WebSocket
wscat -c ws://localhost:3001/ws
```

### Log Analysis

```bash
# Search for errors
grep -i error /opt/pma-backend/logs/pma.log

# Monitor real-time logs
tail -f /opt/pma-backend/logs/pma.log | jq .

# Performance analysis
grep "response_time" /opt/pma-backend/logs/pma.log | jq '.response_time'
```

---

For more information, see the [PMA Backend Go Documentation](../README.md) and [Configuration Reference](CONFIGURATION.md).