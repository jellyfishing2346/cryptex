package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
	"github.com/jellyfishing2346/cryptex/internal/risk"
)

func TestRiskChecksIntegration(t *testing.T) {
	book := orderbook.New("BTC-USD")
	riskConfig := &risk.Config{
		MaxPositionSize: 10.0,
		MinPrice:        0.01,
		MaxPrice:        1000000.0,
	}
	server := NewServerWithRisk(book, nil, riskConfig)
	router := server.Router()

	userID := uuid.New()

	// Test 1: Valid order should be accepted
	t.Run("valid order accepted", func(t *testing.T) {
		reqBody := placeOrderRequest{
			TradingPair: "BTC-USD",
			Side:        models.SideBuy,
			Type:        models.OrderTypeLimit,
			Price:       50000.0,
			Quantity:    1.0,
			UserID:      userID,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}
	})

	// Test 2: Order exceeding position limit should be rejected
	t.Run("position limit exceeded", func(t *testing.T) {
		// Add existing orders
		for i := 0; i < 8; i++ {
			order := &models.Order{
				ID:          uuid.New(),
				TradingPair: "BTC-USD",
				Side:        models.SideBuy,
				Type:        models.OrderTypeLimit,
				Price:       50000.0,
				Quantity:    1.0,
				UserID:      userID,
			}
			book.Add(order)
		}

		reqBody := placeOrderRequest{
			TradingPair: "BTC-USD",
			Side:        models.SideBuy,
			Type:        models.OrderTypeLimit,
			Price:       50000.0,
			Quantity:    3.0, // This would exceed the limit (8 + 3 = 11 > 10)
			UserID:      userID,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}

		var response map[string]string
		json.Unmarshal(w.Body.Bytes(), &response)
		if response["error"] == "" {
			t.Error("expected error message in response")
		}
	})

	// Test 3: Order below min price should be rejected
	t.Run("price below collar", func(t *testing.T) {
		reqBody := placeOrderRequest{
			TradingPair: "BTC-USD",
			Side:        models.SideBuy,
			Type:        models.OrderTypeLimit,
			Price:       0.001, // Below min price of 0.01
			Quantity:    1.0,
			UserID:      uuid.New(),
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	// Test 4: Order above max price should be rejected
	t.Run("price above collar", func(t *testing.T) {
		reqBody := placeOrderRequest{
			TradingPair: "BTC-USD",
			Side:        models.SideBuy,
			Type:        models.OrderTypeLimit,
			Price:       2000000.0, // Above max price of 1000000.0
			Quantity:    1.0,
			UserID:      uuid.New(),
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	// Test 5: Self-trade prevention
	t.Run("self-trade prevention", func(t *testing.T) {
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
		book.Add(sellOrder)

		// Try to place a buy order that would cross
		reqBody := placeOrderRequest{
			TradingPair: "BTC-USD",
			Side:        models.SideBuy,
			Type:        models.OrderTypeLimit,
			Price:       51000.0, // Higher than sell order, would cross
			Quantity:    1.0,
			UserID:      userID,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/orders", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestDashboardEndpoints(t *testing.T) {
	book := orderbook.New("BTC-USD")
	riskConfig := &risk.Config{
		MaxPositionSize: 1000.0,
		MinPrice:        0.01,
		MaxPrice:        1000000.0,
	}
	server := NewServerWithRisk(book, nil, riskConfig)
	router := server.Router()

	// Test health endpoint
	t.Run("health endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/healthz", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test orderbook snapshot endpoint
	t.Run("orderbook snapshot", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/orderbook", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var snapshot models.OrderBookSnapshot
		if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil {
			t.Errorf("failed to unmarshal snapshot: %v", err)
		}

		if snapshot.TradingPair != "BTC-USD" {
			t.Errorf("expected trading pair BTC-USD, got %s", snapshot.TradingPair)
		}
	})
}
