// Package matching implements the core order matching algorithm.
//
// When a new order arrives, the engine checks the opposite side of the
// book for crossing orders (price overlap) and executes trades at the
// resting order's price — this is standard exchange behavior, since the
// order that was already in the book set the price first.
package matching

import (
	"time"

	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
	"github.com/jellyfishing2346/cryptex/internal/risk"
)

// Engine matches incoming orders against the resting orders in an OrderBook.
type Engine struct {
	book   *orderbook.OrderBook
	checker *risk.Checker
}

// New creates a matching engine for the given order book.
func New(book *orderbook.OrderBook) *Engine {
	return &Engine{book: book}
}

// NewWithRiskChecks creates a matching engine with risk management checks.
func NewWithRiskChecks(book *orderbook.OrderBook, checker *risk.Checker) *Engine {
	return &Engine{book: book, checker: checker}
}

// Result holds the outcome of submitting an order: the order itself
// (now updated with fill status) and any trades that were executed.
type Result struct {
	Order  *models.Order
	Trades []*models.Trade
}

// Submit processes a new order: it first tries to match against the
// resting book, then adds any unfilled remainder to the book (limit
// orders only — market orders that can't fully fill are not booked).
func (e *Engine) Submit(order *models.Order) (*Result, error) {
	// Perform risk checks before locking the book
	if e.checker != nil {
		if err := e.checker.CheckOrder(order); err != nil {
			return nil, err
		}
	}

	e.book.Lock()
	defer e.book.Unlock()

	now := time.Now().UTC()
	order.CreatedAt = now
	order.UpdatedAt = now
	order.Status = models.OrderStatusOpen

	trades := e.match(order)

	// Update order status based on fill state
	switch {
	case order.IsFilled():
		order.Status = models.OrderStatusFilled
	case order.Filled > 0:
		order.Status = models.OrderStatusPartial
	default:
		order.Status = models.OrderStatusOpen
	}
	order.UpdatedAt = time.Now().UTC()

	// Market orders are never added to the book — any unfilled
	// remainder is simply dropped (this matches real exchange behavior).
	if order.Type == models.OrderTypeMarket {
		e.book.CleanEmptyLevels()
		return &Result{Order: order, Trades: trades}, nil
	}

	// Limit orders: book the remainder if not fully filled
	if !order.IsFilled() {
		if err := e.book.AddLocked(order); err != nil {
			return nil, err
		}
	}

	e.book.CleanEmptyLevels()
	return &Result{Order: order, Trades: trades}, nil
}

// match crosses the incoming order against the opposite side of the book.
// Returns the list of trades executed. Mutates order.Filled in place.
// Caller must hold the book's write lock.
func (e *Engine) match(incoming *models.Order) []*models.Trade {
	trades := make([]*models.Trade, 0)

	var levels []*orderbook.PriceLevel
	if incoming.Side == models.SideBuy {
		levels = e.book.AsksForMatching()
	} else {
		levels = e.book.BidsForMatching()
	}

	for _, level := range levels {
		if incoming.IsFilled() {
			break
		}
		if !e.crosses(incoming, level.Price) {
			break // levels are sorted, so no further levels will cross either
		}

		// Match against orders at this level, FIFO (oldest first)
		i := 0
		for i < len(level.Orders) && !incoming.IsFilled() {
			resting := level.Orders[i]
			if resting.IsFilled() || resting.Status == models.OrderStatusCancelled {
				i++
				continue
			}

			fillQty := min(incoming.Remaining(), resting.Remaining())
			incoming.Filled += fillQty
			resting.Filled += fillQty
			resting.UpdatedAt = time.Now().UTC()

			if resting.IsFilled() {
				resting.Status = models.OrderStatusFilled
			} else {
				resting.Status = models.OrderStatusPartial
			}

			trade := &models.Trade{
				ID:          uuid.New(),
				TradingPair: e.book.TradingPair,
				Price:       level.Price, // trades execute at the resting order's price
				Quantity:    fillQty,
				ExecutedAt:  time.Now().UTC(),
			}
			if incoming.Side == models.SideBuy {
				trade.BuyOrderID = incoming.ID
				trade.SellOrderID = resting.ID
			} else {
				trade.BuyOrderID = resting.ID
				trade.SellOrderID = incoming.ID
			}
			trades = append(trades, trade)

			if resting.IsFilled() {
				i++
			}
		}

		// Remove fully filled orders from this level
		remaining := level.Orders[:0]
		for _, o := range level.Orders {
			if !o.IsFilled() {
				remaining = append(remaining, o)
			}
		}
		level.Orders = remaining
	}

	return trades
}

// crosses returns true if the incoming order's price crosses the given
// resting price level — i.e. a match is possible at this level.
// Market orders always cross (they accept any price).
func (e *Engine) crosses(incoming *models.Order, restingPrice float64) bool {
	if incoming.Type == models.OrderTypeMarket {
		return true
	}
	if incoming.Side == models.SideBuy {
		return incoming.Price >= restingPrice
	}
	return incoming.Price <= restingPrice
}

// min returns the smaller of two float64 values.
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
