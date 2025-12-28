package activities

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/cx-tal-miterani/flight-booking-system/shared/models"
	"go.temporal.io/sdk/activity"
)

const (
	PaymentFailureRate = 0.15 // 15% failure rate
	SeatHoldDuration   = 15 * time.Minute
)

// SeatInventory manages seat state in memory (can be replaced with DB)
type SeatInventory struct {
	mu     sync.RWMutex
	seats  map[string]*models.Seat // seatID -> Seat
	holds  map[string]string       // seatID -> orderID
	expiry map[string]time.Time    // seatID -> expiry time
}

var inventory = &SeatInventory{
	seats:  make(map[string]*models.Seat),
	holds:  make(map[string]string),
	expiry: make(map[string]time.Time),
}

// InitializeInventory sets up initial seat inventory for a flight
func InitializeInventory(flightID string, rows int, columns []string, pricePerSeat float64) {
	inventory.mu.Lock()
	defer inventory.mu.Unlock()

	for row := 1; row <= rows; row++ {
		for _, col := range columns {
			seatID := fmt.Sprintf("%s-%d%s", flightID, row, col)
			inventory.seats[seatID] = &models.Seat{
				ID:       seatID,
				FlightID: flightID,
				Row:      row,
				Column:   col,
				Class:    models.SeatClassEconomy,
				Status:   models.SeatStatusAvailable,
				Price:    pricePerSeat,
			}
		}
	}
}

// GetAvailableSeats returns all available seats for a flight
func GetAvailableSeats(flightID string) []*models.Seat {
	inventory.mu.RLock()
	defer inventory.mu.RUnlock()

	var available []*models.Seat
	for _, seat := range inventory.seats {
		if seat.FlightID == flightID && seat.Status == models.SeatStatusAvailable {
			available = append(available, seat)
		}
	}
	return available
}

// ReserveSeats activity - reserves seats for an order
func ReserveSeats(ctx context.Context, orderID, flightID string, seatIDs []string) (*models.ReserveSeatsResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Reserving seats", "orderID", orderID, "seats", seatIDs)

	inventory.mu.Lock()
	defer inventory.mu.Unlock()

	// First check all seats are available
	var totalAmount float64
	for _, seatID := range seatIDs {
		seat, exists := inventory.seats[seatID]
		if !exists {
			return &models.ReserveSeatsResult{
				Success: false,
				Error:   fmt.Sprintf("Seat %s not found", seatID),
			}, nil
		}

		// Check if seat is available or held by same order
		if seat.Status != models.SeatStatusAvailable {
			if existingOrder, held := inventory.holds[seatID]; held && existingOrder == orderID {
				// Same order, refresh the hold
				continue
			}
			// Check if hold has expired
			if expiry, hasExpiry := inventory.expiry[seatID]; hasExpiry && time.Now().After(expiry) {
				// Hold expired, seat can be claimed
				seat.Status = models.SeatStatusAvailable
				delete(inventory.holds, seatID)
				delete(inventory.expiry, seatID)
			} else {
				return &models.ReserveSeatsResult{
					Success: false,
					Error:   fmt.Sprintf("Seat %s is not available", seatID),
				}, nil
			}
		}
		totalAmount += seat.Price
	}

	// Reserve all seats
	holdExpiry := time.Now().Add(SeatHoldDuration)
	for _, seatID := range seatIDs {
		seat := inventory.seats[seatID]
		seat.Status = models.SeatStatusHeld
		inventory.holds[seatID] = orderID
		inventory.expiry[seatID] = holdExpiry
	}

	logger.Info("Seats reserved successfully", "orderID", orderID, "total", totalAmount)

	return &models.ReserveSeatsResult{
		Success:     true,
		SeatIDs:     seatIDs,
		TotalAmount: totalAmount,
		HoldExpiry:  holdExpiry,
	}, nil
}

// ReleaseSeats activity - releases held seats
func ReleaseSeats(ctx context.Context, orderID string, seatIDs []string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Releasing seats", "orderID", orderID, "seats", seatIDs)

	inventory.mu.Lock()
	defer inventory.mu.Unlock()

	for _, seatID := range seatIDs {
		seat, exists := inventory.seats[seatID]
		if !exists {
			continue
		}

		// Only release if held by this order
		if holdOrder, held := inventory.holds[seatID]; held && holdOrder == orderID {
			seat.Status = models.SeatStatusAvailable
			delete(inventory.holds, seatID)
			delete(inventory.expiry, seatID)
		}
	}

	return nil
}

// ValidatePayment activity - validates payment code with simulated failures
func ValidatePayment(ctx context.Context, orderID, paymentCode string, amount float64) (*models.ValidatePaymentResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Validating payment", "orderID", orderID, "amount", amount)

	// Validate payment code format (5 digits)
	if len(paymentCode) != 5 {
		return &models.ValidatePaymentResult{
			Success:  false,
			Error:    "Payment code must be 5 digits",
			CanRetry: false,
		}, nil
	}

	for _, c := range paymentCode {
		if c < '0' || c > '9' {
			return &models.ValidatePaymentResult{
				Success:  false,
				Error:    "Payment code must contain only digits",
				CanRetry: false,
			}, nil
		}
	}

	// Simulate payment processing delay
	time.Sleep(500 * time.Millisecond)

	// Simulate 15% failure rate
	if rand.Float64() < PaymentFailureRate {
		logger.Warn("Payment failed (simulated)", "orderID", orderID)
		return &models.ValidatePaymentResult{
			Success:  false,
			Error:    "Payment declined by provider",
			CanRetry: true,
		}, nil
	}

	logger.Info("Payment validated successfully", "orderID", orderID)
	return &models.ValidatePaymentResult{
		Success: true,
	}, nil
}

// ConfirmBooking activity - confirms the booking and marks seats as booked
func ConfirmBooking(ctx context.Context, orderID string, seatIDs []string) (*models.ConfirmBookingResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Confirming booking", "orderID", orderID, "seats", seatIDs)

	inventory.mu.Lock()
	defer inventory.mu.Unlock()

	// Mark all seats as booked
	for _, seatID := range seatIDs {
		seat, exists := inventory.seats[seatID]
		if !exists {
			return &models.ConfirmBookingResult{
				Success: false,
				Error:   fmt.Sprintf("Seat %s not found", seatID),
			}, nil
		}

		// Verify seat is held by this order
		if holdOrder, held := inventory.holds[seatID]; !held || holdOrder != orderID {
			return &models.ConfirmBookingResult{
				Success: false,
				Error:   fmt.Sprintf("Seat %s is not held by this order", seatID),
			}, nil
		}

		seat.Status = models.SeatStatusBooked
		delete(inventory.holds, seatID)
		delete(inventory.expiry, seatID)
	}

	// Generate confirmation code
	confirmationCode := fmt.Sprintf("FLT%s%d", orderID[:4], time.Now().Unix()%10000)

	logger.Info("Booking confirmed", "orderID", orderID, "confirmation", confirmationCode)
	return &models.ConfirmBookingResult{
		Success:          true,
		ConfirmationCode: confirmationCode,
	}, nil
}

// GetSeatInventory returns the current inventory (for testing/debugging)
func GetSeatInventory() *SeatInventory {
	return inventory
}

// ResetInventory clears all seats (for testing)
func ResetInventory() {
	inventory.mu.Lock()
	defer inventory.mu.Unlock()
	inventory.seats = make(map[string]*models.Seat)
	inventory.holds = make(map[string]string)
	inventory.expiry = make(map[string]time.Time)
}

