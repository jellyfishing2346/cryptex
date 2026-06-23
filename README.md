# Cryptex

A high-performance cryptocurrency exchange matching engine with real-time trade event streaming.

## Features

- **Order Matching Engine**: Price-time priority matching algorithm for limit and market orders
- **REST API**: HTTP endpoints for order placement, cancellation, and order book snapshots
- **WebSocket Real-time Feed**: Live order book updates
- **Redis Persistence**: Durable storage of resting orders
- **NATS Trade Event Streaming**: Publish trade events to NATS for downstream consumers (analytics, notifications)

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   REST API  │────▶│   Matching   │────▶│   NATS      │
│             │     │   Engine     │     │   Publisher │
└─────────────┘     └──────────────┘     └─────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │   Redis      │
                     │   Storage    │
                     └──────────────┘
```

## Getting Started

### Prerequisites

- Go 1.26+
- Redis (for persistence)
- NATS (for trade event streaming, optional)

### Running with Docker Compose

Start Redis and NATS:

```bash
docker-compose -f docker/docker-compose.yml up -d
```

### Running the Server

```bash
# Build
go build -o bin/server ./cmd/server

# Run with NATS enabled
export TRADING_PAIR="BTC-USD"
export REDIS_ADDR="localhost:6379"
export NATS_URL="nats://localhost:4222"
./bin/server

# Run without NATS (trade event streaming disabled)
export TRADING_PAIR="BTC-USD"
export REDIS_ADDR="localhost:6379"
./bin/server
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TRADING_PAIR` | Trading pair to match | `BTC-USD` |
| `REDIS_ADDR` | Redis server address | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password (optional) | - |
| `REDIS_DB` | Redis database number | `0` |
| `NATS_URL` | NATS server URL (optional) | - |
| `PORT` | HTTP server port | `8080` |

## API Endpoints

### Place Order

```bash
POST /orders
Content-Type: application/json

{
  "trading_pair": "BTC-USD",
  "side": "buy",
  "type": "limit",
  "price": 50000.0,
  "quantity": 1.5,
  "user_id": "uuid-here"
}
```

### Cancel Order

```bash
DELETE /orders/:id
```

### Order Book Snapshot

```bash
GET /orderbook?depth=10
```

### Health Check

```bash
GET /healthz
```

## NATS Trade Event Streaming

When `NATS_URL` is configured, the server publishes trade events to NATS subjects:

### Trade Subjects

- `trades.<trading_pair>` - Raw trade data
- `trade-events.<trading_pair>` - Structured trade events with metadata

### Example Trade Event

```json
{
  "trade": {
    "id": "uuid",
    "trading_pair": "BTC-USD",
    "buy_order_id": "uuid",
    "sell_order_id": "uuid",
    "price": 50000.0,
    "quantity": 1.5,
    "executed_at": "2024-06-22T20:00:00Z"
  },
  "timestamp": "2024-06-22T20:00:00Z",
  "event_type": "trade.executed"
}
```

### Consuming Trade Events

```go
nc, _ := nats.Connect("nats://localhost:4222")
sub, _ := nc.Subscribe("trades.BTC-USD", func(msg *nats.Msg) {
    var trade models.Trade
    json.Unmarshal(msg.Data, &trade)
    // Process trade
})
```

## Testing

Run all tests:

```bash
go test ./...
```

Run specific package tests:

```bash
go test ./internal/matching/...
go test ./internal/nats/...
```

## Development

### Project Structure

```
cmd/server/          # Main application entry point
internal/
  api/               # HTTP handlers and routing
  matching/          # Order matching engine
  models/            # Data models (Order, Trade, etc.)
  nats/              # NATS publisher for trade events
  orderbook/         # Order book data structure
  persistence/       # Redis storage layer
  ws/                # WebSocket real-time feed
docker/              # Docker Compose configuration
```

### Week-by-Week Implementation

- **Week 1**: Order book data structure
- **Week 2**: Matching engine with price-time priority
- **Week 3**: REST API for order management
- **Week 4**: Redis persistence
- **Week 5**: WebSocket real-time feed
- **Week 6**: NATS trade event streaming

## License

MIT
