# Cryptex Deployment Guide

## Week 7 & 8 Features

### Risk Management (Week 7)
- **Position Limits**: Users cannot exceed maximum position size per trading pair
- **Self-Trade Prevention**: Orders are rejected if they would trade against the user's own orders
- **Price Collars**: Orders are rejected if price is outside configured min/max bounds

### Dashboard (Week 8)
- **Live Order Book Visualization**: Real-time display of bids and asks
- **WebSocket Integration**: Real-time updates via WebSocket connection
- **Statistics**: Spread, total volume, best bid/ask prices

## Environment Variables

### Risk Management
- `MAX_POSITION_SIZE`: Maximum position size per user (default: 1000.0)
- `MIN_PRICE`: Minimum allowed price (default: 0.01)
- `MAX_PRICE`: Maximum allowed price (default: 1000000.0)

### Infrastructure
- `TRADING_PAIR`: Trading pair to track (default: BTC-USD)
- `REDIS_ADDR`: Redis server address (default: localhost:6379)
- `REDIS_PASSWORD`: Redis password (optional)
- `REDIS_DB`: Redis database number (default: 0)
- `NATS_URL`: NATS server URL for trade event streaming (optional)
- `PORT`: Server port (default: 8080)

## Local Development

### Prerequisites
- Go 1.26+
- Redis server
- NATS server (optional, for trade event streaming)

### Running with Docker Compose
```bash
docker-compose up
```

This will start:
- Redis on port 6379
- NATS on port 4222
- Cryptex API on port 8080

### Running Locally
```bash
# Start Redis
redis-server

# Start NATS (optional)
nats-server -js

# Build and run
go build -o cryptex ./cmd/server
./cryptex
```

### Accessing the Dashboard
Open your browser to: `http://localhost:8080`

## Production Deployment

### Docker Build
```bash
docker build -t cryptex:latest .
```

### Kubernetes Deployment
```bash
kubectl apply -f deploy/production.yaml
```

This will deploy:
- Redis (1 replica)
- NATS (1 replica)
- Cryptex (3 replicas with LoadBalancer)

### Manual Deployment
1. Build the binary:
```bash
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cryptex ./cmd/server
```

2. Copy binary and web assets to server
3. Set environment variables
4. Run the binary

## API Endpoints

### Health Check
```
GET /healthz
```

### Place Order
```
POST /orders
Content-Type: application/json

{
  "trading_pair": "BTC-USD",
  "side": "buy",
  "type": "limit",
  "price": 50000.0,
  "quantity": 1.0,
  "user_id": "uuid"
}
```

### Cancel Order
```
DELETE /orders/:id
```

### Order Book Snapshot
```
GET /orderbook?depth=10
```

### WebSocket
```
WS /ws
```

## Testing

### Run All Tests
```bash
go test ./...
```

### Run Specific Tests
```bash
go test ./internal/risk/...
go test ./internal/api/... -run Integration
```

## Risk Check Examples

### Position Limit Exceeded
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "limit",
    "price": 50000.0,
    "quantity": 1500.0,
    "user_id": "user-uuid"
  }'
# Response: 400 Bad Request - "position limit exceeded"
```

### Self-Trade Prevention
```bash
# First place a sell order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "sell",
    "type": "limit",
    "price": 50000.0,
    "quantity": 1.0,
    "user_id": "user-uuid"
  }'

# Then try to place a crossing buy order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "limit",
    "price": 51000.0,
    "quantity": 1.0,
    "user_id": "user-uuid"
  }'
# Response: 400 Bad Request - "self-trade prevention triggered"
```

### Price Collar Violation
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "limit",
    "price": 0.001,
    "quantity": 1.0,
    "user_id": "user-uuid"
  }'
# Response: 400 Bad Request - "price below minimum collar"
```
