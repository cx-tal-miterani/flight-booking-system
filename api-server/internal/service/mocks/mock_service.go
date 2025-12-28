package mocks

import (
	"context"

	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/database"
	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/service"
	"github.com/stretchr/testify/mock"
)

// MockService is a mock implementation of the Service interface
type MockService struct {
	mock.Mock
}

// Ensure MockService implements service.Service
var _ service.Service = (*MockService)(nil)

func (m *MockService) GetFlights(ctx context.Context) ([]database.Flight, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]database.Flight), args.Error(1)
}

func (m *MockService) GetFlight(ctx context.Context, id string) (*database.Flight, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.Flight), args.Error(1)
}

func (m *MockService) GetFlightSeats(ctx context.Context, flightID string) ([]database.Seat, error) {
	args := m.Called(ctx, flightID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]database.Seat), args.Error(1)
}

func (m *MockService) CreateOrder(ctx context.Context, req service.CreateOrderRequest) (*database.Order, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*database.Order), args.Error(1)
}

func (m *MockService) GetOrder(ctx context.Context, id string) (*service.OrderStatusResponse, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.OrderStatusResponse), args.Error(1)
}

func (m *MockService) SelectSeats(ctx context.Context, orderID string, seatIDs []string) (*service.OrderStatusResponse, error) {
	args := m.Called(ctx, orderID, seatIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.OrderStatusResponse), args.Error(1)
}

func (m *MockService) SubmitPayment(ctx context.Context, orderID string, paymentCode string) (*service.OrderStatusResponse, error) {
	args := m.Called(ctx, orderID, paymentCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.OrderStatusResponse), args.Error(1)
}

func (m *MockService) CancelOrder(ctx context.Context, orderID string) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
}
