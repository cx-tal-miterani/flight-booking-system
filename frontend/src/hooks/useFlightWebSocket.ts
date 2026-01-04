import { useEffect, useRef, useCallback } from 'react';

export type WebSocketMessageType = 
  | 'seats_updated' 
  | 'seat_conflict' 
  | 'order_completed' 
  | 'order_expired'
  | 'seats_released';

export interface WebSocketMessage {
  type: WebSocketMessageType;
  flightId: string;
  seatIds?: string[];
  orderId?: string;
  status?: string;
  timestamp: number;
}

interface UseFlightWebSocketOptions {
  flightId: string | undefined;
  orderId?: string;
  onSeatsUpdated?: (seatIds: string[], status: string, orderId?: string) => void;
  onSeatConflict?: (seatIds: string[]) => void;
  onOrderCompleted?: (orderId: string, seatIds: string[]) => void;
  onOrderExpired?: (orderId: string, seatIds: string[]) => void;
  onSeatsReleased?: (seatIds: string[], orderId?: string) => void;
}

export function useFlightWebSocket({
  flightId,
  orderId,
  onSeatsUpdated,
  onSeatConflict,
  onOrderCompleted,
  onOrderExpired,
  onSeatsReleased,
}: UseFlightWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttempts = useRef(0);
  const maxReconnectAttempts = 5;

  const connect = useCallback(() => {
    if (!flightId) return;

    // Build WebSocket URL
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    let wsUrl = `${protocol}//${host}/api/flights/${flightId}/ws`;
    if (orderId) {
      wsUrl += `?orderId=${orderId}`;
    }

    // Close existing connection
    if (wsRef.current) {
      wsRef.current.close();
    }

    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    ws.onopen = () => {
      console.log('WebSocket connected for flight:', flightId);
      reconnectAttempts.current = 0;
    };

    ws.onmessage = (event) => {
      try {
        // Handle multiple messages separated by newlines
        const messages = event.data.split('\n').filter(Boolean);
        
        for (const msgStr of messages) {
          const message: WebSocketMessage = JSON.parse(msgStr);
          
          switch (message.type) {
            case 'seats_updated':
              if (message.seatIds && message.status) {
                onSeatsUpdated?.(message.seatIds, message.status, message.orderId);
              }
              break;
              
            case 'seat_conflict':
              if (message.seatIds) {
                onSeatConflict?.(message.seatIds);
              }
              break;
              
            case 'order_completed':
              if (message.orderId && message.seatIds) {
                onOrderCompleted?.(message.orderId, message.seatIds);
              }
              break;
              
            case 'order_expired':
              if (message.orderId && message.seatIds) {
                onOrderExpired?.(message.orderId, message.seatIds);
              }
              break;
              
            case 'seats_released':
              if (message.seatIds) {
                onSeatsReleased?.(message.seatIds, message.orderId);
              }
              break;
          }
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };

    ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    ws.onclose = (event) => {
      console.log('WebSocket closed:', event.code, event.reason);
      
      // Attempt reconnection if not a normal close
      if (event.code !== 1000 && reconnectAttempts.current < maxReconnectAttempts) {
        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), 30000);
        reconnectAttempts.current++;
        
        console.log(`WebSocket reconnecting in ${delay}ms (attempt ${reconnectAttempts.current})`);
        reconnectTimeoutRef.current = setTimeout(connect, delay);
      }
    };
  }, [flightId, orderId, onSeatsUpdated, onSeatConflict, onOrderCompleted, onOrderExpired, onSeatsReleased]);

  useEffect(() => {
    connect();

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close(1000, 'Component unmounting');
      }
    };
  }, [connect]);

  return {
    isConnected: wsRef.current?.readyState === WebSocket.OPEN,
  };
}

