import type { Flight, Seat, Order, OrderStatusResponse } from './types';

const API_BASE = '/api';

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }));
    throw new Error(error.error || `HTTP ${response.status}`);
  }
  return response.json();
}

export interface CreateOrderRequest {
  flightId: string;
  customerEmail: string;
  customerName: string;
}

export const api = {
  // Flights
  getFlights: async (): Promise<Flight[]> => {
    const response = await fetch(`${API_BASE}/flights`);
    return handleResponse<Flight[]>(response);
  },

  getFlight: async (id: string): Promise<Flight> => {
    const response = await fetch(`${API_BASE}/flights/${id}`);
    return handleResponse<Flight>(response);
  },

  getFlightSeats: async (flightId: string): Promise<Seat[]> => {
    const response = await fetch(`${API_BASE}/flights/${flightId}/seats`);
    return handleResponse<Seat[]>(response);
  },

  // Orders
  createOrder: async (request: CreateOrderRequest): Promise<Order> => {
    const response = await fetch(`${API_BASE}/orders`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });
    return handleResponse<Order>(response);
  },

  getOrderStatus: async (orderId: string): Promise<OrderStatusResponse> => {
    const response = await fetch(`${API_BASE}/orders/${orderId}`);
    return handleResponse<OrderStatusResponse>(response);
  },

  selectSeats: async (orderId: string, seatIds: string[]): Promise<OrderStatusResponse> => {
    const response = await fetch(`${API_BASE}/orders/${orderId}/seats`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ seatIds }),
    });
    return handleResponse<OrderStatusResponse>(response);
  },

  submitPayment: async (orderId: string, paymentCode: string): Promise<OrderStatusResponse> => {
    const response = await fetch(`${API_BASE}/orders/${orderId}/pay`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ paymentCode }),
    });
    return handleResponse<OrderStatusResponse>(response);
  },

  cancelOrder: async (orderId: string): Promise<void> => {
    const response = await fetch(`${API_BASE}/orders/${orderId}`, {
      method: 'DELETE',
    });
    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Unknown error' }));
      throw new Error(error.error || `HTTP ${response.status}`);
    }
  },

  refreshTimer: async (orderId: string): Promise<OrderStatusResponse> => {
    const response = await fetch(`${API_BASE}/orders/${orderId}/refresh`, {
      method: 'POST',
    });
    return handleResponse<OrderStatusResponse>(response);
  },
};
