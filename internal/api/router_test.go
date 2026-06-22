package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/api"
	"github.com/jellyfishing2346/cryptex/internal/matching"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
)

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	book := orderbook.New("BTC-USD")
	engine := matching.New(book)
	return api.NewServer(book, engine).Router()
}

func doJSON(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			t.Fatalf("encode request body: %v", err)
		}
	}

	req := httptest.NewRequest(method, path, &payload)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(rec.Body.Bytes(), &value); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
	return value
}

func limitOrder(side models.Side, price, quantity float64) map[string]any {
	return map[string]any{
		"trading_pair": "BTC-USD",
		"side":         side,
		"type":         models.OrderTypeLimit,
		"price":        price,
		"quantity":     quantity,
		"user_id":      uuid.New(),
	}
}

func TestPlaceOrderBooksLimitOrder(t *testing.T) {
	router := newTestRouter()

	rec := doJSON(t, router, http.MethodPost, "/orders", limitOrder(models.SideBuy, 50000, 1.25))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, rec.Code, rec.Body.String())
	}

	result := decodeJSON[matching.Result](t, rec)
	if result.Order.ID == uuid.Nil {
		t.Fatal("expected generated order id")
	}
	if result.Order.Status != models.OrderStatusOpen {
		t.Fatalf("expected open order, got %s", result.Order.Status)
	}
	if len(result.Trades) != 0 {
		t.Fatalf("expected no trades, got %d", len(result.Trades))
	}

	snapshotRec := doJSON(t, router, http.MethodGet, "/orderbook", nil)
	if snapshotRec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, snapshotRec.Code)
	}
	snapshot := decodeJSON[models.OrderBookSnapshot](t, snapshotRec)
	if len(snapshot.Bids) != 1 || snapshot.Bids[0].Price != 50000 {
		t.Fatalf("expected bid at 50000, got %+v", snapshot.Bids)
	}
}

func TestPlaceOrderReturnsTrades(t *testing.T) {
	router := newTestRouter()

	sellRec := doJSON(t, router, http.MethodPost, "/orders", limitOrder(models.SideSell, 50000, 1))
	if sellRec.Code != http.StatusCreated {
		t.Fatalf("expected sell status %d, got %d: %s", http.StatusCreated, sellRec.Code, sellRec.Body.String())
	}

	buyRec := doJSON(t, router, http.MethodPost, "/orders", limitOrder(models.SideBuy, 50000, 1))
	if buyRec.Code != http.StatusCreated {
		t.Fatalf("expected buy status %d, got %d: %s", http.StatusCreated, buyRec.Code, buyRec.Body.String())
	}

	result := decodeJSON[matching.Result](t, buyRec)
	if result.Order.Status != models.OrderStatusFilled {
		t.Fatalf("expected filled order, got %s", result.Order.Status)
	}
	if len(result.Trades) != 1 {
		t.Fatalf("expected one trade, got %d", len(result.Trades))
	}
	if result.Trades[0].Price != 50000 {
		t.Fatalf("expected trade at 50000, got %f", result.Trades[0].Price)
	}
}

func TestCancelOrder(t *testing.T) {
	router := newTestRouter()

	createRec := doJSON(t, router, http.MethodPost, "/orders", limitOrder(models.SideBuy, 50000, 1))
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create status %d, got %d: %s", http.StatusCreated, createRec.Code, createRec.Body.String())
	}
	result := decodeJSON[matching.Result](t, createRec)

	cancelRec := doJSON(t, router, http.MethodDelete, "/orders/"+result.Order.ID.String(), nil)
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("expected cancel status %d, got %d: %s", http.StatusOK, cancelRec.Code, cancelRec.Body.String())
	}
	cancelled := decodeJSON[models.Order](t, cancelRec)
	if cancelled.Status != models.OrderStatusCancelled {
		t.Fatalf("expected cancelled order, got %s", cancelled.Status)
	}
}

func TestRejectsInvalidOrder(t *testing.T) {
	router := newTestRouter()

	body := limitOrder(models.SideBuy, 0, 1)
	rec := doJSON(t, router, http.MethodPost, "/orders", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestRejectsInvalidDepth(t *testing.T) {
	router := newTestRouter()

	rec := doJSON(t, router, http.MethodGet, "/orderbook?depth=-1", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}
