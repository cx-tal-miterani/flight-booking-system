# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY shared/ /shared/
COPY api-server/go.mod api-server/go.sum* ./
RUN go mod edit -replace github.com/cx-tal-miterani/flight-booking-system/shared=/shared
RUN go mod download

COPY api-server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /api-server ./cmd/server

# Runtime stage
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /api-server .
EXPOSE 8080
CMD ["./api-server"]

