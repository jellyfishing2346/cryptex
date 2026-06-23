package nats

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublisher_New(t *testing.T) {
	// Test with invalid URL
	_, err := New("invalid://url")
	assert.Error(t, err)
}

func TestPublisher_PublishTrade(t *testing.T) {
	// Skip if NATS is not available
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skip("NATS server not available, skipping integration test")
		return
	}
	defer nc.Close()

	publisher, err := New(nats.DefaultURL)
	require.NoError(t, err)
	defer publisher.Close()

	// Subscribe to verify the message was published
	sub, err := nc.Subscribe("trades.BTC-USD", func(msg *nats.Msg) {
		var trade models.Trade
		err := json.Unmarshal(msg.Data, &trade)
		assert.NoError(t, err)
		assert.Equal(t, "BTC-USD", trade.TradingPair)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Create a test trade
	trade := &models.Trade{
		TradingPair: "BTC-USD",
		Price:       50000.0,
		Quantity:    1.5,
		ExecutedAt:  time.Now().UTC(),
	}

	// Publish the trade
	err = publisher.PublishTrade(trade)
	assert.NoError(t, err)

	// Wait a bit for the message to be processed
	time.Sleep(100 * time.Millisecond)
}

func TestPublisher_PublishOrder(t *testing.T) {
	// Skip if NATS is not available
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skip("NATS server not available, skipping integration test")
		return
	}
	defer nc.Close()

	publisher, err := New(nats.DefaultURL)
	require.NoError(t, err)
	defer publisher.Close()

	sub, err := nc.Subscribe("orders.BTC-USD", func(msg *nats.Msg) {
		var order models.Order
		err := json.Unmarshal(msg.Data, &order)
		assert.NoError(t, err)
		assert.Equal(t, "BTC-USD", order.TradingPair)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	order := &models.Order{
		TradingPair: "BTC-USD",
		Price:       50000.0,
		Quantity:    1.5,
	}

	err = publisher.PublishOrder(order)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
}

func TestPublisher_PublishTradeEvent(t *testing.T) {
	// Skip if NATS is not available
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Skip("NATS server not available, skipping integration test")
		return
	}
	defer nc.Close()

	publisher, err := New(nats.DefaultURL)
	require.NoError(t, err)
	defer publisher.Close()

	sub, err := nc.Subscribe("trade-events.BTC-USD", func(msg *nats.Msg) {
		var event TradeEvent
		err := json.Unmarshal(msg.Data, &event)
		assert.NoError(t, err)
		assert.Equal(t, "trade.executed", event.EventType)
		assert.Equal(t, "BTC-USD", event.Trade.TradingPair)
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	event := &TradeEvent{
		Trade: &models.Trade{
			TradingPair: "BTC-USD",
			Price:       50000.0,
			Quantity:    1.5,
			ExecutedAt:  time.Now().UTC(),
		},
		Timestamp: time.Now().UTC(),
		EventType: "trade.executed",
	}

	err = publisher.PublishTradeEvent(event)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
}

func TestPublisher_Close(t *testing.T) {
	publisher, err := New(nats.DefaultURL)
	if err != nil {
		t.Skip("NATS server not available, skipping integration test")
		return
	}

	err = publisher.Close()
	assert.NoError(t, err)

	// Closing again should not error
	err = publisher.Close()
	assert.NoError(t, err)
}
