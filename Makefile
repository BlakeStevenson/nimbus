.PHONY: help build run test clean migrate-up migrate-down sqlc-generate dev install-tools docker-up docker-down

# Default target
help:
	@echo "Available targets:"
	@echo "  make build          - Build the server binary"
	@echo "  make run            - Run the server"
	@echo "  make dev            - Run the server in development mode with auto-reload"
	@echo "  make test           - Run tests"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make migrate-up     - Run database migrations"
	@echo "  make migrate-down   - Rollback last database migration"
	@echo "  make sqlc-generate  - Generate sqlc code"
	@echo "  make install-tools  - Install development tools"
	@echo "  make docker-up      - Start PostgreSQL with Docker"
	@echo "  make docker-down    - Stop PostgreSQL Docker container"

# Build the server binary
build:
	@echo "Building server..."
	go build -o bin/server ./cmd/server

# Run the server
run: build
	@echo "Starting server..."
	./bin/server

# Run in development mode
dev:
	@echo "Starting development server..."
	@echo "Loading configuration from .env file..."
	go run ./cmd/server

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run database migrations
migrate-up:
	@echo "Running migrations..."
	psql $(DATABASE_URL) -f internal/db/migrations/0001_init.sql

# Note: For a production app, use a proper migration tool like golang-migrate
# This is just a simple example for initial setup

# Generate sqlc code
sqlc-generate:
	@echo "Generating sqlc code..."
	cd internal/db && sqlc generate

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@echo "Tools installed successfully!"

# Start PostgreSQL with Docker
docker-up:
	@echo "Starting PostgreSQL..."
	docker run -d \
		--name nimbus-postgres \
		-e POSTGRES_USER=nimbus \
		-e POSTGRES_PASSWORD=nimbus \
		-e POSTGRES_DB=nimbus \
		-p 5432:5432 \
		postgres:16-alpine
	@echo "Waiting for PostgreSQL to be ready..."
	sleep 3
	@echo "PostgreSQL is ready!"

# Stop PostgreSQL Docker container
docker-down:
	@echo "Stopping PostgreSQL..."
	docker stop nimbus-postgres || true
	docker rm nimbus-postgres || true

# Setup development environment
setup-dev: install-tools docker-up
	@echo "Waiting for database to be ready..."
	sleep 3
	@echo "Running migrations..."
	DATABASE_URL="postgres://nimbus:nimbus@localhost:5432/nimbus?sslmode=disable" make migrate-up
	@echo "Generating sqlc code..."
	make sqlc-generate
	@echo "Development environment ready!"
	@echo "Run 'make dev' to start the server"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy
