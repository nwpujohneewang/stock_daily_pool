.PHONY: build run test lint fmt migrate-up clean

# Build
build:
	go build -o bin/stock-monitor ./cmd/server

# Run
run:
	go run ./cmd/server

# Test
test:
	go test ./... -v -count=1 -race

test-cover:
	go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

# Lint
lint:
	golangci-lint run ./...

# Format
fmt:
	gofmt -w .

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out

# Dependencies
deps:
	go mod download
	go mod tidy

# Database migrations (requires psql)
migrate-up:
	@echo "Run migrations manually with: psql -h localhost -U stock_monitor -d stock_monitor -f migrations/001_create_stock_basic_info.sql"
	@echo "Then run subsequent migration files in order."

# Development
dev:
	GIN_MODE=debug go run ./cmd/server

# Help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binary"
	@echo "  run         - Run the server"
	@echo "  test        - Run tests"
	@echo "  lint        - Run linter"
	@echo "  fmt         - Format code"
	@echo "  deps        - Download dependencies"
	@echo "  clean       - Clean build artifacts"
	@echo "  dev         - Run in development mode"
