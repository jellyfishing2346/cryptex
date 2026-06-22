// Package ws implements the WebSocket hub that fans out real-time
// order book and trade events to all connected clients.
package ws

import (
	"encoding/json"
	"log"
	"sync"
)

// EventType identifies what kind of event is being broadcast.
type EventType string

const (
	EventTypeTrade     EventType = "trade"
	EventTypeOrderBook EventType = "orderbook"
)

// Event is the envelope written to every connected client.
type Event struct {
	Type    EventType `json:"type"`
	Payload any       `json:"payload"`
}

// client represents a single WebSocket connection.
type client struct {
	send chan []byte
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

// NewHub creates a ready-to-use Hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[*client]struct{}),
	}
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
}

// Broadcast serialises payload as an Event and sends it to every client.
// Slow or disconnected clients are dropped without blocking the caller.
func (h *Hub) Broadcast(eventType EventType, payload any) {
	event := Event{Type: eventType, Payload: payload}
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("ws: marshal event %s: %v", eventType, err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for c := range h.clients {
		select {
		case c.send <- data:
		default:
			log.Printf("ws: dropped message for slow client")
		}
	}
}

// ClientCount returns the number of currently connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
