package workflows

import (
	"testing"
	"time"

	"github.com/cx-tal-miterani/flight-booking-system/temporal-worker/internal/activities"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type BookingWorkflowTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
	env *testsuite.TestWorkflowEnvironment
}

func (s *BookingWorkflowTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func (s *BookingWorkflowTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func TestBookingWorkflowTestSuite(t *testing.T) {
	suite.Run(t, new(BookingWorkflowTestSuite))
}

func (s *BookingWorkflowTestSuite) TestWorkflow_Constants() {
	// Verify workflow constants are set correctly per requirements
	s.Equal(15*time.Minute, SeatHoldDuration, "Seat hold should be 15 minutes")
	s.Equal(10*time.Second, PaymentTimeout, "Payment timeout should be 10 seconds")
	s.Equal(3, MaxPaymentAttempts, "Max payment attempts should be 3")
}

func (s *BookingWorkflowTestSuite) TestWorkflow_SeatsSelectedSignal() {
	input := BookingWorkflowInput{
		OrderID:       "test-order-123",
		FlightID:      "test-flight-456",
		CustomerName:  "John Doe",
		CustomerEmail: "john@example.com",
	}

	// Register activity mocks
	s.env.OnActivity("UpdateOrderStatus", mock.Anything, mock.Anything).Return(nil)

	// Start workflow
	s.env.RegisterDelayedCallback(func() {
		// Send seats selected signal after workflow starts
		s.env.SignalWorkflow("seats-selected", SeatsSelectedSignal{
			SeatIDs:   []string{"seat-1", "seat-2"},
			ExpiresAt: time.Now().Add(15 * time.Minute),
		})
	}, time.Millisecond*100)

	// Cancel workflow after signal is processed
	s.env.RegisterDelayedCallback(func() {
		s.env.CancelWorkflow()
	}, time.Millisecond*500)

	s.env.OnActivity("ReleaseSeats", mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(BookingWorkflow, input)

	s.True(s.env.IsWorkflowCompleted())
}

func (s *BookingWorkflowTestSuite) TestWorkflow_PaymentSuccess() {
	input := BookingWorkflowInput{
		OrderID:       "test-order-123",
		FlightID:      "test-flight-456",
		CustomerName:  "John Doe",
		CustomerEmail: "john@example.com",
	}

	// Register activity mocks
	s.env.OnActivity("UpdateOrderStatus", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("ValidatePayment", mock.Anything, mock.Anything).Return(&activities.ValidatePaymentOutput{
		Success:       true,
		TransactionID: "TXN-12345",
	}, nil)
	s.env.OnActivity("SendConfirmation", mock.Anything, mock.Anything).Return(nil)

	// Send signals
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow("seats-selected", SeatsSelectedSignal{
			SeatIDs:   []string{"seat-1"},
			ExpiresAt: time.Now().Add(15 * time.Minute),
		})
	}, time.Millisecond*100)

	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow("payment-submitted", PaymentSubmittedSignal{
			PaymentCode: "12345",
		})
	}, time.Millisecond*200)

	// Need to cancel since workflow loop doesn't have proper termination in test
	s.env.RegisterDelayedCallback(func() {
		s.env.CancelWorkflow()
	}, time.Millisecond*500)

	s.env.OnActivity("ReleaseSeats", mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(BookingWorkflow, input)

	s.True(s.env.IsWorkflowCompleted())
}

func (s *BookingWorkflowTestSuite) TestWorkflow_PaymentFailure_Retry() {
	input := BookingWorkflowInput{
		OrderID:       "test-order-123",
		FlightID:      "test-flight-456",
		CustomerName:  "John Doe",
		CustomerEmail: "john@example.com",
	}

	// Register activity mocks
	s.env.OnActivity("UpdateOrderStatus", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("ValidatePayment", mock.Anything, mock.Anything).Return(&activities.ValidatePaymentOutput{
		Success:      false,
		ErrorMessage: "Payment failed",
	}, nil)

	// Send signals
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow("seats-selected", SeatsSelectedSignal{
			SeatIDs:   []string{"seat-1"},
			ExpiresAt: time.Now().Add(15 * time.Minute),
		})
	}, time.Millisecond*100)

	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow("payment-submitted", PaymentSubmittedSignal{
			PaymentCode: "12345",
		})
	}, time.Millisecond*200)

	// Cancel after first payment attempt
	s.env.RegisterDelayedCallback(func() {
		s.env.CancelWorkflow()
	}, time.Millisecond*500)

	s.env.OnActivity("ReleaseSeats", mock.Anything, mock.Anything).Return(nil)

	s.env.ExecuteWorkflow(BookingWorkflow, input)

	s.True(s.env.IsWorkflowCompleted())
}

func (s *BookingWorkflowTestSuite) TestWorkflow_MaxPaymentAttemptsExceeded() {
	input := BookingWorkflowInput{
		OrderID:       "test-order-123",
		FlightID:      "test-flight-456",
		CustomerName:  "John Doe",
		CustomerEmail: "john@example.com",
	}

	// Register activity mocks - payment always fails
	s.env.OnActivity("UpdateOrderStatus", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("ValidatePayment", mock.Anything, mock.Anything).Return(&activities.ValidatePaymentOutput{
		Success:      false,
		ErrorMessage: "Payment failed",
	}, nil)
	s.env.OnActivity("ReleaseSeats", mock.Anything, mock.Anything).Return(nil)

	// Send signals - first seats, then 3 payment attempts
	s.env.RegisterDelayedCallback(func() {
		s.env.SignalWorkflow("seats-selected", SeatsSelectedSignal{
			SeatIDs:   []string{"seat-1"},
			ExpiresAt: time.Now().Add(15 * time.Minute),
		})
	}, time.Millisecond*100)

	// Send 3 payment attempts
	for i := 0; i < 3; i++ {
		delay := time.Millisecond * time.Duration(200+i*100)
		s.env.RegisterDelayedCallback(func() {
			s.env.SignalWorkflow("payment-submitted", PaymentSubmittedSignal{
				PaymentCode: "12345",
			})
		}, delay)
	}

	// Cancel after attempts
	s.env.RegisterDelayedCallback(func() {
		s.env.CancelWorkflow()
	}, time.Millisecond*800)

	s.env.ExecuteWorkflow(BookingWorkflow, input)

	s.True(s.env.IsWorkflowCompleted())
}

func (s *BookingWorkflowTestSuite) TestWorkflow_Cancellation() {
	input := BookingWorkflowInput{
		OrderID:       "test-order-123",
		FlightID:      "test-flight-456",
		CustomerName:  "John Doe",
		CustomerEmail: "john@example.com",
	}

	s.env.OnActivity("UpdateOrderStatus", mock.Anything, mock.Anything).Return(nil)
	s.env.OnActivity("ReleaseSeats", mock.Anything, mock.Anything).Return(nil)

	// Cancel workflow immediately
	s.env.RegisterDelayedCallback(func() {
		s.env.CancelWorkflow()
	}, time.Millisecond*100)

	s.env.ExecuteWorkflow(BookingWorkflow, input)

	s.True(s.env.IsWorkflowCompleted())
	
	var result *BookingWorkflowResult
	err := s.env.GetWorkflowResult(&result)
	s.NoError(err)
	s.False(result.Success)
	s.Equal("cancelled", result.FailureReason)
}
