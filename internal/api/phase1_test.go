package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jellyfishing2346/cryptex/internal/api"
	"github.com/jellyfishing2346/cryptex/internal/matching"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
	"github.com/jellyfishing2346/cryptex/internal/ws"
)

func newServer(opts api.Options) (*api.Server, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	book := orderbook.New("BTC-USD")
	server := api.NewServer(book, matching.New(book), nil).WithOptions(opts)
	return server, server.Router()
}

func orderFor(user uuid.UUID, side models.Side, price, quantity float64) map[string]any {
	o := limitOrder(side, price, quantity)
	o["user_id"] = user
	return o
}

func TestCancelRequiresOwner(t *testing.T) {
	_, router := newServer(api.Options{})
	owner, stranger := uuid.New(), uuid.New()

	created := decodeJSON[matching.Result](t, doJSON(t, router, http.MethodPost, "/orders", orderFor(owner, models.SideBuy, 50000, 1)))
	path := "/orders/" + created.Order.ID.String()

	if rec := doJSON(t, router, http.MethodDelete, path, nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("cancel without user_id: expected 400, got %d", rec.Code)
	}
	if rec := doJSON(t, router, http.MethodDelete, path+"?user_id="+stranger.String(), nil); rec.Code != http.StatusForbidden {
		t.Fatalf("cancel by another user: expected 403, got %d", rec.Code)
	}
	if rec := doJSON(t, router, http.MethodDelete, path+"?user_id="+owner.String(), nil); rec.Code != http.StatusOK {
		t.Fatalf("cancel by owner: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if rec := doJSON(t, router, http.MethodDelete, path+"?user_id="+owner.String(), nil); rec.Code != http.StatusNotFound {
		t.Fatalf("cancel twice: expected 404, got %d", rec.Code)
	}
}

func TestListOrdersReturnsOnlyTheUsersRestingOrders(t *testing.T) {
	_, router := newServer(api.Options{})
	alice, bob := uuid.New(), uuid.New()
	doJSON(t, router, http.MethodPost, "/orders", orderFor(alice, models.SideBuy, 49000, 1))
	doJSON(t, router, http.MethodPost, "/orders", orderFor(alice, models.SideSell, 51000, 2))
	doJSON(t, router, http.MethodPost, "/orders", orderFor(bob, models.SideBuy, 48000, 1))

	rec := doJSON(t, router, http.MethodGet, "/orders?user_id="+alice.String(), nil)
	got := decodeJSON[struct{ Orders []models.Order }](t, rec)
	if len(got.Orders) != 2 {
		t.Fatalf("expected alice's 2 orders, got %d", len(got.Orders))
	}
	for _, o := range got.Orders {
		if o.UserID != alice {
			t.Fatalf("returned another user's order: %+v", o)
		}
	}
	if rec := doJSON(t, router, http.MethodGet, "/orders", nil); rec.Code != http.StatusBadRequest {
		t.Fatalf("missing user_id: expected 400, got %d", rec.Code)
	}
}

func TestTradesEndpointReturnsNewestFirst(t *testing.T) {
	_, router := newServer(api.Options{})
	seller, buyer := uuid.New(), uuid.New()
	doJSON(t, router, http.MethodPost, "/orders", orderFor(seller, models.SideSell, 50000, 1))
	doJSON(t, router, http.MethodPost, "/orders", orderFor(seller, models.SideSell, 50100, 1))
	doJSON(t, router, http.MethodPost, "/orders", orderFor(buyer, models.SideBuy, 50000, 1))
	doJSON(t, router, http.MethodPost, "/orders", orderFor(buyer, models.SideBuy, 50100, 1))

	got := decodeJSON[struct{ Trades []models.Trade }](t, doJSON(t, router, http.MethodGet, "/trades", nil))
	if len(got.Trades) != 2 {
		t.Fatalf("expected 2 trades, got %d", len(got.Trades))
	}
	if got.Trades[0].Price != 50100 || got.Trades[1].Price != 50000 {
		t.Fatalf("expected newest first, got prices %v, %v", got.Trades[0].Price, got.Trades[1].Price)
	}

	limited := decodeJSON[struct{ Trades []models.Trade }](t, doJSON(t, router, http.MethodGet, "/trades?limit=1", nil))
	if len(limited.Trades) != 1 {
		t.Fatalf("expected limit=1 to return 1 trade, got %d", len(limited.Trades))
	}
	for _, bad := range []string{"0", "-1", "abc", "1000"} {
		if rec := doJSON(t, router, http.MethodGet, "/trades?limit="+bad, nil); rec.Code != http.StatusBadRequest {
			t.Fatalf("limit=%s: expected 400, got %d", bad, rec.Code)
		}
	}
}

func TestMaxOrderQuantity(t *testing.T) {
	_, router := newServer(api.Options{MaxOrderQuantity: 5})
	if rec := doJSON(t, router, http.MethodPost, "/orders", limitOrder(models.SideBuy, 50000, 5)); rec.Code != http.StatusCreated {
		t.Fatalf("at the cap: expected 201, got %d", rec.Code)
	}
	rec := doJSON(t, router, http.MethodPost, "/orders", limitOrder(models.SideBuy, 50000, 5.01))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "limit per order") {
		t.Fatalf("over the cap: expected 400 with message, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRateLimitAppliesToWritesOnly(t *testing.T) {
	_, router := newServer(api.Options{RateLimitPerSecond: 0.001, RateLimitBurst: 2})
	codes := []int{}
	for i := 0; i < 3; i++ {
		codes = append(codes, doJSON(t, router, http.MethodPost, "/orders", limitOrder(models.SideBuy, 50000, 1)).Code)
	}
	if codes[0] != http.StatusCreated || codes[1] != http.StatusCreated || codes[2] != http.StatusTooManyRequests {
		t.Fatalf("expected 201, 201, 429; got %v", codes)
	}
	for i := 0; i < 5; i++ {
		if rec := doJSON(t, router, http.MethodGet, "/orderbook", nil); rec.Code != http.StatusOK {
			t.Fatalf("reads should not be rate limited, got %d", rec.Code)
		}
	}
}

func TestCORSAllowsOnlyConfiguredOrigins(t *testing.T) {
	_, router := newServer(api.Options{AllowedOrigins: []string{"https://cryptex.vercel.app/"}})

	request := func(method, origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/orderbook", nil)
		req.Header.Set("Origin", origin)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}

	ok := request(http.MethodGet, "https://cryptex.vercel.app")
	if got := ok.Header().Get("Access-Control-Allow-Origin"); got != "https://cryptex.vercel.app" {
		t.Fatalf("allowed origin: got header %q", got)
	}
	if got := request(http.MethodGet, "https://evil.example").Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("other origin should get no CORS header, got %q", got)
	}
	if rec := request(http.MethodOptions, "https://cryptex.vercel.app"); rec.Code != http.StatusNoContent {
		t.Fatalf("preflight: expected 204, got %d", rec.Code)
	}
}

func TestWebSocketStreamsTradesAndChecksOrigin(t *testing.T) {
	server, router := newServer(api.Options{AllowedOrigins: []string{"https://cryptex.vercel.app"}})
	srv := httptest.NewServer(router)
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"

	header := http.Header{"Origin": []string{"https://evil.example"}}
	if _, resp, err := websocket.DefaultDialer.Dial(wsURL, header); err == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("disallowed origin should be rejected with 403, got err=%v", err)
	}

	header = http.Header{"Origin": []string{"https://cryptex.vercel.app"}}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("allowed origin: dial: %v", err)
	}
	defer func() { _ = conn.Close() }()
	time.Sleep(50 * time.Millisecond) // let the hub register the client

	ctx := context.Background()
	seller, buyer := uuid.New(), uuid.New()
	if _, err := server.PlaceOrder(ctx, "BTC-USD", models.SideSell, models.OrderTypeLimit, 50000, 1, seller); err != nil {
		t.Fatal(err)
	}
	if _, err := server.PlaceOrder(ctx, "BTC-USD", models.SideBuy, models.OrderTypeLimit, 50000, 1, buyer); err != nil {
		t.Fatal(err)
	}

	sawTrade := false
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for i := 0; i < 5 && !sawTrade; i++ {
		var event ws.Event
		if err := conn.ReadJSON(&event); err != nil {
			t.Fatalf("read event: %v", err)
		}
		sawTrade = event.Type == ws.EventTypeTrade
	}
	if !sawTrade {
		t.Fatal("expected a trade event on the WebSocket")
	}
	if server.LastTradePrice() != 50000 {
		t.Fatalf("expected last trade price 50000, got %v", server.LastTradePrice())
	}
}
