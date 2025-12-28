package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/service/mocks"
	"github.com/cx-tal-miterani/flight-booking-system/shared/models"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTestRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/api/flights", h.GetFlights)
	r.Get("/api/flights/{id}", h.GetFlight)
	r.Get("/api/flights/{id}/seats", h.GetFlightSeats)
	r.Post("/api/orders", h.CreateOrder)
	r.Get("/api/orders/{id}", h.GetOrder)
	r.Post("/api/orders/{id}/seats", h.SelectSeats)
	r.Post("/api/orders/{id}/pay", h.SubmitPayment)
	r.Delete("/api/orders/{id}", h.CancelOrder)
	r.Get("/health", h.HealthCheck)
	return r
}

func TestHandler_GetFlights(t *testing.T) {
	mockService := new(mocks.MockBookingService)
	handler := NewHandler(mockService)
	router := setupTestRouter(handler)

	expectedFlights := []*models.Flight{
		{
			ID:           "FL001",
			FlightNumber: "AA123",
			Origin:       "New York",
			Destination:  "Los Angeles",
			PricePerSeat: 150.00,
		},
	}

	mockService.On("GetFlights", mock.Anything).Return(expectedFlights)

	req := httptest.NewRequest(http.MethodGet, "/api/flights", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []*models.Flight
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)
	assert.Len(t, response, 1)

	mockService.AssertExpectations(t)
}

func TestHandler_GetFlight(t *testing.T) {
	tests := []struct {
		name           string
		flightID       string
		mockReturn     *models.Flight
		mockError      error
		expectedStatus int
	}{
		{
			name:     "flight found",
			flightID: "FL001",
			mockReturn: &models.Flight{
				ID:           "FL001",
				FlightNumber: "AA123",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "flight not found",
			flightID:       "FL999",
			mockReturn:     nil,
			mockError:      errors.New("flight not found"),
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.MockBookingService)
			handler := NewHandler(mockService)
			router := setupTestRouter(handler)

			mockService.On("GetFlight", mock.Anything, tt.flightID).Return(tt.mockReturn, tt.mockError)

			req := httptest.NewRequest(http.MethodGet, "/api/flights/"+tt.flightID, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_CreateOrder(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockReturn     *models.Order
		mockError      error
		expectedStatus int
	}{
		{
			name: "valid order creation",
			requestBody: models.CreateOrderRequest{
				FlightID:      "FL001",
				CustomerEmail: "test@example.com",
				CustomerName:  "John Doe",
			},
			mockReturn: &models.Order{
				ID:            "abc12345",
				FlightID:      "FL001",
				CustomerEmail: "test@example.com",
				CustomerName:  "John Doe",
				Status:        models.OrderStatusPending,
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
		},
		{
			name: "missing flight ID",
			requestBody: models.CreateOrderRequest{
				CustomerEmail: "test@example.com",
				CustomerName:  "John Doe",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.MockBookingService)
			handler := NewHandler(mockService)
			router := setupTestRouter(handler)

			body, _ := json.Marshal(tt.requestBody)

			if tt.mockReturn != nil {
				mockService.On("CreateOrder", mock.Anything, mock.AnythingOfType("*models.CreateOrderRequest")).Return(tt.mockReturn, tt.mockError)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestHandler_SubmitPayment(t *testing.T) {
	tests := []struct {
		name           string
		orderID        string
		paymentCode    string
		expectedStatus int
	}{
		{
			name:           "valid payment code",
			orderID:        "abc12345",
			paymentCode:    "12345",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid payment code - too short",
			orderID:        "abc12345",
			paymentCode:    "1234",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid payment code - contains letters",
			orderID:        "abc12345",
			paymentCode:    "1234a",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.MockBookingService)
			handler := NewHandler(mockService)
			router := setupTestRouter(handler)

			body, _ := json.Marshal(models.PaymentRequest{PaymentCode: tt.paymentCode})

			if tt.expectedStatus == http.StatusOK {
				mockService.On("SubmitPayment", mock.Anything, tt.orderID, tt.paymentCode).Return(nil)
				mockService.On("GetOrderStatus", mock.Anything, tt.orderID).Return(&models.OrderStatusResponse{
					Order: &models.Order{ID: tt.orderID, Status: models.OrderStatusProcessing},
				}, nil)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/orders/"+tt.orderID+"/pay", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestHandler_HealthCheck(t *testing.T) {
	mockService := new(mocks.MockBookingService)
	handler := NewHandler(mockService)
	router := setupTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]string
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

