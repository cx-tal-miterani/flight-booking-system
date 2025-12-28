# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY shared/ /shared/
COPY temporal-worker/go.mod temporal-worker/go.sum* ./
RUN go mod edit -replace github.com/cx-tal-miterani/flight-booking-system/shared=/shared
RUN go mod download

COPY temporal-worker/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./cmd/worker

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /worker .
CMD ["./worker"]

