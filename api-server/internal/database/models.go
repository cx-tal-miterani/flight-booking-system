package database

import (
	"time"

	"github.com/google/uuid"
)

// Flight represents a flight in the database
type Flight struct {
	ID             uuid.UUID `json:"id"`
	FlightNumber   string    `json:"flightNumber"`
	Origin         string    `json:"origin"`
	Destination    string    `json:"destination"`
	DepartureTime  time.Time `json:"departureTime"`
	ArrivalTime    time.Time `json:"arrivalTime"`
	TotalSeats     int       `json:"totalSeats"`
	AvailableSeats int       `json:"availableSeats"`
	PricePerSeat   float64   `json:"pricePerSeat"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// SeatStatus represents the status of a seat
type SeatStatus string

const (
	SeatStatusAvailable SeatStatus = "available"
	SeatStatusHeld      SeatStatus = "held"
	SeatStatusBooked    SeatStatus = "booked"
)

// Seat represents a seat in the database
type Seat struct {
	ID           uuid.UUID   `json:"id"`
	FlightID     uuid.UUID   `json:"flightId"`
	SeatNumber   string      `json:"seatNumber"`
	RowNumber    int         `json:"row"`
	ColumnLetter string      `json:"column"`
	Class        string      `json:"class"`
	Status       SeatStatus  `json:"status"`
	Price        float64     `json:"price"`
	HeldUntil    *time.Time  `json:"heldUntil,omitempty"`
	HeldByOrder  *uuid.UUID  `json:"heldByOrder,omitempty"`
	CreatedAt    time.Time   `json:"createdAt"`
	UpdatedAt    time.Time   `json:"updatedAt"`
}

// OrderStatus represents the status of an order
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

// Order represents an order in the database
type Order struct {
	ID                   uuid.UUID   `json:"id"`
	FlightID             uuid.UUID   `json:"flightId"`
	CustomerName         string      `json:"customerName"`
	CustomerEmail        string      `json:"customerEmail"`
	Status               OrderStatus `json:"status"`
	TotalAmount          float64     `json:"totalAmount"`
	PaymentAttempts      int         `json:"paymentAttempts"`
	FailureReason        *string     `json:"failureReason,omitempty"`
	WorkflowID           *string     `json:"workflowId,omitempty"`
	WorkflowRunID        *string     `json:"workflowRunId,omitempty"`
	ReservationExpiresAt *time.Time  `json:"reservationExpiresAt,omitempty"`
	CreatedAt            time.Time   `json:"createdAt"`
	UpdatedAt            time.Time   `json:"updatedAt"`
	Seats                []string    `json:"seats,omitempty"`
}

// OrderSeat represents the junction between orders and seats
type OrderSeat struct {
	ID        uuid.UUID `json:"id"`
	OrderID   uuid.UUID `json:"orderId"`
	SeatID    uuid.UUID `json:"seatId"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"createdAt"`
}

