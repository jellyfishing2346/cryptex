# Cryptex Development Guide

## Overview

This guide covers development practices, testing strategies, and contribution guidelines for the Cryptex cryptocurrency exchange matching engine.

## Development Environment

### Prerequisites

- **Go**: 1.26 or higher
- **Redis**: 6.0 or higher (for persistence)
- **NATS**: 2.9 or higher (optional, for event streaming)
- **Docker**: For containerized development
- **Make**: For build automation (optional)

### Setting Up

```bash
# Clone the repository
git clone https://github.com/jellyfishing2346/cryptex.git
cd cryptex

# Install dependencies
go mod download

# Run Redis and NATS (optional)
docker-compose -f docker/docker-compose.yml up -d

# Build the project
go build -o cryptex ./cmd/server

# Run the server
./cryptex
```

### Environment Configuration

Create a `.env` file or set environment variables:

```bash
# Required
export TRADING_PAIR="BTC-USD"
export REDIS_ADDR="localhost:6379"

# Optional
export REDIS_PASSWORD=""
export REDIS_DB="0"
export NATS_URL="nats://localhost:4222"
export PORT="8080"

# Risk Management
export MAX_POSITION_SIZE="1000.0"
export MIN_PRICE="0.01"
export MAX_PRICE="1000000.0"
```

## Project Structure

```
cryptex/
├── cmd/
│   └── server/
│       ├── main.go           # Application entry point
│       └── main_test.go      # Integration tests
├── internal/
│   ├── api/
│   │   ├── router.go         # HTTP handlers and routing
│   │   ├── router_test.go    # API tests
│   │   └── integration_test.go
│   ├── matching/
│   │   ├── engine.go         # Matching engine implementation
│   │   └── engine_test.go    # Matching tests
│   ├── models/
│   │   └── order.go          # Data models
│   ├── nats/
│   │   ├── publisher.go      # NATS event publisher
│   │   └── publisher_test.go
│   ├── orderbook/
│   │   ├── orderbook.go      # Order book data structure
│   │   └── orderbook_test.go
│   ├── persistence/
│   │   ├── redis_store.go    # Redis persistence
│   │   └── redis_store_test.go
│   ├── risk/
│   │   ├── checks.go         # Risk management
│   │   └── checks_test.go
│   └── ws/
│       ├── handler.go        # WebSocket handler
│       ├── hub.go            # WebSocket hub
│       └── hub_test.go
├── web/
│   └── dashboard.html        # Web dashboard
├── docker/
│   └── docker-compose.yml     # Development containers
├── docs/
│   ├── ARCHITECTURE.md       # Architecture documentation
│   ├── API.md                # API documentation
│   └── DEVELOPMENT.md        # This file
├── Dockerfile               # Production image
├── go.mod                   # Go module definition
├── go.sum                   # Go dependencies
├── README.md                # Project README
└── DEPLOYMENT.md            # Deployment guide
```

## Code Style

### Go Conventions

We follow standard Go conventions as outlined in [Effective Go](https://golang.org/doc/effective_go) and the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments).

### Naming

- **Packages**: Short, lowercase, single words when possible
- **Exports**: PascalCase for exported, camelCase for private
- **Interfaces**: Usually -er suffix (e.g., `Checker`, `Store`)
- **Constants**: PascalCase for exported, camelCase for private

### Formatting

```bash
# Format all Go files
go fmt ./...

# Run linter
golangci-lint run
```

### Comments

- **Package comments**: Describe the package's purpose
- **Exported functions**: Document behavior, parameters, and return values
- **Complex logic**: Explain non-obvious algorithms
- **TODO**: Mark future work with TODO comments

Example:
```go
// Package matching implements the core order matching algorithm.
//
// When a new order arrives, the engine checks the opposite side of the
// book for crossing orders (price overlap) and executes trades at the
// resting order's price — this is standard exchange behavior.
package matching

// Submit processes a new order: it first tries to match against the
// resting book, then adds any unfilled remainder to the book (limit
// orders only — market orders that can't fully fill are not booked).
func (e *Engine) Submit(order *models.Order) (*Result, error) {
    // Implementation...
}
```

## Testing

### Test Structure

We use table-driven tests for multiple scenarios:

```go
func TestEngineSubmit(t *testing.T) {
    tests := []struct {
        name     string
        order    *models.Order
        setup    func(*orderbook.OrderBook)
        wantErr  bool
        validate func(*Result, error)
    }{
        {
            name: "market order fills completely",
            order: &models.Order{
                Side:     models.SideBuy,
                Type:     models.OrderTypeMarket,
                Quantity: 1.0,
            },
            setup: func(book *orderbook.OrderBook) {
                // Setup order book state
            },
            wantErr: false,
            validate: func(result *Result, err error) {
                // Validate result
            },
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run specific package tests
go test ./internal/matching/...
go test ./internal/orderbook/...

# Run specific test
go test -run TestEngineSubmit ./internal/matching/

# Run benchmarks
go test -bench=. ./internal/orderbook/

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Test Coverage Goals

- **Core matching**: >95%
- **Risk management**: >90%
- **API layer**: >85%
- **Persistence**: >80%
- **Overall**: >85%

### Writing Tests

**Unit Tests**: Test individual functions and methods in isolation
```go
func TestOrderBookAdd(t *testing.T) {
    book := orderbook.New("BTC-USD")
    order := &models.Order{
        ID:       uuid.New(),
        Side:     models.SideBuy,
        Price:    50000.0,
        Quantity: 1.0,
    }

    err := book.Add(order)
    if err != nil {
        t.Fatalf("Add() error = %v", err)
    }

    // Verify order was added
    _, exists := book.GetOrder(order.ID)
    if !exists {
        t.Error("Order not found in book")
    }
}
```

**Integration Tests**: Test component interactions
```go
func TestAPIPlaceOrderIntegration(t *testing.T) {
    // Setup test server
    book := orderbook.New("BTC-USD")
    engine := matching.New(book)
    server := api.NewServer(book, engine, nil)

    // Make HTTP request
    // Verify response
    // Check order book state
}
```

**Benchmark Tests**: Measure performance
```go
func BenchmarkOrderBookAdd(b *testing.B) {
    book := orderbook.New("BTC-USD")

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        order := &models.Order{
            ID:       uuid.New(),
            Side:     models.SideBuy,
            Price:    50000.0 + float64(i),
            Quantity: 1.0,
        }
        book.Add(order)
    }
}
```

## Debugging

### Logging

Add structured logging for debugging:

```go
import "log"

log.Printf("Processing order: %+v", order)
log.Printf("Match result: %d trades", len(result.Trades))
```

### Delve Debugger

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug main program
dlv debug ./cmd/server

# Debug test
dlv test ./internal/matching/
```

### Common Debugging Scenarios

**Order not matching:**
```go
// Add logging in matching engine
log.Printf("Incoming order: side=%s, price=%f, quantity=%f",
    order.Side, order.Price, order.Quantity)
log.Printf("Best bid: %f, Best ask: %f", book.BestBid(), book.BestAsk())
```

**Persistence issues:**
```go
// Add logging in Redis store
log.Printf("Saving %d orders to Redis", len(orders))
log.Printf("Redis key: %s", s.key)
```

**WebSocket connection issues:**
```go
// Add logging in WebSocket handler
log.Printf("WebSocket connection from %s", c.Request.RemoteAddr)
log.Printf("Total clients: %d", hub.ClientCount())
```

## Build and Release

### Building

```bash
# Build for current platform
go build -o cryptex ./cmd/server

# Build for Linux
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cryptex ./cmd/server

# Build for macOS
CGO_ENABLED=0 GOOS=darwin go build -a -installsuffix cgo -o cryptex ./cmd/server

# Build with version info
VERSION=$(git describe --tags --always --dirty)
go build -ldflags "-X main.Version=$VERSION" -o cryptex ./cmd/server
```

### Docker

```bash
# Build image
docker build -t cryptex:latest .

# Run container
docker run -p 8080:8080 \
  -e TRADING_PAIR=BTC-USD \
  -e REDIS_ADDR=redis:6379 \
  cryptex:latest
```

### Release Process

1. Update version in code
2. Update CHANGELOG.md
3. Create git tag
4. Build release binaries
5. Create GitHub release
6. Update documentation

## Adding Features

### Adding a New Order Type

1. **Update models** (`internal/models/order.go`):
```go
const (
    OrderTypeLimit    OrderType = "limit"
    OrderTypeMarket   OrderType = "market"
    OrderTypeStopLimit OrderType = "stop-limit"  // New type
)
```

2. **Update validation** (`internal/api/router.go`):
```go
func validatePlaceOrderRequest(req placeOrderRequest, tradingPair string) error {
    // ... existing validation ...
    if req.Type == models.OrderTypeStopLimit {
        if req.StopPrice <= 0 {
            return errors.New("stop-limit order requires stop price")
        }
    }
    return nil
}
```

3. **Update matching engine** (`internal/matching/engine.go`):
```go
func (e *Engine) match(incoming *models.Order) []*models.Trade {
    // Add stop-limit logic
    if incoming.Type == models.OrderTypeStopLimit {
        return e.matchStopLimit(incoming)
    }
    // ... existing logic ...
}
```

4. **Add tests** for the new order type

### Adding a New Risk Check

1. **Add check method** (`internal/risk/checks.go`):
```go
func (r *Checker) checkNewRiskRule(order *models.Order) error {
    // Implement check logic
    if someCondition {
        return errors.New("new risk rule violated")
    }
    return nil
}
```

2. **Call from main checker**:
```go
func (r *Checker) CheckOrder(order *models.Order) error {
    // ... existing checks ...
    if err := r.checkNewRiskRule(order); err != nil {
        return err
    }
    return nil
}
```

3. **Add configuration** if needed:
```go
type Config struct {
    MaxPositionSize float64
    MinPrice        float64
    MaxPrice        float64
    NewRiskParam    float64  // New parameter
}
```

4. **Add tests** for the new check

### Adding a New API Endpoint

1. **Add handler** (`internal/api/router.go`):
```go
func (s *Server) newEndpoint(c *gin.Context) {
    // Implement handler logic
    c.JSON(http.StatusOK, response)
}
```

2. **Register route**:
```go
func (s *Server) Router() *gin.Engine {
    router := gin.New()
    // ... existing routes ...
    router.GET("/new-endpoint", s.newEndpoint)
    return router
}
```

3. **Add request/response models** if needed
4. **Add tests** for the new endpoint

## Performance Optimization

### Profiling

```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=. ./internal/matching/
go tool pprof cpu.prof

# Memory profiling
go test -memprofile=mem.prof -bench=. ./internal/matching/
go tool pprof mem.prof

# HTTP profiling
import _ "net/http/pprof"
# Then access http://localhost:8080/debug/pprof/
```

### Optimization Strategies

**Order Book Optimization:**
- Use binary search for price level insertion
- Pool order objects to reduce GC pressure
- Optimize serialization for snapshots

**Matching Engine Optimization:**
- Reduce lock contention
- Batch order processing
- Optimize trade generation

**API Optimization:**
- Add response caching
- Implement request batching
- Use connection pooling

## Contributing

### Workflow

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make changes and commit
4. Run tests: `go test ./...`
5. Run linter: `golangci-lint run`
6. Push branch: `git push origin feature/my-feature`
7. Create pull request

### Commit Messages

Follow conventional commits:

```
feat: add stop-limit order support
fix: resolve race condition in order book
docs: update API documentation
test: add integration tests for risk checks
refactor: simplify matching engine logic
```

### Pull Request Checklist

- [ ] Tests pass
- [ ] Linter passes
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] No breaking changes (or documented)
- [ ] Performance impact considered

## Troubleshooting

### Common Issues

**Redis connection fails:**
```bash
# Check Redis is running
redis-cli ping

# Check connection settings
echo $REDIS_ADDR
```

**NATS connection fails:**
```bash
# Check NATS is running
nats-server -js

# Verify NATS URL
echo $NATS_URL
```

**Tests fail randomly:**
```bash
# Run with race detection
go test -race ./...

# Increase test timeouts
go test -timeout 30s ./...
```

**Build fails:**
```bash
# Clean module cache
go clean -modcache

# Re-download dependencies
go mod download
```

## Resources

### Go Documentation
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Go Blog](https://blog.golang.org)

### Project Documentation
- [Architecture Documentation](./ARCHITECTURE.md)
- [API Documentation](./API.md)
- [Deployment Guide](../DEPLOYMENT.md)

### External Libraries
- [Gin Web Framework](https://gin-gonic.com/docs/)
- [Redis Go Client](https://redis.uptrace.dev/)
- [NATS Go Client](https://github.com/nats-io/nats.go)

## Support

For questions or issues:
- Open a GitHub issue
- Check existing documentation
- Review test files for examples
