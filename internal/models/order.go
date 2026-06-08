package models

import (
	"time"

	"github.com/google/uuid"
)

// Side represents whether an order is a buy or sell.
type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

// OrderType represents the type of order.
type OrderType string

const (
	OrderTypeLimit  OrderType = "limit"
	OrderTypeMarket OrderType = "market"
)

// OrderStatus represents the current state of an order.
type OrderStatus string

const (
	OrderStatusOpen      OrderStatus = "open"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusPartial   OrderStatus = "partial"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// Order represents a single order in the order book.
type Order struct {
	ID          uuid.UUID   `json:"id"`
	TradingPair string      `json:"trading_pair"` // e.g. "BTC-USD"
	Side        Side        `json:"side"`
	Type        OrderType   `json:"type"`
	Price       float64     `json:"price"`    // 0 for market orders
	Quantity    float64     `json:"quantity"` // original quantity
	Filled      float64     `json:"filled"`   // how much has been filled
	Status      OrderStatus `json:"status"`
	UserID      uuid.UUID   `json:"user_id"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Remaining returns how much of the order is left to fill.
func (o *Order) Remaining() float64 {
	return o.Quantity - o.Filled
}

// IsFilled returns true if the order has been completely filled.
func (o *Order) IsFilled() bool {
	return o.Filled >= o.Quantity
}

// Trade represents a completed match between a buy and sell order.
type Trade struct {
	ID          uuid.UUID `json:"id"`
	TradingPair string    `json:"trading_pair"`
	BuyOrderID  uuid.UUID `json:"buy_order_id"`
	SellOrderID uuid.UUID `json:"sell_order_id"`
	Price       float64   `json:"price"`
	Quantity    float64   `json:"quantity"`
	ExecutedAt  time.Time `json:"executed_at"`
}

// OrderBookLevel represents a price level in the order book.
// Each level aggregates all orders at the same price.
type OrderBookLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Orders   int     `json:"orders"`
}

// OrderBookSnapshot is what gets sent to clients — the current state.
type OrderBookSnapshot struct {
	TradingPair string           `json:"trading_pair"`
	Bids        []OrderBookLevel `json:"bids"` // buy orders, highest price first
	Asks        []OrderBookLevel `json:"asks"` // sell orders, lowest price first
	Timestamp   time.Time        `json:"timestamp"`
}
