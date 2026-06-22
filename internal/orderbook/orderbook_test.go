package orderbook_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

func newOrder(side models.Side, price, qty float64) *models.Order {
	return &models.Order{
		ID:          uuid.New(),
		TradingPair: "BTC-USD",
		Side:        side,
		Type:        models.OrderTypeLimit,
		Price:       price,
		Quantity:    qty,
		Status:      models.OrderStatusOpen,
		UserID:      uuid.New(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

func addOrder(t *testing.T, ob *orderbook.OrderBook, order *models.Order) {
	t.Helper()
	if err := ob.Add(order); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestNewOrderBook(t *testing.T) {
	ob := orderbook.New("BTC-USD")
	if ob.TradingPair != "BTC-USD" {
		t.Fatalf("expected BTC-USD, got %s", ob.TradingPair)
	}
	if ob.BestBid() != 0 {
		t.Fatal("expected empty best bid")
	}
	if ob.BestAsk() != 0 {
		t.Fatal("expected empty best ask")
	}
}

func TestAddBidOrder(t *testing.T) {
	ob := orderbook.New("BTC-USD")
	order := newOrder(models.SideBuy, 50000.0, 1.0)

	if err := ob.Add(order); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ob.BestBid() != 50000.0 {
		t.Fatalf("expected best bid 50000, got %f", ob.BestBid())
	}
}

func TestAddAskOrder(t *testing.T) {
	ob := orderbook.New("BTC-USD")
	order := newOrder(models.SideSell, 51000.0, 1.0)

	if err := ob.Add(order); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ob.BestAsk() != 51000.0 {
		t.Fatalf("expected best ask 51000, got %f", ob.BestAsk())
	}
}

func TestBidsOrderedDescending(t *testing.T) {
	ob := orderbook.New("BTC-USD")

	// Add bids out of order
	addOrder(t, ob, newOrder(models.SideBuy, 49000.0, 1.0))
	addOrder(t, ob, newOrder(models.SideBuy, 51000.0, 1.0))
	addOrder(t, ob, newOrder(models.SideBuy, 50000.0, 1.0))

	// Best bid should be 51000 (highest)
	if ob.BestBid() != 51000.0 {
		t.Fatalf("expected best bid 51000, got %f", ob.BestBid())
	}
}

func TestAsksOrderedAscending(t *testing.T) {
	ob := orderbook.New("BTC-USD")

	// Add asks out of order
	addOrder(t, ob, newOrder(models.SideSell, 53000.0, 1.0))
	addOrder(t, ob, newOrder(models.SideSell, 51000.0, 1.0))
	addOrder(t, ob, newOrder(models.SideSell, 52000.0, 1.0))

	// Best ask should be 51000 (lowest)
	if ob.BestAsk() != 51000.0 {
		t.Fatalf("expected best ask 51000, got %f", ob.BestAsk())
	}
}

func TestSpread(t *testing.T) {
	ob := orderbook.New("BTC-USD")
	addOrder(t, ob, newOrder(models.SideBuy, 50000.0, 1.0))
	addOrder(t, ob, newOrder(models.SideSell, 51000.0, 1.0))

	spread := ob.Spread()
	if spread != 1000.0 {
		t.Fatalf("expected spread 1000, got %f", spread)
	}
}

func TestCancelOrder(t *testing.T) {
	ob := orderbook.New("BTC-USD")
	order := newOrder(models.SideBuy, 50000.0, 1.0)
	addOrder(t, ob, order)

	cancelled, err := ob.Cancel(order.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cancelled.Status != models.OrderStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", cancelled.Status)
	}
	// Best bid should now be empty
	if ob.BestBid() != 0 {
		t.Fatalf("expected empty book after cancel, got %f", ob.BestBid())
	}
}

func TestCancelNonExistentOrder(t *testing.T) {
	ob := orderbook.New("BTC-USD")
	_, err := ob.Cancel(uuid.New())
	if err == nil {
		t.Fatal("expected error cancelling non-existent order")
	}
}

func TestDuplicateOrder(t *testing.T) {
	ob := orderbook.New("BTC-USD")
	order := newOrder(models.SideBuy, 50000.0, 1.0)
	addOrder(t, ob, order)

	err := ob.Add(order)
	if err == nil {
		t.Fatal("expected error adding duplicate order")
	}
}

func TestInvalidPrice(t *testing.T) {
	ob := orderbook.New("BTC-USD")
	order := newOrder(models.SideBuy, -100.0, 1.0)
	if err := ob.Add(order); err == nil {
		t.Fatal("expected error for negative price")
	}
}

func TestSnapshot(t *testing.T) {
	ob := orderbook.New("BTC-USD")
	addOrder(t, ob, newOrder(models.SideBuy, 50000.0, 2.0))
	addOrder(t, ob, newOrder(models.SideBuy, 49000.0, 1.0))
	addOrder(t, ob, newOrder(models.SideSell, 51000.0, 1.5))

	snap := ob.Snapshot(10)
	if snap.TradingPair != "BTC-USD" {
		t.Fatalf("wrong trading pair: %s", snap.TradingPair)
	}
	if len(snap.Bids) != 2 {
		t.Fatalf("expected 2 bid levels, got %d", len(snap.Bids))
	}
	if len(snap.Asks) != 1 {
		t.Fatalf("expected 1 ask level, got %d", len(snap.Asks))
	}
	// Best bid first
	if snap.Bids[0].Price != 50000.0 {
		t.Fatalf("expected first bid 50000, got %f", snap.Bids[0].Price)
	}
}

func TestMultipleOrdersSamePrice(t *testing.T) {
	ob := orderbook.New("BTC-USD")
	addOrder(t, ob, newOrder(models.SideBuy, 50000.0, 1.0))
	addOrder(t, ob, newOrder(models.SideBuy, 50000.0, 2.0))
	addOrder(t, ob, newOrder(models.SideBuy, 50000.0, 0.5))

	snap := ob.Snapshot(10)
	if len(snap.Bids) != 1 {
		t.Fatalf("expected 1 price level, got %d", len(snap.Bids))
	}
	if snap.Bids[0].Quantity != 3.5 {
		t.Fatalf("expected total qty 3.5, got %f", snap.Bids[0].Quantity)
	}
	if snap.Bids[0].Orders != 3 {
		t.Fatalf("expected 3 orders, got %d", snap.Bids[0].Orders)
	}
}
