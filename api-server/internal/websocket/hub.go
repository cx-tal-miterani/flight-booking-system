package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeSeatsUpdated   MessageType = "seats_updated"
	MessageTypeSeatConflict   MessageType = "seat_conflict"
	MessageTypeOrderCompleted MessageType = "order_completed"
	MessageTypeOrderExpired   MessageType = "order_expired"
)

// SeatUpdate represents a seat status change
type SeatUpdate struct {
	SeatID string `json:"seatId"`
	Status string `json:"status"` // available, held, booked
	HeldBy string `json:"heldBy,omitempty"`
}

// Message represents a WebSocket message
type Message struct {
	Type      MessageType  `json:"type"`
	FlightID  string       `json:"flightId"`
	Seats     []SeatUpdate `json:"seats,omitempty"`
	OrderID   string       `json:"orderId,omitempty"`
	Message   string       `json:"message,omitempty"`
	Timestamp int64        `json:"timestamp"`
}

// Client represents a WebSocket client connection
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	flightID uuid.UUID
	orderID  *uuid.UUID
}

// Hub manages WebSocket connections per flight
type Hub struct {
	clients    map[uuid.UUID]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan *Message
	mu         sync.RWMutex
}

var globalHub *Hub
var hubOnce sync.Once

// GetHub returns the global hub instance
func GetHub() *Hub {
	hubOnce.Do(func() {
		globalHub = NewHub()
		go globalHub.Run()
	})
	return globalHub
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *Message, 256),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.flightID] == nil {
				h.clients[client.flightID] = make(map[*Client]bool)
			}
			h.clients[client.flightID][client] = true
			log.Printf("WebSocket: Client registered for flight %s (total: %d)", client.flightID, len(h.clients[client.flightID]))
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.flightID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					log.Printf("WebSocket: Client unregistered from flight %s (remaining: %d)", client.flightID, len(clients))
					if len(clients) == 0 {
						delete(h.clients, client.flightID)
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			flightID, err := uuid.Parse(message.FlightID)
			if err != nil {
				log.Printf("WebSocket: Invalid flight ID in broadcast: %s", message.FlightID)
				continue
			}

			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("WebSocket: Failed to marshal message: %v", err)
				continue
			}

			h.mu.RLock()
			clients := h.clients[flightID]
			h.mu.RUnlock()

			log.Printf("WebSocket: Broadcasting %s to %d clients for flight %s", message.Type, len(clients), message.FlightID)

			for client := range clients {
				select {
				case client.send <- data:
				default:
					h.mu.Lock()
					delete(h.clients[flightID], client)
					close(client.send)
					h.mu.Unlock()
				}
			}
		}
	}
}

// BroadcastSeatUpdate broadcasts seat status changes to all clients watching a flight
func (h *Hub) BroadcastSeatUpdate(flightID string, seats []SeatUpdate) {
	msg := &Message{
		Type:      MessageTypeSeatsUpdated,
		FlightID:  flightID,
		Seats:     seats,
		Timestamp: time.Now().UnixMilli(),
	}
	h.broadcast <- msg
}

// BroadcastSeatsHeld broadcasts that seats were held by an order
func (h *Hub) BroadcastSeatsHeld(flightID string, orderID string, seatIDs []string) {
	seats := make([]SeatUpdate, len(seatIDs))
	for i, seatID := range seatIDs {
		seats[i] = SeatUpdate{
			SeatID: seatID,
			Status: "held",
			HeldBy: orderID,
		}
	}

	msg := &Message{
		Type:      MessageTypeSeatsUpdated,
		FlightID:  flightID,
		OrderID:   orderID,
		Seats:     seats,
		Timestamp: time.Now().UnixMilli(),
	}
	h.broadcast <- msg
}

// BroadcastOrderCompleted notifies clients that an order was completed
func (h *Hub) BroadcastOrderCompleted(flightID string, orderID string, seatIDs []string) {
	seats := make([]SeatUpdate, len(seatIDs))
	for i, seatID := range seatIDs {
		seats[i] = SeatUpdate{
			SeatID: seatID,
			Status: "booked",
		}
	}

	msg := &Message{
		Type:      MessageTypeOrderCompleted,
		FlightID:  flightID,
		OrderID:   orderID,
		Seats:     seats,
		Message:   "Seats have been booked",
		Timestamp: time.Now().UnixMilli(),
	}
	h.broadcast <- msg
}

// BroadcastOrderExpired notifies clients that an order expired
func (h *Hub) BroadcastOrderExpired(flightID string, orderID string, seatIDs []string) {
	seats := make([]SeatUpdate, len(seatIDs))
	for i, seatID := range seatIDs {
		seats[i] = SeatUpdate{
			SeatID: seatID,
			Status: "available",
		}
	}

	msg := &Message{
		Type:      MessageTypeOrderExpired,
		FlightID:  flightID,
		OrderID:   orderID,
		Seats:     seats,
		Message:   "Reservation expired - seats are now available",
		Timestamp: time.Now().UnixMilli(),
	}
	h.broadcast <- msg
}

// NotifySeatConflict sends a conflict notification to a specific order's client
func (h *Hub) NotifySeatConflict(flightID string, orderID string, conflictingSeatIDs []string) {
	seats := make([]SeatUpdate, len(conflictingSeatIDs))
	for i, seatID := range conflictingSeatIDs {
		seats[i] = SeatUpdate{
			SeatID: seatID,
			Status: "held",
		}
	}

	msg := &Message{
		Type:      MessageTypeSeatConflict,
		FlightID:  flightID,
		OrderID:   orderID,
		Seats:     seats,
		Message:   "Some seats you selected are no longer available",
		Timestamp: time.Now().UnixMilli(),
	}
	h.broadcast <- msg
}

// GetClientCount returns the number of clients watching a flight
func (h *Hub) GetClientCount(flightID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[flightID])
}
