package matching_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/matching"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
)

// ─── Helpers ─────────────────────────────────────────────────────────────────

func newLimitOrder(side models.Side, price, qty float64) *models.Order {
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

func newMarketOrder(side models.Side, qty float64) *models.Order {
	return &models.Order{
		ID:          uuid.New(),
		TradingPair: "BTC-USD",
		Side:        side,
		Type:        models.OrderTypeMarket,
		Quantity:    qty,
		Status:      models.OrderStatusOpen,
		UserID:      uuid.New(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestNoMatchEmptyBook(t *testing.T) {
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)

	order := newLimitOrder(models.SideBuy, 50000.0, 1.0)
	result, err := engine.Submit(order)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Trades) != 0 {
		t.Fatalf("expected no trades, got %d", len(result.Trades))
	}
	if result.Order.Status != models.OrderStatusOpen {
		t.Fatalf("expected open status, got %s", result.Order.Status)
	}
}

func TestFullMatch(t *testing.T) {
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)

	// Resting sell order
	sell := newLimitOrder(models.SideSell, 50000.0, 1.0)
	engine.Submit(sell)

	// Incoming buy that crosses
	buy := newLimitOrder(models.SideBuy, 50000.0, 1.0)
	result, err := engine.Submit(buy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	trade := result.Trades[0]
	if trade.Price != 50000.0 {
		t.Fatalf("expected trade price 50000, got %f", trade.Price)
	}
	if trade.Quantity != 1.0 {
		t.Fatalf("expected trade qty 1.0, got %f", trade.Quantity)
	}
	if result.Order.Status != models.OrderStatusFilled {
		t.Fatalf("expected filled status, got %s", result.Order.Status)
	}

	// Book should now be empty
	if book.BestBid() != 0 || book.BestAsk() != 0 {
		t.Fatal("expected empty book after full match")
	}
}

func TestPartialMatch(t *testing.T) {
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)

	// Resting sell: 2.0 BTC
	sell := newLimitOrder(models.SideSell, 50000.0, 2.0)
	engine.Submit(sell)

	// Incoming buy: only 0.5 BTC
	buy := newLimitOrder(models.SideBuy, 50000.0, 0.5)
	result, err := engine.Submit(buy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	if result.Trades[0].Quantity != 0.5 {
		t.Fatalf("expected fill qty 0.5, got %f", result.Trades[0].Quantity)
	}
	if result.Order.Status != models.OrderStatusFilled {
		t.Fatalf("expected incoming order fully filled, got %s", result.Order.Status)
	}

	// Resting sell should have 1.5 remaining
	if book.BestAsk() != 50000.0 {
		t.Fatalf("expected resting ask still at 50000, got %f", book.BestAsk())
	}
}

func TestIncomingOrderPartiallyFilledThenBooked(t *testing.T) {
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)

	// Resting sell: only 0.5 BTC available
	sell := newLimitOrder(models.SideSell, 50000.0, 0.5)
	engine.Submit(sell)

	// Incoming buy wants 2.0 BTC — only 0.5 can fill
	buy := newLimitOrder(models.SideBuy, 50000.0, 2.0)
	result, err := engine.Submit(buy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	if result.Order.Status != models.OrderStatusPartial {
		t.Fatalf("expected partial status, got %s", result.Order.Status)
	}
	if result.Order.Remaining() != 1.5 {
		t.Fatalf("expected 1.5 remaining, got %f", result.Order.Remaining())
	}

	// The remainder should now be resting in the book as a bid
	if book.BestBid() != 50000.0 {
		t.Fatalf("expected remainder booked at 50000, got %f", book.BestBid())
	}
}

func TestNoMatchWhenPricesDontCross(t *testing.T) {
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)

	// Resting sell at 51000
	sell := newLimitOrder(models.SideSell, 51000.0, 1.0)
	engine.Submit(sell)

	// Incoming buy at 50000 — doesn't cross
	buy := newLimitOrder(models.SideBuy, 50000.0, 1.0)
	result, err := engine.Submit(buy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Trades) != 0 {
		t.Fatalf("expected no trades, got %d", len(result.Trades))
	}
	// Both orders should now be resting
	if book.BestBid() != 50000.0 {
		t.Fatalf("expected bid 50000, got %f", book.BestBid())
	}
	if book.BestAsk() != 51000.0 {
		t.Fatalf("expected ask 51000, got %f", book.BestAsk())
	}
}

func TestFIFOPriceTimePriority(t *testing.T) {
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)

	// Two resting sells at the same price, added in order
	sell1 := newLimitOrder(models.SideSell, 50000.0, 1.0)
	engine.Submit(sell1)
	sell2 := newLimitOrder(models.SideSell, 50000.0, 1.0)
	engine.Submit(sell2)

	// Incoming buy for 1.0 should match sell1 first (FIFO)
	buy := newLimitOrder(models.SideBuy, 50000.0, 1.0)
	result, _ := engine.Submit(buy)

	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	if result.Trades[0].SellOrderID != sell1.ID {
		t.Fatal("expected FIFO match against the first resting order")
	}
}

func TestTradePriceIsRestingOrderPrice(t *testing.T) {
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)

	// Resting sell at 49000 (better than buyer's limit)
	sell := newLimitOrder(models.SideSell, 49000.0, 1.0)
	engine.Submit(sell)

	// Incoming buy willing to pay up to 50000
	buy := newLimitOrder(models.SideBuy, 50000.0, 1.0)
	result, _ := engine.Submit(buy)

	// Trade should execute at the resting price (49000), not 50000
	if result.Trades[0].Price != 49000.0 {
		t.Fatalf("expected trade at resting price 49000, got %f", result.Trades[0].Price)
	}
}

func TestMarketOrderMatchesAnyPrice(t *testing.T) {
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)

	sell := newLimitOrder(models.SideSell, 52000.0, 1.0)
	engine.Submit(sell)

	marketBuy := newMarketOrder(models.SideBuy, 1.0)
	result, err := engine.Submit(marketBuy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Trades) != 1 {
		t.Fatalf("expected 1 trade, got %d", len(result.Trades))
	}
	if result.Trades[0].Price != 52000.0 {
		t.Fatalf("expected fill at 52000, got %f", result.Trades[0].Price)
	}
}

func TestMarketOrderNotBookedIfUnfilled(t *testing.T) {
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)

	// No resting liquidity at all
	marketBuy := newMarketOrder(models.SideBuy, 1.0)
	result, err := engine.Submit(marketBuy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Trades) != 0 {
		t.Fatalf("expected no trades, got %d", len(result.Trades))
	}
	// Market order should NOT be resting in the book
	if book.BestBid() != 0 {
		t.Fatalf("market order should not be booked, but best bid is %f", book.BestBid())
	}
}

func TestMultiLevelSweep(t *testing.T) {
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)

	// Three sell levels
	engine.Submit(newLimitOrder(models.SideSell, 50000.0, 1.0))
	engine.Submit(newLimitOrder(models.SideSell, 50500.0, 1.0))
	engine.Submit(newLimitOrder(models.SideSell, 51000.0, 1.0))

	// Large buy that sweeps through all three levels
	buy := newLimitOrder(models.SideBuy, 51000.0, 3.0)
	result, err := engine.Submit(buy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Trades) != 3 {
		t.Fatalf("expected 3 trades sweeping all levels, got %d", len(result.Trades))
	}
	if result.Order.Status != models.OrderStatusFilled {
		t.Fatalf("expected fully filled, got %s", result.Order.Status)
	}
	if book.BestAsk() != 0 {
		t.Fatalf("expected book swept clean, best ask = %f", book.BestAsk())
	}
}
