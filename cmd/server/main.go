package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jellyfishing2346/cryptex/internal/api"
	"github.com/jellyfishing2346/cryptex/internal/matching"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
	"github.com/jellyfishing2346/cryptex/internal/persistence"
	"github.com/redis/go-redis/v9"
)

func main() {
	tradingPair := env("TRADING_PAIR", "BTC-USD")
	book := orderbook.New(tradingPair)
	client := redis.NewClient(&redis.Options{
		Addr:     env("REDIS_ADDR", "localhost:6379"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       redisDB(),
	})
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
	if err := restoreOrders(book, orders); err != nil {
		log.Fatalf("restore order book: %v", err)
	}
	log.Printf("loaded %d resting orders for %s", len(orders), tradingPair)

	engine := matching.New(book)
	router := api.NewServer(book, engine, store).Router()

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
