# Flight Booking System

A microservice-based flight booking system built with **Go**, **React**, **Temporal**, and **PostgreSQL**.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                              Frontend                                │
│                         (React + Vite)                              │
│                           Port: 3000                                │
└─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────┐
│                            API Server                                │
│                          (Go + Gorilla)                              │
│                           Port: 8081                                 │
└─────────────────────────────────────────────────────────────────────┘
                    │                               │
                    ▼                               ▼
┌───────────────────────────────┐   ┌─────────────────────────────────┐
│         PostgreSQL            │   │           Temporal              │
│         Port: 5432            │   │           Port: 7233            │
│   (Flights, Seats, Orders)    │   │    (Workflow Orchestration)     │
└───────────────────────────────┘   └─────────────────────────────────┘
                                                    │
                                                    ▼
                                    ┌─────────────────────────────────┐
                                    │       Temporal Worker           │
                                    │    (Booking Workflows)          │
                                    └─────────────────────────────────┘
```

## Features

- ✅ **Flight Listing**: Browse available flights with seat availability
- ✅ **Seat Selection**: Interactive seat map with real-time availability
- ✅ **15-Minute Timer**: Seat reservation hold with countdown (refreshes on changes)
- ✅ **5-Digit Payment**: Secure payment code validation
- ✅ **85% Success Rate**: Simulated payment validation with retry logic
- ✅ **3 Retry Attempts**: Automatic retry handling for failed payments
- ✅ **Real-time Updates**: Polling for order status changes
- ✅ **Workflow Orchestration**: Temporal-based booking workflow

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.21, Gorilla Mux |
| Frontend | React 18, TypeScript, Vite, Tailwind CSS |
| Database | PostgreSQL 16 |
| Workflow Engine | Temporal |
| UI Components | Radix UI, Shadcn-style |
| Testing | Go: testify, Jest for React |
| Containerization | Docker, Podman |

## Quick Start

### Prerequisites

- Docker or Podman
- Docker Compose
- Go 1.21+ (for local development)
- Node.js 20+ (for frontend development)

### Run with Docker/Podman

```bash
# Start all services
docker-compose up -d

# Or with Podman
podman-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### Access the Application

| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| API Server | http://localhost:8081 |
| Temporal UI | http://localhost:8080 |
| PostgreSQL | localhost:5432 |

### Local Development

#### Database Setup

```bash
# Start PostgreSQL only
docker-compose up -d postgres

# Or run PostgreSQL locally and apply migrations
psql -U flightbooking -d flightbooking -f database/init/001_schema.sql
psql -U flightbooking -d flightbooking -f database/init/002_seed_data.sql
```

#### API Server

```bash
cd api-server
go mod download
DATABASE_URL="postgres://flightbooking:flightbooking123@localhost:5432/flightbooking?sslmode=disable" \
TEMPORAL_HOST="localhost:7233" \
go run cmd/server/main.go
```

#### Temporal Worker

```bash
cd temporal-worker
go mod download
DATABASE_URL="postgres://flightbooking:flightbooking123@localhost:5432/flightbooking?sslmode=disable" \
TEMPORAL_HOST="localhost:7233" \
go run cmd/worker/main.go
```

#### Frontend

```bash
cd frontend
npm install
npm run dev
```

## Testing

### Go Tests

```bash
# API Server
cd api-server
go test -v ./...

# Temporal Worker
cd temporal-worker
go test -v ./...
```

### Frontend Tests

```bash
cd frontend
npm test
npm test -- --coverage
```

## Database Schema

### Tables

| Table | Description |
|-------|-------------|
| `flights` | Flight information (number, origin, destination, times, pricing) |
| `seats` | Seat details (row, column, class, status, price) |
| `orders` | Booking orders (customer info, status, payment attempts) |
| `order_seats` | Junction table for order-seat relationships |

### Seat Statuses

- `available` - Can be selected
- `held` - Temporarily reserved (15-min hold)
- `booked` - Permanently booked after payment

### Order Statuses

- `pending` - Order created
- `seats_selected` - Seats held
- `awaiting_payment` - Waiting for payment
- `processing` - Payment being validated
- `confirmed` - Payment successful
- `failed` - Payment failed after 3 attempts
- `cancelled` - Order cancelled by user
- `expired` - Reservation timer expired

## API Endpoints

### Flights

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/flights` | List all available flights |
| GET | `/api/flights/:id` | Get flight details |
| GET | `/api/flights/:id/seats` | Get seats for a flight |

### Orders

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/orders` | Create a new order |
| GET | `/api/orders/:id` | Get order status |
| POST | `/api/orders/:id/seats` | Select seats (starts/refreshes 15-min timer) |
| POST | `/api/orders/:id/pay` | Submit payment code |
| DELETE | `/api/orders/:id` | Cancel order |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8081 | API server port |
| `DATABASE_URL` | (see docker-compose) | PostgreSQL connection string |
| `TEMPORAL_HOST` | localhost:7233 | Temporal server address |

## Booking Flow

```
1. User selects flight
       │
       ▼
2. User enters customer info → Order created → Workflow started
       │
       ▼
3. User selects seats → 15-minute timer starts
       │
       ├── User modifies seats → Timer refreshes
       │
       ▼
4. User enters 5-digit payment code
       │
       ▼
5. Payment validation (10 seconds, 85% success)
       │
       ├── Success → Seats booked → Confirmation
       │
       └── Failure → Retry (up to 3 times)
               │
               └── 3 failures → Order failed → Seats released
```

## License

MIT
