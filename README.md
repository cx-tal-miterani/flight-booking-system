# Flight Booking System - Temporal Architecture

A microservices-based flight booking system using Go, Temporal, and React.

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────────┐
│   React     │────▶│  API Server │────▶│ Temporal Server │
│   Frontend  │◀────│    (Go)     │◀────│                 │
└─────────────┘     └─────────────┘     └────────┬────────┘
                                                 │
                                        ┌────────▼────────┐
                                        │ Temporal Worker │
                                        │      (Go)       │
                                        └─────────────────┘
```

## Services

- **api-server**: RESTful API for flight booking operations
- **temporal-worker**: Temporal workflow and activity workers
- **frontend**: React-based booking interface

## Business Logic

- **Seat Reservation**: 15-minute hold with auto-release, refreshable timer
- **Payment Validation**: 5-digit code, 10s timeout, 3 retries, 15% failure simulation
- **Order Management**: Status tracking, failure handling, confirmations

## Prerequisites

- Go 1.21+
- Node.js 20+
- Podman or Docker
- Temporal CLI (for local development)

## Quick Start

### Using Podman/Docker Compose

```bash
# Start all services
podman-compose up -d

# Or with Docker
docker compose up -d
```

### Manual Development

```bash
# Terminal 1: Start Temporal Server
temporal server start-dev

# Terminal 2: Start API Server
cd api-server && go run ./cmd/server

# Terminal 3: Start Temporal Worker
cd temporal-worker && go run ./cmd/worker

# Terminal 4: Start Frontend
cd frontend && npm install && npm run dev
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | /api/flights | List available flights |
| POST | /api/orders | Create new booking order |
| GET | /api/orders/:id | Get order status |
| POST | /api/orders/:id/seats | Select/update seats |
| POST | /api/orders/:id/pay | Submit payment code |
| DELETE | /api/orders/:id | Cancel order |

## Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific service tests
cd api-server && go test ./... -v
cd temporal-worker && go test ./... -v
```

## Project Structure

```
flight-booking-system/
├── api-server/           # REST API service
│   ├── cmd/server/       # Entry point
│   ├── internal/
│   │   ├── handlers/     # HTTP handlers
│   │   ├── service/      # Business logic
│   │   └── repository/   # Data access
│   └── Dockerfile
├── temporal-worker/      # Temporal workers
│   ├── cmd/worker/       # Entry point
│   ├── internal/
│   │   ├── workflows/    # Workflow definitions
│   │   └── activities/   # Activity implementations
│   └── Dockerfile
├── shared/               # Shared code
│   └── models/           # Common data models
├── frontend/             # React application
└── docker-compose.yml
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| API_PORT | 8080 | API server port |
| TEMPORAL_HOST | localhost:7233 | Temporal server address |
| TEMPORAL_NAMESPACE | default | Temporal namespace |
| SEAT_HOLD_TIMEOUT | 15m | Seat reservation timeout |
| PAYMENT_TIMEOUT | 10s | Payment validation timeout |
| PAYMENT_RETRIES | 3 | Payment retry attempts |

## License

MIT

