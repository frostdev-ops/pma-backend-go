# PMA Backend Go - Development Guide

This document provides comprehensive guidance for setting up a development environment and contributing to PMA Backend Go.

## Table of Contents

- [Development Environment Setup](#development-environment-setup)
- [Project Structure](#project-structure)
- [Coding Standards](#coding-standards)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Debugging](#debugging)
- [Database Development](#database-development)
- [API Development](#api-development)
- [WebSocket Development](#websocket-development)
- [Integration Development](#integration-development)
- [Performance Development](#performance-development)
- [Documentation](#documentation)
- [Contribution Guidelines](#contribution-guidelines)

## Development Environment Setup

### Prerequisites

- **Go**: 1.23.0 or later
- **Git**: For version control
- **SQLite3**: Database (usually bundled with Go)
- **Make**: Build automation
- **Docker**: For containerized development (optional)
- **IDE/Editor**: VS Code, GoLand, or Vim with Go plugins

### Go Installation

```bash
# Install Go (latest version)
wget https://go.dev/dl/go1.23.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

### Development Tools

```bash
# Install essential Go tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/air-verse/air@latest
go install github.com/swaggo/swag/cmd/swag@latest
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Install debugging tools
go install github.com/go-delve/delve/cmd/dlv@latest

# Install testing tools
go install github.com/onsi/ginkgo/v2/ginkgo@latest
go install gotest.tools/gotestsum@latest
```

### Project Setup

```bash
# Clone the repository
git clone https://github.com/frostdev-ops/pma-backend-go.git
cd pma-backend-go

# Install dependencies
go mod download

# Verify setup
make test

# Build the project
make build
```

### Development Configuration

Create a development configuration file:

```yaml
# configs/config.development.yaml
server:
  port: 3001
  host: "127.0.0.1"
  mode: "development"

database:
  path: "./data/pma_dev.db"
  migrations_path: "./migrations"

auth:
  enabled: true
  jwt_secret: "development-jwt-secret-key-here"
  token_expiry: 86400  # 24 hours for development

home_assistant:
  url: "http://localhost:8123"
  token: "your-development-token"
  sync:
    enabled: true
    full_sync_interval: "30m"

logging:
  level: "debug"
  format: "text"  # Easier to read during development
  output: "stdout"

ai:
  enabled: true
  default_provider: "openai"
  providers:
    openai:
      enabled: true
      api_key: "your-development-key"

performance:
  cache:
    enabled: true
    default_ttl: "5m"  # Shorter for development
```

### Environment Variables

Create a `.env.development` file:

```bash
# Development environment variables
APP_ENV=development
LOG_LEVEL=debug
JWT_SECRET=development-jwt-secret-key-here
HOME_ASSISTANT_URL=http://localhost:8123
HOME_ASSISTANT_TOKEN=your-development-token
OPENAI_API_KEY=your-development-key
DATABASE_PATH=./data/pma_dev.db
```

### IDE Configuration

#### VS Code Setup

Create `.vscode/settings.json`:

```json
{
  "go.useLanguageServer": true,
  "go.formatTool": "goimports",
  "go.lintTool": "golangci-lint",
  "go.testFlags": ["-v"],
  "go.testTimeout": "30s",
  "go.coverOnSave": true,
  "go.coverOnSingleTest": true,
  "go.coverageDecorator": {
    "type": "gutter",
    "coveredHighlightColor": "rgba(64,128,64,0.5)",
    "uncoveredHighlightColor": "rgba(128,64,64,0.25)"
  },
  "files.exclude": {
    "**/.git": true,
    "**/node_modules": true,
    "**/vendor": true,
    "**/*.exe": true
  }
}
```

Create `.vscode/launch.json` for debugging:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch PMA Backend",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/server",
      "env": {
        "APP_ENV": "development",
        "LOG_LEVEL": "debug"
      },
      "args": []
    },
    {
      "name": "Test Current Package",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${workspaceFolder}/${relativeFileDirname}"
    }
  ]
}
```

Create `.vscode/tasks.json`:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Build",
      "type": "shell",
      "command": "make build",
      "group": "build",
      "presentation": {
        "echo": true,
        "reveal": "silent",
        "focus": false,
        "panel": "shared"
      }
    },
    {
      "label": "Test",
      "type": "shell",
      "command": "make test",
      "group": "test",
      "presentation": {
        "echo": true,
        "reveal": "always",
        "focus": false,
        "panel": "shared"
      }
    }
  ]
}
```

## Project Structure

Understanding the project structure is crucial for development:

```
pma-backend-go/
├── cmd/                    # Application entry points
│   ├── server/            # Main server application
│   │   └── main.go
│   └── migrate/           # Database migration tool
│       └── main.go
├── internal/              # Private application code
│   ├── api/               # HTTP API layer
│   │   ├── handlers/      # HTTP request handlers
│   │   ├── middleware/    # HTTP middleware
│   │   └── router.go      # Route definitions
│   ├── core/              # Core business logic
│   │   ├── automation/    # Automation engine
│   │   ├── entities/      # Entity management
│   │   ├── types/         # Unified type system
│   │   └── unified/       # Unified services
│   ├── adapters/          # External service adapters
│   │   ├── homeassistant/ # Home Assistant adapter
│   │   ├── ring/          # Ring security adapter
│   │   └── shelly/        # Shelly devices adapter
│   ├── ai/                # AI/LLM integration
│   ├── database/          # Database layer
│   │   ├── sqlite/        # SQLite implementation
│   │   ├── models/        # Data models
│   │   └── repositories/  # Data access layer
│   ├── websocket/         # WebSocket implementation
│   └── config/            # Configuration management
├── pkg/                   # Public packages
│   ├── logger/            # Logging utilities
│   ├── errors/            # Error handling
│   └── utils/             # Common utilities
├── migrations/            # Database migrations
├── configs/               # Configuration files
├── docs/                  # Documentation
├── tests/                 # Test files
│   ├── integration/       # Integration tests
│   └── e2e/              # End-to-end tests
└── scripts/               # Build and utility scripts
```

### Package Organization

- **cmd/**: Application entry points (main packages)
- **internal/**: Private application code that cannot be imported by other projects
- **pkg/**: Public packages that can be imported by other projects
- **api/**: HTTP API layer with handlers, middleware, and routing
- **core/**: Core business logic and domain models
- **adapters/**: External service integrations
- **database/**: Data persistence layer

## Coding Standards

### Go Code Style

Follow standard Go conventions and best practices:

```go
// Good: Use clear, descriptive names
func (s *EntityService) GetEntityByID(ctx context.Context, id string) (*Entity, error) {
    // Implementation
}

// Good: Use consistent error handling
func (r *Repository) Save(entity *Entity) error {
    if err := r.validate(entity); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    // Save logic
    return nil
}

// Good: Use context for cancellation and timeouts
func (s *Service) ProcessRequest(ctx context.Context, req *Request) error {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()
    // Processing logic
}
```

### Package Structure Guidelines

```go
// package: internal/core/entities

// types.go - Define types and interfaces
type Entity interface {
    GetID() string
    GetType() EntityType
    GetState() EntityState
}

// service.go - Business logic
type Service struct {
    repo Repository
    logger *logrus.Logger
}

func NewService(repo Repository, logger *logrus.Logger) *Service {
    return &Service{repo: repo, logger: logger}
}

// repository.go - Data access interface
type Repository interface {
    Save(ctx context.Context, entity *Entity) error
    GetByID(ctx context.Context, id string) (*Entity, error)
    GetAll(ctx context.Context) ([]*Entity, error)
}
```

### Error Handling

```go
// Use custom error types for different error categories
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

// Wrap errors with context
func (s *Service) ProcessEntity(entity *Entity) error {
    if err := s.validate(entity); err != nil {
        return fmt.Errorf("entity processing failed: %w", err)
    }
    
    if err := s.repo.Save(entity); err != nil {
        return fmt.Errorf("failed to save entity %s: %w", entity.GetID(), err)
    }
    
    return nil
}
```

### Logging Standards

```go
// Use structured logging with consistent fields
func (s *Service) ProcessEntity(ctx context.Context, entity *Entity) error {
    logger := s.logger.WithFields(logrus.Fields{
        "entity_id":   entity.GetID(),
        "entity_type": entity.GetType(),
        "operation":   "process_entity",
    })
    
    logger.Info("Processing entity")
    
    if err := s.doProcessing(entity); err != nil {
        logger.WithError(err).Error("Entity processing failed")
        return err
    }
    
    logger.Info("Entity processed successfully")
    return nil
}
```

### Configuration Management

```go
// Use configuration structs with validation
type ServiceConfig struct {
    Timeout     time.Duration `mapstructure:"timeout" validate:"required,min=1s"`
    MaxRetries  int           `mapstructure:"max_retries" validate:"min=0,max=10"`
    EnableCache bool          `mapstructure:"enable_cache"`
}

func (c *ServiceConfig) Validate() error {
    validate := validator.New()
    return validate.Struct(c)
}
```

## Development Workflow

### Daily Development

1. **Pull Latest Changes**
```bash
git pull origin main
go mod download
```

2. **Create Feature Branch**
```bash
git checkout -b feature/your-feature-name
```

3. **Development Cycle**
```bash
# Run with hot reload
make dev

# Run tests
make test

# Check code quality
make lint

# Format code
make fmt
```

4. **Commit Changes**
```bash
git add .
git commit -m "feat: add new feature description"
git push origin feature/your-feature-name
```

### Makefile Targets

```makefile
# Development targets
.PHONY: dev
dev:
	air -c .air.toml

.PHONY: build
build:
	go build -o bin/pma-backend ./cmd/server

.PHONY: test
test:
	go test -v ./...

.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: lint
lint:
	golangci-lint run

.PHONY: fmt
fmt:
	goimports -w .
	go fmt ./...

.PHONY: clean
clean:
	rm -rf bin/ coverage.out coverage.html

.PHONY: deps
deps:
	go mod download
	go mod tidy

.PHONY: migrate
migrate:
	migrate -database "sqlite3://$(DATABASE_PATH)" -path ./migrations up

.PHONY: migrate-down
migrate-down:
	migrate -database "sqlite3://$(DATABASE_PATH)" -path ./migrations down 1
```

### Hot Reload Configuration

Create `.air.toml`:

```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/server"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata", "node_modules", "docs"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html", "yaml", "yml"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
```

## Testing

### Test Structure

Organize tests following Go conventions:

```
internal/
├── core/
│   ├── entities/
│   │   ├── service.go
│   │   ├── service_test.go      # Unit tests
│   │   └── integration_test.go  # Integration tests
│   └── automation/
│       ├── engine.go
│       └── engine_test.go
```

### Unit Testing

```go
// service_test.go
package entities

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
)

// Mock repository
type MockRepository struct {
    mock.Mock
}

func (m *MockRepository) Save(ctx context.Context, entity *Entity) error {
    args := m.Called(ctx, entity)
    return args.Error(0)
}

func (m *MockRepository) GetByID(ctx context.Context, id string) (*Entity, error) {
    args := m.Called(ctx, id)
    return args.Get(0).(*Entity), args.Error(1)
}

func TestService_ProcessEntity(t *testing.T) {
    tests := []struct {
        name        string
        entity      *Entity
        setupMocks  func(*MockRepository)
        expectedErr bool
    }{
        {
            name: "successful processing",
            entity: &Entity{
                ID:   "test-entity",
                Type: EntityTypeLight,
            },
            setupMocks: func(repo *MockRepository) {
                repo.On("Save", mock.Anything, mock.AnythingOfType("*Entity")).Return(nil)
            },
            expectedErr: false,
        },
        {
            name: "repository error",
            entity: &Entity{
                ID:   "test-entity",
                Type: EntityTypeLight,
            },
            setupMocks: func(repo *MockRepository) {
                repo.On("Save", mock.Anything, mock.AnythingOfType("*Entity")).Return(errors.New("save failed"))
            },
            expectedErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := &MockRepository{}
            tt.setupMocks(repo)
            
            service := NewService(repo, logrus.New())
            
            err := service.ProcessEntity(context.Background(), tt.entity)
            
            if tt.expectedErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
            
            repo.AssertExpectations(t)
        })
    }
}
```

### Integration Testing

```go
// integration_test.go
//go:build integration

package entities

import (
    "context"
    "database/sql"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    _ "github.com/mattn/go-sqlite3"
)

func TestEntityService_Integration(t *testing.T) {
    // Setup test database
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)
    defer db.Close()
    
    // Run migrations
    runTestMigrations(t, db)
    
    // Create repository
    repo := NewSQLiteRepository(db)
    service := NewService(repo, logrus.New())
    
    // Test entity lifecycle
    entity := &Entity{
        ID:   "test-entity",
        Type: EntityTypeLight,
        State: EntityStateOff,
    }
    
    // Create entity
    err = service.ProcessEntity(context.Background(), entity)
    require.NoError(t, err)
    
    // Retrieve entity
    retrieved, err := service.GetEntityByID(context.Background(), "test-entity")
    require.NoError(t, err)
    assert.Equal(t, entity.ID, retrieved.ID)
    assert.Equal(t, entity.Type, retrieved.Type)
}

func runTestMigrations(t *testing.T, db *sql.DB) {
    // Apply test schema
    schema := `
    CREATE TABLE entities (
        id TEXT PRIMARY KEY,
        type TEXT NOT NULL,
        state TEXT NOT NULL
    );
    `
    _, err := db.Exec(schema)
    require.NoError(t, err)
}
```

### API Testing

```go
// handlers_test.go
package handlers

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
)

func TestEntityHandler_GetEntity(t *testing.T) {
    gin.SetMode(gin.TestMode)
    
    // Setup
    mockService := &MockEntityService{}
    handler := NewEntityHandler(mockService)
    
    router := gin.New()
    router.GET("/entities/:id", handler.GetEntity)
    
    tests := []struct {
        name           string
        entityID       string
        setupMock      func(*MockEntityService)
        expectedStatus int
    }{
        {
            name:     "existing entity",
            entityID: "test-entity",
            setupMock: func(service *MockEntityService) {
                entity := &Entity{ID: "test-entity", Type: EntityTypeLight}
                service.On("GetEntityByID", mock.Anything, "test-entity").Return(entity, nil)
            },
            expectedStatus: http.StatusOK,
        },
        {
            name:     "non-existent entity",
            entityID: "unknown",
            setupMock: func(service *MockEntityService) {
                service.On("GetEntityByID", mock.Anything, "unknown").Return(nil, ErrEntityNotFound)
            },
            expectedStatus: http.StatusNotFound,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockService.ExpectedCalls = nil
            tt.setupMock(mockService)
            
            req := httptest.NewRequest("GET", "/entities/"+tt.entityID, nil)
            resp := httptest.NewRecorder()
            
            router.ServeHTTP(resp, req)
            
            assert.Equal(t, tt.expectedStatus, resp.Code)
            mockService.AssertExpectations(t)
        })
    }
}
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/core/entities

# Run integration tests
go test -tags=integration ./tests/integration/...

# Run tests with race detection
go test -race ./...

# Benchmark tests
go test -bench=. ./...
```

## Debugging

### Using Delve Debugger

```bash
# Debug main application
dlv debug ./cmd/server

# Debug specific test
dlv test ./internal/core/entities

# Remote debugging
dlv debug --headless --listen=:2345 --api-version=2 ./cmd/server
```

### Debugging Configuration

```go
// Add debug endpoints in development
if config.IsDevelopment() {
    router.GET("/debug/pprof/*any", gin.WrapH(http.DefaultServeMux))
    router.GET("/debug/vars", gin.WrapH(http.DefaultServeMux))
}
```

### Logging for Development

```go
// Enhanced development logging
func (s *Service) debugLog(ctx context.Context, msg string, fields map[string]interface{}) {
    if s.config.IsDevelopment() {
        logger := s.logger.WithFields(logrus.Fields(fields))
        logger.Debug(msg)
        
        // Add stack trace for errors in development
        if level == logrus.ErrorLevel {
            logger.WithField("stack", string(debug.Stack())).Error(msg)
        }
    }
}
```

## Database Development

### Migration Development

```bash
# Create new migration
migrate create -ext sql -dir migrations -seq add_new_table

# Apply migrations
make migrate

# Rollback migration
make migrate-down

# Check migration status
migrate -database "sqlite3://./data/pma_dev.db" -path ./migrations version
```

### Migration Example

```sql
-- migrations/001_create_entities.up.sql
CREATE TABLE entities (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    state TEXT NOT NULL,
    attributes TEXT, -- JSON
    source TEXT NOT NULL,
    room_id TEXT,
    area_id TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_entities_type ON entities(type);
CREATE INDEX idx_entities_state ON entities(state);
CREATE INDEX idx_entities_room ON entities(room_id);

-- migrations/001_create_entities.down.sql
DROP INDEX IF EXISTS idx_entities_room;
DROP INDEX IF EXISTS idx_entities_state;
DROP INDEX IF EXISTS idx_entities_type;
DROP TABLE IF EXISTS entities;
```

### Repository Development

```go
// Use interfaces for testability
type EntityRepository interface {
    Save(ctx context.Context, entity *Entity) error
    GetByID(ctx context.Context, id string) (*Entity, error)
    GetAll(ctx context.Context, filters EntityFilters) ([]*Entity, error)
    Delete(ctx context.Context, id string) error
}

// SQLite implementation
type sqliteEntityRepository struct {
    db *sql.DB
}

func (r *sqliteEntityRepository) Save(ctx context.Context, entity *Entity) error {
    query := `
        INSERT OR REPLACE INTO entities (id, name, type, state, attributes, source, room_id, area_id, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
    `
    
    attributesJSON, err := json.Marshal(entity.Attributes)
    if err != nil {
        return fmt.Errorf("failed to marshal attributes: %w", err)
    }
    
    _, err = r.db.ExecContext(ctx, query,
        entity.ID, entity.Name, entity.Type, entity.State,
        string(attributesJSON), entity.Source, entity.RoomID, entity.AreaID)
    
    return err
}
```

## API Development

### Handler Development

```go
// Follow consistent handler patterns
type EntityHandler struct {
    service EntityService
    logger  *logrus.Logger
}

func NewEntityHandler(service EntityService, logger *logrus.Logger) *EntityHandler {
    return &EntityHandler{service: service, logger: logger}
}

func (h *EntityHandler) GetEntity(c *gin.Context) {
    entityID := c.Param("id")
    if entityID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "entity ID is required"})
        return
    }
    
    entity, err := h.service.GetEntityByID(c.Request.Context(), entityID)
    if err != nil {
        if errors.Is(err, ErrEntityNotFound) {
            c.JSON(http.StatusNotFound, gin.H{"error": "entity not found"})
            return
        }
        
        h.logger.WithError(err).Error("Failed to get entity")
        c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
        return
    }
    
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data":    entity,
    })
}
```

### Middleware Development

```go
// Development middleware for request logging
func RequestLoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        logger.WithFields(logrus.Fields{
            "status":     param.StatusCode,
            "method":     param.Method,
            "path":       param.Path,
            "latency":    param.Latency,
            "client_ip":  param.ClientIP,
            "user_agent": param.Request.UserAgent(),
        }).Info("HTTP Request")
        
        return ""
    })
}
```

### API Documentation

Use swagger/OpenAPI for API documentation:

```go
// @title PMA Backend API
// @version 1.0
// @description Personal Management Assistant Backend API
// @host localhost:3001
// @BasePath /api/v1

// @tag.name entities
// @tag.description Entity management operations

// GetEntity retrieves an entity by ID
// @Summary Get entity by ID
// @Description Get detailed information about a specific entity
// @Tags entities
// @Accept json
// @Produce json
// @Param id path string true "Entity ID"
// @Success 200 {object} EntityResponse
// @Failure 404 {object} ErrorResponse
// @Router /entities/{id} [get]
func (h *EntityHandler) GetEntity(c *gin.Context) {
    // Implementation
}
```

## WebSocket Development

### WebSocket Handler Development

```go
// WebSocket message handling
type MessageHandler struct {
    hub    *Hub
    logger *logrus.Logger
}

func (h *MessageHandler) HandleMessage(client *Client, message []byte) error {
    var msg Message
    if err := json.Unmarshal(message, &msg); err != nil {
        return fmt.Errorf("invalid message format: %w", err)
    }
    
    switch msg.Type {
    case "subscribe_entities":
        return h.handleSubscribeEntities(client, msg.Data)
    case "unsubscribe_entities":
        return h.handleUnsubscribeEntities(client, msg.Data)
    default:
        h.logger.WithField("type", msg.Type).Warn("Unknown message type")
        return nil
    }
}

func (h *MessageHandler) handleSubscribeEntities(client *Client, data map[string]interface{}) error {
    entityIDs, ok := data["entity_ids"].([]interface{})
    if !ok {
        return fmt.Errorf("invalid entity_ids format")
    }
    
    for _, entityID := range entityIDs {
        if id, ok := entityID.(string); ok {
            client.SubscribeToEntity(id)
        }
    }
    
    return nil
}
```

### WebSocket Testing

```go
func TestWebSocketHandler(t *testing.T) {
    // Setup WebSocket test server
    hub := NewHub(logrus.New())
    go hub.Run()
    
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        HandleWebSocket(hub, w, r)
    }))
    defer server.Close()
    
    // Connect WebSocket client
    wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
    ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
    require.NoError(t, err)
    defer ws.Close()
    
    // Test message exchange
    testMessage := Message{
        Type: "subscribe_entities",
        Data: map[string]interface{}{
            "entity_ids": []string{"light.test"},
        },
    }
    
    err = ws.WriteJSON(testMessage)
    require.NoError(t, err)
    
    // Verify response
    var response Message
    err = ws.ReadJSON(&response)
    require.NoError(t, err)
    assert.Equal(t, "subscription_confirmed", response.Type)
}
```

## Integration Development

### Adapter Development

```go
// Create adapters following the interface
type Adapter interface {
    GetID() string
    GetSourceType() types.PMASourceType
    GetVersion() string
    IsConnected() bool
    GetHealth() *types.AdapterHealth
    
    Connect(ctx context.Context) error
    Disconnect(ctx context.Context) error
    
    GetEntities(ctx context.Context) ([]types.PMAEntity, error)
    ExecuteAction(ctx context.Context, entityID string, action types.PMAAction) error
}

// Example adapter implementation
type MyServiceAdapter struct {
    client *MyServiceClient
    logger *logrus.Logger
    config MyServiceConfig
}

func NewMyServiceAdapter(config MyServiceConfig, logger *logrus.Logger) *MyServiceAdapter {
    return &MyServiceAdapter{
        config: config,
        logger: logger,
    }
}

func (a *MyServiceAdapter) Connect(ctx context.Context) error {
    client, err := NewMyServiceClient(a.config)
    if err != nil {
        return fmt.Errorf("failed to create client: %w", err)
    }
    
    a.client = client
    return nil
}

func (a *MyServiceAdapter) GetEntities(ctx context.Context) ([]types.PMAEntity, error) {
    devices, err := a.client.GetDevices(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to get devices: %w", err)
    }
    
    entities := make([]types.PMAEntity, len(devices))
    for i, device := range devices {
        entities[i] = a.convertDevice(device)
    }
    
    return entities, nil
}
```

## Performance Development

### Profiling

```go
// Add profiling endpoints for development
import _ "net/http/pprof"

func setupProfiling() {
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
}

// Use pprof for performance analysis
// go tool pprof http://localhost:6060/debug/pprof/profile
// go tool pprof http://localhost:6060/debug/pprof/heap
```

### Benchmarking

```go
// Benchmark critical paths
func BenchmarkEntityProcessing(b *testing.B) {
    service := setupTestService()
    entity := createTestEntity()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        err := service.ProcessEntity(context.Background(), entity)
        if err != nil {
            b.Fatalf("ProcessEntity failed: %v", err)
        }
    }
}

func BenchmarkDatabaseOperations(b *testing.B) {
    db := setupTestDB()
    defer db.Close()
    
    b.Run("Save", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            entity := createTestEntity()
            err := saveEntity(db, entity)
            if err != nil {
                b.Fatalf("Save failed: %v", err)
            }
        }
    })
    
    b.Run("Get", func(b *testing.B) {
        for i := 0; i < b.N; i++ {
            _, err := getEntity(db, "test-id")
            if err != nil {
                b.Fatalf("Get failed: %v", err)
            }
        }
    })
}
```

## Documentation

### Code Documentation

```go
// Package documentation
// Package entities provides entity management functionality for the PMA system.
// It includes entity lifecycle management, state tracking, and integration
// with various smart home platforms.
package entities

// Type documentation
// Entity represents a smart home device or service that can be controlled
// and monitored through the PMA system.
type Entity struct {
    // ID is the unique identifier for the entity across all sources
    ID string `json:"id"`
    
    // Name is the human-readable name of the entity
    Name string `json:"name"`
    
    // Type indicates the category of the entity (light, switch, sensor, etc.)
    Type EntityType `json:"type"`
}

// Function documentation
// ProcessEntity validates and processes an entity, updating its state
// and persisting changes to the database. It returns an error if validation
// fails or if the database operation encounters an issue.
//
// The function performs the following steps:
// 1. Validates the entity structure and data
// 2. Updates the entity's last modified timestamp
// 3. Persists the entity to the database
// 4. Triggers any registered event handlers
func (s *Service) ProcessEntity(ctx context.Context, entity *Entity) error {
    // Implementation
}
```

### API Documentation

Generate API documentation:

```bash
# Install swag
go install github.com/swaggo/swag/cmd/swag@latest

# Generate docs
swag init -g cmd/server/main.go

# View docs at http://localhost:3001/swagger/index.html
```

## Contribution Guidelines

### Before Contributing

1. **Read the Documentation**: Familiarize yourself with the project structure and goals
2. **Check Issues**: Look for existing issues or create a new one
3. **Fork the Repository**: Create your own fork for development
4. **Set Up Development Environment**: Follow this guide to set up your environment

### Development Process

1. **Create Feature Branch**
```bash
git checkout -b feature/your-feature-name
```

2. **Write Tests First** (TDD approach)
```bash
# Write failing tests
make test  # Should fail

# Implement feature
make test  # Should pass
```

3. **Follow Code Standards**
```bash
# Format code
make fmt

# Run linter
make lint

# Run all tests
make test
```

4. **Update Documentation**
- Update relevant documentation
- Add API documentation for new endpoints
- Update README if needed

5. **Commit Changes**
```bash
git add .
git commit -m "feat: add new feature description"
```

### Commit Message Convention

Follow conventional commits:

```
type(scope): description

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or modifying tests
- `chore`: Maintenance tasks

Examples:
```
feat(api): add entity filtering endpoint
fix(websocket): resolve connection timeout issue
docs(readme): update installation instructions
```

### Pull Request Process

1. **Ensure All Tests Pass**
```bash
make test
make lint
```

2. **Update Documentation**
3. **Create Pull Request** with:
   - Clear description of changes
   - Reference to related issues
   - Screenshots if UI changes
   - Testing instructions

4. **Code Review Process**
   - Address reviewer feedback
   - Keep PR focused and small
   - Ensure CI passes

### Code Review Guidelines

When reviewing code:

1. **Functionality**: Does the code work as intended?
2. **Testing**: Are there adequate tests?
3. **Performance**: Are there any performance implications?
4. **Security**: Are there any security concerns?
5. **Maintainability**: Is the code readable and maintainable?
6. **Documentation**: Is the code properly documented?

For more information, see the [PMA Backend Go Documentation](../README.md).