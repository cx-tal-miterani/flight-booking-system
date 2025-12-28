package repository

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
	ErrNotFound = errors.New("not found")
)

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

// Repository handles database operations for the worker
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository creates a new repository
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetOrderStatus returns the current status of an order
func (r *Repository) GetOrderStatus(ctx context.Context, orderID uuid.UUID) (OrderStatus, error) {
	var status OrderStatus
	err := r.pool.QueryRow(ctx, `
		SELECT status FROM orders WHERE id = $1
	`, orderID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get order status: %w", err)
	}
	return status, nil
}

// UpdateOrderStatus updates the status of an order
func (r *Repository) UpdateOrderStatus(ctx context.Context, orderID uuid.UUID, status OrderStatus) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE orders SET status = $1 WHERE id = $2
	`, status, orderID)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}
	return nil
}

// UpdateOrderPayment updates payment-related fields
func (r *Repository) UpdateOrderPayment(ctx context.Context, orderID uuid.UUID, attempts int, failureReason *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE orders SET payment_attempts = $1, failure_reason = $2 WHERE id = $3
	`, attempts, failureReason, orderID)
	if err != nil {
		return fmt.Errorf("failed to update order payment: %w", err)
	}
	return nil
}

// BookSeats permanently books seats after payment
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

// ReleaseSeats releases held seats
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

// GetReservationExpiry returns when the reservation expires
func (r *Repository) GetReservationExpiry(ctx context.Context, orderID uuid.UUID) (*time.Time, error) {
	var expiresAt *time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT reservation_expires_at FROM orders WHERE id = $1
	`, orderID).Scan(&expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get expiration: %w", err)
	}
	return expiresAt, nil
}

// CheckSeatsHeld checks if seats are still held for an order
func (r *Repository) CheckSeatsHeld(ctx context.Context, orderID uuid.UUID) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM seats WHERE held_by_order = $1 AND status = 'held'
	`, orderID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check seats: %w", err)
	}
	return count > 0, nil
}

