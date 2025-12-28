package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/handlers"
	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/router"
	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/service"
	"go.temporal.io/sdk/client"
)

const (
	DefaultPort         = "8080"
	DefaultTemporalHost = "localhost:7233"
)

func main() {
	// Get configuration from environment
	port := os.Getenv("API_PORT")
	if port == "" {
		port = DefaultPort
	}

	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = DefaultTemporalHost
	}

	// Create Temporal client
	temporalClient, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer temporalClient.Close()

	// Initialize services
	bookingService := service.NewBookingService(temporalClient)

	// Initialize handlers
	h := handlers.NewHandler(bookingService)

	// Create router
	r := router.NewRouter(h)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("API Server starting on port %s", port)
		log.Printf("Connected to Temporal server at %s", temporalHost)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

