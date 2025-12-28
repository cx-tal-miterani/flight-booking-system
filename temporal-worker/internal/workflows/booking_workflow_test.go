package workflows

import (
	"testing"
	"time"

	"github.com/cx-tal-miterani/flight-booking-system/shared/models"
	"github.com/cx-tal-miterani/flight-booking-system/temporal-worker/internal/activities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestBookingWorkflow_BasicFlow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(BookingWorkflow)

	env.OnActivity(activities.ReserveSeats, mock.Anything, "order-1", "FL001", []string{"FL001-1A"}).Return(
		&models.ReserveSeatsResult{
			Success:     true,
			SeatIDs:     []string{"FL001-1A"},
			TotalAmount: 150.00,
			HoldExpiry:  time.Now().Add(15 * time.Minute),
		}, nil)

	env.OnActivity(activities.ValidatePayment, mock.Anything, "order-1", "12345", 150.00).Return(
		&models.ValidatePaymentResult{
			Success: true,
		}, nil)

	env.OnActivity(activities.ConfirmBooking, mock.Anything, "order-1", []string{"FL001-1A"}).Return(
		&models.ConfirmBookingResult{
			Success:          true,
			ConfirmationCode: "FLTorde1234",
		}, nil)

	input := models.BookingWorkflowInput{
		OrderID:       "order-1",
		FlightID:      "FL001",
		CustomerEmail: "test@example.com",
		CustomerName:  "John Doe",
		SeatIDs:       []string{"FL001-1A"},
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(models.SignalSubmitPayment, models.SubmitPaymentSignal{
			PaymentCode: "12345",
		})
	}, time.Millisecond*100)

	env.ExecuteWorkflow(BookingWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())

	var result *models.Order
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, models.OrderStatusConfirmed, result.Status)
}

func TestBookingWorkflow_PaymentFailure(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(BookingWorkflow)

	env.OnActivity(activities.ReserveSeats, mock.Anything, "order-1", "FL001", []string{"FL001-1A"}).Return(
		&models.ReserveSeatsResult{
			Success:     true,
			SeatIDs:     []string{"FL001-1A"},
			TotalAmount: 150.00,
			HoldExpiry:  time.Now().Add(15 * time.Minute),
		}, nil)

	env.OnActivity(activities.ValidatePayment, mock.Anything, "order-1", "99999", 150.00).Return(
		&models.ValidatePaymentResult{
			Success:  false,
			Error:    "Payment declined",
			CanRetry: true,
		}, nil)

	env.OnActivity(activities.ReleaseSeats, mock.Anything, "order-1", []string{"FL001-1A"}).Return(nil)

	input := models.BookingWorkflowInput{
		OrderID:       "order-1",
		FlightID:      "FL001",
		CustomerEmail: "test@example.com",
		CustomerName:  "John Doe",
		SeatIDs:       []string{"FL001-1A"},
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(models.SignalSubmitPayment, models.SubmitPaymentSignal{
			PaymentCode: "99999",
		})
	}, time.Millisecond*100)

	env.ExecuteWorkflow(BookingWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())

	var result *models.Order
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, models.OrderStatusFailed, result.Status)
	assert.Equal(t, 3, result.PaymentAttempts)
}

func TestBookingWorkflow_CancelOrder(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterWorkflow(BookingWorkflow)

	env.OnActivity(activities.ReserveSeats, mock.Anything, "order-1", "FL001", []string{"FL001-1A"}).Return(
		&models.ReserveSeatsResult{
			Success:     true,
			SeatIDs:     []string{"FL001-1A"},
			TotalAmount: 150.00,
			HoldExpiry:  time.Now().Add(15 * time.Minute),
		}, nil)

	env.OnActivity(activities.ReleaseSeats, mock.Anything, "order-1", []string{"FL001-1A"}).Return(nil)

	input := models.BookingWorkflowInput{
		OrderID:       "order-1",
		FlightID:      "FL001",
		CustomerEmail: "test@example.com",
		CustomerName:  "John Doe",
		SeatIDs:       []string{"FL001-1A"},
	}

	env.RegisterDelayedCallback(func() {
		env.SignalWorkflow(models.SignalCancelOrder, nil)
	}, time.Millisecond*100)

	env.ExecuteWorkflow(BookingWorkflow, input)

	require.True(t, env.IsWorkflowCompleted())

	var result *models.Order
	require.NoError(t, env.GetWorkflowResult(&result))
	assert.Equal(t, models.OrderStatusCancelled, result.Status)
}

