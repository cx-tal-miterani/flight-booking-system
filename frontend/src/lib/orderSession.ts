/**
 * Order Session Management
 * 
 * Persists order IDs in localStorage so users can resume their booking
 * after a page refresh. The actual order state (seats, timer, etc.) is
 * always fetched fresh from the server to ensure accuracy.
 */

const STORAGE_KEY_PREFIX = 'flight-booking-order-';

interface OrderSession {
  orderId: string;
  customerName: string;
  customerEmail: string;
  createdAt: number;
}

/**
 * Save an order session for a flight
 */
export function saveOrderSession(flightId: string, orderId: string, customerName: string, customerEmail: string): void {
  const session: OrderSession = {
    orderId,
    customerName,
    customerEmail,
    createdAt: Date.now(),
  };
  
  try {
    localStorage.setItem(`${STORAGE_KEY_PREFIX}${flightId}`, JSON.stringify(session));
  } catch (err) {
    console.error('Failed to save order session:', err);
  }
}

/**
 * Get the saved order session for a flight
 */
export function getOrderSession(flightId: string): OrderSession | null {
  try {
    const data = localStorage.getItem(`${STORAGE_KEY_PREFIX}${flightId}`);
    if (!data) return null;
    
    const session: OrderSession = JSON.parse(data);
    
    // Expire sessions older than 20 minutes (buffer beyond 15-minute timer)
    const maxAge = 20 * 60 * 1000;
    if (Date.now() - session.createdAt > maxAge) {
      clearOrderSession(flightId);
      return null;
    }
    
    return session;
  } catch (err) {
    console.error('Failed to get order session:', err);
    return null;
  }
}

/**
 * Clear the order session for a flight
 */
export function clearOrderSession(flightId: string): void {
  try {
    localStorage.removeItem(`${STORAGE_KEY_PREFIX}${flightId}`);
  } catch (err) {
    console.error('Failed to clear order session:', err);
  }
}

/**
 * Clear all expired order sessions (cleanup utility)
 */
export function clearExpiredSessions(): void {
  try {
    const maxAge = 20 * 60 * 1000;
    const now = Date.now();
    
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key?.startsWith(STORAGE_KEY_PREFIX)) {
        const data = localStorage.getItem(key);
        if (data) {
          try {
            const session: OrderSession = JSON.parse(data);
            if (now - session.createdAt > maxAge) {
              localStorage.removeItem(key);
            }
          } catch {
            // Invalid data, remove it
            localStorage.removeItem(key);
          }
        }
      }
    }
  } catch (err) {
    console.error('Failed to clear expired sessions:', err);
  }
}

