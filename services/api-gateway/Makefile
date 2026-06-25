.PHONY: help build run test clean swagger lint

# Default target
help:
	@echo "Available targets:"
	@echo "  make build        - Build the API Gateway binary"
	@echo "  make run          - Run the API Gateway locally"
	@echo "  make dev          - Run with hot-reload (requires air)"
	@echo "  make test         - Run tests"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make swagger      - Generate Swagger documentation"
	@echo "  make lint         - Run linter (requires golangci-lint)"
	@echo "  make deps         - Download dependencies"

# Build the binary
build:
	@echo "Building API Gateway..."
	@go build -v -o bin/api-gateway ./cmd/main.go

# Run locally
run: build
	@echo "Running API Gateway..."
	@./bin/api-gateway

# Development mode with hot-reload
dev:
	@echo "Running with hot-reload (requires air)..."
	@air

# Run tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out

# Clean
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out
	@go clean

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Generate Swagger docs (requires swag)
swagger:
	@echo "Generating Swagger documentation..."
	@swag init -g cmd/main.go

# Run linter (requires golangci-lint)
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Docker build
docker-build:
	@echo "Building Docker image..."
	@docker build -t dms-api-gateway:latest .

# Docker run
docker-run: docker-build
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --env-file .env dms-api-gateway:latest
