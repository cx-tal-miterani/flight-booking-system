package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cx-tal-miterani/flight-booking-system/temporal-worker/internal/activities"
	"github.com/cx-tal-miterani/flight-booking-system/temporal-worker/internal/repository"
	"github.com/cx-tal-miterani/flight-booking-system/temporal-worker/internal/workflows"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	ctx := context.Background()

	// Get configuration
	temporalHost := getEnv("TEMPORAL_HOST", "localhost:7233")
	dbURL := getEnv("DATABASE_URL", "postgres://flightbooking:flightbooking123@localhost:5432/flightbooking?sslmode=disable")

	// Connect to database
	log.Println("Connecting to database...")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database")

	// Create repository
	repo := repository.NewRepository(pool)

	// Connect to Temporal
	log.Printf("Connecting to Temporal at %s...", temporalHost)
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}
	defer c.Close()
	log.Println("Connected to Temporal")

	// Create worker
	w := worker.New(c, "flight-booking-queue", worker.Options{})

	// Register workflows
	w.RegisterWorkflow(workflows.BookingWorkflow)

	// Create and register activities
	acts := activities.NewActivities(repo)
	w.RegisterActivityWithOptions(acts.ValidatePayment, worker.RegisterActivityOptions{Name: "ValidatePayment"})
	w.RegisterActivityWithOptions(acts.ReserveSeats, worker.RegisterActivityOptions{Name: "ReserveSeats"})
	w.RegisterActivityWithOptions(acts.ReleaseSeats, worker.RegisterActivityOptions{Name: "ReleaseSeats"})
	w.RegisterActivityWithOptions(acts.SendConfirmation, worker.RegisterActivityOptions{Name: "SendConfirmation"})
	w.RegisterActivityWithOptions(acts.CheckReservationExpiry, worker.RegisterActivityOptions{Name: "CheckReservationExpiry"})
	w.RegisterActivityWithOptions(acts.UpdateOrderStatus, worker.RegisterActivityOptions{Name: "UpdateOrderStatus"})

	// Start worker
	log.Println("Starting Temporal worker...")
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
