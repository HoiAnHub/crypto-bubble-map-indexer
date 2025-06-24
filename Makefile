.PHONY: help setup build run test clean up down logs status cleanup

# Default target
help:
	@echo "Available commands:"
	@echo "  setup     - Setup development environment"
	@echo "  build     - Build the application"
	@echo "  run       - Run the application locally"
	@echo "  test      - Run tests"
	@echo "  clean     - Clean build artifacts"
	@echo "  up        - Start services with Docker Compose"
	@echo "  down      - Stop services"
	@echo "  logs      - View logs"
	@echo "  status    - Check service status"
	@echo "  cleanup   - Remove Transaction nodes and convert to direct Wallet relationships"

# Setup development environment
setup:
	@echo "Setting up development environment..."
	go mod download
	go mod tidy
	@echo "Setup complete!"

# Build the application
build:
	@echo "Building application..."
	go build -o bin/indexer cmd/indexer/main.go
	@echo "Build complete!"

# Run the application locally
run:
	@echo "Running application..."
	go run cmd/indexer/main.go

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "Clean complete!"

# Start services with Docker Compose
up:
	@echo "Starting services..."
	docker-compose up -d

# Stop services
down:
	@echo "Stopping services..."
	docker-compose down

# View logs
logs:
	docker-compose logs -f

# Check service status
status:
	@echo "Service status:"
	docker-compose ps

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t crypto-bubble-map-indexer .

# Run with fresh build
up-fresh: clean build up

# Development mode with hot reload
dev:
	@echo "Starting development mode..."
	air

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/cosmtrek/air@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Generate mocks
generate:
	@echo "Generating mocks..."
	mockgen -source=internal/domain/repository/wallet_repository.go -destination=internal/domain/repository/mocks/wallet_repository_mock.go
	mockgen -source=internal/domain/repository/transaction_repository.go -destination=internal/domain/repository/mocks/transaction_repository_mock.go

# Run the database cleanup script to remove Transaction nodes
cleanup:
	@echo "Removing Transaction nodes and creating direct Wallet relationships..."
	go run scripts/cleanup_transaction_nodes.go
	@echo "Cleanup complete!"