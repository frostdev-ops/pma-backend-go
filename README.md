# PMA Backend Go

A high-performance, enterprise-grade Go backend for the Personal Management Assistant (PMA) home automation system. Built for scalability, performance, and reliability with comprehensive smart home integration capabilities.

## 🌟 Overview

PMA Backend Go is a complete rewrite of the original Node.js backend, delivering superior performance, lower memory usage, and enhanced concurrency for modern smart home environments. It provides a unified platform for managing smart devices, automation rules, real-time monitoring, and AI-powered interactions across multiple protocols and platforms.

### Key Features

- **🏠 Universal Smart Home Integration**: Seamless connectivity with Home Assistant, Ring, Shelly, UPS systems, and network devices
- **🤖 AI-Powered Assistant**: Integrated LLM support (LlamaCpp) with MCP (Model Context Protocol) tools
- **⚡ Real-time Communication**: WebSocket-based live updates with subscription management and message queuing
- **🎯 Advanced Automation**: Rule-based automation engine with triggers, conditions, actions, and circuit breaker protection
- **📊 Analytics & Monitoring**: Comprehensive system monitoring, performance analytics, predictive insights, and historical data
- **🔐 Enterprise Security**: JWT authentication, PIN-based access, rate limiting, CORS protection, and advanced security middleware
- **📱 Cross-Platform API**: RESTful API with mobile and web frontend support, API versioning, and comprehensive error handling
- **🎨 Area Management**: Hierarchical room and area organization with advanced entity grouping and conflict resolution
- **🚀 High Performance**: Optimized database operations, memory management, concurrent processing, and intelligent caching
- **🔄 Unified Type System**: Centralized entity management with adapter registry, conflict resolution, and source prioritization
- **📈 Scalability**: Horizontal scaling support, connection pooling, and resource optimization
- **🔧 Developer Experience**: Hot reload development, comprehensive testing, detailed logging, and extensive documentation

## 📋 Table of Contents

- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [Installation](#installation)
- [Configuration](#configuration)
- [API Documentation](#api-documentation)
- [WebSocket Communication](#websocket-communication)
- [Development](#development)
- [Deployment](#deployment)
- [Performance](#performance)
- [Security](#security)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)
- [Changelog](#changelog)
- [License](#license)

## 🚀 Quick Start

### Prerequisites

- **Go 1.24.0+** (with Go 1.24.5 toolchain recommended)
- **SQLite3** for database storage
- **Git** for version control
- **Home Assistant** (optional but recommended for full functionality)
- **Make** (for build automation)

### Fast Installation

```bash
# Clone the repository
git clone https://github.com/frostdev-ops/pma-backend-go.git
cd pma-backend-go

# Install dependencies and build
make build

# Create data directory and copy configuration
mkdir -p data/backups data/temp data/cache logs
cp configs/config.yaml configs/config.local.yaml

# Edit configuration (see Configuration section below)
nano configs/config.local.yaml

# Run database migrations
make migrate

# Start the server
./bin/pma-server
```

### Quick Configuration

Edit `configs/config.local.yaml`:

```yaml
# Basic server configuration
server:
  port: 3001
  host: "0.0.0.0"
  mode: "development"

# Database configuration
database:
  path: "./data/pma.db"
  max_connections: 25
  backup_enabled: true
  backup_path: "./data/backups"
  max_idle_conns: 10
  conn_max_lifetime: "1h"
  query_timeout: "30s"
  enable_query_cache: true
  cache_ttl: "30m"

# Authentication (currently disabled for development)
auth:
  enabled: false  # Set to true for production
  jwt_secret: "your-secret-key"
  token_expiry: 1800

# Home Assistant integration
home_assistant:
  url: "http://192.168.100.2:8123"
  token: "your-home-assistant-token"
  sync:
    enabled: true
    full_sync_interval: "1h"
    supported_domains: ["light", "switch", "sensor", "binary_sensor", "climate", "cover"]

# AI Configuration
ai:
  enabled: true
  default_model: "LFM2-1.2B"
  default_provider: "llamacpp"
  fallback_enabled: true
  ollama:
    enabled: true
    url: "http://localhost:11434"
    auto_start: true
```

## 🏗️ Architecture

### Core Components

The PMA Backend Go follows a modular, service-oriented architecture with the following key components:

#### 1. **API Layer** (`internal/api/`)
- **Router**: Gin-based HTTP router with middleware support
- **Handlers**: RESTful API handlers for all system operations
- **Middleware**: Authentication, CORS, rate limiting, error handling

#### 2. **Core Services** (`internal/core/`)
- **Unified Entity Service**: Centralized device management across platforms
- **Automation Engine**: Rule-based automation with triggers and actions
- **AI Services**: LLM integration with conversation management
- **WebSocket Hub**: Real-time communication and event broadcasting
- **Monitoring**: System health, performance metrics, and analytics
- **File Management**: Media processing, backup, and storage
- **Kiosk Management**: Device pairing and remote control
- **Energy Management**: Power monitoring and cost tracking

#### 3. **Database Layer** (`internal/database/`)
- **SQLite**: Primary database with connection pooling
- **Migrations**: Version-controlled schema management
- **Repositories**: Data access layer with type safety
- **Enhanced DB**: Performance optimizations and caching

#### 4. **Device Adapters** (`internal/adapters/`)
- **Home Assistant**: Primary smart home platform integration
- **Ring**: Camera and security device management
- **Shelly**: IoT device control and configuration
- **UPS**: Power monitoring and backup systems
- **Network**: Device discovery and network management

#### 5. **AI Integration** (`internal/ai/`)
- **LLM Manager**: Multi-provider AI model management
- **Conversation Service**: Chat history and context management
- **MCP Tools**: Model Context Protocol tool execution
- **Smart Model Selector**: Intelligent provider selection

### Data Flow

```
Client Request → API Router → Handler → Service → Repository → Database
                ↓
            WebSocket Hub → Real-time Updates → Connected Clients
```

### Service Communication

- **Event-Driven**: Services communicate via WebSocket events
- **Unified Interface**: All devices accessed through unified entity service
- **Adapter Pattern**: Platform-specific logic isolated in adapters
- **Repository Pattern**: Data access abstracted through repositories

## 📁 Project Structure

```
pma-backend-go/
├── cmd/                    # Application entry points
│   ├── server/            # Main server application
│   └── migrate/           # Database migration tool
├── internal/              # Internal application code
│   ├── api/              # HTTP API layer
│   │   ├── handlers/     # API endpoint handlers
│   │   ├── middleware/   # HTTP middleware
│   │   └── router.go     # Route definitions
│   ├── core/             # Core business logic
│   │   ├── unified/      # Unified entity management
│   │   ├── automation/   # Automation engine
│   │   ├── ai/           # AI services
│   │   ├── monitoring/   # System monitoring
│   │   ├── kiosk/        # Kiosk management
│   │   ├── energy/       # Energy management
│   │   └── ...           # Other core services
│   ├── adapters/         # External platform adapters
│   │   ├── homeassistant/
│   │   ├── ring/
│   │   ├── shelly/
│   │   └── ups/
│   ├── config/           # Configuration management
│   ├── database/         # Database layer
│   │   ├── repositories/ # Data access layer
│   │   └── sqlite/       # SQLite implementation
│   └── websocket/        # Real-time communication
├── pkg/                  # Public packages
│   ├── logger/           # Logging utilities
│   ├── errors/           # Error handling
│   └── utils/            # Common utilities
├── configs/              # Configuration files
├── migrations/           # Database migrations
├── data/                 # Runtime data
├── tests/                # Test files
├── scripts/              # Build and deployment scripts
└── docs/                 # Documentation
```

## ⚙️ Installation

### Development Setup

```bash
# Clone repository
git clone https://github.com/frostdev-ops/pma-backend-go.git
cd pma-backend-go

# Install Go dependencies
go mod download

# Build the application
make build

# Run database migrations
make migrate

# Start development server with hot reload
make dev
```

### Production Deployment

```bash
# Build for production
make build-prod

# Create systemd service
sudo cp deploy/systemd/pma-backend.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable pma-backend
sudo systemctl start pma-backend
```

### Docker Deployment

```bash
# Build Docker image
docker build -t pma-backend-go .

# Run container
docker run -d \
  --name pma-backend \
  -p 3001:3001 \
  -v $(pwd)/data:/app/data \
  -v $(pwd)/configs:/app/configs \
  pma-backend-go
```

## 🔧 Configuration

### Configuration Files

The application uses Viper for configuration management with support for:

- **YAML files**: `configs/config.yaml`
- **Environment variables**: `PMA_*` prefixed variables
- **Command line flags**: `--config` flag for custom config file

### Key Configuration Sections

#### Server Configuration
```yaml
server:
  port: 3001
  host: "0.0.0.0"
  mode: "production"  # development, production
```

#### Database Configuration
```yaml
database:
  path: "./data/pma.db"
  migrations_path: "./migrations"
  backup_enabled: true
  backup_path: "./data/backups"
  max_connections: 25
  max_idle_conns: 10
  conn_max_lifetime: "1h"
  query_timeout: "30s"
  enable_query_cache: true
  cache_ttl: "30m"
```

#### Authentication Configuration
```yaml
auth:
  enabled: true
  jwt_secret: "your-secret-key"
  token_expiry: 1800  # 30 minutes
  api_secret: "api-secret-key"
  allow_localhost_bypass: false
```

#### Home Assistant Integration
```yaml
home_assistant:
  url: "http://192.168.100.2:8123"
  token: "your-home-assistant-token"
  sync:
    enabled: true
    full_sync_interval: "1h"
    supported_domains: ["light", "switch", "sensor", "binary_sensor", "climate", "cover"]
    conflict_resolution: "homeassistant_wins"
    batch_size: 100
    retry_attempts: 3
```

#### AI Configuration
```yaml
ai:
  enabled: true
  default_model: "LFM2-1.2B"
  default_provider: "llamacpp"
  fallback_enabled: true
  ollama:
    enabled: true
    url: "http://localhost:11434"
    auto_start: true
    timeout: "30s"
    max_retries: 3
  hugot:
    enabled: false
    models_dir: "./models"
    default_model: "hugot-model"
  vllm:
    enabled: false
    base_url: "http://localhost:8000"
    api_key: ""
    default_model: "vllm-model"
  llamacpp:
    enabled: true
    base_url: "http://localhost:8080"
    default_model: "LFM2-1.2B"
    auto_start: true
    binary_path: "/usr/local/bin/llama-cpp-server"
    model_path: "./models/LFM2-1.2B.gguf"
    server_port: 8080
```

#### Performance Configuration
```yaml
performance:
  database:
    max_connections: 25
    max_idle_conns: 10
    conn_max_lifetime: "1h"
    query_timeout: "30s"
    enable_query_cache: true
    cache_ttl: "30m"
  memory:
    gc_target: 70
    heap_limit: 1073741824  # 1GB
    enable_pooling: true
  api:
    enable_compression: true
    max_request_size: 10485760  # 10MB
    rate_limit_requests: 1000
    rate_limit_window: "1m"
  websocket:
    max_connections: 1000
    message_buffer_size: 256
    compression_enabled: true
```

## 🔌 API Documentation

The PMA Backend Go provides a comprehensive REST API with the following endpoints:

### Authentication Endpoints
- `POST /api/v1/auth/verify-pin` - PIN-based authentication
- `POST /api/v1/auth/set-pin` - Set PIN for authentication
- `POST /api/v1/auth/change-pin` - Change existing PIN
- `POST /api/v1/auth/validate` - Validate JWT token

### Entity Management
- `GET /api/v1/entities` - List all entities
- `GET /api/v1/entities/:id` - Get entity details
- `POST /api/v1/entities/:id/action` - Execute entity action
- `GET /api/v1/entities/sync` - Trigger entity synchronization

### Room and Area Management
- `GET /api/v1/rooms` - List all rooms
- `POST /api/v1/rooms` - Create new room
- `PUT /api/v1/rooms/:id` - Update room
- `DELETE /api/v1/rooms/:id` - Delete room
- `GET /api/v1/areas` - List all areas
- `POST /api/v1/areas` - Create new area

### Automation Engine
- `GET /api/v1/automation/rules` - List automation rules
- `POST /api/v1/automation/rules` - Create automation rule
- `PUT /api/v1/automation/rules/:id` - Update automation rule
- `DELETE /api/v1/automation/rules/:id` - Delete automation rule
- `POST /api/v1/automation/rules/:id/test` - Test automation rule

### AI and Conversation
- `POST /api/v1/ai/chat` - Send chat message
- `GET /api/v1/ai/conversations` - List conversations
- `POST /api/v1/ai/conversations` - Create conversation
- `GET /api/v1/ai/conversations/:id` - Get conversation
- `POST /api/v1/ai/conversations/:id/messages` - Add message to conversation

### System Management
- `GET /api/v1/system/status` - Get system status
- `GET /api/v1/system/health` - Health check
- `POST /api/v1/system/reboot` - Reboot system
- `POST /api/v1/system/shutdown` - Shutdown system
- `GET /api/v1/system/metrics` - System metrics

### Monitoring and Analytics
- `GET /api/v1/monitoring/metrics` - Get monitoring metrics
- `GET /api/v1/monitoring/alerts` - Get active alerts
- `GET /api/v1/analytics/events` - Get analytics events
- `GET /api/v1/analytics/reports` - Generate analytics reports

### File Management
- `GET /api/v1/files` - List files
- `POST /api/v1/files/upload` - Upload file
- `GET /api/v1/files/:id` - Download file
- `DELETE /api/v1/files/:id` - Delete file

### Kiosk Management
- `GET /api/v1/kiosk/devices` - List kiosk devices
- `POST /api/v1/kiosk/pair` - Pair new kiosk device
- `GET /api/v1/kiosk/devices/:id/status` - Get device status
- `POST /api/v1/kiosk/devices/:id/command` - Send command to device

### Energy Management
- `GET /api/v1/energy/current` - Get current energy usage
- `GET /api/v1/energy/history` - Get energy history
- `GET /api/v1/energy/settings` - Get energy settings
- `PUT /api/v1/energy/settings` - Update energy settings

### WebSocket Endpoints
- `GET /ws` - WebSocket connection for real-time updates

For complete API documentation, see [API Reference Guide](docs/API_REFERENCE.md).

## 🔄 WebSocket Communication

The PMA Backend Go provides real-time communication through WebSocket connections:

### Connection
```javascript
const ws = new WebSocket('ws://localhost:3001/ws');
```

### Event Types

#### Entity Events
```json
{
  "type": "entity_updated",
  "data": {
    "entity_id": "light.living_room",
    "state": "on",
    "attributes": {...},
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

#### System Events
```json
{
  "type": "system_status",
  "data": {
    "status": "online",
    "uptime": 3600,
    "memory_usage": 0.65,
    "cpu_usage": 0.12
  }
}
```

#### Automation Events
```json
{
  "type": "automation_triggered",
  "data": {
    "rule_id": "rule_123",
    "trigger": "entity_state_changed",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

### Subscription Management
```javascript
// Subscribe to entity updates
ws.send(JSON.stringify({
  "type": "subscribe",
  "topic": "entities",
  "filter": ["light", "switch"]
}));

// Unsubscribe
ws.send(JSON.stringify({
  "type": "unsubscribe",
  "topic": "entities"
}));
```

## 🛠️ Development

### Development Environment

```bash
# Install development dependencies
go mod download

# Start development server with hot reload
make dev

# Run tests
make test

# Run specific test package
go test ./internal/core/automation -v

# Run benchmarks
go test ./internal/core/automation -bench=.
```

### Code Structure

The codebase follows Go best practices:

- **Package Organization**: Clear separation of concerns
- **Interface Design**: Dependency injection through interfaces
- **Error Handling**: Comprehensive error handling with custom error types
- **Testing**: Unit tests, integration tests, and benchmarks
- **Documentation**: Comprehensive code documentation

### Adding New Features

1. **Create Service**: Add new service in `internal/core/`
2. **Add Repository**: Create data access layer in `internal/database/repositories/`
3. **Add Handler**: Create API handler in `internal/api/handlers/`
4. **Add Routes**: Register routes in `internal/api/router.go`
5. **Add Tests**: Create comprehensive tests
6. **Update Documentation**: Update API documentation

### Code Style

The project follows Go formatting standards:

```bash
# Format code
go fmt ./...

# Run linter
golangci-lint run

# Check for security issues
gosec ./...
```

## 🚀 Deployment

### Production Build

```bash
# Build for production
make build-prod

# Build for specific platform
GOOS=linux GOARCH=arm64 make build-prod
```

### Systemd Service

Create `/etc/systemd/system/pma-backend.service`:

```ini
[Unit]
Description=PMA Backend Go
After=network.target

[Service]
Type=simple
User=pma
WorkingDirectory=/opt/pma/pma-backend-go
ExecStart=/opt/pma/pma-backend-go/bin/pma-server
Restart=always
RestartSec=5
Environment=PMA_CONFIG_FILE=/opt/pma/pma-backend-go/configs/config.yaml

[Install]
WantedBy=multi-user.target
```

### Docker Deployment

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/pma-server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite
WORKDIR /app
COPY --from=builder /app/bin/pma-server .
COPY configs/ ./configs/
COPY migrations/ ./migrations/

EXPOSE 3001
CMD ["./pma-server"]
```

### Environment Variables

Key environment variables:

```bash
# Server configuration
PMA_SERVER_PORT=3001
PMA_SERVER_HOST=0.0.0.0
PMA_SERVER_MODE=production

# Database configuration
PMA_DATABASE_PATH=./data/pma.db
PMA_DATABASE_MAX_CONNECTIONS=25

# Authentication
PMA_AUTH_ENABLED=true
PMA_AUTH_JWT_SECRET=your-secret-key

# Home Assistant
PMA_HOME_ASSISTANT_URL=http://192.168.100.2:8123
PMA_HOME_ASSISTANT_TOKEN=your-token

# AI Configuration
PMA_AI_ENABLED=true
PMA_AI_DEFAULT_PROVIDER=llamacpp
```

## 📊 Performance

### Performance Characteristics

- **Response Time**: < 100ms for most API calls
- **Concurrent Connections**: 1000+ WebSocket connections
- **Database Operations**: Optimized with connection pooling and caching
- **Memory Usage**: < 200MB typical usage
- **CPU Usage**: < 10% under normal load

### Optimization Features

- **Connection Pooling**: Database connection reuse
- **Query Caching**: Intelligent query result caching
- **Memory Management**: Automatic garbage collection optimization
- **Concurrent Processing**: Goroutine-based parallel processing
- **Compression**: HTTP response compression
- **Rate Limiting**: API rate limiting to prevent abuse

### Monitoring

The system provides comprehensive monitoring:

- **System Metrics**: CPU, memory, disk usage
- **Application Metrics**: Request rates, response times, error rates
- **Database Metrics**: Query performance, connection pool status
- **Custom Metrics**: Business-specific metrics and analytics

## 🔐 Security

### Security Features

- **Authentication**: JWT-based authentication with PIN fallback
- **Authorization**: Role-based access control
- **Rate Limiting**: API rate limiting to prevent abuse
- **CORS Protection**: Cross-origin resource sharing protection
- **Input Validation**: Comprehensive input validation and sanitization
- **Error Handling**: Secure error messages without information leakage
- **Audit Logging**: Comprehensive audit trail for security events

### Security Best Practices

1. **Use HTTPS**: Always use HTTPS in production
2. **Secure Secrets**: Store secrets in environment variables
3. **Regular Updates**: Keep dependencies updated
4. **Access Control**: Implement proper access controls
5. **Monitoring**: Monitor for security events
6. **Backup**: Regular secure backups

### Security Configuration

```yaml
security:
  enable_cors: true
  allowed_origins: ["https://yourdomain.com"]
  rate_limiting:
    enabled: true
    requests_per_minute: 1000
    burst_size: 200
```

## 🧪 Testing

### Test Structure

```bash
# Run all tests
make test

# Run specific test package
go test ./internal/core/automation -v

# Run tests with coverage
go test ./... -cover

# Run benchmarks
go test ./internal/core/automation -bench=.

# Run integration tests
go test ./tests/integration -v
```

### Test Categories

- **Unit Tests**: Individual component testing
- **Integration Tests**: Service interaction testing
- **API Tests**: HTTP endpoint testing
- **Database Tests**: Data access layer testing
- **Performance Tests**: Load and stress testing

### Test Coverage

The project maintains high test coverage:

- **Core Services**: > 80% coverage
- **API Handlers**: > 75% coverage
- **Database Layer**: > 85% coverage
- **Overall**: > 80% coverage

## 🔧 Troubleshooting

### Common Issues

#### Database Connection Issues
```bash
# Check database file permissions
ls -la data/pma.db

# Verify database integrity
sqlite3 data/pma.db "PRAGMA integrity_check;"

# Check database size
du -h data/pma.db
```

#### Memory Issues
```bash
# Check memory usage
ps aux | grep pma-server

# Force garbage collection
curl -X POST http://localhost:3001/api/v1/system/gc

# Check goroutine count
curl http://localhost:3001/api/v1/system/metrics
```

#### WebSocket Issues
```bash
# Check WebSocket connections
curl http://localhost:3001/api/v1/websocket/status

# Test WebSocket connection
wscat -c ws://localhost:3001/ws
```

#### Home Assistant Integration Issues
```bash
# Test HA connection
curl http://localhost:3001/api/v1/system/test-ha

# Check sync status
curl http://localhost:3001/api/v1/entities/sync/status

# Force full sync
curl -X POST http://localhost:3001/api/v1/entities/sync/full
```

### Logging

The application provides comprehensive logging:

```bash
# View application logs
tail -f logs/pma-backend.log

# View debug logs
tail -f logs/debug.log

# View error logs
tail -f logs/error.log
```

### Debug Mode

Enable debug mode for detailed logging:

```bash
# Set debug environment variable
export PMA_DEBUG=true

# Or set in configuration
logging:
  debug:
    enabled: true
    level: "debug"
```

## 🤝 Contributing

### Development Workflow

1. **Fork Repository**: Fork the repository on GitHub
2. **Create Branch**: Create feature branch from main
3. **Make Changes**: Implement your changes
4. **Add Tests**: Add comprehensive tests
5. **Run Tests**: Ensure all tests pass
6. **Submit PR**: Create pull request with description

### Code Standards

- **Go Format**: Use `go fmt` for code formatting
- **Linting**: Use `golangci-lint` for code quality
- **Documentation**: Add comprehensive documentation
- **Testing**: Maintain high test coverage
- **Performance**: Consider performance implications

### Pull Request Guidelines

- **Clear Description**: Describe changes clearly
- **Tests**: Include relevant tests
- **Documentation**: Update documentation as needed
- **Performance**: Consider performance impact
- **Security**: Ensure security best practices

## 📝 Changelog

### Version 1.0.0 (Current)

#### Features
- Complete Go rewrite of Node.js backend
- Unified entity management system
- Advanced automation engine
- AI integration with multiple providers
- Real-time WebSocket communication
- Comprehensive monitoring and analytics
- Kiosk device management
- Energy monitoring and management
- File management and media processing
- Security and authentication system

#### Performance
- 10x performance improvement over Node.js version
- < 100ms API response times
- Support for 1000+ concurrent WebSocket connections
- Optimized database operations with connection pooling
- Intelligent caching system

#### Architecture
- Modular service-oriented architecture
- Clean separation of concerns
- Comprehensive error handling
- Extensive logging and monitoring
- Production-ready deployment configuration

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Home Assistant**: For the excellent smart home platform
- **Gin Framework**: For the high-performance HTTP framework
- **SQLite**: For the reliable embedded database
- **Go Community**: For the excellent Go ecosystem

## 📞 Support

For support and questions:

- **Issues**: Create an issue on GitHub
- **Discussions**: Use GitHub Discussions
- **Documentation**: Check the docs directory
- **Examples**: See the examples directory

---

**PMA Backend Go** - Enterprise-grade smart home automation backend built with Go. 