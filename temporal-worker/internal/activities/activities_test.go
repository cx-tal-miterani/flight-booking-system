package activities

import (
	"context"
	"testing"

	"github.com/cx-tal-miterani/flight-booking-system/shared/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestInventory() {
	ResetInventory()
	InitializeInventory("TEST001", 5, []string{"A", "B", "C"}, 100.00)
}

func TestReserveSeats_Success(t *testing.T) {
	setupTestInventory()
	ctx := context.Background()

	seatIDs := []string{"TEST001-1A", "TEST001-1B"}
	result, err := ReserveSeats(ctx, "order-1", "TEST001", seatIDs)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, seatIDs, result.SeatIDs)
	assert.Equal(t, 200.00, result.TotalAmount)
	assert.False(t, result.HoldExpiry.IsZero())

	// Verify seats are held
	inv := GetSeatInventory()
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	assert.Equal(t, models.SeatStatusHeld, inv.seats["TEST001-1A"].Status)
	assert.Equal(t, models.SeatStatusHeld, inv.seats["TEST001-1B"].Status)
}

func TestReserveSeats_SeatNotFound(t *testing.T) {
	setupTestInventory()
	ctx := context.Background()

	seatIDs := []string{"TEST001-99Z"}
	result, err := ReserveSeats(ctx, "order-1", "TEST001", seatIDs)

	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not found")
}

func TestReserveSeats_SeatAlreadyHeld(t *testing.T) {
	setupTestInventory()
	ctx := context.Background()

	// First reservation
	seatIDs := []string{"TEST001-1A"}
	result, err := ReserveSeats(ctx, "order-1", "TEST001", seatIDs)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Second reservation by different order
	result, err = ReserveSeats(ctx, "order-2", "TEST001", seatIDs)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not available")
}

func TestReleaseSeats_Success(t *testing.T) {
	setupTestInventory()
	ctx := context.Background()

	// First reserve
	seatIDs := []string{"TEST001-1A", "TEST001-1B"}
	_, err := ReserveSeats(ctx, "order-1", "TEST001", seatIDs)
	require.NoError(t, err)

	// Then release
	err = ReleaseSeats(ctx, "order-1", seatIDs)
	require.NoError(t, err)

	// Verify seats are available
	inv := GetSeatInventory()
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	assert.Equal(t, models.SeatStatusAvailable, inv.seats["TEST001-1A"].Status)
	assert.Equal(t, models.SeatStatusAvailable, inv.seats["TEST001-1B"].Status)
}

func TestValidatePayment_InvalidCode_TooShort(t *testing.T) {
	ctx := context.Background()

	result, err := ValidatePayment(ctx, "order-1", "1234", 100.00)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "5 digits")
	assert.False(t, result.CanRetry)
}

func TestValidatePayment_InvalidCode_NonNumeric(t *testing.T) {
	ctx := context.Background()

	result, err := ValidatePayment(ctx, "order-1", "1234a", 100.00)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "only digits")
}

func TestConfirmBooking_Success(t *testing.T) {
	setupTestInventory()
	ctx := context.Background()

	// First reserve seats
	seatIDs := []string{"TEST001-1A", "TEST001-1B"}
	_, err := ReserveSeats(ctx, "order-1", "TEST001", seatIDs)
	require.NoError(t, err)

	// Then confirm
	result, err := ConfirmBooking(ctx, "order-1", seatIDs)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.ConfirmationCode)

	// Verify seats are booked
	inv := GetSeatInventory()
	inv.mu.RLock()
	defer inv.mu.RUnlock()
	assert.Equal(t, models.SeatStatusBooked, inv.seats["TEST001-1A"].Status)
	assert.Equal(t, models.SeatStatusBooked, inv.seats["TEST001-1B"].Status)
}

func TestConfirmBooking_NotHeld(t *testing.T) {
	setupTestInventory()
	ctx := context.Background()

	// Try to confirm without reservation
	seatIDs := []string{"TEST001-1A"}
	result, err := ConfirmBooking(ctx, "order-1", seatIDs)
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "not held")
}

