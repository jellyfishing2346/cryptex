// Package orderbook implements a price-time priority order book.
//
// The order book maintains two sides:
//   - Bids (buy orders): sorted by price descending, then time ascending
//   - Asks (sell orders): sorted by price ascending, then time ascending
//
// This is the standard price-time priority (FIFO) matching used by
// most exchanges including Coinbase, Binance, and NYSE.
package orderbook

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/models"
)

// PriceLevel holds all orders at a specific price point.
// Orders within a level are sorted by time (FIFO — first in, first out).
type PriceLevel struct {
	Price  float64
	Orders []*models.Order // sorted by CreatedAt ascending
}

// TotalQuantity returns the total remaining quantity at this price level.
func (pl *PriceLevel) TotalQuantity() float64 {
	total := 0.0
	for _, o := range pl.Orders {
		total += o.Remaining()
	}
	return total
}

// OrderBook holds all open limit orders for a single trading pair.
// It is safe for concurrent use.
type OrderBook struct {
	mu          sync.RWMutex
	TradingPair string

	// bids: buy orders, sorted price descending (highest buyer first)
	bids []*PriceLevel

	// asks: sell orders, sorted price ascending (lowest seller first)
	asks []*PriceLevel

	// index for O(1) order lookup by ID
	orders map[uuid.UUID]*models.Order
}

// New creates an empty order book for the given trading pair.
func New(tradingPair string) *OrderBook {
	return &OrderBook{
		TradingPair: tradingPair,
		bids:        make([]*PriceLevel, 0),
		asks:        make([]*PriceLevel, 0),
		orders:      make(map[uuid.UUID]*models.Order),
	}
}

// Add inserts a limit order into the order book.
// Returns an error if the order already exists or is invalid.
// Acquires the write lock — do not call while already holding it
// (use AddLocked from within matching engine code instead).
func (ob *OrderBook) Add(order *models.Order) error {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	return ob.addLocked(order)
}

// AddLocked inserts a limit order into the book WITHOUT acquiring the
// lock. Callers must already hold the write lock (e.g. via Lock()).
// This is used by the matching engine, which locks once for the whole
// Submit() operation rather than locking per sub-step.
func (ob *OrderBook) AddLocked(order *models.Order) error {
	return ob.addLocked(order)
}

func (ob *OrderBook) addLocked(order *models.Order) error {
	if _, exists := ob.orders[order.ID]; exists {
		return fmt.Errorf("order %s already exists", order.ID)
	}
	if order.Price <= 0 {
		return fmt.Errorf("limit order price must be positive, got %f", order.Price)
	}
	if order.Quantity <= 0 {
		return fmt.Errorf("order quantity must be positive, got %f", order.Quantity)
	}

	ob.orders[order.ID] = order

	if order.Side == models.SideBuy {
		ob.addToBids(order)
	} else {
		ob.addToAsks(order)
	}

	return nil
}

// Cancel removes an open order from the book.
// Returns an error if the order is not found or already filled/cancelled.
func (ob *OrderBook) Cancel(orderID uuid.UUID) (*models.Order, error) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	order, exists := ob.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order %s not found", orderID)
	}
	if order.Status == models.OrderStatusFilled {
		return nil, fmt.Errorf("order %s is already filled", orderID)
	}
	if order.Status == models.OrderStatusCancelled {
		return nil, fmt.Errorf("order %s is already cancelled", orderID)
	}

	order.Status = models.OrderStatusCancelled
	order.UpdatedAt = time.Now().UTC()

	// Remove from price level
	if order.Side == models.SideBuy {
		ob.removeFromBids(order)
	} else {
		ob.removeFromAsks(order)
	}

	delete(ob.orders, orderID)
	return order, nil
}

// BestBid returns the highest buy price in the book, or 0 if empty.
func (ob *OrderBook) BestBid() float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if len(ob.bids) == 0 {
		return 0
	}
	return ob.bids[0].Price
}

// BestAsk returns the lowest sell price in the book, or 0 if empty.
func (ob *OrderBook) BestAsk() float64 {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if len(ob.asks) == 0 {
		return 0
	}
	return ob.asks[0].Price
}

// Spread returns the difference between best ask and best bid.
// Returns 0 if either side is empty.
func (ob *OrderBook) Spread() float64 {
	bid := ob.BestBid()
	ask := ob.BestAsk()
	if bid == 0 || ask == 0 {
		return 0
	}
	return ask - bid
}

// Snapshot returns the current state of the order book,
// aggregated by price level. Depth limits how many levels to return.
func (ob *OrderBook) Snapshot(depth int) *models.OrderBookSnapshot {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids := make([]models.OrderBookLevel, 0)
	for i, level := range ob.bids {
		if depth > 0 && i >= depth {
			break
		}
		bids = append(bids, models.OrderBookLevel{
			Price:    level.Price,
			Quantity: level.TotalQuantity(),
			Orders:   len(level.Orders),
		})
	}

	asks := make([]models.OrderBookLevel, 0)
	for i, level := range ob.asks {
		if depth > 0 && i >= depth {
			break
		}
		asks = append(asks, models.OrderBookLevel{
			Price:    level.Price,
			Quantity: level.TotalQuantity(),
			Orders:   len(level.Orders),
		})
	}

	return &models.OrderBookSnapshot{
		TradingPair: ob.TradingPair,
		Bids:        bids,
		Asks:        asks,
		Timestamp:   time.Now().UTC(),
	}
}

// GetOrder returns an order by ID.
func (ob *OrderBook) GetOrder(orderID uuid.UUID) (*models.Order, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	order, ok := ob.orders[orderID]
	return order, ok
}

// RestingOrders returns all orders currently resting in the book.
// Orders are returned in book priority order: bids first, then asks.
func (ob *OrderBook) RestingOrders() []*models.Order {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	orders := make([]*models.Order, 0, len(ob.orders))
	for _, level := range ob.bids {
		for _, order := range level.Orders {
			if order.Remaining() > 0 && order.Status != models.OrderStatusCancelled {
				orders = append(orders, cloneOrder(order))
			}
		}
	}
	for _, level := range ob.asks {
		for _, order := range level.Orders {
			if order.Remaining() > 0 && order.Status != models.OrderStatusCancelled {
				orders = append(orders, cloneOrder(order))
			}
		}
	}
	return orders
}

// BidsForMatching returns bid price levels for the matching engine.
// Caller must hold the write lock.
func (ob *OrderBook) BidsForMatching() []*PriceLevel {
	return ob.bids
}

// AsksForMatching returns ask price levels for the matching engine.
// Caller must hold the write lock.
func (ob *OrderBook) AsksForMatching() []*PriceLevel {
	return ob.asks
}

// Lock acquires the write lock. Used by the matching engine.
func (ob *OrderBook) Lock() {
	ob.mu.Lock()
}

// Unlock releases the write lock. Used by the matching engine.
func (ob *OrderBook) Unlock() {
	ob.mu.Unlock()
}

// CleanEmptyLevels removes price levels with no remaining orders.
// Called by the matching engine after fills.
func (ob *OrderBook) CleanEmptyLevels() {
	ob.bids = filterEmptyLevels(ob.bids)
	ob.asks = filterEmptyLevels(ob.asks)
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (ob *OrderBook) addToBids(order *models.Order) {
	ob.bids = addToLevels(ob.bids, order, true)
}

func (ob *OrderBook) addToAsks(order *models.Order) {
	ob.asks = addToLevels(ob.asks, order, false)
}

func (ob *OrderBook) removeFromBids(order *models.Order) {
	ob.bids = removeFromLevels(ob.bids, order)
}

func (ob *OrderBook) removeFromAsks(order *models.Order) {
	ob.asks = removeFromLevels(ob.asks, order)
}

// addToLevels inserts an order into the correct price level.
// descending=true for bids (highest price first),
// descending=false for asks (lowest price first).
func addToLevels(levels []*PriceLevel, order *models.Order, descending bool) []*PriceLevel {
	// Find existing level at this price
	for _, level := range levels {
		if level.Price == order.Price {
			level.Orders = append(level.Orders, order)
			return levels
		}
	}

	// No existing level — create a new one
	newLevel := &PriceLevel{
		Price:  order.Price,
		Orders: []*models.Order{order},
	}
	levels = append(levels, newLevel)

	// Re-sort: bids descending, asks ascending
	sort.Slice(levels, func(i, j int) bool {
		if descending {
			return levels[i].Price > levels[j].Price
		}
		return levels[i].Price < levels[j].Price
	})

	return levels
}

// removeFromLevels removes a specific order from its price level.
func removeFromLevels(levels []*PriceLevel, order *models.Order) []*PriceLevel {
	for _, level := range levels {
		if level.Price == order.Price {
			for i, o := range level.Orders {
				if o.ID == order.ID {
					level.Orders = append(level.Orders[:i], level.Orders[i+1:]...)
					break
				}
			}
			break
		}
	}
	return filterEmptyLevels(levels)
}

// filterEmptyLevels removes price levels with no orders.
func filterEmptyLevels(levels []*PriceLevel) []*PriceLevel {
	result := levels[:0]
	for _, level := range levels {
		if len(level.Orders) > 0 {
			result = append(result, level)
		}
	}
	return result
}

func cloneOrder(order *models.Order) *models.Order {
	clone := *order
	return &clone
}
