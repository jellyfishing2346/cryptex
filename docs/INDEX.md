# Cryptex Documentation Index

Welcome to the Cryptex documentation. This comprehensive guide covers everything you need to know about the cryptocurrency exchange matching engine.

## 📚 Documentation Overview

### Getting Started
- **[README.md](../README.md)** - Project overview, features, and quick introduction
- **[Quick Start Guide](QUICKSTART.md)** - Get up and running in 5 minutes with practical examples

### Technical Documentation
- **[Architecture Documentation](ARCHITECTURE.md)** - Deep dive into system design, components, and data flow
- **[API Documentation](API.md)** - Complete API reference with examples for all endpoints
- **[Development Guide](DEVELOPMENT.md)** - Development setup, coding standards, and contribution guidelines

### Deployment & Operations
- **[Deployment Guide](../DEPLOYMENT.md)** - Production deployment instructions and configuration

## 🎯 Documentation by Use Case

### For New Users
1. Start with the [README.md](../README.md) to understand what Cryptex is
2. Follow the [Quick Start Guide](QUICKSTART.md) to get running immediately
3. Explore the [API Documentation](API.md) to integrate with your applications

### For Developers
1. Read the [Architecture Documentation](ARCHITECTURE.md) to understand the system design
2. Review the [Development Guide](DEVELOPMENT.md) for coding standards and practices
3. Check the [API Documentation](API.md) for endpoint details
4. Run the examples in the [Quick Start Guide](QUICKSTART.md) to understand the flow

### For DevOps Engineers
1. Review the [Deployment Guide](../DEPLOYMENT.md) for production setup
2. Understand the architecture from [ARCHITECTURE.md](ARCHITECTURE.md)
3. Configure environment variables as specified in deployment docs

### For Contributors
1. Read the [Development Guide](DEVELOPMENT.md) thoroughly
2. Understand the [Architecture Documentation](ARCHITECTURE.md)
3. Review existing code and tests
4. Follow the contribution guidelines

## 📖 Key Concepts

### Core Components
- **Order Book**: Price-time priority data structure for managing orders
- **Matching Engine**: Core trading logic with FIFO algorithm
- **Risk Management**: Position limits, self-trade prevention, price collars
- **Persistence Layer**: Redis-based durable storage
- **Event Streaming**: NATS-based trade event distribution

### API Interfaces
- **REST API**: HTTP endpoints for order management
- **WebSocket**: Real-time order book updates
- **Web Dashboard**: Browser-based trading interface

### Data Models
- **Order**: Limit and market orders with full lifecycle tracking
- **Trade**: Executed trades with price and quantity information
- **Order Book Snapshot**: Aggregated view of current market state

## 🔧 Common Tasks

### Running Cryptex
```bash
# Quick start with Docker
docker-compose up

# Manual setup
go build -o cryptex ./cmd/server
./cryptex
```

### Testing the API
```bash
# Place an order
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"trading_pair":"BTC-USD","side":"buy","type":"limit","price":50000.0,"quantity":1.0,"user_id":"user-id"}'

# Get order book
curl http://localhost:8080/orderbook
```

### Running Tests
```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Specific package
go test ./internal/matching/...
```

## 📊 System Architecture Overview

```
Client → REST API/WebSocket → Matching Engine → Order Book
                                    ↓
                              Risk Checks
                                    ↓
                              Redis Persistence
                                    ↓
                              NATS Event Streaming
```

## 🔍 Documentation Structure

```
docs/
├── INDEX.md              # This file - documentation index
├── ARCHITECTURE.md       # System architecture and design
├── API.md                # Complete API reference
├── DEVELOPMENT.md        # Development guide and contributions
└── QUICKSTART.md         # Quick start guide with examples
```

## 🆘 Getting Help

### Documentation Issues
If you find errors or unclear sections in the documentation:
1. Check for existing issues on GitHub
2. Create a new issue with the "documentation" label
3. Suggest improvements via pull requests

### Technical Issues
For technical problems:
1. Check the [Deployment Guide](../DEPLOYMENT.md) for common issues
2. Review the [Quick Start Guide](QUICKSTART.md) troubleshooting section
3. Open a GitHub issue with detailed information

### Feature Requests
For new features or improvements:
1. Check existing feature requests
2. Open a new issue with the "enhancement" label
3. Provide use cases and requirements

## 📝 Documentation Conventions

### Code Examples
- Language-agnostic where possible
- Include complete, runnable examples
- Show expected responses
- Use realistic data

### Diagrams
- ASCII art for text-based documentation
- Clear component labels
- Data flow indications
- Consistent styling

### Formatting
- Markdown for all documentation
- Code blocks with syntax highlighting
- Tables for configuration and parameters
- Headers for logical organization

## 🔗 External Resources

### Go Documentation
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Standard Library](https://golang.org/pkg/)

### Dependencies
- [Gin Web Framework](https://gin-gonic.com/docs/)
- [Redis Go Client](https://redis.uptrace.dev/)
- [NATS Go Client](https://github.com/nats-io/nats.go)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)

### Related Technologies
- [Redis Documentation](https://redis.io/documentation)
- [NATS Documentation](https://docs.nats.io/)
- [WebSocket RFC](https://tools.ietf.org/html/rfc6455)

## 📈 Documentation Metrics

- **Total Documentation Files**: 5
- **Architecture Sections**: 15+
- **API Endpoints Documented**: 6
- **Code Examples**: 20+
- **Diagrams**: 5+

## 🔄 Keeping Documentation Updated

Documentation should be updated when:
- New features are added
- API endpoints change
- Configuration options are added/modified
- Architecture changes occur
- Bugs are found in documentation

## 📞 Contact

For documentation-specific questions:
- Open a GitHub issue with the "documentation" label
- Contact maintainers via GitHub discussions
- Improve documentation via pull requests

---

**Last Updated**: 2024-06-23
**Documentation Version**: 1.0.0
