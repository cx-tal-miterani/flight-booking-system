package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cx-tal-miterani/flight-booking-system/api-server/internal/database"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"
)

// Service defines the interface for business logic
type Service interface {
	// Flights
	GetFlights(ctx context.Context) ([]database.Flight, error)
	GetFlight(ctx context.Context, id string) (*database.Flight, error)
	GetFlightSeats(ctx context.Context, flightID string) ([]database.Seat, error)

	// Orders
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*database.Order, error)
	GetOrder(ctx context.Context, id string) (*OrderStatusResponse, error)
	SelectSeats(ctx context.Context, orderID string, seatIDs []string) (*OrderStatusResponse, error)
	SubmitPayment(ctx context.Context, orderID string, paymentCode string) (*OrderStatusResponse, error)
	CancelOrder(ctx context.Context, orderID string) error
}

// CreateOrderRequest represents a request to create an order
type CreateOrderRequest struct {
	FlightID      string `json:"flightId"`
	CustomerName  string `json:"customerName"`
	CustomerEmail string `json:"customerEmail"`
}

// OrderStatusResponse represents the response for order status
type OrderStatusResponse struct {
	Order            *database.Order `json:"order"`
	RemainingSeconds int             `json:"remainingSeconds"`
}

// BookingService implements the Service interface
type BookingService struct {
	repo           *database.Repository
	temporalClient client.Client
}

// NewBookingService creates a new booking service
func NewBookingService(repo *database.Repository, temporalClient client.Client) *BookingService {
	return &BookingService{
		repo:           repo,
		temporalClient: temporalClient,
	}
}

// GetFlights returns all available flights
func (s *BookingService) GetFlights(ctx context.Context) ([]database.Flight, error) {
	return s.repo.GetAllFlights(ctx)
}

// GetFlight returns a flight by ID
func (s *BookingService) GetFlight(ctx context.Context, id string) (*database.Flight, error) {
	flightID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid flight ID: %w", err)
	}
	return s.repo.GetFlightByID(ctx, flightID)
}

// GetFlightSeats returns seats for a flight
func (s *BookingService) GetFlightSeats(ctx context.Context, flightID string) ([]database.Seat, error) {
	id, err := uuid.Parse(flightID)
	if err != nil {
		return nil, fmt.Errorf("invalid flight ID: %w", err)
	}
	return s.repo.GetFlightSeats(ctx, id)
}

// CreateOrder creates a new booking order and starts the Temporal workflow
func (s *BookingService) CreateOrder(ctx context.Context, req CreateOrderRequest) (*database.Order, error) {
	flightID, err := uuid.Parse(req.FlightID)
	if err != nil {
		return nil, fmt.Errorf("invalid flight ID: %w", err)
	}

	// Verify flight exists
	_, err = s.repo.GetFlightByID(ctx, flightID)
	if err != nil {
		return nil, fmt.Errorf("flight not found: %w", err)
	}

	// Create order
	order := &database.Order{
		ID:            uuid.New(),
		FlightID:      flightID,
		CustomerName:  req.CustomerName,
		CustomerEmail: req.CustomerEmail,
		Status:        database.OrderStatusPending,
	}

	// Start Temporal workflow
	workflowOptions := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("booking-%s", order.ID.String()),
		TaskQueue: "flight-booking-queue",
	}

	workflowInput := map[string]interface{}{
		"orderId":       order.ID.String(),
		"flightId":      flightID.String(),
		"customerName":  req.CustomerName,
		"customerEmail": req.CustomerEmail,
	}

	we, err := s.temporalClient.ExecuteWorkflow(ctx, workflowOptions, "BookingWorkflow", workflowInput)
	if err != nil {
		return nil, fmt.Errorf("failed to start workflow: %w", err)
	}

	workflowID := we.GetID()
	runID := we.GetRunID()
	order.WorkflowID = &workflowID
	order.WorkflowRunID = &runID

	// Save order to database
	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return order, nil
}

// GetOrder returns order status
func (s *BookingService) GetOrder(ctx context.Context, id string) (*OrderStatusResponse, error) {
	orderID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	remaining, _ := s.repo.GetOrderRemainingSeconds(ctx, orderID)

	return &OrderStatusResponse{
		Order:            order,
		RemainingSeconds: remaining,
	}, nil
}

// SelectSeats selects seats for an order
func (s *BookingService) SelectSeats(ctx context.Context, orderID string, seatIDs []string) (*OrderStatusResponse, error) {
	oid, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	order, err := s.repo.GetOrderByID(ctx, oid)
	if err != nil {
		return nil, err
	}

	// Parse seat IDs (they come as "flightID-seatNumber" format from frontend)
	var seatUUIDs []uuid.UUID
	var seatNumbers []string
	
	for _, sid := range seatIDs {
		// Try parsing as UUID first
		if id, err := uuid.Parse(sid); err == nil {
			seatUUIDs = append(seatUUIDs, id)
		} else {
			// Extract seat number from "flightID-seatNumber" format
			seatNumbers = append(seatNumbers, extractSeatNumber(sid))
		}
	}

	// If we have seat numbers, convert to UUIDs
	if len(seatNumbers) > 0 {
		ids, err := s.repo.GetSeatIDsByFlightAndNumbers(ctx, order.FlightID, seatNumbers)
		if err != nil {
			return nil, fmt.Errorf("failed to get seat IDs: %w", err)
		}
		seatUUIDs = append(seatUUIDs, ids...)
	}

	if len(seatUUIDs) == 0 {
		return nil, errors.New("no valid seats selected")
	}

	// Hold seats (this refreshes the 15-minute timer)
	if err := s.repo.HoldSeats(ctx, oid, seatUUIDs); err != nil {
		return nil, fmt.Errorf("failed to hold seats: %w", err)
	}

	// Update order seats
	if err := s.repo.SetOrderSeats(ctx, oid, seatUUIDs); err != nil {
		return nil, fmt.Errorf("failed to set order seats: %w", err)
	}

	// Signal workflow about seat selection
	if order.WorkflowID != nil {
		err = s.temporalClient.SignalWorkflow(ctx, *order.WorkflowID, "", "seats-selected", map[string]interface{}{
			"seatIds":   seatIDs,
			"expiresAt": time.Now().Add(15 * time.Minute),
		})
		if err != nil {
			// Log but don't fail - order is already updated
			fmt.Printf("Warning: failed to signal workflow: %v\n", err)
		}
	}

	return s.GetOrder(ctx, orderID)
}

// SubmitPayment submits payment for an order
func (s *BookingService) SubmitPayment(ctx context.Context, orderID string, paymentCode string) (*OrderStatusResponse, error) {
	oid, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	order, err := s.repo.GetOrderByID(ctx, oid)
	if err != nil {
		return nil, err
	}

	// Check if reservation expired
	remaining, _ := s.repo.GetOrderRemainingSeconds(ctx, oid)
	if remaining <= 0 {
		s.repo.UpdateOrderStatus(ctx, oid, database.OrderStatusExpired)
		s.repo.ReleaseSeats(ctx, oid)
		return nil, database.ErrOrderExpired
	}

	// Update status to processing
	s.repo.UpdateOrderStatus(ctx, oid, database.OrderStatusProcessing)

	// Signal workflow to process payment
	if order.WorkflowID != nil {
		err = s.temporalClient.SignalWorkflow(ctx, *order.WorkflowID, "", "payment-submitted", map[string]interface{}{
			"paymentCode": paymentCode,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to signal payment: %w", err)
		}
	}

	return s.GetOrder(ctx, orderID)
}

// CancelOrder cancels an order
func (s *BookingService) CancelOrder(ctx context.Context, orderID string) error {
	oid, err := uuid.Parse(orderID)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

	order, err := s.repo.GetOrderByID(ctx, oid)
	if err != nil {
		return err
	}

	// Release seats
	s.repo.ReleaseSeats(ctx, oid)

	// Update status
	s.repo.UpdateOrderStatus(ctx, oid, database.OrderStatusCancelled)

	// Cancel workflow
	if order.WorkflowID != nil {
		s.temporalClient.CancelWorkflow(ctx, *order.WorkflowID, "")
	}

	return nil
}

// extractSeatNumber extracts seat number from "flightID-seatNumber" format
func extractSeatNumber(seatID string) string {
	// Handle format like "550e8400-e29b-41d4-a716-446655440001-1A"
	// or just "1A"
	for i := len(seatID) - 1; i >= 0; i-- {
		if seatID[i] == '-' {
			return seatID[i+1:]
		}
	}
	return seatID
}
