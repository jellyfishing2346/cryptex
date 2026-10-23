package orderbook_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
)

// Book sizes (price levels per side) used across benchmarks. Levels are
// kept in sorted slices, so costs grow with depth; measuring several sizes
// shows where that starts to matter.
var depths = []int{10, 100, 1000, 10000}

func benchOrder(side models.Side, price float64) *models.Order {
	now := time.Now().UTC()
	return &models.Order{
		ID: uuid.New(), TradingPair: "BTC-USD", Side: side, Type: models.OrderTypeLimit,
		Price: price, Quantity: 1, Status: models.OrderStatusOpen, UserID: uuid.New(),
		CreatedAt: now, UpdatedAt: now,
	}
}

// filledBook returns a book with `levels` bid levels below 50,000 and ask
// levels above 50,001, one order each.
func filledBook(b *testing.B, levels int) *orderbook.OrderBook {
	b.Helper()
	book := orderbook.New("BTC-USD")
	for i := 0; i < levels; i++ {
		if err := book.Add(benchOrder(models.SideBuy, 50000-float64(i))); err != nil {
			b.Fatal(err)
		}
		if err := book.Add(benchOrder(models.SideSell, 50001+float64(i))); err != nil {
			b.Fatal(err)
		}
	}
	return book
}

// BenchmarkAddCancelNewLevel adds an order at a new best-bid price (creating
// a price level) and cancels it, leaving the book unchanged between runs.
func BenchmarkAddCancelNewLevel(b *testing.B) {
	for _, levels := range depths {
		b.Run(fmt.Sprintf("levels=%d", levels), func(b *testing.B) {
			book := filledBook(b, levels)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				o := benchOrder(models.SideBuy, 50000.5)
				if err := book.Add(o); err != nil {
					b.Fatal(err)
				}
				if _, err := book.Cancel(o.ID); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkSnapshot builds the top-10 snapshot served by GET /orderbook and
// broadcast after every order.
func BenchmarkSnapshot(b *testing.B) {
	for _, levels := range depths {
		b.Run(fmt.Sprintf("levels=%d", levels), func(b *testing.B) {
			book := filledBook(b, levels)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if snap := book.Snapshot(10); len(snap.Bids) == 0 {
					b.Fatal("empty snapshot")
				}
			}
		})
	}
}
