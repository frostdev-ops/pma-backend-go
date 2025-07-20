.PHONY: build run test clean migrate dev version build-prod

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS := -X github.com/frostdev-ops/pma-backend-go/pkg/version.Version=$(VERSION)
LDFLAGS += -X github.com/frostdev-ops/pma-backend-go/pkg/version.GitCommit=$(GIT_COMMIT)
LDFLAGS += -X github.com/frostdev-ops/pma-backend-go/pkg/version.BuildDate=$(BUILD_DATE)

# Build the application
build:
	go build -ldflags="$(LDFLAGS)" -o bin/pma-server cmd/server/main.go

# Run the application
run:
	go run cmd/server/main.go

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -rf bin/

# Run database migrations
migrate:
	go run ./cmd/migrate/main.go migrations "sqlite3://data/pma.db" up

# Development with hot reload
dev:
	air

# Show version information
version:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

# Build for production
build-prod:
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w $(LDFLAGS)" -o bin/pma-server cmd/server/main.go 