# PMA Backend Go

A high-performance Go backend for the Personal Management Assistant (PMA) home automation system.

## Overview

This is a complete rewrite of the PMA backend from Node.js/TypeScript to Go, providing improved performance, lower memory usage, and better concurrency for home automation tasks.

## Features

- **Home Assistant Integration**: Complete integration with Home Assistant API
- **Entity Management**: Centralized management of smart home entities
- **Room-based Organization**: Logical grouping of devices by rooms
- **Automation Engine**: Rule-based automation system
- **Real-time Updates**: WebSocket-based real-time communication
- **Multi-service Support**: Adapters for Ollama, Ring, Shelly, and other services
- **SQLite Database**: Lightweight, embedded database with migrations
- **JWT Authentication**: Secure API access
- **Configuration Management**: YAML-based configuration with environment overrides

## Project Structure

```
pma-backend-go/
├── cmd/
│   └── server/           # Main application entry point
├── internal/             # Private application code
│   ├── api/              # HTTP handlers and routing
│   │   ├── handlers/     # Request handlers
│   │   └── middleware/   # HTTP middleware
│   ├── core/             # Core business logic
│   │   ├── entities/     # Entity management
│   │   ├── rooms/        # Room management
│   │   ├── automation/   # Automation engine
│   │   └── services/     # Core services
│   ├── adapters/         # External service adapters
│   │   ├── homeassistant/
│   │   ├── ollama/
│   │   ├── ring/
│   │   └── shelly/
│   ├── database/         # Database layer
│   │   ├── sqlite/       # SQLite implementation
│   │   ├── migrations/   # Database migrations
│   │   └── models/       # Database models
│   ├── websocket/        # WebSocket handling
│   └── config/           # Configuration management
├── pkg/                  # Public packages
│   ├── logger/           # Logging utilities
│   ├── errors/           # Error handling
│   └── utils/            # Common utilities
├── configs/              # Configuration files
├── migrations/           # SQL migration files
├── scripts/              # Build and deployment scripts
└── tests/                # Test files
```

## Prerequisites

- Go 1.21 or later
- SQLite3
- Home Assistant instance (optional for development)

## Setup

1. **Clone and setup the project:**
   ```bash
   cd pma-backend-go
   go mod download
   ```

2. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env with your specific configuration
   ```

3. **Initialize database:**
   ```bash
   make migrate
   ```

4. **Build the application:**
   ```bash
   make build
   ```

## Development

### Running the application

```bash
# Development mode
make run

# With hot reload (requires air)
make dev

# Run tests
make test
```

### Configuration

The application uses a layered configuration approach:

1. Default values in `configs/config.yaml`
2. Environment variable overrides
3. Command-line flag overrides

Key environment variables:
- `PORT`: Server port (default: 3001)
- `HOME_ASSISTANT_URL`: Home Assistant URL
- `HOME_ASSISTANT_TOKEN`: Home Assistant long-lived access token
- `JWT_SECRET`: JWT signing secret
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

### Database

The application uses SQLite with automatic migrations. Database files are stored in the `data/` directory.

To run migrations manually:
```bash
make migrate
```

### API Documentation

The REST API provides endpoints for:
- Entity management (`/api/entities`)
- Room management (`/api/rooms`)
- Automation rules (`/api/automation`)
- WebSocket connections (`/ws`)

### WebSocket Events

Real-time updates are provided via WebSocket:
- Entity state changes
- Room updates
- Automation triggers
- System status

## Production Deployment

1. **Build for production:**
   ```bash
   make build-prod
   ```

2. **Deploy configuration:**
   - Update `configs/config.yaml` for production
   - Set appropriate environment variables
   - Configure reverse proxy (nginx recommended)

3. **Service management:**
   The application can be run as a systemd service. See the `scripts/` directory for deployment helpers.

## Testing

```bash
# Run all tests
make test

# Run tests with coverage
go test -v -cover ./...
```

## Contributing

1. Follow Go conventions and best practices
2. Add tests for new functionality
3. Update documentation as needed
4. Use `make test` before submitting changes

## Architecture Notes

This Go backend is designed to replace the existing Node.js backend while maintaining API compatibility. Key improvements include:

- **Performance**: Go's superior performance for concurrent operations
- **Memory efficiency**: Lower memory footprint
- **Type safety**: Strong typing throughout the application
- **Concurrency**: Better handling of multiple simultaneous connections
- **Deployment**: Single binary deployment with no runtime dependencies

## License

This project is part of the PMA (Personal Management Assistant) system. 