package demo

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/api"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
	"github.com/jellyfishing2346/cryptex/internal/risk"
)

func newExchange(t *testing.T) (*api.Server, *orderbook.OrderBook) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	book := orderbook.New("BTC-USD")
	riskCfg := risk.DefaultConfig()
	return api.NewServerWithRisk(book, nil, &riskCfg), book
}

func quiet(cfg Config) Config {
	cfg.TakerProbability = 0
	return cfg
}

func TestFirstTickQuotesBothSidesWithoutCrossing(t *testing.T) {
	server, book := newExchange(t)
	cfg := quiet(DefaultConfig("BTC-USD"))
	mm := NewMarketMaker(server, cfg, 1)
	mm.Tick(context.Background())

	snap := book.Snapshot(100)
	if len(snap.Bids) != cfg.Levels || len(snap.Asks) != cfg.Levels {
		t.Fatalf("expected %d levels per side, got %d bids, %d asks", cfg.Levels, len(snap.Bids), len(snap.Asks))
	}
	if snap.Bids[0].Price >= snap.Asks[0].Price {
		t.Fatalf("book is crossed: best bid %.2f >= best ask %.2f", snap.Bids[0].Price, snap.Asks[0].Price)
	}
	for _, o := range server.RestingOrders() {
		if o.UserID != MakerID {
			t.Fatalf("unexpected order owner %s", o.UserID)
		}
	}
}

func TestRequoteOnlyReplacesLevelsThatMoved(t *testing.T) {
	server, _ := newExchange(t)
	cfg := quiet(DefaultConfig("BTC-USD"))
	cfg.Volatility, cfg.MeanReversion = 0, 0 // price stays put
	mm := NewMarketMaker(server, cfg, 1)
	ctx := context.Background()

	mm.Tick(ctx)
	before := map[uuid.UUID]bool{}
	for _, o := range server.RestingOrders() {
		before[o.ID] = true
	}
	mm.Tick(ctx)
	for _, o := range server.RestingOrders() {
		if !before[o.ID] {
			t.Fatal("unchanged price should keep every existing quote")
		}
	}
	if len(server.RestingOrders()) != 2*cfg.Levels {
		t.Fatalf("expected %d quotes, got %d", 2*cfg.Levels, len(server.RestingOrders()))
	}
}

func TestTakerProducesTrades(t *testing.T) {
	server, _ := newExchange(t)
	cfg := DefaultConfig("BTC-USD")
	cfg.TakerProbability = 1
	mm := NewMarketMaker(server, cfg, 7)
	for i := 0; i < 20; i++ {
		mm.Tick(context.Background())
	}
	if server.LastTradePrice() == 0 {
		t.Fatal("expected the taker to have traded")
	}
	if p := server.LastTradePrice(); p < cfg.StartPrice*0.9 || p > cfg.StartPrice*1.1 {
		t.Fatalf("trade price %.2f far from the market", p)
	}
}

func TestPriceStaysNearStartOverLongRuns(t *testing.T) {
	server, _ := newExchange(t)
	cfg := quiet(DefaultConfig("BTC-USD"))
	mm := NewMarketMaker(server, cfg, 42)
	for i := 0; i < 5000; i++ {
		mm.movePrice()
	}
	if mid := mm.Mid(); mid < cfg.StartPrice*0.8 || mid > cfg.StartPrice*1.2 {
		t.Fatalf("mean reversion should keep the price near %.0f, got %.2f", cfg.StartPrice, mid)
	}
}

func TestStaleVisitorOrdersExpireButBotsAndFreshOrdersStay(t *testing.T) {
	server, _ := newExchange(t)
	cfg := quiet(DefaultConfig("BTC-USD"))
	cfg.UserOrderTTL = 30 * time.Minute
	mm := NewMarketMaker(server, cfg, 1)
	ctx := context.Background()

	visitor := uuid.New()
	res, err := server.PlaceOrder(ctx, "BTC-USD", models.SideBuy, models.OrderTypeLimit, 1000, 1, visitor)
	if err != nil {
		t.Fatal(err)
	}
	staleID := res.Order.ID

	mm.Tick(ctx) // order is fresh: kept
	if len(server.OrdersFor(visitor)) != 1 {
		t.Fatal("fresh visitor order should be kept")
	}

	mm.now = func() time.Time { return time.Now().Add(31 * time.Minute) }
	mm.Tick(ctx)

	for _, o := range server.OrdersFor(visitor) {
		if o.ID == staleID {
			t.Fatal("stale visitor order should be cancelled")
		}
	}
	makerQuotes := 0
	for _, o := range server.RestingOrders() {
		if o.UserID == MakerID {
			makerQuotes++
		}
	}
	if makerQuotes != 2*cfg.Levels {
		t.Fatalf("maker quotes must survive expiry, got %d", makerQuotes)
	}
}

var _ Exchange = (*api.Server)(nil)
