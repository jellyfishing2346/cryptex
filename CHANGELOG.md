# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Comprehensive documentation suite
  - Architecture documentation with system design details
  - Complete API reference with examples
  - Development guide with contribution guidelines
  - Quick start guide with practical examples
  - Documentation index for navigation
- Modern README with professional formatting
- Contributing guidelines
- Security documentation
- Troubleshooting guide
- FAQ section

### Changed
- Enhanced README with emojis, badges, and ASCII diagrams
- Improved documentation structure and organization

### Fixed
- Documentation formatting issues (end-of-file, trailing whitespace)

## [0.8.0] - 2024-06-22

### Added
- Risk management system
  - Position limits per user
  - Self-trade prevention
  - Price collars (min/max price bounds)
- Web dashboard for real-time trading visualization
  - Live order book display
  - WebSocket integration
  - Trade statistics
  - Price spread visualization
- Risk configuration via environment variables
  - MAX_POSITION_SIZE
  - MIN_PRICE
  - MAX_PRICE

### Changed
- Enhanced matching engine with risk check integration
- Updated API server to include risk management
- Improved order validation with risk checks

### Fixed
- Risk check edge cases and validation logic

## [0.7.0] - 2024-06-15

### Added
- NATS trade event streaming
  - Trade event publisher
  - Order event publisher
  - Structured trade events with metadata
- NATS subjects for different event types
  - trades.{trading_pair}
  - trade-events.{trading_pair}
  - orders.{trading_pair}
- Configuration for NATS URL

### Changed
- Enhanced API server with NATS integration
- Trade execution now publishes to NATS
- Optional NATS support (graceful degradation)

## [0.6.0] - 2024-06-08

### Added
- WebSocket real-time feed
  - WebSocket hub for connection management
  - Real-time order book updates
  - Automatic ping/pong keepalive
  - Graceful connection handling
- WebSocket endpoint `/ws`
- Order book broadcast on state changes

### Changed
- Enhanced API server with WebSocket support
- Order placement triggers WebSocket updates
- Order cancellation triggers WebSocket updates

## [0.5.0] - 2024-06-01

### Added
- Redis persistence layer
  - Order book state persistence
  - Automatic restoration on startup
  - Redis client configuration
- Redis storage interface
- Environment variables for Redis configuration
  - REDIS_ADDR
  - REDIS_PASSWORD
  - REDIS_DB

### Changed
- Enhanced main application with persistence
- Order book state saved on every change
- Automatic order book restoration on startup

## [0.4.0] - 2024-05-25

### Added
- REST API for order management
  - POST /orders - Place orders
  - DELETE /orders/:id - Cancel orders
  - GET /orderbook - Get order book snapshot
  - GET /healthz - Health check
- Gin web framework integration
- Order validation
- JSON request/response handling
- Error handling and status codes

### Changed
- Separated API layer from matching engine
- Enhanced order models with API support
- Added HTTP server to main application

## [0.3.0] - 2024-05-18

### Added
- Matching engine with price-time priority
  - Limit order matching
  - Market order matching
  - FIFO execution within price levels
  - Trade generation
- Order status management
  - Open, Partial, Filled, Cancelled
- Trade model with execution details
- Result structure for order submission

### Changed
- Enhanced order book with matching integration
- Improved order lifecycle management
- Added trade execution logic

## [0.2.0] - 2024-05-11

### Added
- Order book data structure
  - Two-sided book (bids and asks)
  - Price-time priority (FIFO)
  - Price level aggregation
  - O(1) order lookup via hash map
- Thread-safe operations with RWMutex
- Order addition and cancellation
- Order book snapshots
- Best bid/ask and spread calculation

### Changed
- Initial order book implementation
- Added concurrent access support

## [0.1.0] - 2024-05-04

### Added
- Initial project structure
- Order and Trade models
- Basic data structures
- Go module setup
- Project documentation (basic README)

## [0.0.1] - 2024-04-27

### Added
- Project initialization
- Basic repository setup
- Initial commit

---

## Version History

| Version | Date | Description |
|---------|------|-------------|
| 0.8.0 | 2024-06-22 | Risk management and web dashboard |
| 0.7.0 | 2024-06-15 | NATS event streaming |
| 0.6.0 | 2024-06-08 | WebSocket real-time feed |
| 0.5.0 | 2024-06-01 | Redis persistence |
| 0.4.0 | 2024-05-25 | REST API |
| 0.3.0 | 2024-05-18 | Matching engine |
| 0.2.0 | 2024-05-11 | Order book |
| 0.1.0 | 2024-05-04 | Initial models |
| 0.0.1 | 2024-04-27 | Project initialization |

---

## Upcoming Features

### Planned
- [ ] Additional order types (stop-limit, trailing stop)
- [ ] Multi-trading pair support
- [ ] Advanced order types (OCO, iceberg)
- [ ] User authentication and authorization
- [ ] Order history and trade history API
- [ ] Market data API (candles, statistics)
- [ ] Webhook notifications
- [ ] Admin dashboard
- [ ] Performance monitoring and metrics
- [ ] Horizontal scaling support

### Under Consideration
- [ ] GraphQL API
- [ ] gRPC API
- [ ] Additional message brokers (Kafka, RabbitMQ)
- [ ] Alternative persistence (PostgreSQL, MySQL)
- [ ] Distributed order book
- [ ] High-frequency trading optimizations

---

## Notes

- Version 0.x.x indicates development/alpha status
- Breaking changes may occur in minor versions until 1.0.0
- API stability will be guaranteed from 1.0.0 onwards
- Documentation will be updated with each release
