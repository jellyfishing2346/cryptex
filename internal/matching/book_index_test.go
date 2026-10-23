package matching_test

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/matching"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
	"github.com/jellyfishing2346/cryptex/internal/risk"
)

func orderFor(user uuid.UUID, side models.Side, price, qty float64) *models.Order {
	o := limit(side, price, qty)
	o.UserID = user
	return o
}

func TestFilledOrdersAreRemovedFromTheBook(t *testing.T) {
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)
	seller := uuid.New()

	sell := orderFor(seller, models.SideSell, 50000, 1)
	if _, err := engine.Submit(sell); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Submit(orderFor(uuid.New(), models.SideBuy, 50000, 1)); err != nil {
		t.Fatal(err)
	}

	if _, ok := book.GetOrder(sell.ID); ok {
		t.Fatal("a fully filled order should no longer be held by the book")
	}
	if q := book.UserRestingQuantity(seller, models.SideSell); q != 0 {
		t.Fatalf("filled order still counted in the user's exposure: %v", q)
	}
}

func TestUserExposureTracksPartialFillsAndCancels(t *testing.T) {
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)
	alice := uuid.New()

	a := orderFor(alice, models.SideSell, 50000, 2)
	b := orderFor(alice, models.SideSell, 50100, 3)
	for _, o := range []*models.Order{a, b} {
		if _, err := engine.Submit(o); err != nil {
			t.Fatal(err)
		}
	}
	if q := book.UserRestingQuantity(alice, models.SideSell); q != 5 {
		t.Fatalf("expected 5 resting, got %v", q)
	}

	if _, err := engine.Submit(orderFor(uuid.New(), models.SideBuy, 50000, 0.5)); err != nil {
		t.Fatal(err)
	}
	if q := book.UserRestingQuantity(alice, models.SideSell); q != 4.5 {
		t.Fatalf("after a partial fill expected 4.5, got %v", q)
	}

	if _, err := book.Cancel(b.ID); err != nil {
		t.Fatal(err)
	}
	if q := book.UserRestingQuantity(alice, models.SideSell); q != 1.5 {
		t.Fatalf("after cancelling expected 1.5, got %v", q)
	}
	if q := book.UserRestingQuantity(alice, models.SideBuy); q != 0 {
		t.Fatalf("no buy orders, got %v", q)
	}
}

func TestCrossingOrderOf(t *testing.T) {
	book := orderbook.New("BTC-USD")
	bob := uuid.New()
	ask := orderFor(bob, models.SideSell, 50000, 1)
	if err := book.Add(ask); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name   string
		side   models.Side
		price  float64
		market bool
		want   bool
	}{
		{"buy below own ask", models.SideBuy, 49999, false, false},
		{"buy at own ask", models.SideBuy, 50000, false, true},
		{"market buy", models.SideBuy, 0, true, true},
		{"same side", models.SideSell, 50000, false, false},
	}
	for _, c := range cases {
		if _, got := book.CrossingOrderOf(bob, c.side, c.price, c.market); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	if _, got := book.CrossingOrderOf(uuid.New(), models.SideBuy, 50000, false); got {
		t.Error("another user's order must not count")
	}
}

// Many goroutines placing, matching and cancelling at once, with risk
// checks on. The checker used to take the book's read lock twice in a row,
// which can deadlock once a writer queues between the two.
func TestConcurrentSubmitsWithRiskChecksDoNotDeadlock(t *testing.T) {
	book := orderbook.New("BTC-USD")
	cfg := risk.DefaultConfig()
	engine := matching.NewWithRiskChecks(book, risk.NewWithConfig(&cfg, book))

	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		for g := 0; g < 8; g++ {
			wg.Add(1)
			go func(g int) {
				defer wg.Done()
				user := uuid.New()
				for i := 0; i < 400; i++ {
					side := models.SideBuy
					if (g+i)%2 == 0 {
						side = models.SideSell
					}
					o := orderFor(user, side, 50000+float64(i%7), 0.1)
					if _, err := engine.Submit(o); err == nil && i%3 == 0 {
						_, _ = book.Cancel(o.ID)
					}
					_ = book.Snapshot(10)
				}
			}(g)
		}
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(20 * time.Second):
		t.Fatal("concurrent submits did not finish: possible deadlock")
	}
}
