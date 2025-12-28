.PHONY: all build test clean docker-build docker-up docker-down help

all: test build

build:
	cd api-server && go build -o ../bin/api-server ./cmd/server
	cd temporal-worker && go build -o ../bin/worker ./cmd/worker

test:
	cd api-server && go test -v ./...
	cd temporal-worker && go test -v ./...

clean:
	rm -rf bin/

deps:
	cd api-server && go mod download
	cd temporal-worker && go mod download

tidy:
	cd api-server && go mod tidy
	cd temporal-worker && go mod tidy

# Docker/Podman commands
docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

podman-build:
	podman-compose build

podman-up:
	podman-compose up -d

podman-down:
	podman-compose down

# Development helpers
dev-temporal:
	temporal server start-dev

dev-api:
	cd api-server && go run ./cmd/server

dev-worker:
	cd temporal-worker && go run ./cmd/worker

help:
	@echo "Flight Booking System - Makefile Commands"
	@echo ""
	@echo "  make build        - Build all services"
	@echo "  make test         - Run all tests"
	@echo "  make docker-up    - Start with Docker"
	@echo "  make podman-up    - Start with Podman"
	@echo "  make dev-temporal - Start Temporal server"
	@echo "  make dev-api      - Start API server"
	@echo "  make dev-worker   - Start worker"

