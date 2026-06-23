// Package risk implements risk management checks for order acceptance.
package risk

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
)

var (
	ErrPositionLimitExceeded    = errors.New("position limit exceeded")
	ErrSelfTradePrevention      = errors.New("self-trade prevention triggered")
	ErrPriceBelowCollar         = errors.New("price below minimum collar")
	ErrPriceAboveCollar         = errors.New("price above maximum collar")
	ErrInvalidOrderForRiskCheck = errors.New("invalid order for risk check")
)

// Config holds risk management configuration.
type Config struct {
	MaxPositionSize float64 // Maximum position size per user (in base currency)
	MinPrice        float64 // Minimum allowed price (price collar floor)
	MaxPrice        float64 // Maximum allowed price (price collar ceiling)
}

// DefaultConfig returns sensible default risk limits.
func DefaultConfig() Config {
	return Config{
		MaxPositionSize: 1000.0, // Default: 1000 units max position
		MinPrice:        0.01,  // Default: $0.01 minimum price
		MaxPrice:        1000000.0, // Default: $1M maximum price
	}
}

// Checker performs risk checks on orders before they're accepted.
type Checker struct {
	config Config
	book   *orderbook.OrderBook
}

// New creates a new risk checker with the given configuration.
func New(config Config, book *orderbook.OrderBook) *Checker {
	return &Checker{
		config: config,
		book:   book,
	}
}

// NewWithConfig creates a new risk checker with a pointer to the configuration.
func NewWithConfig(config *Config, book *orderbook.OrderBook) *Checker {
	return &Checker{
		config: *config,
		book:   book,
	}
}

// CheckOrder performs all risk checks on an order before submission.
// Returns an error if any risk check fails.
func (r *Checker) CheckOrder(order *models.Order) error {
	if order == nil {
		return ErrInvalidOrderForRiskCheck
	}

	// Check position limits
	if err := r.checkPositionLimit(order); err != nil {
		return err
	}

	// Check for self-trade prevention
	if err := r.checkSelfTradePrevention(order); err != nil {
		return err
	}

	// Check price collars (for limit orders only)
	if order.Type == models.OrderTypeLimit {
		if err := r.checkPriceCollar(order); err != nil {
			return err
		}
	}

	return nil
}

// checkPositionLimit ensures the user doesn't exceed their maximum position size.
func (r *Checker) checkPositionLimit(order *models.Order) error {
	currentPosition := r.calculateUserPosition(order.UserID, order.Side)
	newPosition := currentPosition + order.Quantity

	if newPosition > r.config.MaxPositionSize {
		return fmt.Errorf("%w: current position %.2f, new position %.2f exceeds limit %.2f",
			ErrPositionLimitExceeded, currentPosition, newPosition, r.config.MaxPositionSize)
	}

	return nil
}

// calculateUserPosition calculates the current position size for a user on a given side.
func (r *Checker) calculateUserPosition(userID uuid.UUID, side models.Side) float64 {
	r.book.RLock()
	defer r.book.RUnlock()

	var position float64
	orders := r.book.RestingOrders()

	for _, o := range orders {
		if o.UserID == userID && o.Side == side {
			position += o.Remaining()
		}
	}

	return position
}

// checkSelfTradePrevention ensures the order doesn't trade against the user's own orders.
func (r *Checker) checkSelfTradePrevention(order *models.Order) error {
	r.book.RLock()
	defer r.book.RUnlock()

	// Get the opposite side of the book
	var levels []*orderbook.PriceLevel
	if order.Side == models.SideBuy {
		levels = r.book.GetAsks()
	} else {
		levels = r.book.GetBids()
	}

	// Check if the order would cross any of the user's own orders
	for _, level := range levels {
		// For limit orders, check if price crosses
		if order.Type == models.OrderTypeLimit {
			if order.Side == models.SideBuy && order.Price < level.Price {
				continue
			}
			if order.Side == models.SideSell && order.Price > level.Price {
				continue
			}
		}

		// Check if any orders at this level belong to the same user
		for _, restingOrder := range level.Orders {
			if restingOrder.UserID == order.UserID {
				return fmt.Errorf("%w: order would cross with user's own order %s",
					ErrSelfTradePrevention, restingOrder.ID)
			}
		}
	}

	return nil
}

// checkPriceCollar ensures the order price is within acceptable bounds.
func (r *Checker) checkPriceCollar(order *models.Order) error {
	if order.Price < r.config.MinPrice {
		return fmt.Errorf("%w: price %.2f is below minimum %.2f",
			ErrPriceBelowCollar, order.Price, r.config.MinPrice)
	}

	if order.Price > r.config.MaxPrice {
		return fmt.Errorf("%w: price %.2f is above maximum %.2f",
			ErrPriceAboveCollar, order.Price, r.config.MaxPrice)
	}

	return nil
}
