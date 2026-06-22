package persistence_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/persistence"
	"github.com/redis/go-redis/v9"
)

func TestRedisOrderBookStoreSaveLoad(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := persistence.NewRedisOrderBookStore(client, "BTC-USD")

	now := time.Now().UTC()
	orders := []*models.Order{
		{
			ID:          uuid.New(),
			TradingPair: "BTC-USD",
			Side:        models.SideBuy,
			Type:        models.OrderTypeLimit,
			Price:       50000,
			Quantity:    2,
			Filled:      0.5,
			Status:      models.OrderStatusPartial,
			UserID:      uuid.New(),
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	if err := store.Save(context.Background(), orders); err != nil {
		t.Fatalf("save order book: %v", err)
	}

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load order book: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 loaded order, got %d", len(loaded))
	}
	if loaded[0].ID != orders[0].ID {
		t.Fatalf("expected loaded order %s, got %s", orders[0].ID, loaded[0].ID)
	}
	if loaded[0].Remaining() != 1.5 {
		t.Fatalf("expected remaining quantity 1.5, got %f", loaded[0].Remaining())
	}
}

func TestRedisOrderBookStoreLoadMissingKey(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	store := persistence.NewRedisOrderBookStore(client, "BTC-USD")

	loaded, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("load missing order book: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected empty order book, got %d orders", len(loaded))
	}
}
