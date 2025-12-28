package models

import "time"

// Order represents a flight booking order
type Order struct {
	ID              string      `json:"id"`
	FlightID        string      `json:"flightId"`
	CustomerEmail   string      `json:"customerEmail"`
	CustomerName    string      `json:"customerName"`
	Seats           []string    `json:"seats"` // Seat IDs
	Status          OrderStatus `json:"status"`
	TotalAmount     float64     `json:"totalAmount"`
	PaymentCode     string      `json:"paymentCode,omitempty"`
	PaymentAttempts int         `json:"paymentAttempts"`
	SeatHoldExpiry  time.Time   `json:"seatHoldExpiry"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
	ConfirmedAt     *time.Time  `json:"confirmedAt,omitempty"`
	FailureReason   string      `json:"failureReason,omitempty"`
}

type OrderStatus string

const (
	OrderStatusPending         OrderStatus = "pending"
	OrderStatusSeatsSelected   OrderStatus = "seats_selected"
	OrderStatusAwaitingPayment OrderStatus = "awaiting_payment"
	OrderStatusProcessing      OrderStatus = "processing"
	OrderStatusConfirmed       OrderStatus = "confirmed"
	OrderStatusFailed          OrderStatus = "failed"
	OrderStatusCancelled       OrderStatus = "cancelled"
	OrderStatusExpired         OrderStatus = "expired"
)

// CreateOrderRequest represents a request to create a new order
type CreateOrderRequest struct {
	FlightID      string `json:"flightId" validate:"required"`
	CustomerEmail string `json:"customerEmail" validate:"required,email"`
	CustomerName  string `json:"customerName" validate:"required"`
}

// SelectSeatsRequest represents a request to select seats
type SelectSeatsRequest struct {
	SeatIDs []string `json:"seatIds" validate:"required,min=1"`
}

// PaymentRequest represents a payment submission
type PaymentRequest struct {
	PaymentCode string `json:"paymentCode" validate:"required,len=5,numeric"`
}

// OrderStatusResponse represents real-time order status
type OrderStatusResponse struct {
	Order            *Order `json:"order"`
	RemainingSeconds int    `json:"remainingSeconds"`
	Message          string `json:"message,omitempty"`
}

