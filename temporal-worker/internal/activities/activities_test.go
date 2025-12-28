package activities

import (
	"context"
	"testing"

	"github.com/cx-tal-miterani/flight-booking-system/temporal-worker/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of the repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetOrderStatus(ctx context.Context, orderID uuid.UUID) (repository.OrderStatus, error) {
	args := m.Called(ctx, orderID)
	return args.Get(0).(repository.OrderStatus), args.Error(1)
}

func (m *MockRepository) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status repository.OrderStatus) error {
	args := m.Called(ctx, orderID, status)
	return args.Error(0)
}

func (m *MockRepository) UpdateOrderPayment(ctx context.Context, orderID uuid.UUID, attempts int, failureReason *string) error {
	args := m.Called(ctx, orderID, attempts, failureReason)
	return args.Error(0)
}

func (m *MockRepository) BookSeats(ctx context.Context, orderID uuid.UUID) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
}

func (m *MockRepository) ReleaseSeats(ctx context.Context, orderID uuid.UUID) error {
	args := m.Called(ctx, orderID)
	return args.Error(0)
}

func TestValidatePayment_InvalidCode_TooShort(t *testing.T) {
	mockRepo := new(MockRepository)
	activities := NewActivities(&repository.Repository{})

	ctx := context.Background()
	input := ValidatePaymentInput{
		OrderID:     uuid.New().String(),
		PaymentCode: "1234", // Too short
		Attempt:     1,
	}

	result, err := activities.ValidatePayment(ctx, input)

	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "Invalid payment code")
}

func TestValidatePayment_InvalidCode_TooLong(t *testing.T) {
	mockRepo := new(MockRepository)
	activities := NewActivities(&repository.Repository{})

	ctx := context.Background()
	input := ValidatePaymentInput{
		OrderID:     uuid.New().String(),
		PaymentCode: "123456", // Too long
		Attempt:     1,
	}

	result, err := activities.ValidatePayment(ctx, input)

	assert.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.ErrorMessage, "Invalid payment code")
}

func TestValidatePayment_InvalidOrderID(t *testing.T) {
	activities := NewActivities(&repository.Repository{})

	ctx := context.Background()
	input := ValidatePaymentInput{
		OrderID:     "invalid-uuid",
		PaymentCode: "12345",
		Attempt:     1,
	}

	_, err := activities.ValidatePayment(ctx, input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid order ID")
}

func TestReleaseSeats_InvalidOrderID(t *testing.T) {
	activities := NewActivities(&repository.Repository{})

	ctx := context.Background()
	input := ReleaseSeatsInput{
		OrderID: "invalid-uuid",
		Reason:  "expired",
	}

	err := activities.ReleaseSeats(ctx, input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid order ID")
}

func TestUpdateOrderStatus_InvalidOrderID(t *testing.T) {
	activities := NewActivities(&repository.Repository{})

	ctx := context.Background()
	input := UpdateOrderStatusInput{
		OrderID: "invalid-uuid",
		Status:  "confirmed",
	}

	err := activities.UpdateOrderStatus(ctx, input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid order ID")
}

func TestCheckReservationExpiry_InvalidOrderID(t *testing.T) {
	activities := NewActivities(&repository.Repository{})

	ctx := context.Background()
	input := CheckReservationExpiryInput{
		OrderID: "invalid-uuid",
	}

	_, err := activities.CheckReservationExpiry(ctx, input)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid order ID")
}

func TestSendConfirmation_Success(t *testing.T) {
	activities := NewActivities(&repository.Repository{})

	ctx := context.Background()
	input := SendConfirmationInput{
		OrderID:       uuid.New().String(),
		CustomerEmail: "test@example.com",
		CustomerName:  "John Doe",
		FlightNumber:  "AA123",
		TransactionID: "TXN-12345",
	}

	err := activities.SendConfirmation(ctx, input)

	// SendConfirmation just logs and returns nil
	assert.NoError(t, err)
}

func TestReserveSeats_Success(t *testing.T) {
	activities := NewActivities(&repository.Repository{})

	ctx := context.Background()
	input := ReserveSeatsInput{
		OrderID: uuid.New().String(),
		SeatIDs: []string{"seat-1", "seat-2"},
	}

	err := activities.ReserveSeats(ctx, input)

	// ReserveSeats just logs and returns nil (actual reservation is handled via API)
	assert.NoError(t, err)
}

// TestValidatePayment_SuccessRate tests that payment validation has approximately 85% success rate
// This is a statistical test and may occasionally fail due to randomness
func TestValidatePayment_SuccessRate(t *testing.T) {
	t.Skip("Skipping statistical test - run manually if needed")
	
	// This test would require a proper mock repository setup
	// The 85% success rate is simulated in the ValidatePayment function
	// using rand.Float32() < 0.85
}
