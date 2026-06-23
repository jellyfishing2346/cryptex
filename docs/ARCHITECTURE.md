# Cryptex Architecture Documentation

## Overview

Cryptex is a high-performance cryptocurrency exchange matching engine built in Go. It implements a price-time priority (FIFO) matching algorithm with real-time trade event streaming, persistent storage, and comprehensive risk management.

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Cryptex Exchange                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐      │
│  │   REST API   │    │  WebSocket   │    │   Dashboard  │      │
│  │   (Gin)      │    │   (Gorilla)  │    │   (HTML/JS)   │      │
│  └──────┬───────┘    └──────┬───────┘    └──────┬───────┘      │
│         │                   │                   │              │
│         └───────────────────┼───────────────────┘              │
│                             │                                  │
│                    ┌────────▼────────┐                         │
│                    │  API Router     │                         │
│                    │  (Order mgmt)   │                         │
│                    └────────┬────────┘                         │
│                             │                                  │
│                    ┌────────▼────────┐                         │
│                    │ Matching Engine │                         │
│                    │ (Price-Time)    │                         │
│                    └────────┬────────┘                         │
│                             │                                  │
│         ┌───────────────────┼───────────────────┐              │
│         │                   │                   │              │
│  ┌──────▼──────┐    ┌──────▼──────┐    ┌──────▼──────┐       │
│  │ Order Book  │    │Risk Checker │    │ NATS Pub.   │       │
│  │ (Bids/Asks) │    │ (Limits)    │    │ (Events)    │       │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘       │
│         │                   │                   │              │
│  ┌──────▼──────┐    ┌──────▼──────┐                   │       │
│  │ Redis Store │    │             │                   │       │
│  │ (Persistence│    │             │                   │       │
│  └─────────────┘    │             │                   │       │
│                     │             │                   │       │
└─────────────────────┴─────────────┴───────────────────┘       │
                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Order Book (`internal/orderbook`)

The order book is the heart of the matching engine, implementing a price-time priority data structure.

**Key Features:**
- **Two-sided book**: Bids (buy orders) and Asks (sell orders)
- **Price-time priority**: Orders sorted by price, then by time (FIFO)
- **Thread-safe**: Uses `sync.RWMutex` for concurrent access
- **O(1) order lookup**: Hash map index for order ID lookups

**Data Structure:**
```go
type OrderBook struct {
    mu          sync.RWMutex
    TradingPair string
    bids        []*PriceLevel  // Sorted descending (highest first)
    asks        []*PriceLevel  // Sorted ascending (lowest first)
    orders      map[uuid.UUID]*models.Order  // O(1) lookup
}

type PriceLevel struct {
    Price  float64
    Orders []*models.Order  // Sorted by CreatedAt (FIFO)
}
```

**Key Operations:**
- `Add(order)`: Insert limit order into appropriate price level
- `Cancel(orderID)`: Remove order from book
- `Snapshot(depth)`: Get current state aggregated by price
- `BestBid()/BestAsk()`: Get best prices on each side
- `Spread()`: Calculate bid-ask spread

### 2. Matching Engine (`internal/matching`)

The matching engine implements the core trading logic using price-time priority matching.

**Algorithm:**
1. When a new order arrives, check the opposite side of the book for crossing orders
2. Execute trades at the resting order's price (standard exchange behavior)
3. Process orders FIFO within each price level
4. For limit orders, add unfilled remainder to the book
5. Market orders are never booked - unfilled remainder is dropped

**Key Features:**
- **Atomic operations**: Single lock for entire matching cycle
- **Trade generation**: Creates trade records for each fill
- **Order status updates**: Open → Partial → Filled
- **Risk checks integration**: Pre-match validation

**Match Logic:**
```go
func (e *Engine) match(incoming *models.Order) []*models.Trade {
    // 1. Get opposite side price levels
    // 2. For each level that crosses:
    //    a. Match against orders FIFO
    //    b. Execute at resting order's price
    //    c. Update fill quantities
    //    d. Create trade records
    // 3. Remove fully filled orders
    // 4. Return all trades executed
}
```

### 3. Risk Management (`internal/risk`)

Comprehensive risk checks before order acceptance to prevent problematic trades.

**Risk Checks:**
1. **Position Limits**: Maximum position size per user per trading pair
2. **Self-Trade Prevention**: Prevents users from trading against themselves
3. **Price Collars**: Rejects orders outside min/max price bounds

**Configuration:**
```go
type Config struct {
    MaxPositionSize float64  // Default: 1000.0
    MinPrice        float64  // Default: 0.01
    MaxPrice        float64  // Default: 1000000.0
}
```

**Check Sequence:**
1. Position limit check (calculates current user position)
2. Self-trade prevention (scans opposite side for user's orders)
3. Price collar validation (for limit orders only)

### 4. API Layer (`internal/api`)

REST API built with Gin framework for order management and market data.

**Endpoints:**
- `POST /orders`: Place new order (limit or market)
- `DELETE /orders/:id`: Cancel existing order
- `GET /orderbook`: Get order book snapshot with depth
- `GET /healthz`: Health check endpoint
- `GET /ws`: WebSocket upgrade endpoint
- `GET /`: Web dashboard

**Request/Response Flow:**
1. Validate request (trading pair, side, type, price, quantity)
2. Create order with UUID and timestamps
3. Submit to matching engine
4. Persist to Redis if successful
5. Broadcast WebSocket update
6. Publish to NATS if trades executed
7. Return result to client

### 5. Persistence Layer (`internal/persistence`)

Redis-based persistence for durable order book storage.

**Storage Strategy:**
- Single key per trading pair: `cryptex:orderbook:{pair}:orders`
- JSON serialization of all resting orders
- Full snapshot on each state change
- Automatic restoration on startup

**Operations:**
- `Save(ctx, orders)`: Persist current order book state
- `Load(ctx)`: Restore order book from Redis

### 6. WebSocket Real-time Feed (`internal/ws`)

Real-time order book updates using WebSocket connections.

**Architecture:**
- **Hub pattern**: Central coordinator for all connections
- **Broadcast model**: Single message to all connected clients
- **Event types**: Order book updates, trade notifications
- **Connection management**: Ping/pong keepalive, graceful cleanup

**Message Flow:**
1. Client connects via WebSocket upgrade
2. Registered with hub
3. Receives real-time order book updates
4. Automatic reconnection handling

### 7. NATS Event Streaming (`internal/nats`)

Publish-subscribe messaging for trade event distribution.

**Subjects:**
- `trades.{trading_pair}`: Raw trade data
- `trade-events.{trading_pair}`: Structured events with metadata
- `orders.{trading_pair}`: Order lifecycle events

**Event Structure:**
```go
type TradeEvent struct {
    Trade     *models.Trade
    Timestamp time.Time
    EventType string  // "trade.executed"
}
```

**Use Cases:**
- Real-time analytics
- Notification systems
- Audit logging
- Downstream processing

## Data Models

### Order Model
```go
type Order struct {
    ID          uuid.UUID
    TradingPair string      // e.g., "BTC-USD"
    Side        Side        // "buy" or "sell"
    Type        OrderType   // "limit" or "market"
    Price       float64     // 0 for market orders
    Quantity    float64     // Original order quantity
    Filled      float64     // Quantity filled so far
    Status      OrderStatus // "open", "partial", "filled", "cancelled"
    UserID      uuid.UUID
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### Trade Model
```go
type Trade struct {
    ID          uuid.UUID
    TradingPair string
    BuyOrderID  uuid.UUID
    SellOrderID uuid.UUID
    Price       float64
    Quantity    float64
    ExecutedAt  time.Time
}
```

### Order Book Snapshot
```go
type OrderBookSnapshot struct {
    TradingPair string
    Bids        []OrderBookLevel  // Highest price first
    Asks        []OrderBookLevel  // Lowest price first
    Timestamp   time.Time
}

type OrderBookLevel struct {
    Price    float64
    Quantity float64  // Total quantity at this price
    Orders   int      // Number of orders at this price
}
```

## Concurrency Model

### Locking Strategy
- **Order Book**: Read-write mutex for concurrent access
- **Matching Engine**: Single write lock for entire match cycle
- **Risk Checker**: Read lock for position calculations
- **WebSocket Hub**: Channel-based communication (no locks)

### Thread Safety Guarantees
- Order book operations are thread-safe
- Matching engine processes orders atomically
- WebSocket broadcasts are concurrent-safe
- Redis operations use context for cancellation

## Performance Characteristics

### Time Complexity
- Order addition: O(n) where n = number of price levels
- Order cancellation: O(n) for price level search
- Order lookup: O(1) via hash map
- Matching: O(m) where m = number of crossed orders
- Snapshot generation: O(d) where d = requested depth

### Memory Usage
- Linear in number of orders and price levels
- Hash map adds constant overhead per order
- WebSocket connections buffered per client

## Deployment Architecture

### Single Instance
```
┌─────────────────────────────────────┐
│         Single Server                │
│  ┌─────────┐    ┌─────────┐         │
│  │ Cryptex │    │ Redis   │         │
│  │ API     │◄──►│ (local) │         │
│  └─────────┘    └─────────┘         │
│       │                               │
│       └──► NATS (optional)           │
└─────────────────────────────────────┘
```

### Production Deployment
```
                    ┌─────────────┐
                    │   Load Bal. │
                    └──────┬──────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
    ┌─────▼─────┐    ┌─────▼─────┐    ┌─────▼─────┐
    │  Cryptex  │    │  Cryptex  │    │  Cryptex  │
    │  Instance │    │  Instance │    │  Instance │
    └─────┬─────┘    └─────┬─────┘    └─────┬─────┘
          │                │                │
          └────────────────┼────────────────┘
                           │
                    ┌──────▼──────┐
                    │   Redis     │
                    │  Cluster    │
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │    NATS     │
                    │   Cluster   │
                    └─────────────┘
```

## Extension Points

### Adding New Order Types
1. Extend `OrderType` enum in `models/order.go`
2. Add matching logic in `matching/engine.go`
3. Update validation in `api/router.go`

### Additional Risk Checks
1. Add check method in `risk/checks.go`
2. Call from `CheckOrder()` method
3. Add configuration to `Config` struct

### Custom Persistence
1. Implement `OrderBookStore` interface
2. Add constructor in `persistence` package
3. Pass to API server constructor

### Alternative Message Broker
1. Create publisher package similar to `nats`
2. Implement trade publishing interface
3. Integrate in API layer

## Monitoring and Observability

### Key Metrics
- Order latency (submission to match)
- Match rate (orders per second)
- Trade rate (trades per second)
- WebSocket connection count
- Redis operation latency
- NATS publish latency

### Logging Points
- Order submission and result
- Trade execution
- Risk check failures
- WebSocket connection events
- Persistence operations
- External service errors

## Security Considerations

### Input Validation
- All API inputs validated before processing
- UUID validation for order and user IDs
- Price and quantity range checks
- Trading pair whitelist

### Risk Controls
- Position limits prevent excessive exposure
- Self-trade prevention avoids accidental conflicts
- Price collars prevent extreme price manipulation
- Order size limits prevent market disruption

### Operational Security
- Redis password support
- No secrets in logs
- Context-based cancellation
- Graceful shutdown handling

## Testing Strategy

### Unit Tests
- Order book operations (add, cancel, snapshot)
- Matching engine logic (various scenarios)
- Risk check validation
- Model serialization

### Integration Tests
- API endpoint testing
- Redis persistence
- WebSocket connectivity
- NATS publishing

### Test Coverage
- Core matching algorithm: >95%
- Risk management: >90%
- API layer: >85%
- Persistence: >80%
