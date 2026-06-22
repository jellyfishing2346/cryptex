// Package persistence stores and loads order book state.
package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/redis/go-redis/v9"
)

// RedisOrderBookStore persists resting orders for one trading pair.
type RedisOrderBookStore struct {
	client *redis.Client
	key    string
}

// NewRedisOrderBookStore creates a Redis-backed order book store.
func NewRedisOrderBookStore(client *redis.Client, tradingPair string) *RedisOrderBookStore {
	return &RedisOrderBookStore{
		client: client,
		key:    fmt.Sprintf("cryptex:orderbook:%s:orders", tradingPair),
	}
}

// Save replaces the persisted order book with the given resting orders.
func (s *RedisOrderBookStore) Save(ctx context.Context, orders []*models.Order) error {
	if orders == nil {
		orders = make([]*models.Order, 0)
	}

	payload, err := json.Marshal(orders)
	if err != nil {
		return fmt.Errorf("marshal order book: %w", err)
	}
	if err := s.client.Set(ctx, s.key, payload, 0).Err(); err != nil {
		return fmt.Errorf("save order book to redis: %w", err)
	}
	return nil
}

// Load returns the persisted resting orders. A missing key is an empty book.
func (s *RedisOrderBookStore) Load(ctx context.Context) ([]*models.Order, error) {
	payload, err := s.client.Get(ctx, s.key).Bytes()
	if errors.Is(err, redis.Nil) {
		return []*models.Order{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load order book from redis: %w", err)
	}

	var orders []*models.Order
	if err := json.Unmarshal(payload, &orders); err != nil {
		return nil, fmt.Errorf("unmarshal order book: %w", err)
	}
	if orders == nil {
		orders = []*models.Order{}
	}
	return orders, nil
}
