package mocks

import (
	"context"

	"github.com/cx-tal-miterani/flight-booking-system/shared/models"
	"github.com/stretchr/testify/mock"
)

// MockBookingService is a mock implementation of BookingService
type MockBookingService struct {
	mock.Mock
}

func (m *MockBookingService) GetFlights(ctx context.Context) []*models.Flight {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]*models.Flight)
}

func (m *MockBookingService) GetFlight(ctx context.Context, flightID string) (*models.Flight, error) {
	args := m.Called(ctx, flightID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Flight), args.Error(1)
}

func (m *MockBookingService) GetAvailableSeats(ctx context.Context, flightID string) ([]*models.Seat, error) {
	args := m.Called(ctx, flightID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Seat), args.Error(1)
}

func (m *MockBookingService) CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Order), args.Error(1)
}

func (m *MockBookingService) GetOrderStatus(ctx context.Context, orderID string) (*models.OrderStatusResponse, error) {
	args := m.Called(ctx, orderID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrderStatusResponse), args.Error(1)
}

func (m *MockBookingService) SelectSeats(ctx context.Context, orderID string, seatIDs []string) error {
	args := m.Called(ctx, orderID, seatIDs)
	return args.Error(0)
}

func (m *MockBookingService) SubmitPayment(ctx context.Context, orderID string, paymentCode string) error {
	args := m.Called(ctx, orderID, paymentCode)
	return args.Error(0)
}

func (m *MockBookingService) CancelOrder(ctx context.Context, orderID string) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
}

func (m *MockBookingService) RefreshTimer(ctx context.Context, orderID string) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
}

