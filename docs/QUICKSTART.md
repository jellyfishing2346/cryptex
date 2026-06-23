# Cryptex Quick Start Guide

This guide will help you get Cryptex up and running in minutes with practical examples.

## 🚀 Quick Setup (5 minutes)

### 1. Start with Docker (Recommended)

The fastest way to get started:

```bash
# Clone the repository
git clone https://github.com/jellyfishing2346/cryptex.git
cd cryptex

# Start all services (Redis, NATS, Cryptex)
docker-compose up

# Access the dashboard
open http://localhost:8080
```

That's it! You now have a fully functional matching engine running.

### 2. Manual Setup

If you prefer to run components manually:

```bash
# Start Redis
redis-server

# Start NATS (optional)
nats-server -js

# Build Cryptex
go build -o cryptex ./cmd/server

# Run Cryptex
export TRADING_PAIR="BTC-USD"
export REDIS_ADDR="localhost:6379"
./cryptex
```

## 📊 Your First Trade

### Using the Web Dashboard

1. Open http://localhost:8080
2. You'll see the live order book with current bids and asks
3. Use the order form to place your first trade
4. Watch the order book update in real-time

### Using cURL

**Step 1: Check the current order book**

```bash
curl http://localhost:8080/orderbook
```

**Step 2: Place a limit buy order**

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "limit",
    "price": 49000.0,
    "quantity": 1.0,
    "user_id": "550e8400-e29b-41d4-a716-446655440000"
  }'
```

Response:
```json
{
  "order": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "limit",
    "price": 49000.0,
    "quantity": 1.0,
    "filled": 0.0,
    "status": "open",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2024-06-23T00:00:00Z",
    "updated_at": "2024-06-23T00:00:00Z"
  },
  "trades": []
}
```

**Step 3: Place a crossing sell order (will execute a trade)**

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "sell",
    "type": "limit",
    "price": 49000.0,
    "quantity": 1.0,
    "user_id": "660e8400-e29b-41d4-a716-446655440000"
  }'
```

Response with trade:
```json
{
  "order": {
    "id": "660e8400-e29b-41d4-a716-446655440002",
    "trading_pair": "BTC-USD",
    "side": "sell",
    "type": "limit",
    "price": 49000.0,
    "quantity": 1.0,
    "filled": 1.0,
    "status": "filled",
    "user_id": "660e8400-e29b-41d4-a716-446655440000",
    "created_at": "2024-06-23T00:00:01Z",
    "updated_at": "2024-06-23T00:00:01Z"
  },
  "trades": [
    {
      "id": "770e8400-e29b-41d4-a716-446655440003",
      "trading_pair": "BTC-USD",
      "buy_order_id": "550e8400-e29b-41d4-a716-446655440001",
      "sell_order_id": "660e8400-e29b-41d4-a716-446655440002",
      "price": 49000.0,
      "quantity": 1.0,
      "executed_at": "2024-06-23T00:00:01Z"
    }
  ]
}
```

**Step 4: Check the updated order book**

```bash
curl http://localhost:8080/orderbook
```

## 🔌 WebSocket Real-time Updates

### JavaScript Example

Create a file `websocket-client.html`:

```html
<!DOCTYPE html>
<html>
<head>
    <title>Cryptex WebSocket Client</title>
</head>
<body>
    <h1>Cryptex Order Book</h1>
    <div id="orderbook"></div>

    <script>
        const ws = new WebSocket('ws://localhost:8080/ws');
        const orderbookDiv = document.getElementById('orderbook');

        ws.onopen = () => {
            console.log('Connected to Cryptex WebSocket');
        };

        ws.onmessage = (event) => {
            const message = JSON.parse(event.data);

            if (message.type === 'orderbook') {
                const orderbook = message.data;
                displayOrderBook(orderbook);
            }
        };

        function displayOrderBook(orderbook) {
            let html = '<h2>Bids (Buy Orders)</h2>';
            html += '<table border="1">';
            html += '<tr><th>Price</th><th>Quantity</th><th>Orders</th></tr>';

            orderbook.bids.forEach(bid => {
                html += `<tr>
                    <td>${bid.price.toFixed(2)}</td>
                    <td>${bid.quantity.toFixed(4)}</td>
                    <td>${bid.orders}</td>
                </tr>`;
            });

            html += '</table>';

            html += '<h2>Asks (Sell Orders)</h2>';
            html += '<table border="1">';
            html += '<tr><th>Price</th><th>Quantity</th><th>Orders</th></tr>';

            orderbook.asks.forEach(ask => {
                html += `<tr>
                    <td>${ask.price.toFixed(2)}</td>
                    <td>${ask.quantity.toFixed(4)}</td>
                    <td>${ask.orders}</td>
                </tr>`;
            });

            html += '</table>';

            if (orderbook.bids.length > 0 && orderbook.asks.length > 0) {
                const spread = orderbook.asks[0].price - orderbook.bids[0].price;
                html += `<h3>Spread: $${spread.toFixed(2)}</h3>`;
            }

            orderbookDiv.innerHTML = html;
        }
    </script>
</body>
</html>
```

Open this file in your browser to see real-time order book updates.

### Python Example

```python
import websocket
import json
import threading

def on_message(ws, message):
    data = json.loads(message)

    if data['type'] == 'orderbook':
        orderbook = data['data']
        print("\n=== Order Book Update ===")

        print("\nBids (Buy Orders):")
        print("Price\t\tQuantity\tOrders")
        for bid in orderbook['bids'][:5]:
            print(f"${bid['price']:.2f}\t\t{bid['quantity']:.4f}\t\t{bid['orders']}")

        print("\nAsks (Sell Orders):")
        print("Price\t\tQuantity\tOrders")
        for ask in orderbook['asks'][:5]:
            print(f"${ask['price']:.2f}\t\t{ask['quantity']:.4f}\t\t{ask['orders']}")

        if orderbook['bids'] and orderbook['asks']:
            spread = orderbook['asks'][0]['price'] - orderbook['bids'][0]['price']
            print(f"\nSpread: ${spread:.2f}")

def on_error(ws, error):
    print(f"Error: {error}")

def on_close(ws, close_status_code, close_msg):
    print("WebSocket closed")

def on_open(ws):
    print("Connected to Cryptex WebSocket")

if __name__ == "__main__":
    ws = websocket.WebSocketApp(
        "ws://localhost:8080/ws",
        on_open=on_open,
        on_message=on_message,
        on_error=on_error,
        on_close=on_close
    )

    ws.run_forever()
```

Run with: `pip install websocket-client && python websocket_client.py`

## 🧪 Testing the Matching Engine

### Create a Test Script

Create `test_matching.sh`:

```bash
#!/bin/bash

BASE_URL="http://localhost:8080"
USER1_ID="550e8400-e29b-41d4-a716-446655440000"
USER2_ID="660e8400-e29b-41d4-a716-446655440000"

echo "=== Cryptex Matching Engine Test ==="
echo ""

# Place multiple buy orders
echo "Placing buy orders..."
curl -s -X POST $BASE_URL/orders \
  -H "Content-Type: application/json" \
  -d "{\"trading_pair\":\"BTC-USD\",\"side\":\"buy\",\"type\":\"limit\",\"price\":49000.0,\"quantity\":1.0,\"user_id\":\"$USER1_ID\"}" > /dev/null

curl -s -X POST $BASE_URL/orders \
  -H "Content-Type: application/json" \
  -d "{\"trading_pair\":\"BTC-USD\",\"side\":\"buy\",\"type\":\"limit\",\"price\":48500.0,\"quantity\":2.0,\"user_id\":\"$USER1_ID\"}" > /dev/null

curl -s -X POST $BASE_URL/orders \
  -H "Content-Type: application/json" \
  -d "{\"trading_pair\":\"BTC-USD\",\"side\":\"buy\",\"type\":\"limit\",\"price\":48000.0,\"quantity\":1.5,\"user_id\":\"$USER1_ID\"}" > /dev/null

echo "Buy orders placed."
echo ""

# Place a sell order that crosses
echo "Placing crossing sell order..."
TRADE_RESULT=$(curl -s -X POST $BASE_URL/orders \
  -H "Content-Type: application/json" \
  -d "{\"trading_pair\":\"BTC-USD\",\"side\":\"sell\",\"type\":\"limit\",\"price\":49000.0,\"quantity\":1.0,\"user_id\":\"$USER2_ID\"}")

echo "Trade executed:"
echo $TRADE_RESULT | jq '.trades'
echo ""

# Check order book
echo "Current order book:"
curl -s $BASE_URL/orderbook | jq '.'
echo ""

# Place a market order
echo "Placing market buy order..."
MARKET_RESULT=$(curl -s -X POST $BASE_URL/orders \
  -H "Content-Type: application/json" \
  -d "{\"trading_pair\":\"BTC-USD\",\"side\":\"buy\",\"type\":\"market\",\"price\":0.0,\"quantity\":0.5,\"user_id\":\"$USER1_ID\"}")

echo "Market order result:"
echo $MARKET_RESULT | jq '.'
echo ""

echo "=== Test Complete ==="
```

Run with: `chmod +x test_matching.sh && ./test_matching.sh`

## 🎯 Advanced Examples

### Market Order Execution

```bash
# Place a large sell order first
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "sell",
    "type": "limit",
    "price": 50000.0,
    "quantity": 5.0,
    "user_id": "user-seller"
  }'

# Then execute a market buy (will fill at best price)
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "market",
    "price": 0.0,
    "quantity": 2.0,
    "user_id": "user-buyer"
  }'
```

### Partial Fill Scenario

```bash
# Place a small buy order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "limit",
    "price": 50000.0,
    "quantity": 1.0,
    "user_id": "user-buyer"
  }'

# Place a larger sell order (will partially fill)
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "sell",
    "type": "limit",
    "price": 50000.0,
    "quantity": 3.0,
    "user_id": "user-seller"
  }'
```

### Order Cancellation

```bash
# Place an order
ORDER_RESPONSE=$(curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "limit",
    "price": 48000.0,
    "quantity": 1.0,
    "user_id": "user-id"
  }')

# Extract order ID
ORDER_ID=$(echo $ORDER_RESPONSE | jq -r '.order.id')

# Cancel the order
curl -X DELETE http://localhost:8080/orders/$ORDER_ID
```

## 🔍 Monitoring and Debugging

### Check System Health

```bash
# Health check
curl http://localhost:8080/healthz

# Check order book depth
curl http://localhost:8080/orderbook?depth=20

# Check Redis (if running locally)
redis-cli
> KEYS cryptex:*
> GET cryptex:orderbook:BTC-USD:orders
```

### Monitor NATS Events (if configured)

```bash
# Subscribe to trade events
nats sub "trades.>"
nats sub "trade-events.>"

# Subscribe to order events
nats sub "orders.>"
```

## 📈 Performance Testing

### Simple Load Test

```bash
#!/bin/bash

# Load test script
for i in {1..100}; do
  curl -s -X POST http://localhost:8080/orders \
    -H "Content-Type: application/json" \
    -d "{
      \"trading_pair\": \"BTC-USD\",
      \"side\": \"buy\",
      \"type\": \"limit\",
      \"price\": $((49000 + RANDOM % 1000)).0,
      \"quantity\": 0.1,
      \"user_id\": \"user-$i\"
    }" > /dev/null

  if [ $((i % 10)) -eq 0 ]; then
    echo "Placed $i orders..."
  fi
done

echo "Load test complete. Check order book:"
curl http://localhost:8080/orderbook
```

## 🐛 Common Issues and Solutions

### Redis Connection Failed

**Problem:** `connect to redis: connection refused`

**Solution:** Make sure Redis is running:
```bash
redis-server
# or with Docker
docker-compose up redis
```

### Port Already in Use

**Problem:** `bind: address already in use`

**Solution:** Change the port:
```bash
export PORT=8081
./cryptex
```

### NATS Connection Failed

**Problem:** `connect to nats: connection refused`

**Solution:** Either start NATS or run without it:
```bash
# Start NATS
nats-server -js

# Or run without NATS
unset NATS_URL
./cryptex
```

## 📚 Next Steps

Now that you have Cryptex running:

1. **Read the Architecture Documentation** - Understand how the system works
2. **Explore the API Documentation** - Learn about all available endpoints
3. **Check the Development Guide** - Start contributing to the project
4. **Review the Deployment Guide** - Deploy to production

## 🔗 Useful Links

- **Web Dashboard:** http://localhost:8080
- **API Health:** http://localhost:8080/healthz
- **Order Book:** http://localhost:8080/orderbook
- **WebSocket:** ws://localhost:8080/ws

## 💡 Tips

- Use the web dashboard for visual trading
- Use WebSocket for real-time updates in your applications
- Use cURL scripts for automated testing
- Monitor Redis for persistence verification
- Check NATS subjects for event streaming

---

**Need help?** Check the [full documentation](../README.md#-documentation) or open an issue on GitHub.
