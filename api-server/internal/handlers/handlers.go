package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/service"
	"github.com/cx-tal-miterani/flight-booking-system/shared/models"
	"github.com/go-chi/chi/v5"
)

// Handler contains HTTP handlers for the API
type Handler struct {
	bookingService service.BookingService
}

// NewHandler creates a new Handler instance
func NewHandler(bookingService service.BookingService) *Handler {
	return &Handler{
		bookingService: bookingService,
	}
}

// Response helpers
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// GetFlights handles GET /api/flights
func (h *Handler) GetFlights(w http.ResponseWriter, r *http.Request) {
	flights := h.bookingService.GetFlights(r.Context())
	respondJSON(w, http.StatusOK, flights)
}

// GetFlight handles GET /api/flights/{id}
func (h *Handler) GetFlight(w http.ResponseWriter, r *http.Request) {
	flightID := chi.URLParam(r, "id")
	flight, err := h.bookingService.GetFlight(r.Context(), flightID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Flight not found")
		return
	}
	respondJSON(w, http.StatusOK, flight)
}

// GetFlightSeats handles GET /api/flights/{id}/seats
func (h *Handler) GetFlightSeats(w http.ResponseWriter, r *http.Request) {
	flightID := chi.URLParam(r, "id")
	seats, err := h.bookingService.GetAvailableSeats(r.Context(), flightID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Flight not found")
		return
	}
	respondJSON(w, http.StatusOK, seats)
}

// CreateOrder handles POST /api/orders
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if req.FlightID == "" {
		respondError(w, http.StatusBadRequest, "Flight ID is required")
		return
	}
	if req.CustomerEmail == "" {
		respondError(w, http.StatusBadRequest, "Customer email is required")
		return
	}
	if req.CustomerName == "" {
		respondError(w, http.StatusBadRequest, "Customer name is required")
		return
	}

	order, err := h.bookingService.CreateOrder(r.Context(), &req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, order)
}

// GetOrder handles GET /api/orders/{id}
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	status, err := h.bookingService.GetOrderStatus(r.Context(), orderID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Order not found")
		return
	}

	// Calculate remaining seconds
	if !status.Order.SeatHoldExpiry.IsZero() && status.Order.Status == models.OrderStatusSeatsSelected {
		remaining := time.Until(status.Order.SeatHoldExpiry)
		if remaining > 0 {
			status.RemainingSeconds = int(remaining.Seconds())
		}
	}

	respondJSON(w, http.StatusOK, status)
}

// SelectSeats handles POST /api/orders/{id}/seats
func (h *Handler) SelectSeats(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	var req models.SelectSeatsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.SeatIDs) == 0 {
		respondError(w, http.StatusBadRequest, "At least one seat must be selected")
		return
	}

	err := h.bookingService.SelectSeats(r.Context(), orderID, req.SeatIDs)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Return updated order status
	status, _ := h.bookingService.GetOrderStatus(r.Context(), orderID)
	respondJSON(w, http.StatusOK, status)
}

// SubmitPayment handles POST /api/orders/{id}/pay
func (h *Handler) SubmitPayment(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	var req models.PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate payment code format
	if len(req.PaymentCode) != 5 {
		respondError(w, http.StatusBadRequest, "Payment code must be exactly 5 digits")
		return
	}
	for _, c := range req.PaymentCode {
		if c < '0' || c > '9' {
			respondError(w, http.StatusBadRequest, "Payment code must contain only digits")
			return
		}
	}

	err := h.bookingService.SubmitPayment(r.Context(), orderID, req.PaymentCode)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Wait a bit for workflow to process, then return status
	time.Sleep(100 * time.Millisecond)
	status, _ := h.bookingService.GetOrderStatus(r.Context(), orderID)
	respondJSON(w, http.StatusOK, status)
}

// CancelOrder handles DELETE /api/orders/{id}
func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	err := h.bookingService.CancelOrder(r.Context(), orderID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Order cancelled"})
}

// RefreshTimer handles POST /api/orders/{id}/refresh
func (h *Handler) RefreshTimer(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")

	err := h.bookingService.RefreshTimer(r.Context(), orderID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	status, _ := h.bookingService.GetOrderStatus(r.Context(), orderID)
	respondJSON(w, http.StatusOK, status)
}

// HealthCheck handles GET /health
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

