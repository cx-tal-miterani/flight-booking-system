package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/database"
	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/handlers"
	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/router"
	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/service"
	"go.temporal.io/sdk/client"
)

func main() {
	ctx := context.Background()

	// Get configuration from environment
	port := getEnv("PORT", "8081")
	dbURL := getEnv("DATABASE_URL", "postgres://flightbooking:flightbooking123@localhost:5432/flightbooking?sslmode=disable")
	temporalHost := getEnv("TEMPORAL_HOST", "localhost:7233")

	// Connect to database
	log.Println("Connecting to database...")
	dbConfig := database.DefaultConfig(dbURL)
	pool, err := database.Connect(ctx, dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to database")

	// Create repository
	repo := database.NewRepository(pool)

	// Connect to Temporal
	log.Printf("Connecting to Temporal at %s...", temporalHost)
	temporalClient, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Temporal: %v", err)
	}
	defer temporalClient.Close()
	log.Println("Connected to Temporal")

	// Create service and handlers
	svc := service.NewBookingService(repo, temporalClient)
	h := handlers.NewHandler(svc)

	// Setup router
	r := router.SetupRouter(h)

	// Create server
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("API Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
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

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
