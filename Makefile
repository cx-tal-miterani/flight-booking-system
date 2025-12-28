.PHONY: all build test run clean docker-build docker-up docker-down db-init

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod

# Docker
DOCKER=docker
DOCKER_COMPOSE=docker-compose

# Default target
all: test build

# Build all services
build:
	@echo "Building API Server..."
	cd api-server && $(GOBUILD) -o ../bin/api-server ./cmd/server
	@echo "Building Temporal Worker..."
	cd temporal-worker && $(GOBUILD) -o ../bin/temporal-worker ./cmd/worker
	@echo "Build complete!"

# Run tests
test: test-go test-frontend

test-go:
	@echo "Testing API Server..."
	cd api-server && $(GOTEST) -v -race ./...
	@echo "Testing Temporal Worker..."
	cd temporal-worker && $(GOTEST) -v -race ./...

test-frontend:
	@echo "Testing Frontend..."
	cd frontend && npm test -- --watchAll=false

# Download dependencies
deps:
	cd api-server && $(GOMOD) download
	cd temporal-worker && $(GOMOD) download
	cd frontend && npm install

# Clean build artifacts
clean:
	rm -rf bin/
	cd frontend && rm -rf node_modules dist

# Docker operations
docker-build:
	$(DOCKER_COMPOSE) build

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down

docker-logs:
	$(DOCKER_COMPOSE) logs -f

docker-restart:
	$(DOCKER_COMPOSE) restart

# Database operations
db-up:
	$(DOCKER_COMPOSE) up -d postgres
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5

db-init: db-up
	@echo "Database initialized via docker-entrypoint-initdb.d"

db-shell:
	$(DOCKER) exec -it flight-booking-postgres psql -U flightbooking -d flightbooking

db-reset:
	$(DOCKER_COMPOSE) down -v postgres
	$(DOCKER_COMPOSE) up -d postgres

# Development - run locally
dev-api: db-up
	cd api-server && DATABASE_URL="postgres://flightbooking:flightbooking123@localhost:5432/flightbooking?sslmode=disable" \
		TEMPORAL_HOST="localhost:7233" \
		$(GOCMD) run ./cmd/server

dev-worker: db-up
	cd temporal-worker && DATABASE_URL="postgres://flightbooking:flightbooking123@localhost:5432/flightbooking?sslmode=disable" \
		TEMPORAL_HOST="localhost:7233" \
		$(GOCMD) run ./cmd/worker

dev-frontend:
	cd frontend && npm run dev

# Run all in Docker
run: docker-up
	@echo "All services started!"
	@echo "Frontend: http://localhost:3000"
	@echo "API: http://localhost:8081"
	@echo "Temporal UI: http://localhost:8080"

# Stop all
stop: docker-down
	@echo "All services stopped!"

# Linting
lint:
	cd api-server && golangci-lint run
	cd temporal-worker && golangci-lint run

# Help
help:
	@echo "Available targets:"
	@echo "  all          - Run tests and build"
	@echo "  build        - Build Go services"
	@echo "  test         - Run all tests"
	@echo "  test-go      - Run Go tests"
	@echo "  test-frontend - Run frontend tests"
	@echo "  deps         - Download dependencies"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker images"
	@echo "  docker-up    - Start Docker containers"
	@echo "  docker-down  - Stop Docker containers"
	@echo "  docker-logs  - View Docker logs"
	@echo "  db-up        - Start PostgreSQL"
	@echo "  db-shell     - Open PostgreSQL shell"
	@echo "  db-reset     - Reset database"
	@echo "  dev-api      - Run API server locally"
	@echo "  dev-worker   - Run Temporal worker locally"
	@echo "  dev-frontend - Run frontend locally"
	@echo "  run          - Start all services in Docker"
	@echo "  stop         - Stop all services"
	@echo "  lint         - Run linters"
