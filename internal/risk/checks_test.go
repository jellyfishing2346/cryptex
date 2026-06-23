package risk

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
)

func TestCheckOrder(t *testing.T) {
	book := orderbook.New("BTC-USD")
	config := DefaultConfig()
	checker := New(config, book)

	userID := uuid.New()

	tests := []struct {
		name    string
		order   *models.Order
		wantErr error
	}{
		{
			name: "valid order passes all checks",
			order: &models.Order{
				ID:          uuid.New(),
				TradingPair: "BTC-USD",
				Side:        models.SideBuy,
				Type:        models.OrderTypeLimit,
				Price:       50000.0,
				Quantity:    1.0,
				UserID:      userID,
			},
			wantErr: nil,
		},
		{
			name: "nil order fails",
			order: nil,
			wantErr: ErrInvalidOrderForRiskCheck,
		},
		{
			name: "order below min price collar fails",
			order: &models.Order{
				ID:          uuid.New(),
				TradingPair: "BTC-USD",
				Side:        models.SideBuy,
				Type:        models.OrderTypeLimit,
				Price:       0.001,
				Quantity:    1.0,
				UserID:      userID,
			},
			wantErr: ErrPriceBelowCollar,
		},
		{
			name: "order above max price collar fails",
			order: &models.Order{
				ID:          uuid.New(),
				TradingPair: "BTC-USD",
				Side:        models.SideBuy,
				Type:        models.OrderTypeLimit,
				Price:       2000000.0,
				Quantity:    1.0,
				UserID:      userID,
			},
			wantErr: ErrPriceAboveCollar,
		},
		{
			name: "market order skips price collar check",
			order: &models.Order{
				ID:          uuid.New(),
				TradingPair: "BTC-USD",
				Side:        models.SideBuy,
				Type:        models.OrderTypeMarket,
				Price:       0.0,
				Quantity:    1.0,
				UserID:      userID,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.CheckOrder(tt.order)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("CheckOrder() expected error %v, got nil", tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("CheckOrder() unexpected error: %v", err)
			}
		})
	}
}

func TestPositionLimit(t *testing.T) {
	book := orderbook.New("BTC-USD")
	config := Config{
		MaxPositionSize: 10.0,
		MinPrice:        0.01,
		MaxPrice:        1000000.0,
	}
	checker := New(config, book)

	userID := uuid.New()

	// Add existing orders for the user
	existingOrder := &models.Order{
		ID:          uuid.New(),
		TradingPair: "BTC-USD",
		Side:        models.SideBuy,
		Type:        models.OrderTypeLimit,
		Price:       50000.0,
		Quantity:    8.0,
		UserID:      userID,
	}
	if err := book.Add(existingOrder); err != nil {
		t.Fatalf("failed to add existing order: %v", err)
	}

	tests := []struct {
		name    string
		order   *models.Order
		wantErr error
	}{
		{
			name: "order within position limit passes",
			order: &models.Order{
				ID:          uuid.New(),
				TradingPair: "BTC-USD",
				Side:        models.SideBuy,
				Type:        models.OrderTypeLimit,
				Price:       50000.0,
				Quantity:    1.5,
				UserID:      userID,
			},
			wantErr: nil,
		},
		{
			name: "order exceeding position limit fails",
			order: &models.Order{
				ID:          uuid.New(),
				TradingPair: "BTC-USD",
				Side:        models.SideBuy,
				Type:        models.OrderTypeLimit,
				Price:       50000.0,
				Quantity:    3.0,
				UserID:      userID,
			},
			wantErr: ErrPositionLimitExceeded,
		},
		{
			name: "different user not affected by position limit",
			order: &models.Order{
				ID:          uuid.New(),
				TradingPair: "BTC-USD",
				Side:        models.SideBuy,
				Type:        models.OrderTypeLimit,
				Price:       50000.0,
				Quantity:    5.0,
				UserID:      uuid.New(),
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.CheckOrder(tt.order)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("CheckOrder() expected error %v, got nil", tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("CheckOrder() unexpected error: %v", err)
			}
		})
	}
}

func TestSelfTradePrevention(t *testing.T) {
	book := orderbook.New("BTC-USD")
	config := DefaultConfig()
	checker := New(config, book)

	userID := uuid.New()

	// Add a sell order for the user
	sellOrder := &models.Order{
		ID:          uuid.New(),
		TradingPair: "BTC-USD",
		Side:        models.SideSell,
		Type:        models.OrderTypeLimit,
		Price:       50000.0,
		Quantity:    1.0,
		UserID:      userID,
	}
	if err := book.Add(sellOrder); err != nil {
		t.Fatalf("failed to add sell order: %v", err)
	}

	tests := []struct {
		name    string
		order   *models.Order
		wantErr error
	}{
		{
			name: "buy order that would cross own sell order fails",
			order: &models.Order{
				ID:          uuid.New(),
				TradingPair: "BTC-USD",
				Side:        models.SideBuy,
				Type:        models.OrderTypeLimit,
				Price:       51000.0, // Higher than sell order, would cross
				Quantity:    1.0,
				UserID:      userID,
			},
			wantErr: ErrSelfTradePrevention,
		},
		{
			name: "buy order that doesn't cross own sell order passes",
			order: &models.Order{
				ID:          uuid.New(),
				TradingPair: "BTC-USD",
				Side:        models.SideBuy,
				Type:        models.OrderTypeLimit,
				Price:       49000.0, // Lower than sell order, won't cross
				Quantity:    1.0,
				UserID:      userID,
			},
			wantErr: nil,
		},
		{
			name: "different user can cross the order",
			order: &models.Order{
				ID:          uuid.New(),
				TradingPair: "BTC-USD",
				Side:        models.SideBuy,
				Type:        models.OrderTypeLimit,
				Price:       51000.0,
				Quantity:    1.0,
				UserID:      uuid.New(),
			},
			wantErr: nil,
		},
		{
			name: "market order checks for self-trade",
			order: &models.Order{
				ID:          uuid.New(),
				TradingPair: "BTC-USD",
				Side:        models.SideBuy,
				Type:        models.OrderTypeMarket,
				Price:       0.0,
				Quantity:    1.0,
				UserID:      userID,
			},
			wantErr: ErrSelfTradePrevention,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.CheckOrder(tt.order)
			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("CheckOrder() expected error %v, got nil", tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("CheckOrder() unexpected error: %v", err)
			}
		})
	}
}
