package workflows

import (
	"time"

	"github.com/cx-tal-miterani/flight-booking-system/temporal-worker/internal/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	// SeatHoldDuration is how long seats are held (15 minutes)
	SeatHoldDuration = 15 * time.Minute
	// PaymentTimeout is how long to wait for payment validation (10 seconds)
	PaymentTimeout = 10 * time.Second
	// MaxPaymentAttempts is the maximum number of payment retries
	MaxPaymentAttempts = 3
)

// BookingWorkflowInput is the input for the booking workflow
type BookingWorkflowInput struct {
	OrderID       string `json:"orderId"`
	FlightID      string `json:"flightId"`
	CustomerName  string `json:"customerName"`
	CustomerEmail string `json:"customerEmail"`
}

// BookingWorkflowResult is the result of the booking workflow
type BookingWorkflowResult struct {
	Success       bool   `json:"success"`
	TransactionID string `json:"transactionId,omitempty"`
	FailureReason string `json:"failureReason,omitempty"`
}

// SeatsSelectedSignal is the signal for seat selection
type SeatsSelectedSignal struct {
	SeatIDs   []string  `json:"seatIds"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// PaymentSubmittedSignal is the signal for payment submission
type PaymentSubmittedSignal struct {
	PaymentCode string `json:"paymentCode"`
}

// BookingWorkflow orchestrates the flight booking process
func BookingWorkflow(ctx workflow.Context, input BookingWorkflowInput) (*BookingWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Booking workflow started", "orderId", input.OrderID)

	// Activity options
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Payment activity with shorter timeout (10 seconds)
	paymentCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: PaymentTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // No automatic retries for payment
		},
	})

	// Channels for signals
	seatsSelectedCh := workflow.GetSignalChannel(ctx, "seats-selected")
	paymentSubmittedCh := workflow.GetSignalChannel(ctx, "payment-submitted")

	var seatsSelected bool
	var paymentAttempts int
	var reservationExpiry time.Time

	// Update order status to pending
	err := workflow.ExecuteActivity(ctx, "UpdateOrderStatus", activities.UpdateOrderStatusInput{
		OrderID: input.OrderID,
		Status:  "pending",
	}).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to update order status", "error", err)
	}

	// Main workflow loop
	for {
		selector := workflow.NewSelector(ctx)

		// Handle seat selection signal
		selector.AddReceive(seatsSelectedCh, func(c workflow.ReceiveChannel, more bool) {
			var signal SeatsSelectedSignal
			c.Receive(ctx, &signal)
			logger.Info("Seats selected", "seatCount", len(signal.SeatIDs))

			seatsSelected = true
			reservationExpiry = signal.ExpiresAt

			// Update status
			workflow.ExecuteActivity(ctx, "UpdateOrderStatus", activities.UpdateOrderStatusInput{
				OrderID: input.OrderID,
				Status:  "seats_selected",
			})
		})

		// Handle payment submission signal
		selector.AddReceive(paymentSubmittedCh, func(c workflow.ReceiveChannel, more bool) {
			var signal PaymentSubmittedSignal
			c.Receive(ctx, &signal)
			logger.Info("Payment submitted", "attempt", paymentAttempts+1)

			if !seatsSelected {
				logger.Warn("Payment submitted before seats selected")
				return
			}

			// Check if reservation expired
			if workflow.Now(ctx).After(reservationExpiry) {
				logger.Info("Reservation expired before payment")
				workflow.ExecuteActivity(ctx, "ReleaseSeats", activities.ReleaseSeatsInput{
					OrderID: input.OrderID,
					Reason:  "expired",
				})
				return
			}

			paymentAttempts++

			// Update status to processing
			workflow.ExecuteActivity(ctx, "UpdateOrderStatus", activities.UpdateOrderStatusInput{
				OrderID: input.OrderID,
				Status:  "processing",
			})

			// Validate payment
			var result activities.ValidatePaymentOutput
			err := workflow.ExecuteActivity(paymentCtx, "ValidatePayment", activities.ValidatePaymentInput{
				OrderID:     input.OrderID,
				PaymentCode: signal.PaymentCode,
				Attempt:     paymentAttempts,
			}).Get(ctx, &result)

			if err != nil {
				logger.Error("Payment activity failed", "error", err)
				return
			}

			if result.Success {
				logger.Info("Payment successful!", "transactionId", result.TransactionID)

				// Send confirmation
				workflow.ExecuteActivity(ctx, "SendConfirmation", activities.SendConfirmationInput{
					OrderID:       input.OrderID,
					CustomerEmail: input.CustomerEmail,
					CustomerName:  input.CustomerName,
					TransactionID: result.TransactionID,
				})
			} else {
				logger.Info("Payment failed", "attempt", paymentAttempts, "maxAttempts", MaxPaymentAttempts)

				if paymentAttempts >= MaxPaymentAttempts {
					// Max attempts reached - fail order
					workflow.ExecuteActivity(ctx, "ReleaseSeats", activities.ReleaseSeatsInput{
						OrderID: input.OrderID,
						Reason:  "payment_failed",
					})
				} else {
					// Allow retry
					workflow.ExecuteActivity(ctx, "UpdateOrderStatus", activities.UpdateOrderStatusInput{
						OrderID: input.OrderID,
						Status:  "awaiting_payment",
					})
				}
			}
		})

		// Timeout for seat hold expiry
		if seatsSelected && !reservationExpiry.IsZero() {
			timeUntilExpiry := reservationExpiry.Sub(workflow.Now(ctx))
			if timeUntilExpiry > 0 {
				selector.AddFuture(workflow.NewTimer(ctx, timeUntilExpiry), func(f workflow.Future) {
					logger.Info("Reservation timer expired")

					// Check if order is still in progress
					var expired bool
					workflow.ExecuteActivity(ctx, "CheckReservationExpiry", activities.CheckReservationExpiryInput{
						OrderID: input.OrderID,
					}).Get(ctx, &expired)

					if expired {
						workflow.ExecuteActivity(ctx, "ReleaseSeats", activities.ReleaseSeatsInput{
							OrderID: input.OrderID,
							Reason:  "expired",
						})
					}
				})
			}
		}

		selector.Select(ctx)

		// Check for completion conditions
		status, _ := getOrderStatus(ctx, input.OrderID)
		if status == "confirmed" {
			return &BookingWorkflowResult{
				Success: true,
			}, nil
		}
		if status == "failed" || status == "cancelled" || status == "expired" {
			return &BookingWorkflowResult{
				Success:       false,
				FailureReason: string(status),
			}, nil
		}

		// Check for context cancellation
		if ctx.Err() != nil {
			// Release seats on cancellation
			workflow.ExecuteActivity(ctx, "ReleaseSeats", activities.ReleaseSeatsInput{
				OrderID: input.OrderID,
				Reason:  "cancelled",
			})
			return &BookingWorkflowResult{
				Success:       false,
				FailureReason: "cancelled",
			}, nil
		}
	}
}

func getOrderStatus(ctx workflow.Context, orderID string) (string, error) {
	// This is a simplified check - in production you'd query the database
	// For now, we rely on the workflow state
	return "in_progress", nil
}
