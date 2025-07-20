# PMA Backend Go - Deployment Guide

This document provides a comprehensive guide for deploying PMA Backend Go in a production environment, covering various deployment strategies, security hardening, and monitoring.

## Table of Contents

- [Production Build](#production-build)
- [Docker Deployment](#docker-deployment)
  - [Using Docker Compose (Recommended)](#using-docker-compose-recommended)
  - [Manual Docker Deployment](#manual-docker-deployment)
- [Systemd Service Setup](#systemd-service-setup)
- [Reverse Proxy Configuration](#reverse-proxy-configuration)
  - [Nginx Configuration](#nginx-configuration)
  - [Apache Configuration](#apache-configuration)
- [Database Setup & Backup](#database-setup--backup)
- [Environment Configuration](#environment-configuration)
- [SSL/TLS Configuration](#ssltls-configuration)
- [Monitoring and Alerting](#monitoring-and-alerting)
- [Security Hardening](#security-hardening)
- [Performance Tuning](#performance-tuning)
- [Health Checks](#health-checks)
- [Troubleshooting Deployment](#troubleshooting-deployment)

## Production Build

```bash
# Build for production (Linux ARM64 example)
make build-prod

# The optimized binary will be created at: bin/pma-server
```

## Docker Deployment

### Using Docker Compose (Recommended)

```yaml
# docker-compose.yml
version: '3.8'
services:
  pma-backend:
    build: .
    container_name: pma-backend
    ports:
      - "3001:3001"
    volumes:
      - ./data:/app/data
      - ./configs/config.production.yaml:/app/configs/config.production.yaml:ro
    environment:
      - HOME_ASSISTANT_URL=http://homeassistant:8123
      - HOME_ASSISTANT_TOKEN=${HA_TOKEN}
      - JWT_SECRET=${JWT_SECRET}
      - SERVER_MODE=production
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3001/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

```bash
# Deploy with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f pma-backend
```

### Manual Docker Deployment

```bash
# Build Docker image
docker build -t pma-backend:latest .

# Run Docker container
docker run -d \
  --name pma-backend \
  -p 3001:3001 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs/config.production.yaml:/app/configs/config.production.yaml:ro \
  -e HOME_ASSISTANT_URL="http://your-ha:8123" \
  -e HOME_ASSISTANT_TOKEN="your-token" \
  -e JWT_SECRET="your-secret" \
  -e SERVER_MODE="production" \
  --restart unless-stopped \
  pma-backend:latest
```

## Systemd Service Setup

```bash
# Create user for the service
sudo useradd --system --no-create-home --shell /bin/false pma

# Install binary and configuration
sudo mkdir -p /opt/pma-backend
sudo cp bin/pma-server /opt/pma-backend/
sudo cp -r configs /opt/pma-backend/
sudo cp -r migrations /opt/pma-backend/
sudo chown -R pma:pma /opt/pma-backend
```

```ini
# /etc/systemd/system/pma-backend.service
[Unit]
Description=PMA Backend Go
After=network-online.target

[Service]
Type=simple
User=pma
Group=pma
WorkingDirectory=/opt/pma-backend
ExecStart=/opt/pma-backend/pma-server -config /opt/pma-backend/configs/config.production.yaml
Restart=always
RestartSec=5

# Environment variables
EnvironmentFile=/etc/pma-backend/environment

[Install]
WantedBy=multi-user.target
```

```bash
# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable pma-backend
sudo systemctl start pma-backend
```

## Reverse Proxy Configuration

### Nginx Configuration

```nginx
server {
    listen 80;
    server_name pma.yourdomain.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name pma.yourdomain.com;

    # SSL configuration
    ssl_certificate /path/to/fullchain.pem;
    ssl_certificate_key /path/to/privkey.pem;

    location / {
        proxy_pass http://localhost:3001;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location /ws {
        proxy_pass http://localhost:3001;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
}
```

### Apache Configuration

```apache
<VirtualHost *:80>
    ServerName pma.yourdomain.com
    Redirect permanent / https://pma.yourdomain.com/
</VirtualHost>

<VirtualHost *:443>
    ServerName pma.yourdomain.com

    SSLEngine on
    SSLCertificateFile /path/to/fullchain.pem
    SSLCertificateKeyFile /path/to/privkey.pem

    ProxyPass / http://localhost:3001/
    ProxyPassReverse / http://localhost:3001/

    RewriteEngine on
    RewriteCond %{HTTP:Upgrade} websocket [NC]
    RewriteCond %{HTTP:Connection} upgrade [NC]
    RewriteRule ^/ws/?(.*) "ws://localhost:3001/ws/$1" [P,L]
</VirtualHost>
```

## Database Setup & Backup

- **Initial Setup**: Run `make migrate` to create the database schema.
- **Backup**: Use `sqlite3 .backup` or a cron job to regularly back up `data/pma.db`.
- **Performance**: Ensure WAL mode is enabled for better concurrency.

## Environment Configuration

Use a `.env` file or systemd `EnvironmentFile` for production secrets.

```bash
# /etc/pma-backend/environment
JWT_SECRET="your-secure-production-secret"
HOME_ASSISTANT_TOKEN="your-production-token"
OPENAI_API_KEY="your-production-key"
```

## SSL/TLS Configuration

Use Let's Encrypt for free SSL certificates:

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d pma.yourdomain.com
```

## Monitoring and Alerting

- **Prometheus**: Scrape the `/metrics` endpoint for performance data.
- **Grafana**: Use Grafana for creating performance dashboards.
- **Alerting**: Set up alerts in Prometheus/Alertmanager for high error rates, memory usage, or service downtime.

## Security Hardening

- **Firewall**: Use a firewall (e.g., UFW) to restrict access to necessary ports.
- **User Permissions**: Run the application as a non-root user.
- **Secure Configuration**: Use environment variables for all secrets.
- **Regular Updates**: Keep the application and its dependencies up to date.

## Performance Tuning

- **Database**: Use WAL mode and tune SQLite PRAGMA settings.
- **Memory**: Set `GOMEMLIMIT` to prevent out-of-memory issues.
- **Concurrency**: Adjust worker pool sizes based on your hardware.

## Health Checks

- **Basic**: `curl http://localhost:3001/health`
- **Detailed**: `curl -H "Authorization: Bearer TOKEN" http://localhost:3001/api/v1/system/status`

## Troubleshooting Deployment

- **Logs**: Check application logs (`journalctl` or log files) for errors.
- **Permissions**: Ensure correct file and directory permissions.
- **Ports**: Verify that the application port is not blocked.
- **Configuration**: Validate the configuration file syntax.