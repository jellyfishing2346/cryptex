package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/api"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
	"github.com/jellyfishing2346/cryptex/internal/persistence"
	"github.com/jellyfishing2346/cryptex/internal/risk"
	"github.com/redis/go-redis/v9"
)

// benchServer builds the server the way main.go does (risk checks on),
// with `resting` orders already on the book.
func benchServer(b *testing.B, resting int, store api.OrderBookStore) *gin.Engine {
	b.Helper()
	gin.SetMode(gin.ReleaseMode)
	book := orderbook.New("BTC-USD")
	cfg := risk.DefaultConfig()
	var server *api.Server
	if store != nil {
		server = api.NewServerWithRisk(book, nil, &cfg, store)
	} else {
		server = api.NewServerWithRisk(book, nil, &cfg)
	}
	ctx := context.Background()
	for i := 0; i < resting/2; i++ {
		if _, err := server.PlaceOrder(ctx, "BTC-USD", models.SideBuy, models.OrderTypeLimit, 49000-float64(i), 0.01, uuid.New()); err != nil {
			b.Fatal(err)
		}
		if _, err := server.PlaceOrder(ctx, "BTC-USD", models.SideSell, models.OrderTypeLimit, 51000+float64(i), 0.01, uuid.New()); err != nil {
			b.Fatal(err)
		}
	}
	return server.Router()
}

func post(b *testing.B, router http.Handler, side models.Side) {
	body, _ := json.Marshal(map[string]any{
		"trading_pair": "BTC-USD", "side": side, "type": "limit",
		"price": 50000, "quantity": 0.01, "user_id": uuid.New(),
	})
	req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		b.Fatalf("POST /orders: %d %s", rec.Code, rec.Body.String())
	}
}

// benchmarkHTTP posts a resting sell then a buy that matches it, so the book
// size stays constant. Each op is two full requests.
func benchmarkHTTP(b *testing.B, router http.Handler) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		post(b, router, models.SideSell)
		post(b, router, models.SideBuy)
	}
}

// BenchmarkHTTPPlaceOrder measures POST /orders in memory: JSON, validation,
// risk checks, matching and the WebSocket broadcast, without persistence.
func BenchmarkHTTPPlaceOrder(b *testing.B) {
	for _, resting := range []int{20, 1000} {
		b.Run(fmt.Sprintf("resting=%d", resting), func(b *testing.B) {
			benchmarkHTTP(b, benchServer(b, resting, nil))
		})
	}
}

// BenchmarkHTTPPlaceOrderRedis adds Redis persistence, which rewrites the
// resting book after every order. Needs a Redis server:
//
//	BENCH_REDIS_URL=redis://localhost:6379/15 go test -bench=Redis ./internal/api/
//
// The benchmark flushes that database, so point it at a scratch DB.
func BenchmarkHTTPPlaceOrderRedis(b *testing.B) {
	url := os.Getenv("BENCH_REDIS_URL")
	if url == "" {
		b.Skip("set BENCH_REDIS_URL to run against Redis")
	}
	opts, err := redis.ParseURL(url)
	if err != nil {
		b.Fatal(err)
	}
	client := redis.NewClient(opts)
	defer func() { _ = client.Close() }()
	if err := client.Ping(context.Background()).Err(); err != nil {
		b.Skipf("redis unavailable: %v", err)
	}
	for _, resting := range []int{20, 1000} {
		b.Run(fmt.Sprintf("resting=%d", resting), func(b *testing.B) {
			if err := client.FlushDB(context.Background()).Err(); err != nil {
				b.Fatal(err)
			}
			store := persistence.NewRedisOrderBookStore(client, "BTC-USD")
			benchmarkHTTP(b, benchServer(b, resting, store))
		})
	}
}
