.PHONY: build run test clean migrate

# Build the application
build:
	go build -o bin/pma-server cmd/server/main.go

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
	migrate -path migrations -database "sqlite3://data/pma.db" up

# Development with hot reload
dev:
	air

# Build for production
build-prod:
	CGO_ENABLED=1 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/pma-server cmd/server/main.go 