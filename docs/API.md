# Cryptex API Documentation

## Base URL

```
http://localhost:8080
```

## Authentication

Currently, Cryptex does not implement authentication. All endpoints are publicly accessible. In production, you should add authentication middleware.

## Content Type

All API endpoints use `Content-Type: application/json` for requests and responses.

## Response Format

### Success Response
```json
{
  "data": { ... }
}
```

### Error Response
```json
{
  "error": "Error message description"
}
```

## HTTP Status Codes

- `200 OK`: Request succeeded
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

---

## Endpoints

### Health Check

Check if the API server is running.

**Endpoint:** `GET /healthz`

**Response:**
```json
{
  "status": "ok"
}
```

**Example:**
```bash
curl http://localhost:8080/healthz
```

---

### Place Order

Submit a new order to the matching engine.

**Endpoint:** `POST /orders`

**Request Body:**
```json
{
  "trading_pair": "BTC-USD",
  "side": "buy",
  "type": "limit",
  "price": 50000.0,
  "quantity": 1.5,
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

**Request Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `trading_pair` | string | Yes | Trading pair (e.g., "BTC-USD") |
| `side` | string | Yes | Order side: "buy" or "sell" |
| `type` | string | Yes | Order type: "limit" or "market" |
| `price` | float64 | Condition | Price (required for limit orders, must be 0 for market) |
| `quantity` | float64 | Yes | Order quantity (must be positive) |
| `user_id` | string | Yes | User UUID identifier |

**Response (Success):**
```json
{
  "order": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "limit",
    "price": 50000.0,
    "quantity": 1.5,
    "filled": 0.0,
    "status": "open",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2024-06-23T00:00:00Z",
    "updated_at": "2024-06-23T00:00:00Z"
  },
  "trades": []
}
```

**Response (With Trades):**
```json
{
  "order": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "limit",
    "price": 50000.0,
    "quantity": 1.5,
    "filled": 1.5,
    "status": "filled",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2024-06-23T00:00:00Z",
    "updated_at": "2024-06-23T00:00:01Z"
  },
  "trades": [
    {
      "id": "660e8400-e29b-41d4-a716-446655440002",
      "trading_pair": "BTC-USD",
      "buy_order_id": "550e8400-e29b-41d4-a716-446655440001",
      "sell_order_id": "550e8400-e29b-41d4-a716-446655440000",
      "price": 50000.0,
      "quantity": 1.5,
      "executed_at": "2024-06-23T00:00:01Z"
    }
  ]
}
```

**Error Responses:**

**400 Bad Request - Invalid JSON:**
```json
{
  "error": "invalid JSON body"
}
```

**400 Bad Request - Validation Error:**
```json
{
  "error": "trading_pair is required"
}
```

**400 Bad Request - Risk Check Failed:**
```json
{
  "error": "position limit exceeded: current position 500.00, new position 1500.00 exceeds limit 1000.00"
}
```

**Example:**
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "limit",
    "price": 50000.0,
    "quantity": 1.5,
    "user_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

---

### Cancel Order

Cancel an existing open order.

**Endpoint:** `DELETE /orders/:id`

**URL Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | string | Yes | Order UUID |

**Response (Success):**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440001",
  "trading_pair": "BTC-USD",
  "side": "buy",
  "type": "limit",
  "price": 50000.0,
  "quantity": 1.5,
  "filled": 0.0,
  "status": "cancelled",
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "created_at": "2024-06-23T00:00:00Z",
  "updated_at": "2024-06-23T00:01:00Z"
}
```

**Error Responses:**

**400 Bad Request - Invalid UUID:**
```json
{
  "error": "invalid order id"
}
```

**404 Not Found - Order Not Found:**
```json
{
  "error": "order 550e8400-e29b-41d4-a716-446655440001 not found"
}
```

**404 Not Found - Already Filled:**
```json
{
  "error": "order 550e8400-e29b-41d4-a716-446655440001 is already filled"
}
```

**404 Not Found - Already Cancelled:**
```json
{
  "error": "order 550e8400-e29b-41d4-a716-446655440001 is already cancelled"
}
```

**Example:**
```bash
curl -X DELETE http://localhost:8080/orders/550e8400-e29b-41d4-a716-446655440001
```

---

### Order Book Snapshot

Get the current state of the order book.

**Endpoint:** `GET /orderbook`

**Query Parameters:**

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `depth` | integer | No | 10 | Number of price levels to return (0 for all) |

**Response:**
```json
{
  "trading_pair": "BTC-USD",
  "bids": [
    {
      "price": 49900.0,
      "quantity": 2.5,
      "orders": 3
    },
    {
      "price": 49800.0,
      "quantity": 5.0,
      "orders": 5
    }
  ],
  "asks": [
    {
      "price": 50100.0,
      "quantity": 1.5,
      "orders": 2
    },
    {
      "price": 50200.0,
      "quantity": 3.0,
      "orders": 4
    }
  ],
  "timestamp": "2024-06-23T00:00:00Z"
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `trading_pair` | string | Trading pair identifier |
| `bids` | array | Buy orders (highest price first) |
| `asks` | array | Sell orders (lowest price first) |
| `timestamp` | string | Snapshot timestamp (ISO 8601) |

**Price Level Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `price` | float64 | Price level |
| `quantity` | float64 | Total quantity at this price |
| `orders` | integer | Number of orders at this price |

**Example:**
```bash
# Get default depth (10 levels)
curl http://localhost:8080/orderbook

# Get custom depth
curl http://localhost:8080/orderbook?depth=20

# Get all levels
curl http://localhost:8080/orderbook?depth=0
```

---

### WebSocket Connection

Connect to real-time order book updates via WebSocket.

**Endpoint:** `WS /ws`

**Connection:**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Order book update:', data);
};
```

**Message Format:**
```json
{
  "type": "orderbook",
  "data": {
    "trading_pair": "BTC-USD",
    "bids": [
      {
        "price": 49900.0,
        "quantity": 2.5,
        "orders": 3
      }
    ],
    "asks": [
      {
        "price": 50100.0,
        "quantity": 1.5,
        "orders": 2
      }
    ],
    "timestamp": "2024-06-23T00:00:00Z"
  }
}
```

**Event Types:**
- `orderbook`: Order book snapshot updates

**Connection Management:**
- Automatic ping/pong keepalive (60-second interval)
- Graceful reconnection handling
- Connection cleanup on disconnect

**Example (JavaScript):**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

ws.onopen = () => {
  console.log('WebSocket connected');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);

  if (message.type === 'orderbook') {
    const orderbook = message.data;
    console.log('Best bid:', orderbook.bids[0]?.price);
    console.log('Best ask:', orderbook.asks[0]?.price);
    console.log('Spread:', orderbook.asks[0]?.price - orderbook.bids[0]?.price);
  }
};

ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};

ws.onclose = () => {
  console.log('WebSocket disconnected');
  // Implement reconnection logic
};
```

**Example (Python):**
```python
import websocket
import json

def on_message(ws, message):
    data = json.loads(message)
    if data['type'] == 'orderbook':
        orderbook = data['data']
        print(f"Best bid: {orderbook['bids'][0]['price']}")
        print(f"Best ask: {orderbook['asks'][0]['price']}")

def on_error(ws, error):
    print(f"Error: {error}")

def on_close(ws, close_status_code, close_msg):
    print("WebSocket closed")

def on_open(ws):
    print("WebSocket connected")

ws = websocket.WebSocketApp(
    "ws://localhost:8080/ws",
    on_open=on_open,
    on_message=on_message,
    on_error=on_error,
    on_close=on_close
)

ws.run_forever()
```

---

### Web Dashboard

Access the web-based trading dashboard.

**Endpoint:** `GET /`

**Description:**
Returns the HTML dashboard interface for real-time order book visualization.

**Access:**
```
http://localhost:8080
```

**Features:**
- Live order book display
- Real-time WebSocket updates
- Price spread visualization
- Trade statistics
- Order placement form

---

## Order Types

### Limit Orders

Limit orders are added to the order book at a specified price. They execute only when a matching order is available at or better than the specified price.

**Characteristics:**
- Must specify a positive price
- Added to the order book if not immediately filled
- Remain in the book until filled or cancelled
- Execute at the specified price or better

**Example:**
```json
{
  "trading_pair": "BTC-USD",
  "side": "buy",
  "type": "limit",
  "price": 50000.0,
  "quantity": 1.0,
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Market Orders

Market orders execute immediately at the best available price(s). They do not remain in the order book.

**Characteristics:**
- Price must be 0
- Execute immediately against available liquidity
- May partially fill if insufficient liquidity
- Unfilled portion is cancelled (not added to book)
- May execute at multiple price levels

**Example:**
```json
{
  "trading_pair": "BTC-USD",
  "side": "buy",
  "type": "market",
  "price": 0.0,
  "quantity": 1.0,
  "user_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

---

## Order Status Flow

```
┌─────────┐
│  Open   │◄─────── Order placed
└────┬────┘
     │
     ├─────────── Partial fill ────┐
     │                             │
     ▼                             │
┌─────────┐                        │
│ Partial │                        │
└────┬────┘                        │
     │                             │
     ├─────────── Full fill ───────┤
     │                             │
     ▼                             │
┌─────────┐                        │
│ Filled  │                        │
└─────────┘                        │
                                   │
     ┌─────────────────────────────┘
     │
     │ Cancel
     ▼
┌─────────────┐
│  Cancelled  │
└─────────────┘
```

**Status Descriptions:**
- `open`: Order is active in the book with no fills
- `partial`: Order has been partially filled
- `filled`: Order has been completely filled
- `cancelled`: Order has been cancelled by user

---

## Risk Management

### Position Limits

Users cannot exceed maximum position size per trading pair.

**Error Response:**
```json
{
  "error": "position limit exceeded: current position 500.00, new position 1500.00 exceeds limit 1000.00"
}
```

**Configuration:**
- Environment variable: `MAX_POSITION_SIZE`
- Default: 1000.0

### Self-Trade Prevention

Orders are rejected if they would trade against the user's own orders.

**Error Response:**
```json
{
  "error": "self-trade prevention triggered: order would cross with user's own order 550e8400-e29b-41d4-a716-446655440001"
}
```

### Price Collars

Orders are rejected if price is outside configured min/max bounds.

**Error Response:**
```json
{
  "error": "price below minimum collar: price 0.001 is below minimum 0.01"
}
```

**Configuration:**
- Environment variables: `MIN_PRICE`, `MAX_PRICE`
- Defaults: 0.01, 1000000.0

---

## Error Handling

### Common Errors

**Invalid Trading Pair:**
```json
{
  "error": "unsupported trading_pair"
}
```

**Invalid Side:**
```json
{
  "error": "side must be buy or sell"
}
```

**Invalid Order Type:**
```json
{
  "error": "type must be limit or market"
}
```

**Invalid Price:**
```json
{
  "error": "limit order price must be positive"
}
```

**Invalid Quantity:**
```json
{
  "error": "quantity must be positive"
}
```

**Missing User ID:**
```json
{
  "error": "user_id is required"
}
```

---

## Rate Limiting

Currently, Cryptex does not implement rate limiting. In production, you should add rate limiting middleware to prevent abuse.

## CORS

CORS is currently open for all origins. In production, configure appropriate CORS restrictions.

## Examples

### Complete Trading Flow

```bash
# 1. Check current order book
curl http://localhost:8080/orderbook

# 2. Place a limit buy order
BUY_ORDER=$(curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "limit",
    "price": 49000.0,
    "quantity": 1.0,
    "user_id": "550e8400-e29b-41d4-a716-446655440000"
  }')

echo $BUY_ORDER | jq .

# 3. Place a crossing limit sell order
SELL_ORDER=$(curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "sell",
    "type": "limit",
    "price": 49000.0,
    "quantity": 1.0,
    "user_id": "660e8400-e29b-41d4-a716-446655440000"
  }')

echo $SELL_ORDER | jq .

# 4. Check updated order book
curl http://localhost:8080/orderbook

# 5. Cancel remaining order (if any)
ORDER_ID=$(echo $BUY_ORDER | jq -r '.order.id')
curl -X DELETE http://localhost:8080/orders/$ORDER_ID
```

### Market Order Example

```bash
# Place a market buy order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "market",
    "price": 0.0,
    "quantity": 0.5,
    "user_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

### WebSocket Integration Example

```javascript
// Complete WebSocket client with reconnection
class CryptexWebSocket {
  constructor(url) {
    this.url = url;
    this.ws = null;
    this.reconnectInterval = 5000;
    this.connect();
  }

  connect() {
    this.ws = new WebSocket(this.url);

    this.ws.onopen = () => {
      console.log('Connected to Cryptex WebSocket');
    };

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data);
      this.handleMessage(message);
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected, reconnecting...');
      setTimeout(() => this.connect(), this.reconnectInterval);
    };
  }

  handleMessage(message) {
    switch (message.type) {
      case 'orderbook':
        this.updateOrderBook(message.data);
        break;
      default:
        console.log('Unknown message type:', message.type);
    }
  }

  updateOrderBook(orderbook) {
    const bestBid = orderbook.bids[0];
    const bestAsk = orderbook.asks[0];

    if (bestBid && bestAsk) {
      const spread = bestAsk.price - bestBid.price;
      console.log(`Spread: $${spread.toFixed(2)}`);
    }
  }
}

// Usage
const client = new CryptexWebSocket('ws://localhost:8080/ws');
```

---

## Testing the API

### Using cURL

```bash
# Health check
curl http://localhost:8080/healthz

# Place order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "limit",
    "price": 50000.0,
    "quantity": 1.0,
    "user_id": "550e8400-e29b-41d4-a716-446655440000"
  }'

# Get order book
curl http://localhost:8080/orderbook?depth=5

# Cancel order
curl -X DELETE http://localhost:8080/orders/ORDER_ID
```

### Using HTTPie

```bash
# Health check
http GET localhost:8080/healthz

# Place order
http POST localhost:8080/orders \
  trading_pair=BTC-USD \
  side=buy \
  type=limit \
  price:=50000.0 \
  quantity:=1.0 \
  user_id=550e8400-e29b-41d4-a716-446655440000

# Get order book
http GET localhost:8080/orderbook depth==5

# Cancel order
http DELETE localhost:8080/orders/ORDER_ID
```

---

## Performance Considerations

### Expected Latency
- Order submission to match: <1ms
- Order book snapshot: <1ms
- WebSocket message propagation: <5ms

### Throughput
- Orders per second: 10,000+ (single instance)
- Concurrent WebSocket connections: 1,000+

### Optimization Tips
- Use WebSocket instead of polling for real-time updates
- Limit order book depth for faster responses
- Batch order placement when possible
- Use connection pooling for HTTP clients
