<div align="center">

# 🔮 Cryptex

### **High-Performance Cryptocurrency Exchange Matching Engine**

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=flat-square)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen?style=flat-square)]()

A blazing-fast, production-ready cryptocurrency exchange matching engine built in Go with real-time trade event streaming, persistent storage, and comprehensive risk management.

---

**⚡ Features** • **🚀 Quick Start** • **📚 Documentation** • **🧪 Testing**

</div>

---

## ✨ Features

### 🎯 Core Trading Engine
- **Price-Time Priority Matching**: Industry-standard FIFO algorithm for fair order execution
- **Limit & Market Orders**: Support for both order types with instant execution
- **Real-time Matching**: Sub-millisecond order processing latency
- **Thread-Safe Order Book**: Concurrent access with fine-grained locking

### 🌐 API & Connectivity
- **REST API**: Clean HTTP endpoints for order management
- **WebSocket Feed**: Live order book updates and trade notifications
- **Web Dashboard**: Built-in trading interface with real-time visualization
- **NATS Streaming**: Event-driven architecture for downstream consumers

### 💾 Persistence & Reliability
- **Redis Storage**: Durable order book persistence with automatic recovery
- **State Restoration**: Automatic order book reconstruction on startup
- **Atomic Operations**: Consistent state management across all operations

### 🛡️ Risk Management
- **Position Limits**: Configurable maximum position size per user
- **Self-Trade Prevention**: Blocks orders that would trade against user's own orders
- **Price Collars**: Rejects orders outside configured price bounds
- **Pre-Trade Validation**: Comprehensive order validation before acceptance

---

## 🏗️ Architecture

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

---

## 🚀 Quick Start

### Prerequisites

- **Go**: 1.26 or higher
- **Redis**: 6.0+ (for persistence)
- **NATS**: 2.9+ (optional, for event streaming)

### Installation

```bash
# Clone the repository
git clone https://github.com/jellyfishing2346/cryptex.git
cd cryptex

# Install dependencies
go mod download

# Build the binary
go build -o cryptex ./cmd/server
```

### Running with Docker

The easiest way to get started:

```bash
# Start Redis and NATS
docker-compose -f docker/docker-compose.yml up -d

# Run the server
docker-compose up
```

### Running Locally

```bash
# Start Redis
redis-server

# Start NATS (optional)
nats-server -js

# Configure environment
export TRADING_PAIR="BTC-USD"
export REDIS_ADDR="localhost:6379"
export NATS_URL="nats://localhost:4222"  # Optional
export PORT="8080"

# Run the server
./cryptex
```

### Access the Dashboard

Open your browser to: **http://localhost:8080**

---

## ⚙️ Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TRADING_PAIR` | Trading pair to match | `BTC-USD` |
| `REDIS_ADDR` | Redis server address | `localhost:6379` |
| `REDIS_PASSWORD` | Redis password (optional) | - |
| `REDIS_DB` | Redis database number | `0` |
| `NATS_URL` | NATS server URL (optional) | - |
| `PORT` | HTTP server port | `8080` |
| `MAX_POSITION_SIZE` | Max position per user | `1000.0` |
| `MIN_PRICE` | Minimum allowed price | `0.01` |
| `MAX_PRICE` | Maximum allowed price | `1000000.0` |
| `REDIS_URL` | Redis URL, e.g. `rediss://user:pass@host:6380` for TLS; overrides `REDIS_ADDR` | - |
| `DEMO_MODE` | `true` runs a demo market maker and taker so the book is always live | `false` |
| `DEMO_START_PRICE` | Price the demo market drifts around | `65000` |
| `ALLOWED_ORIGINS` | Comma-separated browser origins allowed to call the API (CORS and WebSocket) | - |
| `MAX_ORDER_QUANTITY` | Largest allowed single order; `0` means no cap | `0` |
| `RATE_LIMIT_PER_SECOND` | Order placements and cancellations per second per IP; `0` disables | `0` |
| `RATE_LIMIT_BURST` | Requests allowed in a burst before the rate limit applies | `10` |
| `TRUST_PROXY` | `true` behind a hosting proxy, to read the client IP from `X-Forwarded-For` | `false` |
| `GIN_MODE` | Set to `release` in production | `debug` |

### Running a public demo

`render.yaml` deploys the engine to Render's free tier with a Render Key Value instance for persistence, demo mode on and the guardrails above enabled. In demo mode a market maker quotes both sides of the book around a drifting price, a taker trades against it now and then, and visitor orders expire after 30 minutes. The dashboard at `/` shows the live order book, trades and price chart, and lets visitors place and cancel their own orders.

---

## 📡 API Endpoints

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

### WebSocket Connection

```bash
WS /ws
```

---

## 🔌 NATS Event Streaming

When `NATS_URL` is configured, Cryptex publishes trade events to NATS subjects:

### Trade Subjects

- `trades.<trading_pair>` - Raw trade data
- `trade-events.<trading_pair>` - Structured trade events with metadata
- `orders.<trading_pair>` - Order lifecycle events

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

---

## 🧪 Testing

### Run All Tests

```bash
go test ./...
```

### Run with Coverage

```bash
go test -cover ./...
```

### Run Specific Package Tests

```bash
go test ./internal/matching/...
go test ./internal/orderbook/...
go test ./internal/risk/...
```

### Run Benchmarks

```bash
go test -bench=. ./internal/orderbook/
```

---

## 📁 Project Structure

```
cryptex/
├── cmd/server/          # Main application entry point
├── internal/
│   ├── api/            # HTTP handlers and routing
│   ├── matching/       # Order matching engine
│   ├── models/         # Data models (Order, Trade, etc.)
│   ├── nats/           # NATS publisher for trade events
│   ├── orderbook/      # Order book data structure
│   ├── persistence/    # Redis storage layer
│   ├── risk/           # Risk management checks
│   └── ws/             # WebSocket real-time feed
├── web/                # Web dashboard
├── docker/             # Docker Compose configuration
├── docs/               # Documentation
└── deploy/             # Kubernetes deployment files
```

---

## 📚 Documentation

- **[📖 Documentation Index](docs/INDEX.md)** - Complete documentation overview and navigation
- **[🚀 Quick Start Guide](docs/QUICKSTART.md)** - Get up and running in 5 minutes
- **[🏗️ Architecture Documentation](docs/ARCHITECTURE.md)** - Deep dive into system design and components
- **[📡 API Documentation](docs/API.md)** - Complete API reference with examples
- **[🛠️ Development Guide](docs/DEVELOPMENT.md)** - Development setup and contribution guidelines
- **[🚀 Deployment Guide](DEPLOYMENT.md)** - Production deployment instructions

---

## 🛠️ Development

### Code Style

We follow standard Go conventions:
- Effective Go guidelines
- Standard project layout
- Comprehensive testing
- Clear documentation

### Adding Features

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

### Build for Production

```bash
# Build for Linux
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cryptex ./cmd/server

# Build Docker image
docker build -t cryptex:latest .
```

---

## 🎯 Implementation Timeline

- **Week 1**: ✅ Order book data structure
- **Week 2**: ✅ Matching engine with price-time priority
- **Week 3**: ✅ REST API for order management
- **Week 4**: ✅ Redis persistence
- **Week 5**: ✅ WebSocket real-time feed
- **Week 6**: ✅ NATS trade event streaming
- **Week 7**: ✅ Risk management system
- **Week 8**: ✅ Web dashboard

---

## 🤝 Contributing

Contributions are welcome! Please read our development guide first.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- Built with [Gin](https://gin-gonic.com/) web framework
- Powered by [Redis](https://redis.io/) for persistence
- Event streaming with [NATS](https://nats.io/)
- WebSocket support via [Gorilla WebSocket](https://github.com/gorilla/websocket)

---

<div align="center">

**Built with ❤️ for the crypto community**

[⭐ Star us on GitHub](https://github.com/jellyfishing2346/cryptex) • [🐛 Report Issues](https://github.com/jellyfishing2346/cryptex/issues) • [💡 Feature Requests](https://github.com/jellyfishing2346/cryptex/issues)

</div>
