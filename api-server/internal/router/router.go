package router

import (
	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/handlers"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// NewRouter creates and configures the HTTP router
func NewRouter(h *handlers.Handler) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "http://127.0.0.1:*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health check
	r.Get("/health", h.HealthCheck)

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Flights
		r.Get("/flights", h.GetFlights)
		r.Get("/flights/{id}", h.GetFlight)
		r.Get("/flights/{id}/seats", h.GetFlightSeats)

		// Orders
		r.Post("/orders", h.CreateOrder)
		r.Get("/orders/{id}", h.GetOrder)
		r.Post("/orders/{id}/seats", h.SelectSeats)
		r.Post("/orders/{id}/pay", h.SubmitPayment)
		r.Post("/orders/{id}/refresh", h.RefreshTimer)
		r.Delete("/orders/{id}", h.CancelOrder)
	})

	return r
}

