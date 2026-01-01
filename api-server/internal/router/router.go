package router

import (
	"net/http"

	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/handlers"
	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/websocket"
	"github.com/gorilla/mux"
)

// SetupRouter creates and configures the HTTP router
func SetupRouter(h *handlers.Handler) *mux.Router {
	r := mux.NewRouter()

	// CORS middleware
	r.Use(corsMiddleware)

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Flights
	api.HandleFunc("/flights", h.GetFlights).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/flights/{id}", h.GetFlight).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/flights/{id}/seats", h.GetFlightSeats).Methods(http.MethodGet, http.MethodOptions)

	// Orders
	api.HandleFunc("/orders", h.CreateOrder).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/orders/{id}", h.GetOrder).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/orders/{id}", h.CancelOrder).Methods(http.MethodDelete, http.MethodOptions)
	api.HandleFunc("/orders/{id}/seats", h.SelectSeats).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/orders/{id}/pay", h.SubmitPayment).Methods(http.MethodPost, http.MethodOptions)

	// WebSocket for real-time updates
	api.HandleFunc("/flights/{flightId}/ws", websocket.HandleWebSocket)

	// Health check
	r.HandleFunc("/health", healthCheck).Methods(http.MethodGet)

	return r
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}
