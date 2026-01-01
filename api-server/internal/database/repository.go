package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrSeatNotAvailable = errors.New("seat not available")
	ErrOrderExpired  = errors.New("order reservation expired")
)

// Repository handles all database operations
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// --- Flight Operations ---

// GetAllFlights returns all flights with available seats
func (r *Repository) GetAllFlights(ctx context.Context) ([]Flight, error) {
	query := `
		SELECT id, flight_number, origin, destination, departure_time, arrival_time,
		       total_seats, available_seats, price_per_seat, created_at, updated_at
		FROM flights
		WHERE departure_time > NOW()
		ORDER BY departure_time ASC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query flights: %w", err)
	}
	defer rows.Close()

	var flights []Flight
	for rows.Next() {
		var f Flight
		err := rows.Scan(
			&f.ID, &f.FlightNumber, &f.Origin, &f.Destination,
			&f.DepartureTime, &f.ArrivalTime, &f.TotalSeats, &f.AvailableSeats,
			&f.PricePerSeat, &f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan flight: %w", err)
		}
		flights = append(flights, f)
	}

	return flights, nil
}

// GetFlightByID returns a flight by ID
func (r *Repository) GetFlightByID(ctx context.Context, id uuid.UUID) (*Flight, error) {
	query := `
		SELECT id, flight_number, origin, destination, departure_time, arrival_time,
		       total_seats, available_seats, price_per_seat, created_at, updated_at
		FROM flights
		WHERE id = $1
	`

	var f Flight
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&f.ID, &f.FlightNumber, &f.Origin, &f.Destination,
		&f.DepartureTime, &f.ArrivalTime, &f.TotalSeats, &f.AvailableSeats,
		&f.PricePerSeat, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get flight: %w", err)
	}

	return &f, nil
}

// --- Seat Operations ---

// GetFlightSeats returns all seats for a flight
func (r *Repository) GetFlightSeats(ctx context.Context, flightID uuid.UUID) ([]Seat, error) {
	// First release any expired holds
	_, _ = r.pool.Exec(ctx, "SELECT release_expired_holds()")

	query := `
		SELECT id, flight_id, seat_number, row_number, column_letter, class,
		       status, price, held_until, held_by_order, created_at, updated_at
		FROM seats
		WHERE flight_id = $1
		ORDER BY row_number, column_letter
	`

	rows, err := r.pool.Query(ctx, query, flightID)
	if err != nil {
		return nil, fmt.Errorf("failed to query seats: %w", err)
	}
	defer rows.Close()

	var seats []Seat
	for rows.Next() {
		var s Seat
		err := rows.Scan(
			&s.ID, &s.FlightID, &s.SeatNumber, &s.RowNumber, &s.ColumnLetter,
			&s.Class, &s.Status, &s.Price, &s.HeldUntil, &s.HeldByOrder,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan seat: %w", err)
		}
		seats = append(seats, s)
	}

	return seats, nil
}

// GetSeatByID returns a seat by ID
func (r *Repository) GetSeatByID(ctx context.Context, id uuid.UUID) (*Seat, error) {
	query := `
		SELECT id, flight_id, seat_number, row_number, column_letter, class,
		       status, price, held_until, held_by_order, created_at, updated_at
		FROM seats
		WHERE id = $1
	`

	var s Seat
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.FlightID, &s.SeatNumber, &s.RowNumber, &s.ColumnLetter,
		&s.Class, &s.Status, &s.Price, &s.HeldUntil, &s.HeldByOrder,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get seat: %w", err)
	}

	return &s, nil
}

// HoldSeats holds seats for an order with a 15-minute timer
func (r *Repository) HoldSeats(ctx context.Context, orderID uuid.UUID, seatIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	holdUntil := time.Now().Add(15 * time.Minute)

	// First, release any seats previously held by this order
	_, err = tx.Exec(ctx, `
		UPDATE seats
		SET status = 'available', held_until = NULL, held_by_order = NULL
		WHERE held_by_order = $1
	`, orderID)
	if err != nil {
		return fmt.Errorf("failed to release previous holds: %w", err)
	}

	// Hold new seats
	for _, seatID := range seatIDs {
		result, err := tx.Exec(ctx, `
			UPDATE seats
			SET status = 'held', held_until = $1, held_by_order = $2
			WHERE id = $3 AND (status = 'available' OR held_by_order = $2)
		`, holdUntil, orderID, seatID)
		if err != nil {
			return fmt.Errorf("failed to hold seat: %w", err)
		}
		if result.RowsAffected() == 0 {
			return ErrSeatNotAvailable
		}
	}

	// Update order with new expiration time
	_, err = tx.Exec(ctx, `
		UPDATE orders
		SET reservation_expires_at = $1, status = 'seats_selected'
		WHERE id = $2
	`, holdUntil, orderID)
	if err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	return tx.Commit(ctx)
}

// BookSeats permanently books seats (after successful payment)
func (r *Repository) BookSeats(ctx context.Context, orderID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update seats status to booked
	_, err = tx.Exec(ctx, `
		UPDATE seats
		SET status = 'booked', held_until = NULL
		WHERE held_by_order = $1 AND status = 'held'
	`, orderID)
	if err != nil {
		return fmt.Errorf("failed to book seats: %w", err)
	}

	// Update flight available seats count
	_, err = tx.Exec(ctx, `
		UPDATE flights f
		SET available_seats = (
			SELECT COUNT(*) FROM seats s
			WHERE s.flight_id = f.id AND s.status = 'available'
		)
		WHERE id = (SELECT flight_id FROM orders WHERE id = $1)
	`, orderID)
	if err != nil {
		return fmt.Errorf("failed to update available seats: %w", err)
	}

	return tx.Commit(ctx)
}

// ReleaseSeats releases held seats (on cancellation or expiry)
func (r *Repository) ReleaseSeats(ctx context.Context, orderID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE seats
		SET status = 'available', held_until = NULL, held_by_order = NULL
		WHERE held_by_order = $1
	`, orderID)
	if err != nil {
		return fmt.Errorf("failed to release seats: %w", err)
	}
	return nil
}

// --- Order Operations ---

// CreateOrder creates a new order
func (r *Repository) CreateOrder(ctx context.Context, order *Order) error {
	query := `
		INSERT INTO orders (id, flight_id, customer_name, customer_email, status, workflow_id, workflow_run_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at
	`

	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		order.ID, order.FlightID, order.CustomerName, order.CustomerEmail,
		order.Status, order.WorkflowID, order.WorkflowRunID,
	).Scan(&order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

// GetOrderByID returns an order by ID with its seats
func (r *Repository) GetOrderByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	query := `
		SELECT id, flight_id, customer_name, customer_email, status, total_amount,
		       payment_attempts, failure_reason, workflow_id, workflow_run_id,
		       reservation_expires_at, created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	var o Order
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&o.ID, &o.FlightID, &o.CustomerName, &o.CustomerEmail, &o.Status,
		&o.TotalAmount, &o.PaymentAttempts, &o.FailureReason, &o.WorkflowID,
		&o.WorkflowRunID, &o.ReservationExpiresAt, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Get associated seats
	seatQuery := `
		SELECT s.seat_number
		FROM order_seats os
		JOIN seats s ON s.id = os.seat_id
		WHERE os.order_id = $1
	`
	rows, err := r.pool.Query(ctx, seatQuery, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query order seats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var seatNumber string
		if err := rows.Scan(&seatNumber); err != nil {
			return nil, fmt.Errorf("failed to scan seat number: %w", err)
		}
		o.Seats = append(o.Seats, seatNumber)
	}

	return &o, nil
}

// UpdateOrderStatus updates the status of an order
func (r *Repository) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status OrderStatus) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE orders SET status = $1 WHERE id = $2
	`, status, id)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}

// UpdateOrderPayment updates payment-related fields
func (r *Repository) UpdateOrderPayment(ctx context.Context, id uuid.UUID, attempts int, failureReason *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE orders SET payment_attempts = $1, failure_reason = $2 WHERE id = $3
	`, attempts, failureReason, id)
	if err != nil {
		return fmt.Errorf("failed to update order payment: %w", err)
	}
	return nil
}

// SetOrderSeats sets the seats for an order and calculates total
func (r *Repository) SetOrderSeats(ctx context.Context, orderID uuid.UUID, seatIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Clear existing order seats
	_, err = tx.Exec(ctx, `DELETE FROM order_seats WHERE order_id = $1`, orderID)
	if err != nil {
		return fmt.Errorf("failed to clear order seats: %w", err)
	}

	// Add new seats and calculate total
	var totalAmount float64
	for _, seatID := range seatIDs {
		var price float64
		err := tx.QueryRow(ctx, `SELECT price FROM seats WHERE id = $1`, seatID).Scan(&price)
		if err != nil {
			return fmt.Errorf("failed to get seat price: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO order_seats (order_id, seat_id, price)
			VALUES ($1, $2, $3)
		`, orderID, seatID, price)
		if err != nil {
			return fmt.Errorf("failed to add order seat: %w", err)
		}

		totalAmount += price
	}

	// Update order total
	_, err = tx.Exec(ctx, `UPDATE orders SET total_amount = $1 WHERE id = $2`, totalAmount, orderID)
	if err != nil {
		return fmt.Errorf("failed to update order total: %w", err)
	}

	return tx.Commit(ctx)
}

// GetOrderRemainingSeconds returns seconds until reservation expires
func (r *Repository) GetOrderRemainingSeconds(ctx context.Context, orderID uuid.UUID) (int, error) {
	var expiresAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT reservation_expires_at FROM orders WHERE id = $1
	`, orderID).Scan(&expiresAt)
	if err != nil {
		return 0, fmt.Errorf("failed to get expiration: %w", err)
	}

	if expiresAt == nil {
		return 0, nil
	}

	remaining := int(time.Until(*expiresAt).Seconds())
	if remaining < 0 {
		return 0, nil
	}
	return remaining, nil
}

// GetSeatIDsByFlightAndNumbers returns seat IDs by flight ID and seat numbers
func (r *Repository) GetSeatIDsByFlightAndNumbers(ctx context.Context, flightID uuid.UUID, seatNumbers []string) ([]uuid.UUID, error) {
	query := `
		SELECT id FROM seats
		WHERE flight_id = $1 AND seat_number = ANY($2)
	`

	rows, err := r.pool.Query(ctx, query, flightID, seatNumbers)
	if err != nil {
		return nil, fmt.Errorf("failed to query seats: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan seat id: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

// GetOrderSeatIDs returns the UUIDs of seats associated with an order
func (r *Repository) GetOrderSeatIDs(ctx context.Context, orderID uuid.UUID) ([]uuid.UUID, error) {
	query := `
		SELECT seat_id FROM order_seats WHERE order_id = $1
	`

	rows, err := r.pool.Query(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to query order seats: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan seat id: %w", err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

