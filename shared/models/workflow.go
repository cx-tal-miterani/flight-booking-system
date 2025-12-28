package models

import "time"

// WorkflowInput represents input for the booking workflow
type BookingWorkflowInput struct {
	OrderID       string   `json:"orderId"`
	FlightID      string   `json:"flightId"`
	CustomerEmail string   `json:"customerEmail"`
	CustomerName  string   `json:"customerName"`
	SeatIDs       []string `json:"seatIds,omitempty"`
}

// WorkflowState represents the current state of the booking workflow
type BookingWorkflowState struct {
	OrderID         string      `json:"orderId"`
	Status          OrderStatus `json:"status"`
	SeatIDs         []string    `json:"seatIds"`
	SeatHoldExpiry  time.Time   `json:"seatHoldExpiry"`
	PaymentAttempts int         `json:"paymentAttempts"`
	TotalAmount     float64     `json:"totalAmount"`
	FailureReason   string      `json:"failureReason,omitempty"`
	LastUpdated     time.Time   `json:"lastUpdated"`
}

// Signals for workflow communication
const (
	SignalSelectSeats   = "select_seats"
	SignalSubmitPayment = "submit_payment"
	SignalCancelOrder   = "cancel_order"
	SignalRefreshTimer  = "refresh_timer"
)

// SelectSeatsSignal is sent when user selects/updates seats
type SelectSeatsSignal struct {
	SeatIDs []string `json:"seatIds"`
}

// SubmitPaymentSignal is sent when user submits payment code
type SubmitPaymentSignal struct {
	PaymentCode string `json:"paymentCode"`
}

// Queries for workflow state
const (
	QueryGetState = "get_state"
)

// Activity results
type ReserveSeatsResult struct {
	Success     bool      `json:"success"`
	SeatIDs     []string  `json:"seatIds"`
	TotalAmount float64   `json:"totalAmount"`
	HoldExpiry  time.Time `json:"holdExpiry"`
	Error       string    `json:"error,omitempty"`
}

type ValidatePaymentResult struct {
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
	CanRetry bool   `json:"canRetry"`
}

type ConfirmBookingResult struct {
	Success          bool   `json:"success"`
	ConfirmationCode string `json:"confirmationCode,omitempty"`
	Error            string `json:"error,omitempty"`
}

