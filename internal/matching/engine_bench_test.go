package matching_test

import (
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/matching"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
	"github.com/jellyfishing2346/cryptex/internal/risk"
)

var depths = []int{10, 100, 1000, 10000}

func limit(side models.Side, price, qty float64) *models.Order {
	now := time.Now().UTC()
	return &models.Order{
		ID: uuid.New(), TradingPair: "BTC-USD", Side: side, Type: models.OrderTypeLimit,
		Price: price, Quantity: qty, Status: models.OrderStatusOpen, UserID: uuid.New(),
		CreatedAt: now, UpdatedAt: now,
	}
}

func filledBook(b *testing.B, levels int) *orderbook.OrderBook {
	b.Helper()
	book := orderbook.New("BTC-USD")
	for i := 0; i < levels; i++ {
		if err := book.Add(limit(models.SideBuy, 50000-float64(i), 1)); err != nil {
			b.Fatal(err)
		}
		if err := book.Add(limit(models.SideSell, 50001+float64(i), 1)); err != nil {
			b.Fatal(err)
		}
	}
	return book
}

// reportLatency adds p50/p99 metrics for the recorded per-order durations.
func reportLatency(b *testing.B, samples []time.Duration) {
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	at := func(q float64) float64 { return float64(samples[int(q*float64(len(samples)-1))].Nanoseconds()) }
	b.ReportMetric(at(0.50), "p50-ns/match")
	b.ReportMetric(at(0.99), "p99-ns/match")
}

// benchmarkMatch rests a sell order inside the spread, then submits a buy
// that fully matches it, leaving the book as it started. ns/op covers both
// submissions; the p50/p99 metrics time the matching submission alone.
func benchmarkMatch(b *testing.B, newEngine func(*orderbook.OrderBook) *matching.Engine) {
	for _, levels := range depths {
		b.Run(fmt.Sprintf("levels=%d", levels), func(b *testing.B) {
			book := filledBook(b, levels)
			engine := newEngine(book)
			samples := make([]time.Duration, 0, b.N)
			b.ReportAllocs()
			b.ResetTimer()
			start := time.Now()
			for i := 0; i < b.N; i++ {
				if _, err := engine.Submit(limit(models.SideSell, 50000.5, 1)); err != nil {
					b.Fatal(err)
				}
				t0 := time.Now()
				res, err := engine.Submit(limit(models.SideBuy, 50000.5, 1))
				samples = append(samples, time.Since(t0))
				if err != nil || len(res.Trades) != 1 {
					b.Fatalf("expected one trade, got %v (err %v)", res, err)
				}
			}
			elapsed := time.Since(start)
			b.StopTimer()
			b.ReportMetric(float64(2*b.N)/elapsed.Seconds(), "orders/s")
			reportLatency(b, samples)
		})
	}
}

func BenchmarkMatch(b *testing.B) {
	benchmarkMatch(b, matching.New)
}

// BenchmarkMatchWithRiskChecks is the configuration the server runs.
func BenchmarkMatchWithRiskChecks(b *testing.B) {
	benchmarkMatch(b, func(book *orderbook.OrderBook) *matching.Engine {
		cfg := risk.DefaultConfig()
		return matching.NewWithRiskChecks(book, risk.NewWithConfig(&cfg, book))
	})
}

// BenchmarkMarketOrderSweep sends a market buy that consumes ten ask levels,
// then restores them.
func BenchmarkMarketOrderSweep(b *testing.B) {
	for _, levels := range depths {
		b.Run(fmt.Sprintf("levels=%d", levels), func(b *testing.B) {
			book := filledBook(b, levels)
			engine := matching.New(book)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				market := limit(models.SideBuy, 0, 10)
				market.Type = models.OrderTypeMarket
				res, err := engine.Submit(market)
				if err != nil || len(res.Trades) != 10 {
					b.Fatalf("expected 10 trades, got %d (err %v)", len(res.Trades), err)
				}
				b.StopTimer()
				for l := 0; l < 10; l++ {
					if err := book.Add(limit(models.SideSell, 50001+float64(l), 1)); err != nil {
						b.Fatal(err)
					}
				}
				b.StartTimer()
			}
		})
	}
}
