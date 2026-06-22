package main

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
)

func TestRestoreOrders(t *testing.T) {
	book := orderbook.New("BTC-USD")
	order := persistedOrder("BTC-USD")

	if err := restoreOrders(book, []*models.Order{order}); err != nil {
		t.Fatalf("restore orders: %v", err)
	}

	snapshot := book.Snapshot(10)
	if len(snapshot.Bids) != 1 {
		t.Fatalf("expected 1 bid level, got %d", len(snapshot.Bids))
	}
	if snapshot.Bids[0].Quantity != 1 {
		t.Fatalf("expected restored quantity 1, got %f", snapshot.Bids[0].Quantity)
	}
}

func TestRestoreOrdersRejectsWrongTradingPair(t *testing.T) {
	book := orderbook.New("BTC-USD")
	order := persistedOrder("ETH-USD")

	if err := restoreOrders(book, []*models.Order{order}); err == nil {
		t.Fatal("expected error for wrong trading pair")
	}
}

func persistedOrder(tradingPair string) *models.Order {
	now := time.Now().UTC()
	return &models.Order{
		ID:          uuid.New(),
		TradingPair: tradingPair,
		Side:        models.SideBuy,
		Type:        models.OrderTypeLimit,
		Price:       50000,
		Quantity:    1,
		Status:      models.OrderStatusOpen,
		UserID:      uuid.New(),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
