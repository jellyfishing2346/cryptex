# Cryptex FAQ

Frequently asked questions about Cryptex cryptocurrency exchange matching engine.

## General Questions

### What is Cryptex?

Cryptex is a high-performance cryptocurrency exchange matching engine built in Go. It implements a price-time priority (FIFO) matching algorithm with real-time trade event streaming, persistent storage, and comprehensive risk management.

### What can I use Cryptex for?

- Building a cryptocurrency exchange
- Learning about matching engine architecture
- Testing trading strategies
- Financial application development
- Educational purposes

### Is Cryptex production-ready?

Cryptex is currently in development (version 0.8.x). While it has core functionality, it's missing some production features like authentication, rate limiting, and comprehensive audit logging. Use with caution in production environments.

### What are the system requirements?

- **Go**: 1.26 or higher
- **Redis**: 6.0+ (for persistence)
- **NATS**: 2.9+ (optional, for event streaming)
- **RAM**: Minimum 512MB, recommended 2GB+
- **CPU**: Single core minimum, multi-core recommended

### Is Cryptex free to use?

Yes, Cryptex is open-source and licensed under the MIT License. You can use it for personal and commercial projects.

## Technical Questions

### What matching algorithm does Cryptex use?

Cryptex uses **price-time priority** (FIFO) matching:
- Orders are matched based on price first
- Within the same price, orders are matched by time (oldest first)
- This is the standard algorithm used by most major exchanges

### What order types are supported?

Currently supported:
- **Limit orders**: Execute at specified price or better
- **Market orders**: Execute immediately at best available price

Planned order types:
- Stop-limit orders
- Trailing stop orders
- OCO (One-Cancels-Other) orders
- Iceberg orders

### How does the order book work?

The order book maintains two sides:
- **Bids**: Buy orders sorted by price descending (highest first)
- **Asks**: Sell orders sorted by price ascending (lowest first)

Orders within each price level are sorted by time (FIFO).

### How is persistence handled?

Cryptex uses Redis for persistence:
- All resting orders are saved to Redis after each state change
- On startup, orders are automatically restored from Redis
- Redis provides durability and fast recovery

### What is NATS used for?

NATS is used for event streaming:
- Trade events are published to NATS subjects
- Downstream consumers can subscribe to these events
- Use cases: analytics, notifications, audit logging

### Is the order book thread-safe?

Yes, the order book uses `sync.RWMutex` for thread-safe operations:
- Multiple readers can access simultaneously
- Writers have exclusive access
- Matching engine uses single lock for atomic operations

## Configuration Questions

### How do I configure the trading pair?

Set the `TRADING_PAIR` environment variable:
```bash
export TRADING_PAIR="BTC-USD"
```

### How do I change the server port?

Set the `PORT` environment variable:
```bash
export PORT="8080"
```

### How do I configure Redis?

Set Redis environment variables:
```bash
export REDIS_ADDR="localhost:6379"
export REDIS_PASSWORD="your-password"
export REDIS_DB="0"
```

### How do I configure risk management?

Set risk management environment variables:
```bash
export MAX_POSITION_SIZE="1000.0"
export MIN_PRICE="0.01"
export MAX_PRICE="1000000.0"
```

### Can I run without NATS?

Yes, NATS is optional. Simply don't set the `NATS_URL` environment variable:
```bash
unset NATS_URL
./cryptex
```

## API Questions

### How do I place an order?

Use the REST API:
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{
    "trading_pair": "BTC-USD",
    "side": "buy",
    "type": "limit",
    "price": 50000.0,
    "quantity": 1.0,
    "user_id": "user-id"
  }'
```

### How do I cancel an order?

Use the REST API:
```bash
curl -X DELETE http://localhost:8080/orders/{order-id}
```

### How do I get the order book?

Use the REST API:
```bash
curl http://localhost:8080/orderbook?depth=10
```

### How do I connect via WebSocket?

Connect to the WebSocket endpoint:
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
```

### What is the difference between limit and market orders?

- **Limit orders**: Execute at a specific price or better. If not immediately filled, they're added to the order book.
- **Market orders**: Execute immediately at the best available price. They're never added to the order book.

## Performance Questions

### How many orders per second can Cryptex handle?

Cryptex can handle 10,000+ orders per second on a single instance with proper hardware. Performance depends on:
- Hardware specifications
- Order book depth
- Network latency
- Redis performance

### What is the latency for order matching?

Order matching latency is typically <1ms for local operations. Total latency including API and persistence is typically <5ms.

### Can Cryptex handle multiple trading pairs?

Currently, Cryptex supports a single trading pair per instance. Multi-pair support is planned for future releases.

### How do I scale Cryptex?

For horizontal scaling:
- Deploy multiple instances behind a load balancer
- Use Redis Cluster for persistence
- Use NATS clustering for event streaming
- Implement session affinity if needed

## Integration Questions

### Can I integrate Cryptex with my existing system?

Yes, Cryptex provides:
- REST API for HTTP integration
- WebSocket for real-time updates
- NATS for event streaming
- Redis for data access

### What programming languages can I use?

Any language that can make HTTP requests and handle WebSocket connections:
- JavaScript/TypeScript
- Python
- Java
- C#
- Go
- Rust
- And many more

### Is there a client library available?

Currently, there are no official client libraries. However, the REST API is standard HTTP/JSON, making it easy to integrate with any language.

### Can I use Cryptex with a database other than Redis?

Currently, only Redis is supported for persistence. The persistence layer is designed to be extensible, so other databases could be added in the future.

## Security Questions

### Is Cryptex secure for production use?

Cryptex has basic security features but requires additional security measures for production:
- Add authentication (currently not implemented)
- Enable HTTPS/TLS
- Implement rate limiting
- Add audit logging
- Secure Redis and NATS connections

See [SECURITY.md](../SECURITY.md) for detailed security guidelines.

### Does Cryptex encrypt data?

Currently, Cryptex does not encrypt data. You should:
- Use TLS for network communications
- Enable Redis encryption
- Encrypt data at rest in production

### How do I add authentication?

Authentication is not currently implemented. You can add authentication middleware using:
- JWT tokens
- API keys
- OAuth 2.0

## Troubleshooting Questions

### Why isn't my order matching?

Common reasons:
- Prices don't cross (buy price < sell price)
- Order on same side of the book
- Risk checks blocking the order
- Order book is empty

### Why is Redis connection failing?

Check:
- Redis is running: `redis-cli ping`
- Correct address: `echo $REDIS_ADDR`
- Network connectivity
- Firewall settings

### Why is WebSocket connection failing?

Check:
- Server is running: `curl http://localhost:8080/healthz`
- Correct WebSocket URL: `ws://localhost:8080/ws`
- Network connectivity
- Browser console for errors

## Development Questions

### How do I contribute to Cryptex?

See [CONTRIBUTING.md](../CONTRIBUTING.md) for detailed contribution guidelines.

### How do I run tests?

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/matching/...
```

### What IDE do you recommend?

Any Go-compatible IDE:
- VS Code with Go extension
- GoLand
- Vim/Neovim with vim-go
- Emacs with go-mode

### How do I debug Cryptex?

Use Delve debugger:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug ./cmd/server
```

## Deployment Questions

### How do I deploy Cryptex to production?

See [DEPLOYMENT.md](../DEPLOYMENT.md) for detailed deployment instructions.

### Can I deploy Cryptex on Kubernetes?

Yes, Kubernetes deployment manifests are provided in the `deploy/` directory.

### Can I deploy Cryptex on Docker?

Yes, Docker support is available with a Dockerfile and docker-compose configuration.

### What cloud providers do you recommend?

Cryptex can be deployed on any cloud provider:
- AWS
- Google Cloud Platform
- Azure
- DigitalOcean
- Any VPS provider

## Licensing Questions

### What license does Cryptex use?

Cryptex is licensed under the MIT License. See the [LICENSE](../LICENSE) file for details.

### Can I use Cryptex commercially?

Yes, the MIT License allows commercial use.

### Do I need to attribute Cryptex?

The MIT License requires attribution, but it's quite permissive. See the LICENSE file for details.

## Future Plans

### What features are planned?

See the [CHANGELOG.md](../CHANGELOG.md) for upcoming features including:
- Additional order types
- Multi-trading pair support
- User authentication
- Advanced order types
- Performance optimizations

### When will version 1.0 be released?

There's no specific timeline for 1.0.0. It will be released when the project is production-ready with all core features implemented.

### How can I request a feature?

Open a GitHub issue with the "enhancement" label and describe your feature request.

## Support Questions

### Where can I get help?

- **Documentation**: Check the [docs/](INDEX.md) directory
- **GitHub Issues**: Search or create issues
- **GitHub Discussions**: Ask questions in discussions
- **Troubleshooting Guide**: See [TROUBLESHOOTING.md](TROUBLESHOOTING.md)

### Is commercial support available?

Currently, commercial support is not available. Community support is provided through GitHub.

### How do I report a bug?

Open a GitHub issue with:
- Clear description of the bug
- Steps to reproduce
- Expected vs actual behavior
- Environment details
- Relevant logs

## Miscellaneous

### Why Go?

Go was chosen for:
- Performance and efficiency
- Strong concurrency support
- Excellent standard library
- Fast compilation
- Growing ecosystem

### Why Redis?

Redis was chosen for:
- Performance (in-memory)
- Persistence options
- Rich data structures
- Clustering support
- Active community

### Why NATS?

NATS was chosen for:
- Performance and simplicity
- Cloud-native design
- Clustering support
- Multiple messaging patterns
- Low latency

### Can I use Cryptex for fiat currencies?

Yes, Cryptex is currency-agnostic. You can use it for any asset pair (crypto-crypto, crypto-fiat, fiat-fiat).

---

**Still have questions?** Open a GitHub issue or start a discussion!
