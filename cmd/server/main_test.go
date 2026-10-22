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

func TestRedisOptionsFromURL(t *testing.T) {
	t.Setenv("REDIS_URL", "rediss://default:secret@example.upstash.io:6380/2")
	opts, err := redisOptions()
	if err != nil {
		t.Fatal(err)
	}
	if opts.Addr != "example.upstash.io:6380" || opts.Password != "secret" || opts.DB != 2 || opts.TLSConfig == nil {
		t.Fatalf("unexpected options: addr=%s db=%d tls=%v", opts.Addr, opts.DB, opts.TLSConfig != nil)
	}

	t.Setenv("REDIS_URL", "")
	t.Setenv("REDIS_ADDR", "cache:6379")
	opts, err = redisOptions()
	if err != nil || opts.Addr != "cache:6379" || opts.TLSConfig != nil {
		t.Fatalf("fallback to REDIS_ADDR failed: %+v %v", opts, err)
	}

	t.Setenv("REDIS_URL", "not a url")
	if _, err := redisOptions(); err == nil {
		t.Fatal("expected an error for an invalid REDIS_URL")
	}
}

func TestSplitList(t *testing.T) {
	got := splitList(" https://a.app , ,https://b.app ")
	if len(got) != 2 || got[0] != "https://a.app" || got[1] != "https://b.app" {
		t.Fatalf("got %v", got)
	}
	if splitList("") != nil {
		t.Fatal("empty input should give no origins")
	}
}
