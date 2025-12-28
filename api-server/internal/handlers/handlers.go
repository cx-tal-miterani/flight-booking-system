package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/database"
	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/service"
	"github.com/gorilla/mux"
)

// Handler contains all HTTP handlers
type Handler struct {
	service service.Service
}

// NewHandler creates a new handler
func NewHandler(svc service.Service) *Handler {
	return &Handler{service: svc}
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// respondError sends an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// GetFlights handles GET /api/flights
func (h *Handler) GetFlights(w http.ResponseWriter, r *http.Request) {
	flights, err := h.service.GetFlights(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, flights)
}

// GetFlight handles GET /api/flights/{id}
func (h *Handler) GetFlight(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	flight, err := h.service.GetFlight(r.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "Flight not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, flight)
}

// GetFlightSeats handles GET /api/flights/{id}/seats
func (h *Handler) GetFlightSeats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	flightID := vars["id"]

	seats, err := h.service.GetFlightSeats(r.Context(), flightID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, seats)
}

// CreateOrder handles POST /api/orders
func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req service.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.FlightID == "" || req.CustomerName == "" || req.CustomerEmail == "" {
		respondError(w, http.StatusBadRequest, "Missing required fields")
		return
	}

	order, err := h.service.CreateOrder(r.Context(), req)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, order)
}

// GetOrder handles GET /api/orders/{id}
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	status, err := h.service.GetOrder(r.Context(), id)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "Order not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, status)
}

// SelectSeatsRequest represents the request body for seat selection
type SelectSeatsRequest struct {
	SeatIDs []string `json:"seatIds"`
}

// SelectSeats handles POST /api/orders/{id}/seats
func (h *Handler) SelectSeats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]

	var req SelectSeatsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.SeatIDs) == 0 {
		respondError(w, http.StatusBadRequest, "No seats selected")
		return
	}

	status, err := h.service.SelectSeats(r.Context(), orderID, req.SeatIDs)
	if err != nil {
		if errors.Is(err, database.ErrSeatNotAvailable) {
			respondError(w, http.StatusConflict, "One or more seats are not available")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, status)
}

// PaymentRequest represents the request body for payment
type PaymentRequest struct {
	PaymentCode string `json:"paymentCode"`
}

// SubmitPayment handles POST /api/orders/{id}/pay
func (h *Handler) SubmitPayment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]

	var req PaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.PaymentCode) != 5 {
		respondError(w, http.StatusBadRequest, "Payment code must be 5 digits")
		return
	}

	status, err := h.service.SubmitPayment(r.Context(), orderID, req.PaymentCode)
	if err != nil {
		if errors.Is(err, database.ErrOrderExpired) {
			respondError(w, http.StatusGone, "Reservation has expired")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, status)
}

// CancelOrder handles DELETE /api/orders/{id}
func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	orderID := vars["id"]

	err := h.service.CancelOrder(r.Context(), orderID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			respondError(w, http.StatusNotFound, "Order not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
