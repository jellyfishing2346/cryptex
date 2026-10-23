// Package demo keeps a public demo exchange lively: a market maker quotes
// both sides of the book around a drifting price, a taker occasionally
// trades against it, and stale visitor orders are expired.
package demo

import (
	"context"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/matching"
	"github.com/jellyfishing2346/cryptex/internal/models"
)

// Exchange is the subset of the API server the bots use. Bots go through
// the same path as HTTP clients, so their orders are risk-checked,
// persisted and broadcast like anyone else's.
type Exchange interface {
	PlaceOrder(ctx context.Context, pair string, side models.Side, orderType models.OrderType, price, quantity float64, userID uuid.UUID) (*matching.Result, error)
	CancelOrder(ctx context.Context, orderID, userID uuid.UUID) (*models.Order, error)
	RestingOrders() []*models.Order
}

// Fixed IDs so the bots are recognisable across restarts.
var (
	MakerID = uuid.MustParse("00000000-0000-4000-8000-00000000a001")
	TakerID = uuid.MustParse("00000000-0000-4000-8000-00000000a002")
)

// Config tunes the bots.
type Config struct {
	TradingPair      string
	StartPrice       float64       // centre the price drifts around
	Levels           int           // quotes per side
	LevelSpacing     float64       // gap between levels, as a fraction of price
	HalfSpread       float64       // distance from mid to the best quote, as a fraction
	QuoteSize        float64       // size of the quotes nearest the mid; further levels are larger
	Volatility       float64       // std dev of the per-tick log-return of the mid price
	MeanReversion    float64       // fraction of the gap to StartPrice closed each tick
	TakerProbability float64       // chance per tick that the taker sends a market order
	TakerMaxSize     float64       // largest taker order
	UserOrderTTL     time.Duration // visitor orders older than this are cancelled; 0 keeps them
	Interval         time.Duration // time between ticks
}

// DefaultConfig returns settings that produce a lively but stable book.
func DefaultConfig(pair string) Config {
	return Config{
		TradingPair:      pair,
		StartPrice:       65000,
		Levels:           8,
		LevelSpacing:     0.0004,
		HalfSpread:       0.0002,
		QuoteSize:        0.25,
		Volatility:       0.0006,
		MeanReversion:    0.002,
		TakerProbability: 0.35,
		TakerMaxSize:     0.4,
		UserOrderTTL:     30 * time.Minute,
		Interval:         2 * time.Second,
	}
}

// MarketMaker runs the demo bots.
type MarketMaker struct {
	ex  Exchange
	cfg Config
	rng *rand.Rand
	mid float64
	now func() time.Time
}

// NewMarketMaker creates the bots. seed makes runs reproducible in tests.
func NewMarketMaker(ex Exchange, cfg Config, seed int64) *MarketMaker {
	return &MarketMaker{
		ex:  ex,
		cfg: cfg,
		rng: rand.New(rand.NewSource(seed)),
		mid: cfg.StartPrice,
		now: time.Now,
	}
}

// Mid returns the current reference price.
func (m *MarketMaker) Mid() float64 { return m.mid }

// Run ticks until ctx is cancelled.
func (m *MarketMaker) Run(ctx context.Context) {
	ticker := time.NewTicker(m.cfg.Interval)
	defer ticker.Stop()
	m.Tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.Tick(ctx)
		}
	}
}

// Tick moves the price, refreshes quotes, maybe trades, and expires stale
// visitor orders. Errors are logged, not fatal: the next tick retries.
func (m *MarketMaker) Tick(ctx context.Context) {
	m.movePrice()
	m.requote(ctx)
	m.maybeTake(ctx)
	m.expireVisitorOrders(ctx)
}

func (m *MarketMaker) movePrice() {
	shock := m.rng.NormFloat64() * m.cfg.Volatility
	m.mid *= math.Exp(shock)
	m.mid += (m.cfg.StartPrice - m.mid) * m.cfg.MeanReversion
}

// desiredQuotes returns the ladder of prices the maker wants on each side.
func (m *MarketMaker) desiredQuotes() (bids, asks map[float64]bool) {
	bids, asks = map[float64]bool{}, map[float64]bool{}
	for i := 0; i < m.cfg.Levels; i++ {
		offset := m.cfg.HalfSpread + float64(i)*m.cfg.LevelSpacing
		bids[roundPrice(m.mid*(1-offset))] = true
		asks[roundPrice(m.mid*(1+offset))] = true
	}
	return bids, asks
}

// requote cancels maker orders that are no longer on the ladder and places
// the missing levels, so a small price move only touches a few orders.
func (m *MarketMaker) requote(ctx context.Context) {
	bids, asks := m.desiredQuotes()
	for _, o := range m.ex.RestingOrders() {
		if o.UserID != MakerID {
			continue
		}
		want := asks
		if o.Side == models.SideBuy {
			want = bids
		}
		if want[o.Price] {
			delete(want, o.Price) // already quoted
			continue
		}
		if _, err := m.ex.CancelOrder(ctx, o.ID, MakerID); err != nil {
			log.Printf("demo: cancel stale quote: %v", err)
		}
	}
	m.place(ctx, models.SideBuy, bids)
	m.place(ctx, models.SideSell, asks)
}

// quoteSize grows with distance from the mid, with some noise, so the book
// looks like real liquidity rather than identical orders.
func (m *MarketMaker) quoteSize(price float64) float64 {
	levelsOut := math.Abs(price/m.mid-1) / math.Max(m.cfg.LevelSpacing, 1e-9)
	return roundSize(m.cfg.QuoteSize * (1 + 0.35*levelsOut) * (0.75 + 0.5*m.rng.Float64()))
}

func (m *MarketMaker) place(ctx context.Context, side models.Side, prices map[float64]bool) {
	for price := range prices {
		_, err := m.ex.PlaceOrder(ctx, m.cfg.TradingPair, side, models.OrderTypeLimit, price, m.quoteSize(price), MakerID)
		if err != nil {
			log.Printf("demo: place %s quote at %.2f: %v", side, price, err)
		}
	}
}

func (m *MarketMaker) maybeTake(ctx context.Context) {
	if m.rng.Float64() >= m.cfg.TakerProbability {
		return
	}
	side := models.SideBuy
	if m.rng.Intn(2) == 0 {
		side = models.SideSell
	}
	size := roundSize(0.01 + m.rng.Float64()*(m.cfg.TakerMaxSize-0.01))
	if _, err := m.ex.PlaceOrder(ctx, m.cfg.TradingPair, side, models.OrderTypeMarket, 0, size, TakerID); err != nil {
		log.Printf("demo: taker %s %.4f: %v", side, size, err)
	}
}

// expireVisitorOrders keeps the public book tidy by cancelling orders
// visitors left behind.
func (m *MarketMaker) expireVisitorOrders(ctx context.Context) {
	if m.cfg.UserOrderTTL <= 0 {
		return
	}
	cutoff := m.now().Add(-m.cfg.UserOrderTTL)
	for _, o := range m.ex.RestingOrders() {
		if o.UserID == MakerID || o.UserID == TakerID || !o.CreatedAt.Before(cutoff) {
			continue
		}
		if _, err := m.ex.CancelOrder(ctx, o.ID, o.UserID); err != nil {
			log.Printf("demo: expire visitor order %s: %v", o.ID, err)
		}
	}
}

func roundPrice(p float64) float64 { return math.Round(p*100) / 100 }
func roundSize(q float64) float64  { return math.Round(q*10000) / 10000 }
