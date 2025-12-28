package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cx-tal-miterani/flight-booking-system/shared/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

const (
	TaskQueue = "flight-booking-queue"
)

// BookingService defines the booking service interface
type BookingService interface {
	GetFlights(ctx context.Context) []*models.Flight
	GetFlight(ctx context.Context, flightID string) (*models.Flight, error)
	GetAvailableSeats(ctx context.Context, flightID string) ([]*models.Seat, error)
	CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error)
	GetOrderStatus(ctx context.Context, orderID string) (*models.OrderStatusResponse, error)
	SelectSeats(ctx context.Context, orderID string, seatIDs []string) error
	SubmitPayment(ctx context.Context, orderID string, paymentCode string) error
	CancelOrder(ctx context.Context, orderID string) error
	RefreshTimer(ctx context.Context, orderID string) error
}

// bookingServiceImpl implements BookingService
type bookingServiceImpl struct {
	temporalClient client.Client
	flights        map[string]*models.Flight
}

// NewBookingService creates a new BookingService
func NewBookingService(temporalClient client.Client) BookingService {
	svc := &bookingServiceImpl{
		temporalClient: temporalClient,
		flights:        make(map[string]*models.Flight),
	}
	svc.initializeSampleFlights()
	return svc
}

func (s *bookingServiceImpl) initializeSampleFlights() {
	now := time.Now()
	flights := []*models.Flight{
		{
			ID:             "FL001",
			FlightNumber:   "AA123",
			Origin:         "New York (JFK)",
			Destination:    "Los Angeles (LAX)",
			DepartureTime:  now.Add(24 * time.Hour),
			ArrivalTime:    now.Add(30 * time.Hour),
			TotalSeats:     180,
			AvailableSeats: 180,
			PricePerSeat:   150.00,
		},
		{
			ID:             "FL002",
			FlightNumber:   "UA456",
			Origin:         "Chicago (ORD)",
			Destination:    "Miami (MIA)",
			DepartureTime:  now.Add(48 * time.Hour),
			ArrivalTime:    now.Add(52 * time.Hour),
			TotalSeats:     150,
			AvailableSeats: 150,
			PricePerSeat:   200.00,
		},
		{
			ID:             "FL003",
			FlightNumber:   "DL789",
			Origin:         "San Francisco (SFO)",
			Destination:    "Seattle (SEA)",
			DepartureTime:  now.Add(12 * time.Hour),
			ArrivalTime:    now.Add(14 * time.Hour),
			TotalSeats:     210,
			AvailableSeats: 210,
			PricePerSeat:   120.00,
		},
	}

	for _, f := range flights {
		s.flights[f.ID] = f
	}
}

func (s *bookingServiceImpl) GetFlights(ctx context.Context) []*models.Flight {
	flights := make([]*models.Flight, 0, len(s.flights))
	for _, f := range s.flights {
		flights = append(flights, f)
	}
	return flights
}

func (s *bookingServiceImpl) GetFlight(ctx context.Context, flightID string) (*models.Flight, error) {
	flight, ok := s.flights[flightID]
	if !ok {
		return nil, fmt.Errorf("flight not found: %s", flightID)
	}
	return flight, nil
}

func (s *bookingServiceImpl) GetAvailableSeats(ctx context.Context, flightID string) ([]*models.Seat, error) {
	flight, ok := s.flights[flightID]
	if !ok {
		return nil, fmt.Errorf("flight not found: %s", flightID)
	}

	// Generate seat list (this would come from the worker/database in production)
	columns := []string{"A", "B", "C", "D", "E", "F"}
	rows := flight.TotalSeats / len(columns)

	seats := make([]*models.Seat, 0, flight.TotalSeats)
	for row := 1; row <= rows; row++ {
		for _, col := range columns {
			seatID := fmt.Sprintf("%s-%d%s", flightID, row, col)
			seats = append(seats, &models.Seat{
				ID:       seatID,
				FlightID: flightID,
				Row:      row,
				Column:   col,
				Class:    models.SeatClassEconomy,
				Status:   models.SeatStatusAvailable,
				Price:    flight.PricePerSeat,
			})
		}
	}
	return seats, nil
}

func (s *bookingServiceImpl) CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error) {
	// Validate flight exists
	if _, ok := s.flights[req.FlightID]; !ok {
		return nil, fmt.Errorf("flight not found: %s", req.FlightID)
	}

	// Generate order ID
	orderID := uuid.New().String()[:8]

	// Create workflow input
	input := models.BookingWorkflowInput{
		OrderID:       orderID,
		FlightID:      req.FlightID,
		CustomerEmail: req.CustomerEmail,
		CustomerName:  req.CustomerName,
	}

	// Start the booking workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        "booking-" + orderID,
		TaskQueue: TaskQueue,
	}

	_, err := s.temporalClient.ExecuteWorkflow(ctx, workflowOptions, "BookingWorkflow", input)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}

	// Return initial order state
	order := &models.Order{
		ID:            orderID,
		FlightID:      req.FlightID,
		CustomerEmail: req.CustomerEmail,
		CustomerName:  req.CustomerName,
		Status:        models.OrderStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	return order, nil
}

func (s *bookingServiceImpl) GetOrderStatus(ctx context.Context, orderID string) (*models.OrderStatusResponse, error) {
	workflowID := "booking-" + orderID

	// Query the workflow for current state
	response, err := s.temporalClient.QueryWorkflow(ctx, workflowID, "", models.QueryGetState)
	if err != nil {
		return nil, fmt.Errorf("failed to query workflow: %w", err)
	}

	var state models.BookingWorkflowState
	if err := response.Get(&state); err != nil {
		return nil, fmt.Errorf("failed to decode workflow state: %w", err)
	}

	// Build order from state
	order := &models.Order{
		ID:              orderID,
		Seats:           state.SeatIDs,
		Status:          state.Status,
		TotalAmount:     state.TotalAmount,
		PaymentAttempts: state.PaymentAttempts,
		SeatHoldExpiry:  state.SeatHoldExpiry,
		FailureReason:   state.FailureReason,
		UpdatedAt:       state.LastUpdated,
	}

	// Calculate remaining time
	var remainingSeconds int
	if !state.SeatHoldExpiry.IsZero() && state.Status == models.OrderStatusSeatsSelected {
		remaining := time.Until(state.SeatHoldExpiry)
		if remaining > 0 {
			remainingSeconds = int(remaining.Seconds())
		}
	}

	return &models.OrderStatusResponse{
		Order:            order,
		RemainingSeconds: remainingSeconds,
	}, nil
}

func (s *bookingServiceImpl) SelectSeats(ctx context.Context, orderID string, seatIDs []string) error {
	workflowID := "booking-" + orderID

	signal := models.SelectSeatsSignal{
		SeatIDs: seatIDs,
	}

	return s.temporalClient.SignalWorkflow(ctx, workflowID, "", models.SignalSelectSeats, signal)
}

func (s *bookingServiceImpl) SubmitPayment(ctx context.Context, orderID string, paymentCode string) error {
	workflowID := "booking-" + orderID

	signal := models.SubmitPaymentSignal{
		PaymentCode: paymentCode,
	}

	return s.temporalClient.SignalWorkflow(ctx, workflowID, "", models.SignalSubmitPayment, signal)
}

func (s *bookingServiceImpl) CancelOrder(ctx context.Context, orderID string) error {
	workflowID := "booking-" + orderID
	return s.temporalClient.SignalWorkflow(ctx, workflowID, "", models.SignalCancelOrder, nil)
}

func (s *bookingServiceImpl) RefreshTimer(ctx context.Context, orderID string) error {
	workflowID := "booking-" + orderID
	return s.temporalClient.SignalWorkflow(ctx, workflowID, "", models.SignalRefreshTimer, nil)
}

