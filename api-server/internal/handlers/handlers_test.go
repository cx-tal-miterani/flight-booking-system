package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/database"
	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/service"
	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/service/mocks"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupTestRouter(h *Handler) *mux.Router {
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/flights", h.GetFlights).Methods(http.MethodGet)
	api.HandleFunc("/flights/{id}", h.GetFlight).Methods(http.MethodGet)
	api.HandleFunc("/flights/{id}/seats", h.GetFlightSeats).Methods(http.MethodGet)
	api.HandleFunc("/orders", h.CreateOrder).Methods(http.MethodPost)
	api.HandleFunc("/orders/{id}", h.GetOrder).Methods(http.MethodGet)
	api.HandleFunc("/orders/{id}", h.CancelOrder).Methods(http.MethodDelete)
	api.HandleFunc("/orders/{id}/seats", h.SelectSeats).Methods(http.MethodPost)
	api.HandleFunc("/orders/{id}/pay", h.SubmitPayment).Methods(http.MethodPost)
	return r
}

func TestHandler_GetFlights(t *testing.T) {
	mockService := new(mocks.MockService)
	handler := NewHandler(mockService)
	router := setupTestRouter(handler)

	flightID := uuid.New()
	expectedFlights := []database.Flight{
		{
			ID:             flightID,
			FlightNumber:   "AA123",
			Origin:         "New York",
			Destination:    "Los Angeles",
			PricePerSeat:   150.00,
			AvailableSeats: 100,
		},
	}

	mockService.On("GetFlights", mock.Anything).Return(expectedFlights, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/flights", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var response []database.Flight
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)
	assert.Len(t, response, 1)
	assert.Equal(t, "AA123", response[0].FlightNumber)

	mockService.AssertExpectations(t)
}

func TestHandler_GetFlight(t *testing.T) {
	flightID := uuid.New()

	tests := []struct {
		name           string
		flightID       string
		mockReturn     *database.Flight
		mockError      error
		expectedStatus int
	}{
		{
			name:     "flight found",
			flightID: flightID.String(),
			mockReturn: &database.Flight{
				ID:           flightID,
				FlightNumber: "AA123",
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "flight not found",
			flightID:       uuid.New().String(),
			mockReturn:     nil,
			mockError:      database.ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.MockService)
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
	flightID := uuid.New()
	orderID := uuid.New()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockReturn     *database.Order
		mockError      error
		expectedStatus int
		shouldCallMock bool
	}{
		{
			name: "valid order creation",
			requestBody: service.CreateOrderRequest{
				FlightID:      flightID.String(),
				CustomerEmail: "test@example.com",
				CustomerName:  "John Doe",
			},
			mockReturn: &database.Order{
				ID:            orderID,
				FlightID:      flightID,
				CustomerEmail: "test@example.com",
				CustomerName:  "John Doe",
				Status:        database.OrderStatusPending,
			},
			mockError:      nil,
			expectedStatus: http.StatusCreated,
			shouldCallMock: true,
		},
		{
			name: "missing flight ID",
			requestBody: service.CreateOrderRequest{
				CustomerEmail: "test@example.com",
				CustomerName:  "John Doe",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			shouldCallMock: false,
		},
		{
			name: "missing customer name",
			requestBody: service.CreateOrderRequest{
				FlightID:      flightID.String(),
				CustomerEmail: "test@example.com",
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			shouldCallMock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.MockService)
			handler := NewHandler(mockService)
			router := setupTestRouter(handler)

			body, _ := json.Marshal(tt.requestBody)

			if tt.shouldCallMock {
				mockService.On("CreateOrder", mock.Anything, mock.AnythingOfType("service.CreateOrderRequest")).Return(tt.mockReturn, tt.mockError)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestHandler_SelectSeats(t *testing.T) {
	orderID := uuid.New()

	tests := []struct {
		name           string
		orderID        string
		requestBody    SelectSeatsRequest
		mockReturn     *service.OrderStatusResponse
		mockError      error
		expectedStatus int
		shouldCallMock bool
	}{
		{
			name:    "valid seat selection",
			orderID: orderID.String(),
			requestBody: SelectSeatsRequest{
				SeatIDs: []string{"seat-1", "seat-2"},
			},
			mockReturn: &service.OrderStatusResponse{
				Order:            &database.Order{ID: orderID, Status: database.OrderStatusSeatsSelected},
				RemainingSeconds: 900,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			shouldCallMock: true,
		},
		{
			name:    "no seats selected",
			orderID: orderID.String(),
			requestBody: SelectSeatsRequest{
				SeatIDs: []string{},
			},
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			shouldCallMock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.MockService)
			handler := NewHandler(mockService)
			router := setupTestRouter(handler)

			body, _ := json.Marshal(tt.requestBody)

			if tt.shouldCallMock {
				mockService.On("SelectSeats", mock.Anything, tt.orderID, tt.requestBody.SeatIDs).Return(tt.mockReturn, tt.mockError)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/orders/"+tt.orderID+"/seats", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestHandler_SubmitPayment(t *testing.T) {
	orderID := uuid.New()

	tests := []struct {
		name           string
		orderID        string
		paymentCode    string
		mockReturn     *service.OrderStatusResponse
		mockError      error
		expectedStatus int
		shouldCallMock bool
	}{
		{
			name:        "valid payment code",
			orderID:     orderID.String(),
			paymentCode: "12345",
			mockReturn: &service.OrderStatusResponse{
				Order:            &database.Order{ID: orderID, Status: database.OrderStatusProcessing},
				RemainingSeconds: 800,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			shouldCallMock: true,
		},
		{
			name:           "invalid payment code - too short",
			orderID:        orderID.String(),
			paymentCode:    "1234",
			expectedStatus: http.StatusBadRequest,
			shouldCallMock: false,
		},
		{
			name:           "invalid payment code - too long",
			orderID:        orderID.String(),
			paymentCode:    "123456",
			expectedStatus: http.StatusBadRequest,
			shouldCallMock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.MockService)
			handler := NewHandler(mockService)
			router := setupTestRouter(handler)

			body, _ := json.Marshal(PaymentRequest{PaymentCode: tt.paymentCode})

			if tt.shouldCallMock {
				mockService.On("SubmitPayment", mock.Anything, tt.orderID, tt.paymentCode).Return(tt.mockReturn, tt.mockError)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/orders/"+tt.orderID+"/pay", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
		})
	}
}

func TestHandler_CancelOrder(t *testing.T) {
	orderID := uuid.New()

	tests := []struct {
		name           string
		orderID        string
		mockError      error
		expectedStatus int
	}{
		{
			name:           "successful cancellation",
			orderID:        orderID.String(),
			mockError:      nil,
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "order not found",
			orderID:        uuid.New().String(),
			mockError:      database.ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.MockService)
			handler := NewHandler(mockService)
			router := setupTestRouter(handler)

			mockService.On("CancelOrder", mock.Anything, tt.orderID).Return(tt.mockError)

			req := httptest.NewRequest(http.MethodDelete, "/api/orders/"+tt.orderID, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_GetOrder(t *testing.T) {
	orderID := uuid.New()

	tests := []struct {
		name           string
		orderID        string
		mockReturn     *service.OrderStatusResponse
		mockError      error
		expectedStatus int
	}{
		{
			name:    "order found",
			orderID: orderID.String(),
			mockReturn: &service.OrderStatusResponse{
				Order: &database.Order{
					ID:              orderID,
					Status:          database.OrderStatusSeatsSelected,
					PaymentAttempts: 0,
				},
				RemainingSeconds: 850,
			},
			mockError:      nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "order not found",
			orderID:        uuid.New().String(),
			mockReturn:     nil,
			mockError:      database.ErrNotFound,
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(mocks.MockService)
			handler := NewHandler(mockService)
			router := setupTestRouter(handler)

			mockService.On("GetOrder", mock.Anything, tt.orderID).Return(tt.mockReturn, tt.mockError)

			req := httptest.NewRequest(http.MethodGet, "/api/orders/"+tt.orderID, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			mockService.AssertExpectations(t)
		})
	}
}
