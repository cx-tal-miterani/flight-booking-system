package activities

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/cx-tal-miterani/flight-booking-system/temporal-worker/internal/repository"
	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
)

// Activities contains all workflow activities
type Activities struct {
	repo *repository.Repository
}

// NewActivities creates a new Activities instance
func NewActivities(repo *repository.Repository) *Activities {
	return &Activities{repo: repo}
}

// ValidatePaymentInput is the input for ValidatePayment activity
type ValidatePaymentInput struct {
	OrderID     string `json:"orderId"`
	PaymentCode string `json:"paymentCode"`
	Attempt     int    `json:"attempt"`
}

// ValidatePaymentOutput is the output for ValidatePayment activity
type ValidatePaymentOutput struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transactionId,omitempty"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
}

// ValidatePayment validates a payment code (simulated)
// 85% success rate, must complete within 10 seconds
func (a *Activities) ValidatePayment(ctx context.Context, input ValidatePaymentInput) (*ValidatePaymentOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Validating payment", "orderId", input.OrderID, "attempt", input.Attempt)

	orderID, err := uuid.Parse(input.OrderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	// Validate payment code format
	if len(input.PaymentCode) != 5 {
		return &ValidatePaymentOutput{
			Success:      false,
			ErrorMessage: "Invalid payment code format",
		}, nil
	}

	// Simulate payment processing time (1-3 seconds)
	processingTime := time.Duration(1000+rand.Intn(2000)) * time.Millisecond
	time.Sleep(processingTime)

	// Simulate 85% success rate
	success := rand.Float32() < 0.85

	if success {
		// Update order status and book seats
		if err := a.repo.UpdateOrderStatus(ctx, orderID, repository.OrderStatusConfirmed); err != nil {
			return nil, fmt.Errorf("failed to update order: %w", err)
		}
		if err := a.repo.BookSeats(ctx, orderID); err != nil {
			return nil, fmt.Errorf("failed to book seats: %w", err)
		}

		transactionID := fmt.Sprintf("TXN-%s-%d", input.OrderID[:8], time.Now().Unix())
		logger.Info("Payment successful", "transactionId", transactionID)

		return &ValidatePaymentOutput{
			Success:       true,
			TransactionID: transactionID,
		}, nil
	}

	// Payment failed - update attempts
	failureReason := "Payment validation failed"
	if err := a.repo.UpdateOrderPayment(ctx, orderID, input.Attempt, &failureReason); err != nil {
		logger.Warn("Failed to update payment attempts", "error", err)
	}

	logger.Info("Payment failed", "attempt", input.Attempt)
	return &ValidatePaymentOutput{
		Success:      false,
		ErrorMessage: "Payment validation failed. Please try again.",
	}, nil
}

// ReserveSeatsInput is the input for ReserveSeats activity
type ReserveSeatsInput struct {
	OrderID string   `json:"orderId"`
	SeatIDs []string `json:"seatIds"`
}

// ReserveSeats activity (seat reservation is handled via API, this is for workflow tracking)
func (a *Activities) ReserveSeats(ctx context.Context, input ReserveSeatsInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Seats reserved via API", "orderId", input.OrderID, "seatCount", len(input.SeatIDs))
	return nil
}

// ReleaseSeatsInput is the input for ReleaseSeats activity
type ReleaseSeatsInput struct {
	OrderID string `json:"orderId"`
	Reason  string `json:"reason"`
}

// ReleaseSeats releases held seats
func (a *Activities) ReleaseSeats(ctx context.Context, input ReleaseSeatsInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Releasing seats", "orderId", input.OrderID, "reason", input.Reason)

	orderID, err := uuid.Parse(input.OrderID)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

	if err := a.repo.ReleaseSeats(ctx, orderID); err != nil {
		return fmt.Errorf("failed to release seats: %w", err)
	}

	// Update order status based on reason
	var status repository.OrderStatus
	switch input.Reason {
	case "expired":
		status = repository.OrderStatusExpired
	case "cancelled":
		status = repository.OrderStatusCancelled
	case "payment_failed":
		status = repository.OrderStatusFailed
	default:
		status = repository.OrderStatusFailed
	}

	if err := a.repo.UpdateOrderStatus(ctx, orderID, status); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

// SendConfirmationInput is the input for SendConfirmation activity
type SendConfirmationInput struct {
	OrderID       string `json:"orderId"`
	CustomerEmail string `json:"customerEmail"`
	CustomerName  string `json:"customerName"`
	FlightNumber  string `json:"flightNumber"`
	TransactionID string `json:"transactionId"`
}

// SendConfirmation sends a booking confirmation (simulated)
func (a *Activities) SendConfirmation(ctx context.Context, input SendConfirmationInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending confirmation email",
		"orderId", input.OrderID,
		"email", input.CustomerEmail,
		"transactionId", input.TransactionID,
	)

	// Simulate sending email
	time.Sleep(500 * time.Millisecond)

	logger.Info("Confirmation email sent successfully")
	return nil
}

// CheckReservationExpiryInput is the input for CheckReservationExpiry activity
type CheckReservationExpiryInput struct {
	OrderID string `json:"orderId"`
}

// CheckReservationExpiry checks if the reservation has expired
func (a *Activities) CheckReservationExpiry(ctx context.Context, input CheckReservationExpiryInput) (bool, error) {
	orderID, err := uuid.Parse(input.OrderID)
	if err != nil {
		return false, fmt.Errorf("invalid order ID: %w", err)
	}

	expiresAt, err := a.repo.GetReservationExpiry(ctx, orderID)
	if err != nil {
		return false, err
	}

	if expiresAt == nil {
		return false, errors.New("no reservation found")
	}

	return time.Now().After(*expiresAt), nil
}

// UpdateOrderStatusInput is the input for UpdateOrderStatus activity
type UpdateOrderStatusInput struct {
	OrderID string `json:"orderId"`
	Status  string `json:"status"`
}

// UpdateOrderStatus updates the order status in the database
func (a *Activities) UpdateOrderStatus(ctx context.Context, input UpdateOrderStatusInput) error {
	orderID, err := uuid.Parse(input.OrderID)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

	status := repository.OrderStatus(input.Status)
	return a.repo.UpdateOrderStatus(ctx, orderID, status)
}
