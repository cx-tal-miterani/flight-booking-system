import { render, screen, fireEvent } from '@testing-library/react';
import { SeatMap } from '../components/SeatMap';
import type { Seat } from '../types';

const createMockSeats = (): Seat[] => [
  { id: 'FL001-1A', flightId: 'FL001', row: 1, column: 'A', class: 'economy', status: 'available', price: 150 },
  { id: 'FL001-1B', flightId: 'FL001', row: 1, column: 'B', class: 'economy', status: 'available', price: 150 },
  { id: 'FL001-1C', flightId: 'FL001', row: 1, column: 'C', class: 'economy', status: 'booked', price: 150 },
  { id: 'FL001-1D', flightId: 'FL001', row: 1, column: 'D', class: 'economy', status: 'available', price: 150 },
  { id: 'FL001-1E', flightId: 'FL001', row: 1, column: 'E', class: 'economy', status: 'held', price: 150 },
  { id: 'FL001-1F', flightId: 'FL001', row: 1, column: 'F', class: 'economy', status: 'available', price: 150 },
];

describe('SeatMap', () => {
  it('should render all seats', () => {
    const seats = createMockSeats();
    render(<SeatMap seats={seats} selectedSeats={[]} onSeatSelect={jest.fn()} />);
    
    expect(screen.getByRole('button', { name: /Seat 1A/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Seat 1B/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Seat 1C/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Seat 1D/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Seat 1E/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /Seat 1F/i })).toBeInTheDocument();
  });

  it('should render legend', () => {
    render(<SeatMap seats={createMockSeats()} selectedSeats={[]} onSeatSelect={jest.fn()} />);
    
    expect(screen.getByText('Available')).toBeInTheDocument();
    expect(screen.getByText('Selected')).toBeInTheDocument();
    expect(screen.getByText('Held')).toBeInTheDocument();
    expect(screen.getByText('Booked')).toBeInTheDocument();
  });

  it('should call onSeatSelect when available seat is clicked', () => {
    const onSeatSelect = jest.fn();
    render(<SeatMap seats={createMockSeats()} selectedSeats={[]} onSeatSelect={onSeatSelect} />);
    
    fireEvent.click(screen.getByRole('button', { name: /Seat 1A/i }));
    
    expect(onSeatSelect).toHaveBeenCalledWith('FL001-1A');
  });

  it('should not call onSeatSelect when booked seat is clicked', () => {
    const onSeatSelect = jest.fn();
    render(<SeatMap seats={createMockSeats()} selectedSeats={[]} onSeatSelect={onSeatSelect} />);
    
    const bookedSeat = screen.getByRole('button', { name: /Seat 1C/i });
    fireEvent.click(bookedSeat);
    
    expect(onSeatSelect).not.toHaveBeenCalled();
  });

  it('should not call onSeatSelect when held seat is clicked', () => {
    const onSeatSelect = jest.fn();
    render(<SeatMap seats={createMockSeats()} selectedSeats={[]} onSeatSelect={onSeatSelect} />);
    
    const heldSeat = screen.getByRole('button', { name: /Seat 1E/i });
    fireEvent.click(heldSeat);
    
    expect(onSeatSelect).not.toHaveBeenCalled();
  });

  it('should apply selected class to selected seats', () => {
    render(<SeatMap seats={createMockSeats()} selectedSeats={['FL001-1A', 'FL001-1B']} onSeatSelect={jest.fn()} />);
    
    const seatA = screen.getByRole('button', { name: /Seat 1A/i });
    const seatB = screen.getByRole('button', { name: /Seat 1B/i });
    
    expect(seatA).toHaveClass('seat-selected');
    expect(seatB).toHaveClass('seat-selected');
  });

  it('should allow deselecting a selected seat', () => {
    const onSeatSelect = jest.fn();
    render(<SeatMap seats={createMockSeats()} selectedSeats={['FL001-1A']} onSeatSelect={onSeatSelect} />);
    
    fireEvent.click(screen.getByRole('button', { name: /Seat 1A/i }));
    
    expect(onSeatSelect).toHaveBeenCalledWith('FL001-1A');
  });

  it('should display row numbers', () => {
    render(<SeatMap seats={createMockSeats()} selectedSeats={[]} onSeatSelect={jest.fn()} />);
    
    expect(screen.getByText('1')).toBeInTheDocument();
  });

  it('should display column labels', () => {
    render(<SeatMap seats={createMockSeats()} selectedSeats={[]} onSeatSelect={jest.fn()} />);
    
    // Column labels appear in the footer
    const columnLabels = screen.getAllByText('A');
    expect(columnLabels.length).toBeGreaterThan(0);
  });
});

