package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// MessageType defines the type of WebSocket message
type MessageType string

const (
	MessageTypeSeatsUpdated   MessageType = "seats_updated"
	MessageTypeSeatConflict   MessageType = "seat_conflict"
	MessageTypeOrderCompleted MessageType = "order_completed"
	MessageTypeOrderExpired   MessageType = "order_expired"
	MessageTypeSeatsReleased  MessageType = "seats_released"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType `json:"type"`
	FlightID  string      `json:"flightId"`
	SeatIDs   []string    `json:"seatIds,omitempty"`
	OrderID   string      `json:"orderId,omitempty"`
	Status    string      `json:"status,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// Client represents a WebSocket client
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	flightID uuid.UUID
	orderID  *uuid.UUID
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	// Clients grouped by flight ID
	clients map[uuid.UUID]map[*Client]bool
	
	// Register requests
	register chan *Client
	
	// Unregister requests
	unregister chan *Client
	
	// Broadcast messages to a flight
	broadcast chan *Message
	
	mu sync.RWMutex
}

var (
	globalHub *Hub
	once      sync.Once
)

// GetHub returns the singleton hub instance
func GetHub() *Hub {
	once.Do(func() {
		globalHub = &Hub{
			clients:    make(map[uuid.UUID]map[*Client]bool),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			broadcast:  make(chan *Message, 256),
		}
		go globalHub.Run()
	})
	return globalHub
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
			count := len(h.clients[client.flightID])
			h.mu.Unlock()
			log.Printf("WebSocket: Client registered for flight %s (total: %d)", client.flightID, count)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.flightID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.flightID)
					}
				}
			}
			count := len(h.clients[client.flightID])
			h.mu.Unlock()
			log.Printf("WebSocket: Client unregistered from flight %s (remaining: %d)", client.flightID, count)

		case message := <-h.broadcast:
			h.broadcastToFlight(message)
		}
	}
}

func (h *Hub) broadcastToFlight(message *Message) {
	flightID, err := uuid.Parse(message.FlightID)
	if err != nil {
		log.Printf("WebSocket: Invalid flight ID in broadcast: %s", message.FlightID)
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("WebSocket: Failed to marshal message: %v", err)
		return
	}

	h.mu.RLock()
	clients := h.clients[flightID]
	h.mu.RUnlock()

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

// BroadcastSeatsHeld broadcasts that seats have been held
func (h *Hub) BroadcastSeatsHeld(flightID string, seatIDs []string, orderID string) {
	h.broadcast <- &Message{
		Type:      MessageTypeSeatsUpdated,
		FlightID:  flightID,
		SeatIDs:   seatIDs,
		OrderID:   orderID,
		Status:    "held",
		Timestamp: time.Now().UnixMilli(),
	}
}

// BroadcastSeatsReleased broadcasts that seats have been released
func (h *Hub) BroadcastSeatsReleased(flightID string, seatIDs []string, orderID string) {
	h.broadcast <- &Message{
		Type:      MessageTypeSeatsReleased,
		FlightID:  flightID,
		SeatIDs:   seatIDs,
		OrderID:   orderID,
		Status:    "available",
		Timestamp: time.Now().UnixMilli(),
	}
}

// BroadcastOrderCompleted broadcasts that an order has been completed
func (h *Hub) BroadcastOrderCompleted(flightID string, seatIDs []string, orderID string) {
	h.broadcast <- &Message{
		Type:      MessageTypeOrderCompleted,
		FlightID:  flightID,
		SeatIDs:   seatIDs,
		OrderID:   orderID,
		Status:    "booked",
		Timestamp: time.Now().UnixMilli(),
	}
}

// BroadcastOrderExpired broadcasts that an order has expired
func (h *Hub) BroadcastOrderExpired(flightID string, seatIDs []string, orderID string) {
	h.broadcast <- &Message{
		Type:      MessageTypeOrderExpired,
		FlightID:  flightID,
		SeatIDs:   seatIDs,
		OrderID:   orderID,
		Status:    "available",
		Timestamp: time.Now().UnixMilli(),
	}
}

// NotifySeatConflict notifies a specific client about a seat conflict
func (h *Hub) NotifySeatConflict(flightID string, seatIDs []string, orderID string) {
	h.broadcast <- &Message{
		Type:      MessageTypeSeatConflict,
		FlightID:  flightID,
		SeatIDs:   seatIDs,
		OrderID:   orderID,
		Timestamp: time.Now().UnixMilli(),
	}
}

