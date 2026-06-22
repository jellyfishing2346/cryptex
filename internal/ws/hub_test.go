package ws

import (
	"encoding/json"
	"testing"
)

func TestBroadcastDeliveredToClient(t *testing.T) {
	hub := NewHub()
	cl := &client{send: make(chan []byte, 8)}
	hub.register(cl)

	hub.Broadcast(EventTypeTrade, map[string]any{"price": 50000.0, "quantity": 1.5})

	select {
	case msg := <-cl.send:
		var event Event
		if err := json.Unmarshal(msg, &event); err != nil {
			t.Fatalf("unmarshal event: %v", err)
		}
		if event.Type != EventTypeTrade {
			t.Fatalf("want type %s, got %s", EventTypeTrade, event.Type)
		}
	default:
		t.Fatal("expected a message in client's send channel")
	}
}

func TestBroadcastToMultipleClients(t *testing.T) {
	hub := NewHub()
	clients := make([]*client, 3)
	for i := range clients {
		clients[i] = &client{send: make(chan []byte, 8)}
		hub.register(clients[i])
	}

	hub.Broadcast(EventTypeOrderBook, map[string]any{"bids": []any{}, "asks": []any{}})

	for i, cl := range clients {
		select {
		case <-cl.send:
		default:
			t.Fatalf("client %d did not receive broadcast", i)
		}
	}
}

func TestUnregisterRemovesClient(t *testing.T) {
	hub := NewHub()
	cl := &client{send: make(chan []byte, 8)}
	hub.register(cl)

	if hub.ClientCount() != 1 {
		t.Fatalf("want 1 client, got %d", hub.ClientCount())
	}

	hub.unregister(cl)

	if hub.ClientCount() != 0 {
		t.Fatalf("want 0 clients after unregister, got %d", hub.ClientCount())
	}
}

func TestBroadcastAfterUnregisterDoesNotPanic(t *testing.T) {
	hub := NewHub()
	cl := &client{send: make(chan []byte, 8)}
	hub.register(cl)
	hub.unregister(cl)

	hub.Broadcast(EventTypeTrade, map[string]any{"price": 1.0})
}

func TestSlowClientMessageDropped(t *testing.T) {
	hub := NewHub()
	cl := &client{send: make(chan []byte, 1)}
	hub.register(cl)

	hub.Broadcast(EventTypeTrade, map[string]any{"price": 1.0})
	hub.Broadcast(EventTypeTrade, map[string]any{"price": 2.0})

	if len(cl.send) != 1 {
		t.Fatalf("want 1 message (second dropped), got %d", len(cl.send))
	}
}

func TestClientCountAccurate(t *testing.T) {
	hub := NewHub()

	if hub.ClientCount() != 0 {
		t.Fatalf("want 0, got %d", hub.ClientCount())
	}

	c1 := &client{send: make(chan []byte, 8)}
	c2 := &client{send: make(chan []byte, 8)}
	hub.register(c1)
	hub.register(c2)

	if hub.ClientCount() != 2 {
		t.Fatalf("want 2, got %d", hub.ClientCount())
	}

	hub.unregister(c1)

	if hub.ClientCount() != 1 {
		t.Fatalf("want 1 after one unregister, got %d", hub.ClientCount())
	}
}
