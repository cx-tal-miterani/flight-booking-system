-- Seed data for Flight Booking System

-- Insert sample flights
INSERT INTO flights (id, flight_number, origin, destination, departure_time, arrival_time, total_seats, available_seats, price_per_seat)
VALUES
    ('550e8400-e29b-41d4-a716-446655440001', 'AA101', 'New York (JFK)', 'Los Angeles (LAX)', 
     CURRENT_TIMESTAMP + INTERVAL '2 days' + INTERVAL '8 hours', 
     CURRENT_TIMESTAMP + INTERVAL '2 days' + INTERVAL '14 hours', 
     180, 180, 299.99),
    
    ('550e8400-e29b-41d4-a716-446655440002', 'UA202', 'Chicago (ORD)', 'Miami (MIA)', 
     CURRENT_TIMESTAMP + INTERVAL '3 days' + INTERVAL '10 hours', 
     CURRENT_TIMESTAMP + INTERVAL '3 days' + INTERVAL '14 hours', 
     180, 180, 249.99),
    
    ('550e8400-e29b-41d4-a716-446655440003', 'DL303', 'San Francisco (SFO)', 'Seattle (SEA)', 
     CURRENT_TIMESTAMP + INTERVAL '1 day' + INTERVAL '7 hours', 
     CURRENT_TIMESTAMP + INTERVAL '1 day' + INTERVAL '9 hours', 
     180, 180, 179.99),
    
    ('550e8400-e29b-41d4-a716-446655440004', 'SW404', 'Boston (BOS)', 'Denver (DEN)', 
     CURRENT_TIMESTAMP + INTERVAL '4 days' + INTERVAL '6 hours', 
     CURRENT_TIMESTAMP + INTERVAL '4 days' + INTERVAL '10 hours', 
     180, 180, 329.99),
    
    ('550e8400-e29b-41d4-a716-446655440005', 'JB505', 'Washington (DCA)', 'Orlando (MCO)', 
     CURRENT_TIMESTAMP + INTERVAL '5 days' + INTERVAL '9 hours', 
     CURRENT_TIMESTAMP + INTERVAL '5 days' + INTERVAL '12 hours', 
     180, 180, 199.99);

-- Generate seats for each flight (30 rows, 6 columns: A-F)
DO $$
DECLARE
    flight_record RECORD;
    row_num INTEGER;
    col_letter CHAR(1);
    seat_price DECIMAL(10,2);
    seat_class VARCHAR(20);
BEGIN
    FOR flight_record IN SELECT id, price_per_seat FROM flights LOOP
        FOR row_num IN 1..30 LOOP
            FOR col_letter IN SELECT unnest(ARRAY['A', 'B', 'C', 'D', 'E', 'F']) LOOP
                -- First 5 rows are business class (1.5x price)
                IF row_num <= 5 THEN
                    seat_class := 'business';
                    seat_price := flight_record.price_per_seat * 1.5;
                -- Rows 6-10 are premium economy (1.2x price)
                ELSIF row_num <= 10 THEN
                    seat_class := 'premium';
                    seat_price := flight_record.price_per_seat * 1.2;
                -- Rest is economy
                ELSE
                    seat_class := 'economy';
                    seat_price := flight_record.price_per_seat;
                END IF;
                
                INSERT INTO seats (flight_id, seat_number, row_number, column_letter, class, price)
                VALUES (
                    flight_record.id,
                    row_num || col_letter,
                    row_num,
                    col_letter,
                    seat_class,
                    seat_price
                );
            END LOOP;
        END LOOP;
    END LOOP;
END $$;

-- Randomly mark some seats as booked (to simulate existing bookings)
UPDATE seats
SET status = 'booked'
WHERE id IN (
    SELECT id FROM seats
    WHERE status = 'available'
    ORDER BY RANDOM()
    LIMIT 50
);

-- Update available_seats count for each flight
UPDATE flights f
SET available_seats = (
    SELECT COUNT(*) FROM seats s
    WHERE s.flight_id = f.id AND s.status = 'available'
);

