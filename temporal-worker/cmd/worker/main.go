package main

import (
	"log"
	"os"

	"github.com/cx-tal-miterani/flight-booking-system/temporal-worker/internal/activities"
	"github.com/cx-tal-miterani/flight-booking-system/temporal-worker/internal/workflows"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const (
	TaskQueue   = "flight-booking-queue"
	DefaultHost = "localhost:7233"
)

func main() {
	// Get Temporal host from environment
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = DefaultHost
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer c.Close()

	// Initialize seat inventory with sample data
	initializeSampleData()

	// Create worker
	w := worker.New(c, TaskQueue, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(workflows.BookingWorkflow)

	// Register activities
	w.RegisterActivity(activities.ReserveSeats)
	w.RegisterActivity(activities.ReleaseSeats)
	w.RegisterActivity(activities.ValidatePayment)
	w.RegisterActivity(activities.ConfirmBooking)

	log.Printf("Starting Temporal worker on task queue: %s", TaskQueue)
	log.Printf("Connected to Temporal server at: %s", temporalHost)

	// Start worker
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}

func initializeSampleData() {
	// Initialize sample flights with seats
	flights := []struct {
		id    string
		rows  int
		price float64
	}{
		{"FL001", 30, 150.00},
		{"FL002", 25, 200.00},
		{"FL003", 35, 120.00},
	}

	columns := []string{"A", "B", "C", "D", "E", "F"}

	for _, flight := range flights {
		activities.InitializeInventory(flight.id, flight.rows, columns, flight.price)
		log.Printf("Initialized inventory for flight %s: %d rows x %d columns",
			flight.id, flight.rows, len(columns))
	}
}

