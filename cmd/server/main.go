package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jellyfishing2346/cryptex/internal/api"
	"github.com/jellyfishing2346/cryptex/internal/demo"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/nats"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
	"github.com/jellyfishing2346/cryptex/internal/persistence"
	"github.com/jellyfishing2346/cryptex/internal/risk"
	"github.com/redis/go-redis/v9"
)

func main() {
	tradingPair := env("TRADING_PAIR", "BTC-USD")
	book := orderbook.New(tradingPair)
	redisOpts, err := redisOptions()
	if err != nil {
		log.Fatalf("redis config: %v", err)
	}
	client := redis.NewClient(redisOpts)
	defer func() {
		if err := client.Close(); err != nil {
			log.Printf("close redis client: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("connect to redis: %v", err)
	}

	store := persistence.NewRedisOrderBookStore(client, tradingPair)
	orders, err := store.Load(ctx)
	if err != nil {
		log.Fatalf("load order book: %v", err)
	}
	if err = restoreOrders(book, orders); err != nil {
		log.Fatalf("restore order book: %v", err)
	}
	log.Printf("loaded %d resting orders for %s", len(orders), tradingPair)

	// Configure risk management
	riskConfig := &risk.Config{
		MaxPositionSize: maxPositionSize(),
		MinPrice:        minPrice(),
		MaxPrice:        maxPrice(),
	}

	// Initialize NATS publisher if NATS_URL is configured
	var publisher *nats.Publisher
	natsURL := os.Getenv("NATS_URL")
	if natsURL != "" {
		var err error
		publisher, err = nats.New(natsURL)
		if err != nil {
			log.Fatalf("connect to nats: %v", err)
		}
		defer func() { _ = publisher.Close() }()
		log.Printf("NATS publisher initialized for %s", natsURL)
	} else {
		log.Println("NATS_URL not set, trade event streaming disabled")
	}

	// Create server with risk management enabled
	server := api.NewServerWithRisk(book, publisher, riskConfig, store).WithOptions(api.Options{
		AllowedOrigins:     splitList(os.Getenv("ALLOWED_ORIGINS")),
		MaxOrderQuantity:   floatEnv("MAX_ORDER_QUANTITY", 0),
		RateLimitPerSecond: floatEnv("RATE_LIMIT_PER_SECOND", 0),
		RateLimitBurst:     intEnv("RATE_LIMIT_BURST", 10),
		TrustProxy:         os.Getenv("TRUST_PROXY") == "true",
	})
	router := server.Router()

	if os.Getenv("DEMO_MODE") == "true" {
		cfg := demo.DefaultConfig(tradingPair)
		cfg.StartPrice = floatEnv("DEMO_START_PRICE", cfg.StartPrice)
		go demo.NewMarketMaker(server, cfg, time.Now().UnixNano()).Run(context.Background())
		log.Printf("demo mode: market maker quoting %s around %.2f", tradingPair, cfg.StartPrice)
	}

	addr := ":" + port()
	log.Printf("starting Cryptex API on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func port() string {
	if value := os.Getenv("PORT"); value != "" {
		return value
	}
	return "8080"
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// redisOptions reads REDIS_URL (e.g. rediss://user:pass@host:port for TLS,
// as hosted providers give) or falls back to REDIS_ADDR / REDIS_PASSWORD /
// REDIS_DB.
func redisOptions() (*redis.Options, error) {
	if url := os.Getenv("REDIS_URL"); url != "" {
		opts, err := redis.ParseURL(url)
		if err != nil {
			return nil, fmt.Errorf("invalid REDIS_URL: %w", err)
		}
		return opts, nil
	}
	return &redis.Options{
		Addr:     env("REDIS_ADDR", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       redisDB(),
	}, nil
}

func splitList(raw string) []string {
	var out []string
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func floatEnv(key string, fallback float64) float64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value < 0 {
		log.Fatalf("invalid %s: %q", key, raw)
	}
	return value
}

func intEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		log.Fatalf("invalid %s: %q", key, raw)
	}
	return value
}

func redisDB() int {
	raw := os.Getenv("REDIS_DB")
	if raw == "" {
		return 0
	}
	db, err := strconv.Atoi(raw)
	if err != nil {
		log.Fatalf("invalid REDIS_DB: %v", err)
	}
	return db
}

func maxPositionSize() float64 {
	raw := os.Getenv("MAX_POSITION_SIZE")
	if raw == "" {
		return 1000.0 // default
	}
	size, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		log.Fatalf("invalid MAX_POSITION_SIZE: %v", err)
	}
	return size
}

func minPrice() float64 {
	raw := os.Getenv("MIN_PRICE")
	if raw == "" {
		return 0.01 // default
	}
	price, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		log.Fatalf("invalid MIN_PRICE: %v", err)
	}
	return price
}

func maxPrice() float64 {
	raw := os.Getenv("MAX_PRICE")
	if raw == "" {
		return 1000000.0 // default
	}
	price, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		log.Fatalf("invalid MAX_PRICE: %v", err)
	}
	return price
}

func restoreOrders(book *orderbook.OrderBook, orders []*models.Order) error {
	for _, order := range orders {
		if order == nil {
			return fmt.Errorf("persisted order is nil")
		}
		if order.TradingPair != book.TradingPair {
			return fmt.Errorf("persisted order %s has trading pair %s, want %s", order.ID, order.TradingPair, book.TradingPair)
		}
		if err := book.Add(order); err != nil {
			return err
		}
	}
	return nil
}
