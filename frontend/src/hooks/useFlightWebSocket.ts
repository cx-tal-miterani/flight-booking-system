import { useEffect, useRef, useCallback, useState } from 'react';
import type { Seat } from '../types';

export type WebSocketMessageType = 
  | 'seats_updated'
  | 'seat_conflict'
  | 'order_completed'
  | 'order_expired';

export interface SeatUpdate {
  seatId: string;
  status: 'available' | 'held' | 'booked';
  heldBy?: string;
}

export interface WebSocketMessage {
  type: WebSocketMessageType;
  flightId: string;
  seats?: SeatUpdate[];
  orderId?: string;
  message?: string;
  timestamp: number;
}

interface UseFlightWebSocketOptions {
  flightId: string | undefined;
  orderId?: string;
  onSeatsUpdated?: (seats: SeatUpdate[]) => void;
  onSeatConflict?: (seats: SeatUpdate[], message: string) => void;
  onOrderCompleted?: (orderId: string, seats: SeatUpdate[]) => void;
  onOrderExpired?: (orderId: string, seats: SeatUpdate[]) => void;
}

export function useFlightWebSocket({
  flightId,
  orderId,
  onSeatsUpdated,
  onSeatConflict,
  onOrderCompleted,
  onOrderExpired,
}: UseFlightWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout>();
  const [isConnected, setIsConnected] = useState(false);
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);

  const connect = useCallback(() => {
    if (!flightId) return;

    // Build WebSocket URL
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    let wsUrl = `${protocol}//${host}/api/flights/${flightId}/ws`;
    
    if (orderId) {
      wsUrl += `?orderId=${orderId}`;
    }

    console.log('WebSocket: Connecting to', wsUrl);

    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log('WebSocket: Connected');
      setIsConnected(true);
    };

    ws.onmessage = (event) => {
      try {
        // Handle multiple messages separated by newlines
        const messages = event.data.split('\n').filter(Boolean);
        
        for (const msgStr of messages) {
          const message: WebSocketMessage = JSON.parse(msgStr);
          console.log('WebSocket: Received', message.type, message);
          setLastMessage(message);

          switch (message.type) {
            case 'seats_updated':
              if (message.seats && onSeatsUpdated) {
                onSeatsUpdated(message.seats);
              }
              break;

            case 'seat_conflict':
              if (message.seats && onSeatConflict) {
                onSeatConflict(message.seats, message.message || 'Seat conflict detected');
              }
              break;

            case 'order_completed':
              if (message.orderId && message.seats && onOrderCompleted) {
                onOrderCompleted(message.orderId, message.seats);
              }
              break;

            case 'order_expired':
              if (message.orderId && message.seats && onOrderExpired) {
                onOrderExpired(message.orderId, message.seats);
              }
              break;
          }
        }
      } catch (error) {
        console.error('WebSocket: Failed to parse message', error);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket: Error', error);
    };

    ws.onclose = (event) => {
      console.log('WebSocket: Disconnected', event.code, event.reason);
      setIsConnected(false);
      wsRef.current = null;

      // Attempt to reconnect after 3 seconds
      if (flightId) {
        reconnectTimeoutRef.current = setTimeout(() => {
          console.log('WebSocket: Attempting to reconnect...');
          connect();
        }, 3000);
      }
    };

    wsRef.current = ws;
  }, [flightId, orderId, onSeatsUpdated, onSeatConflict, onOrderCompleted, onOrderExpired]);

  // Connect when flightId changes
  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [connect]);

  return {
    isConnected,
    lastMessage,
  };
}

// Helper function to apply seat updates to a seats array
export function applySeatUpdates(seats: Seat[], updates: SeatUpdate[]): Seat[] {
  const updateMap = new Map(updates.map(u => [u.seatId, u]));
  
  return seats.map(seat => {
    const update = updateMap.get(seat.id);
    if (update) {
      return {
        ...seat,
        status: update.status,
      };
    }
    return seat;
  });
}

