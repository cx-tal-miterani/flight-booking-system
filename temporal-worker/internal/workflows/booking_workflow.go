package workflows

import (
	"time"

	"github.com/cx-tal-miterani/flight-booking-system/shared/models"
	"github.com/cx-tal-miterani/flight-booking-system/temporal-worker/internal/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	SeatHoldTimeout   = 15 * time.Minute
	PaymentTimeout    = 10 * time.Second
	MaxPaymentRetries = 3
)

// BookingWorkflow orchestrates the entire flight booking process
func BookingWorkflow(ctx workflow.Context, input models.BookingWorkflowInput) (*models.Order, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting booking workflow", "orderId", input.OrderID)

	// Initialize workflow state
	state := &models.BookingWorkflowState{
		OrderID:     input.OrderID,
		Status:      models.OrderStatusPending,
		LastUpdated: workflow.Now(ctx),
	}

	// Activity options with retry policy
	activityOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOpts)

	// Set up query handler for state
	if err := workflow.SetQueryHandler(ctx, models.QueryGetState, func() (*models.BookingWorkflowState, error) {
		return state, nil
	}); err != nil {
		return nil, err
	}

	// Channels for signals
	selectSeatsCh := workflow.GetSignalChannel(ctx, models.SignalSelectSeats)
	submitPaymentCh := workflow.GetSignalChannel(ctx, models.SignalSubmitPayment)
	cancelOrderCh := workflow.GetSignalChannel(ctx, models.SignalCancelOrder)
	refreshTimerCh := workflow.GetSignalChannel(ctx, models.SignalRefreshTimer)

	// If seats were provided in input, reserve them immediately
	if len(input.SeatIDs) > 0 {
		result, err := reserveSeats(ctx, input.OrderID, input.FlightID, input.SeatIDs)
		if err != nil {
			state.Status = models.OrderStatusFailed
			state.FailureReason = err.Error()
			return buildOrder(state, input), err
		}
		state.SeatIDs = result.SeatIDs
		state.TotalAmount = result.TotalAmount
		state.SeatHoldExpiry = result.HoldExpiry
		state.Status = models.OrderStatusSeatsSelected
	}

	// Main workflow loop - wait for signals or timeout
	for {
		timerDuration := SeatHoldTimeout
		if !state.SeatHoldExpiry.IsZero() {
			timerDuration = state.SeatHoldExpiry.Sub(workflow.Now(ctx))
			if timerDuration <= 0 {
				// Timer already expired
				state.Status = models.OrderStatusExpired
				state.FailureReason = "Seat hold expired"
				releaseSeats(ctx, input.OrderID, state.SeatIDs)
				return buildOrder(state, input), nil
			}
		}

		selector := workflow.NewSelector(ctx)

		// Handle seat selection signal
		selector.AddReceive(selectSeatsCh, func(c workflow.ReceiveChannel, more bool) {
			var signal models.SelectSeatsSignal
			c.Receive(ctx, &signal)
			logger.Info("Received select seats signal", "seats", signal.SeatIDs)

			// Release previously held seats if any
			if len(state.SeatIDs) > 0 {
				releaseSeats(ctx, input.OrderID, state.SeatIDs)
			}

			// Reserve new seats
			result, err := reserveSeats(ctx, input.OrderID, input.FlightID, signal.SeatIDs)
			if err != nil {
				state.FailureReason = err.Error()
				return
			}
			state.SeatIDs = result.SeatIDs
			state.TotalAmount = result.TotalAmount
			state.SeatHoldExpiry = result.HoldExpiry
			state.Status = models.OrderStatusSeatsSelected
			state.FailureReason = ""
			state.LastUpdated = workflow.Now(ctx)
		})

		// Handle payment submission signal
		selector.AddReceive(submitPaymentCh, func(c workflow.ReceiveChannel, more bool) {
			var signal models.SubmitPaymentSignal
			c.Receive(ctx, &signal)
			logger.Info("Received payment signal")

			if len(state.SeatIDs) == 0 {
				state.FailureReason = "No seats selected"
				return
			}

			state.Status = models.OrderStatusProcessing

			// Process payment with retries
			for attempt := 1; attempt <= MaxPaymentRetries; attempt++ {
				state.PaymentAttempts = attempt
				state.LastUpdated = workflow.Now(ctx)

				result, err := validatePayment(ctx, input.OrderID, signal.PaymentCode, state.TotalAmount)
				if err != nil {
					logger.Error("Payment validation error", "error", err, "attempt", attempt)
					continue
				}

				if result.Success {
					// Payment successful - confirm booking
					confirmResult, err := confirmBooking(ctx, input.OrderID, state.SeatIDs)
					if err != nil {
						state.Status = models.OrderStatusFailed
						state.FailureReason = "Failed to confirm booking: " + err.Error()
						return
					}
					if confirmResult.Success {
						state.Status = models.OrderStatusConfirmed
						state.FailureReason = ""
						return
					}
					state.Status = models.OrderStatusFailed
					state.FailureReason = confirmResult.Error
					return
				}

				if !result.CanRetry || attempt >= MaxPaymentRetries {
					state.Status = models.OrderStatusFailed
					state.FailureReason = result.Error
					releaseSeats(ctx, input.OrderID, state.SeatIDs)
					return
				}

				// Wait before retry
				workflow.Sleep(ctx, time.Second)
			}
		})

		// Handle cancel signal
		selector.AddReceive(cancelOrderCh, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, nil)
			logger.Info("Received cancel signal")
			state.Status = models.OrderStatusCancelled
			if len(state.SeatIDs) > 0 {
				releaseSeats(ctx, input.OrderID, state.SeatIDs)
			}
		})

		// Handle timer refresh signal
		selector.AddReceive(refreshTimerCh, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, nil)
			logger.Info("Received timer refresh signal")
			if len(state.SeatIDs) > 0 && state.Status == models.OrderStatusSeatsSelected {
				state.SeatHoldExpiry = workflow.Now(ctx).Add(SeatHoldTimeout)
				state.LastUpdated = workflow.Now(ctx)
			}
		})

		// Handle timeout
		timerFuture := workflow.NewTimer(ctx, timerDuration)
		selector.AddFuture(timerFuture, func(f workflow.Future) {
			logger.Info("Seat hold timer expired")
			if state.Status == models.OrderStatusSeatsSelected {
				state.Status = models.OrderStatusExpired
				state.FailureReason = "Seat hold expired"
				releaseSeats(ctx, input.OrderID, state.SeatIDs)
			}
		})

		selector.Select(ctx)
		state.LastUpdated = workflow.Now(ctx)

		// Check for terminal states
		if state.Status == models.OrderStatusConfirmed ||
			state.Status == models.OrderStatusFailed ||
			state.Status == models.OrderStatusCancelled ||
			state.Status == models.OrderStatusExpired {
			break
		}
	}

	return buildOrder(state, input), nil
}

func reserveSeats(ctx workflow.Context, orderID, flightID string, seatIDs []string) (*models.ReserveSeatsResult, error) {
	var result models.ReserveSeatsResult
	err := workflow.ExecuteActivity(ctx, activities.ReserveSeats, orderID, flightID, seatIDs).Get(ctx, &result)
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, temporal.NewApplicationError(result.Error, "SEAT_RESERVATION_FAILED")
	}
	return &result, nil
}

func releaseSeats(ctx workflow.Context, orderID string, seatIDs []string) {
	// Fire and forget - use local activity for quick execution
	_ = workflow.ExecuteActivity(ctx, activities.ReleaseSeats, orderID, seatIDs)
}

func validatePayment(ctx workflow.Context, orderID, paymentCode string, amount float64) (*models.ValidatePaymentResult, error) {
	// Payment has its own timeout
	paymentCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: PaymentTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // We handle retries in the workflow
		},
	})

	var result models.ValidatePaymentResult
	err := workflow.ExecuteActivity(paymentCtx, activities.ValidatePayment, orderID, paymentCode, amount).Get(paymentCtx, &result)
	if err != nil {
		return &models.ValidatePaymentResult{
			Success:  false,
			Error:    err.Error(),
			CanRetry: true,
		}, nil
	}
	return &result, nil
}

func confirmBooking(ctx workflow.Context, orderID string, seatIDs []string) (*models.ConfirmBookingResult, error) {
	var result models.ConfirmBookingResult
	err := workflow.ExecuteActivity(ctx, activities.ConfirmBooking, orderID, seatIDs).Get(ctx, &result)
	return &result, err
}

func buildOrder(state *models.BookingWorkflowState, input models.BookingWorkflowInput) *models.Order {
	return &models.Order{
		ID:              input.OrderID,
		FlightID:        input.FlightID,
		CustomerEmail:   input.CustomerEmail,
		CustomerName:    input.CustomerName,
		Seats:           state.SeatIDs,
		Status:          state.Status,
		TotalAmount:     state.TotalAmount,
		PaymentAttempts: state.PaymentAttempts,
		SeatHoldExpiry:  state.SeatHoldExpiry,
		FailureReason:   state.FailureReason,
	}
}

