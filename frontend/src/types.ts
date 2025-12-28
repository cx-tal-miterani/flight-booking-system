export interface Flight {
  id: string;
  flightNumber: string;
  origin: string;
  destination: string;
  departureTime: string;
  arrivalTime: string;
  totalSeats: number;
  availableSeats: number;
  pricePerSeat: number;
}

export interface Seat {
  id: string;
  flightId: string;
  row: number;
  column: string;
  class: 'economy' | 'business' | 'first';
  status: 'available' | 'held' | 'booked';
  price: number;
}

export type OrderStatus =
  | 'pending'
  | 'seats_selected'
  | 'awaiting_payment'
  | 'processing'
  | 'confirmed'
  | 'failed'
  | 'cancelled'
  | 'expired';

export interface Order {
  id: string;
  flightId: string;
  customerEmail: string;
  customerName: string;
  seats: string[];
  status: OrderStatus;
  totalAmount: number;
  paymentAttempts: number;
  seatHoldExpiry: string;
  createdAt: string;
  updatedAt: string;
  failureReason?: string;
}

export interface OrderStatusResponse {
  order: Order;
  remainingSeconds: number;
  message?: string;
}

