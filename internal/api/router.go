// Package api exposes the matching engine over HTTP.
package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/matching"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/nats"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
)

const defaultDepth = 10

// Server owns the HTTP handlers for a single trading pair.
type Server struct {
	book      *orderbook.OrderBook
	engine    *matching.Engine
	store     OrderBookStore
	publisher *nats.Publisher
}

// OrderBookStore persists resting orders after book mutations.
type OrderBookStore interface {
	Save(ctx context.Context, orders []*models.Order) error
}

// NewServer creates an API server backed by the given book and engine.
func NewServer(book *orderbook.OrderBook, engine *matching.Engine, publisher *nats.Publisher, stores ...OrderBookStore) *Server {
	var store OrderBookStore
	if len(stores) > 0 {
		store = stores[0]
	}
	return &Server{book: book, engine: engine, store: store, publisher: publisher}
}

// Router returns a configured Gin engine.
func (s *Server) Router() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/healthz", s.health)
	router.POST("/orders", s.placeOrder)
	router.DELETE("/orders/:id", s.cancelOrder)
	router.GET("/orderbook", s.snapshot)

	return router
}

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type placeOrderRequest struct {
	TradingPair string           `json:"trading_pair"`
	Side        models.Side      `json:"side"`
	Type        models.OrderType `json:"type"`
	Price       float64          `json:"price"`
	Quantity    float64          `json:"quantity"`
	UserID      uuid.UUID        `json:"user_id"`
}

func (s *Server) placeOrder(c *gin.Context) {
	var req placeOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := validatePlaceOrderRequest(req, s.book.TradingPair); err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	req.TradingPair = strings.TrimSpace(req.TradingPair)

	now := time.Now().UTC()
	order := &models.Order{
		ID:          uuid.New(),
		TradingPair: req.TradingPair,
		Side:        req.Side,
		Type:        req.Type,
		Price:       req.Price,
		Quantity:    req.Quantity,
		Status:      models.OrderStatusOpen,
		UserID:      req.UserID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	result, err := s.engine.Submit(order)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.persist(c.Request.Context()); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Publish trades to NATS if publisher is configured
	if s.publisher != nil && len(result.Trades) > 0 {
		for _, trade := range result.Trades {
			if err := s.publisher.PublishTrade(trade); err != nil {
				log.Printf("failed to publish trade to NATS: %v", err)
			}
			// Also publish as a structured event
			event := &nats.TradeEvent{
				Trade:     trade,
				Timestamp: time.Now().UTC(),
				EventType: "trade.executed",
			}
			if err := s.publisher.PublishTradeEvent(event); err != nil {
				log.Printf("failed to publish trade event to NATS: %v", err)
			}
		}
	}

	c.JSON(http.StatusCreated, result)
}

func (s *Server) cancelOrder(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid order id")
		return
	}

	order, err := s.book.Cancel(orderID)
	if err != nil {
		respondError(c, http.StatusNotFound, err.Error())
		return
	}
	if err := s.persist(c.Request.Context()); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, order)
}

func (s *Server) snapshot(c *gin.Context) {
	depth, err := parseDepth(c.Query("depth"))
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, s.book.Snapshot(depth))
}

func validatePlaceOrderRequest(req placeOrderRequest, tradingPair string) error {
	req.TradingPair = strings.TrimSpace(req.TradingPair)
	if req.TradingPair == "" {
		return errors.New("trading_pair is required")
	}
	if req.TradingPair != tradingPair {
		return errors.New("unsupported trading_pair")
	}
	if req.Side != models.SideBuy && req.Side != models.SideSell {
		return errors.New("side must be buy or sell")
	}
	if req.Type != models.OrderTypeLimit && req.Type != models.OrderTypeMarket {
		return errors.New("type must be limit or market")
	}
	if req.Type == models.OrderTypeLimit && req.Price <= 0 {
		return errors.New("limit order price must be positive")
	}
	if req.Type == models.OrderTypeMarket && req.Price != 0 {
		return errors.New("market order price must be 0")
	}
	if req.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	if req.UserID == uuid.Nil {
		return errors.New("user_id is required")
	}
	return nil
}

func parseDepth(raw string) (int, error) {
	if raw == "" {
		return defaultDepth, nil
	}
	depth, err := strconv.Atoi(raw)
	if err != nil || depth < 0 {
		return 0, errors.New("depth must be a non-negative integer")
	}
	return depth, nil
}

func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

func (s *Server) persist(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	if err := s.store.Save(ctx, s.book.RestingOrders()); err != nil {
		return fmt.Errorf("persist order book: %w", err)
	}
	return nil
}
