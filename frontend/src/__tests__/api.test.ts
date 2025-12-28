import { api } from '../api';

describe('API', () => {
  beforeEach(() => {
    (global.fetch as jest.Mock).mockClear();
  });

  describe('getFlights', () => {
    it('should fetch and return flights', async () => {
      const mockFlights = [
        { id: 'FL001', flightNumber: 'AA123', origin: 'NYC', destination: 'LAX', pricePerSeat: 150 },
        { id: 'FL002', flightNumber: 'UA456', origin: 'ORD', destination: 'MIA', pricePerSeat: 200 },
      ];

      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockFlights,
      });

      const result = await api.getFlights();

      expect(fetch).toHaveBeenCalledWith('/api/flights');
      expect(result).toEqual(mockFlights);
    });

    it('should throw error on failed request', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => ({ error: 'Server error' }),
      });

      await expect(api.getFlights()).rejects.toThrow('Server error');
    });
  });

  describe('getFlight', () => {
    it('should fetch a single flight by ID', async () => {
      const mockFlight = { id: 'FL001', flightNumber: 'AA123' };

      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockFlight,
      });

      const result = await api.getFlight('FL001');

      expect(fetch).toHaveBeenCalledWith('/api/flights/FL001');
      expect(result).toEqual(mockFlight);
    });
  });

  describe('getFlightSeats', () => {
    it('should fetch seats for a flight', async () => {
      const mockSeats = [
        { id: 'FL001-1A', row: 1, column: 'A', status: 'available', price: 150 },
        { id: 'FL001-1B', row: 1, column: 'B', status: 'booked', price: 150 },
      ];

      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockSeats,
      });

      const result = await api.getFlightSeats('FL001');

      expect(fetch).toHaveBeenCalledWith('/api/flights/FL001/seats');
      expect(result).toEqual(mockSeats);
    });
  });

  describe('createOrder', () => {
    it('should create a new order', async () => {
      const mockOrder = { id: 'abc123', status: 'pending' };
      const orderRequest = {
        flightId: 'FL001',
        customerEmail: 'test@example.com',
        customerName: 'John Doe',
      };

      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockOrder,
      });

      const result = await api.createOrder(orderRequest);

      expect(fetch).toHaveBeenCalledWith('/api/orders', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(orderRequest),
      });
      expect(result).toEqual(mockOrder);
    });
  });

  describe('getOrderStatus', () => {
    it('should fetch order status', async () => {
      const mockStatus = {
        order: { id: 'abc123', status: 'seats_selected', totalAmount: 300 },
        remainingSeconds: 850,
      };

      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockStatus,
      });

      const result = await api.getOrderStatus('abc123');

      expect(fetch).toHaveBeenCalledWith('/api/orders/abc123');
      expect(result).toEqual(mockStatus);
    });
  });

  describe('selectSeats', () => {
    it('should select seats for an order', async () => {
      const mockResponse = {
        order: { id: 'abc123', seats: ['FL001-1A', 'FL001-1B'] },
        remainingSeconds: 900,
      };

      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      });

      const result = await api.selectSeats('abc123', ['FL001-1A', 'FL001-1B']);

      expect(fetch).toHaveBeenCalledWith('/api/orders/abc123/seats', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ seatIds: ['FL001-1A', 'FL001-1B'] }),
      });
      expect(result).toEqual(mockResponse);
    });
  });

  describe('submitPayment', () => {
    it('should submit payment code', async () => {
      const mockResponse = {
        order: { id: 'abc123', status: 'processing' },
        remainingSeconds: 800,
      };

      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResponse,
      });

      const result = await api.submitPayment('abc123', '12345');

      expect(fetch).toHaveBeenCalledWith('/api/orders/abc123/pay', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ paymentCode: '12345' }),
      });
      expect(result).toEqual(mockResponse);
    });
  });

  describe('cancelOrder', () => {
    it('should cancel an order', async () => {
      (global.fetch as jest.Mock).mockResolvedValueOnce({
        ok: true,
      });

      await api.cancelOrder('abc123');

      expect(fetch).toHaveBeenCalledWith('/api/orders/abc123', {
        method: 'DELETE',
      });
    });
  });
});

