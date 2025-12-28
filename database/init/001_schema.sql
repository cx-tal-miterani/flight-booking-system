-- Flight Booking System Database Schema

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Flights table
CREATE TABLE flights (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    flight_number VARCHAR(10) NOT NULL UNIQUE,
    origin VARCHAR(100) NOT NULL,
    destination VARCHAR(100) NOT NULL,
    departure_time TIMESTAMP WITH TIME ZONE NOT NULL,
    arrival_time TIMESTAMP WITH TIME ZONE NOT NULL,
    total_seats INTEGER NOT NULL DEFAULT 180,
    available_seats INTEGER NOT NULL DEFAULT 180,
    price_per_seat DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Seat status enum
CREATE TYPE seat_status AS ENUM ('available', 'held', 'booked');

-- Seats table
CREATE TABLE seats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    flight_id UUID NOT NULL REFERENCES flights(id) ON DELETE CASCADE,
    seat_number VARCHAR(5) NOT NULL, -- e.g., "1A", "15F"
    row_number INTEGER NOT NULL,
    column_letter CHAR(1) NOT NULL,
    class VARCHAR(20) NOT NULL DEFAULT 'economy',
    status seat_status NOT NULL DEFAULT 'available',
    price DECIMAL(10, 2) NOT NULL,
    held_until TIMESTAMP WITH TIME ZONE,
    held_by_order UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(flight_id, seat_number)
);

-- Order status enum
CREATE TYPE order_status AS ENUM (
    'pending',
    'seats_selected',
    'awaiting_payment',
    'processing',
    'confirmed',
    'failed',
    'cancelled',
    'expired'
);

-- Orders table
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    flight_id UUID NOT NULL REFERENCES flights(id),
    customer_name VARCHAR(100) NOT NULL,
    customer_email VARCHAR(255) NOT NULL,
    status order_status NOT NULL DEFAULT 'pending',
    total_amount DECIMAL(10, 2) NOT NULL DEFAULT 0,
    payment_attempts INTEGER NOT NULL DEFAULT 0,
    failure_reason TEXT,
    workflow_id VARCHAR(255),
    workflow_run_id VARCHAR(255),
    reservation_expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Order seats junction table
CREATE TABLE order_seats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    seat_id UUID NOT NULL REFERENCES seats(id),
    price DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(order_id, seat_id)
);

-- Indexes for performance
CREATE INDEX idx_flights_departure ON flights(departure_time);
CREATE INDEX idx_flights_route ON flights(origin, destination);
CREATE INDEX idx_seats_flight ON seats(flight_id);
CREATE INDEX idx_seats_status ON seats(status);
CREATE INDEX idx_seats_held_until ON seats(held_until) WHERE held_until IS NOT NULL;
CREATE INDEX idx_orders_flight ON orders(flight_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_workflow ON orders(workflow_id);
CREATE INDEX idx_order_seats_order ON order_seats(order_id);
CREATE INDEX idx_order_seats_seat ON order_seats(seat_id);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_flights_updated_at
    BEFORE UPDATE ON flights
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_seats_updated_at
    BEFORE UPDATE ON seats
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Function to release expired seat holds
CREATE OR REPLACE FUNCTION release_expired_holds()
RETURNS INTEGER AS $$
DECLARE
    released_count INTEGER;
BEGIN
    WITH released AS (
        UPDATE seats
        SET status = 'available',
            held_until = NULL,
            held_by_order = NULL
        WHERE status = 'held'
          AND held_until < CURRENT_TIMESTAMP
        RETURNING id
    )
    SELECT COUNT(*) INTO released_count FROM released;
    
    RETURN released_count;
END;
$$ LANGUAGE plpgsql;

