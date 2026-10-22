// Package api exposes the matching engine over HTTP.
package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jellyfishing2346/cryptex/internal/matching"
	"github.com/jellyfishing2346/cryptex/internal/models"
	"github.com/jellyfishing2346/cryptex/internal/nats"
	"github.com/jellyfishing2346/cryptex/internal/orderbook"
	"github.com/jellyfishing2346/cryptex/internal/risk"
	"github.com/jellyfishing2346/cryptex/internal/ws"
)

const (
	defaultDepth      = 10
	tradeLogCapacity  = 200
	defaultTradeLimit = 50
)

var (
	// ErrNotOrderOwner is returned when a user tries to cancel someone else's order.
	ErrNotOrderOwner = errors.New("order belongs to another user")
	// ErrOrderNotFound is returned when an order isn't resting on the book
	// (never existed, already filled, or already cancelled).
	ErrOrderNotFound = errors.New("order not found")
)

// RequestError is a client mistake (bad input or a rejected order), as
// opposed to a server failure; the HTTP layer maps it to 400.
type RequestError struct{ Err error }

func (e *RequestError) Error() string { return e.Err.Error() }
func (e *RequestError) Unwrap() error { return e.Err }

// Options tunes behaviour for public deployments. The zero value keeps the
// original behaviour: no CORS headers, no rate limit, no order size cap.
type Options struct {
	// AllowedOrigins lists browser origins (e.g. a separate frontend) that
	// may call the API and open the WebSocket.
	AllowedOrigins []string
	// MaxOrderQuantity caps a single order's size; 0 means no cap.
	MaxOrderQuantity float64
	// RateLimitPerSecond and RateLimitBurst limit order placement and
	// cancellation per client IP; 0 disables the limit.
	RateLimitPerSecond float64
	RateLimitBurst     int
	// TrustProxy reads the client IP from X-Forwarded-For (set when running
	// behind a hosting proxy such as Render's).
	TrustProxy bool
}

// Server owns the HTTP handlers for a single trading pair.
type Server struct {
	book       *orderbook.OrderBook
	engine     *matching.Engine
	store      OrderBookStore
	publisher  *nats.Publisher
	riskConfig *risk.Config
	hub        *ws.Hub
	trades     *tradeLog
	opts       Options
	// persistMu orders saves so a slower request can't overwrite a newer
	// snapshot with an older one.
	persistMu sync.Mutex
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
	defaultConfig := risk.DefaultConfig()
	return &Server{
		book: book, engine: engine, store: store, publisher: publisher,
		riskConfig: &defaultConfig, hub: ws.NewHub(), trades: newTradeLog(tradeLogCapacity),
	}
}

// NewServerWithRisk creates an API server with risk management enabled.
func NewServerWithRisk(book *orderbook.OrderBook, publisher *nats.Publisher, riskConfig *risk.Config, stores ...OrderBookStore) *Server {
	var store OrderBookStore
	if len(stores) > 0 {
		store = stores[0]
	}
	checker := risk.NewWithConfig(riskConfig, book)
	engine := matching.NewWithRiskChecks(book, checker)
	return &Server{
		book: book, engine: engine, store: store, publisher: publisher,
		riskConfig: riskConfig, hub: ws.NewHub(), trades: newTradeLog(tradeLogCapacity),
	}
}

// WithOptions applies deployment options and returns the server.
func (s *Server) WithOptions(opts Options) *Server {
	s.opts = opts
	return s
}

// Router returns a configured Gin engine.
func (s *Server) Router() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors(s.opts.AllowedOrigins))

	writes := []gin.HandlerFunc{}
	if s.opts.RateLimitPerSecond > 0 {
		burst := s.opts.RateLimitBurst
		if burst < 1 {
			burst = 1
		}
		writes = append(writes, newRateLimiter(s.opts.RateLimitPerSecond, burst).middleware(s.opts.TrustProxy))
	}

	// API routes
	router.GET("/healthz", s.health)
	router.POST("/orders", append(writes, s.placeOrder)...)
	router.DELETE("/orders/:id", append(writes, s.cancelOrder)...)
	router.GET("/orders", s.listOrders)
	router.GET("/orderbook", s.snapshot)
	router.GET("/trades", s.recentTrades)
	router.GET("/ws", ws.Handler(s.hub, s.opts.AllowedOrigins...))

	// Serve static files for dashboard
	router.Static("/assets", "./web/assets")
	router.GET("/", func(c *gin.Context) {
		c.File("./web/dashboard.html")
	})

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
	result, err := s.PlaceOrder(c.Request.Context(), req.TradingPair, req.Side, req.Type, req.Price, req.Quantity, req.UserID)
	if err != nil {
		var reqErr *RequestError
		if errors.As(err, &reqErr) {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, result)
}

// PlaceOrder validates, matches, persists and broadcasts an order. HTTP
// requests and the demo market maker both go through here, so every order
// gets the same checks and every trade reaches the feed.
func (s *Server) PlaceOrder(ctx context.Context, tradingPair string, side models.Side, orderType models.OrderType, price, quantity float64, userID uuid.UUID) (*matching.Result, error) {
	req := placeOrderRequest{
		TradingPair: strings.TrimSpace(tradingPair), Side: side, Type: orderType,
		Price: price, Quantity: quantity, UserID: userID,
	}
	if err := validatePlaceOrderRequest(req, s.book.TradingPair); err != nil {
		return nil, &RequestError{err}
	}
	if s.opts.MaxOrderQuantity > 0 && req.Quantity > s.opts.MaxOrderQuantity {
		return nil, &RequestError{fmt.Errorf("quantity exceeds the %g limit per order", s.opts.MaxOrderQuantity)}
	}

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
		return nil, &RequestError{err}
	}
	if err := s.persist(ctx); err != nil {
		return nil, err
	}

	if len(result.Trades) > 0 {
		s.trades.add(result.Trades...)
		for _, trade := range result.Trades {
			s.hub.Broadcast(ws.EventTypeTrade, trade)
		}
	}
	// Broadcast order book update to WebSocket clients
	s.hub.Broadcast(ws.EventTypeOrderBook, s.book.Snapshot(defaultDepth))

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
	return result, nil
}

func (s *Server) cancelOrder(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "invalid order id")
		return
	}
	userID, err := uuid.Parse(c.Query("user_id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "user_id query parameter is required")
		return
	}

	order, err := s.CancelOrder(c.Request.Context(), orderID, userID)
	switch {
	case errors.Is(err, ErrNotOrderOwner):
		respondError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrOrderNotFound):
		respondError(c, http.StatusNotFound, err.Error())
	case err != nil:
		respondError(c, http.StatusInternalServerError, err.Error())
	default:
		c.JSON(http.StatusOK, order)
	}
}

// CancelOrder cancels a resting order on behalf of its owner.
func (s *Server) CancelOrder(ctx context.Context, orderID, userID uuid.UUID) (*models.Order, error) {
	existing, ok := s.book.GetOrder(orderID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrOrderNotFound, orderID)
	}
	if existing.UserID != userID {
		return nil, ErrNotOrderOwner
	}
	order, err := s.book.Cancel(orderID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrOrderNotFound, err)
	}
	if err := s.persist(ctx); err != nil {
		return nil, err
	}
	// Broadcast order book update to WebSocket clients
	s.hub.Broadcast(ws.EventTypeOrderBook, s.book.Snapshot(defaultDepth))
	return order, nil
}

// listOrders returns a user's resting orders, newest first.
func (s *Server) listOrders(c *gin.Context) {
	userID, err := uuid.Parse(c.Query("user_id"))
	if err != nil {
		respondError(c, http.StatusBadRequest, "user_id query parameter is required")
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": s.OrdersFor(userID)})
}

// OrdersFor returns a user's resting orders, newest first.
func (s *Server) OrdersFor(userID uuid.UUID) []*models.Order {
	mine := []*models.Order{}
	for _, o := range s.book.RestingOrders() {
		if o.UserID == userID {
			mine = append(mine, o)
		}
	}
	sort.Slice(mine, func(i, j int) bool { return mine[i].CreatedAt.After(mine[j].CreatedAt) })
	return mine
}

// RestingOrders returns every order currently on the book.
func (s *Server) RestingOrders() []*models.Order {
	return s.book.RestingOrders()
}

// LastTradePrice returns the most recent trade price, or 0 if none yet.
func (s *Server) LastTradePrice() float64 {
	if t := s.trades.last(); t != nil {
		return t.Price
	}
	return 0
}

func (s *Server) recentTrades(c *gin.Context) {
	limit := defaultTradeLimit
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > tradeLogCapacity {
			respondError(c, http.StatusBadRequest, fmt.Sprintf("limit must be between 1 and %d", tradeLogCapacity))
			return
		}
		limit = n
	}
	c.JSON(http.StatusOK, gin.H{"trades": s.trades.recent(limit)})
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
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	if err := s.store.Save(ctx, s.book.RestingOrders()); err != nil {
		return fmt.Errorf("persist order book: %w", err)
	}
	return nil
}
