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

### Performance Tuning
- `GOMAXPROCS`: Number of CPU cores to use (default: auto-detect)
- `GOGC`: Garbage collection target percentage (default: 100)
- `READ_TIMEOUT`: HTTP read timeout in seconds (default: 30)
- `WRITE_TIMEOUT`: HTTP write timeout in seconds (default: 30)

### Security
- `TLS_CERT_FILE`: Path to TLS certificate file (optional)
- `TLS_KEY_FILE`: Path to TLS private key file (optional)
- `ENABLE_METRICS`: Enable Prometheus metrics (default: false)
- `METRICS_PORT`: Metrics server port (default: 9090)

## Local Development

### Prerequisites
- Go 1.26+
- Redis server
- NATS server (optional, for trade event streaming)
- Docker (optional, for containerized development)

### Running with Docker Compose
```bash
# Using the docker-compose configuration
docker-compose -f docker/docker-compose.yml up

# Or with additional services
docker-compose -f docker/docker-compose.yml -f docker/docker-compose.dev.yml up
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

# Or run directly with go
go run ./cmd/server
```

### Accessing the Dashboard
Open your browser to: `http://localhost:8080`

### Development Tools
```bash
# Run tests
go test ./...

# Run with race detector
go run -race ./cmd/server

# Profile the application
go build -o cryptex ./cmd/server
./cryptex -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

## Production Deployment

### Docker Deployment
See [DOCKER.md](docs/DOCKER.md) for comprehensive Docker deployment guide.

### Kubernetes Deployment
```bash
kubectl apply -f deploy/production.yaml
```

This will deploy:
- Redis (1 replica)
- NATS (1 replica)
- Cryptex (3 replicas with LoadBalancer)

#### Additional Kubernetes Options
```bash
# For development environment
kubectl apply -f deploy/development.yaml

# For staging environment
kubectl apply -f deploy/staging.yaml

# Create Redis secret
kubectl create secret generic redis-secret --from-literal=password=your-redis-password

# Scale the application
kubectl scale deployment cryptex --replicas=5

# Check deployment status
kubectl get pods -l app=cryptex
kubectl logs -f deployment/cryptex
```

### Cloud Deployment

#### AWS Deployment
```bash
# Using Elastic Beanstalk
eb init -p go cryptex
eb create production-env

# Using ECS
ecs-cli compose --file docker-compose.yml up --create-log-groups

# Using EKS
eksctl create cluster --name cryptex --nodes 3
kubectl apply -f deploy/production.yaml
```

#### Google Cloud Platform
```bash
# Using Cloud Run
gcloud run deploy cryptex --image gcr.io/PROJECT-ID/cryptex

# Using GKE
gcloud container clusters create cryptex --num-nodes=3
kubectl apply -f deploy/production.yaml
```

#### Azure Deployment
```bash
# Using Azure Container Instances
az container create --resource-group cryptex-rg --name cryptex --image cryptex:latest

# Using AKS
az aks create --resource-group cryptex-rg --name cryptex --node-count 3
kubectl apply -f deploy/production.yaml
```

#### DigitalOcean
```bash
# Using App Platform
doctl apps create --spec .do/app.yaml

# Using Kubernetes
doctl kubernetes cluster create cryptex --count 3
kubectl apply -f deploy/production.yaml
```

### Manual Deployment
1. Build the binary:
```bash
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o cryptex ./cmd/server
```

2. Copy binary and web assets to server
3. Set environment variables
4. Run the binary with systemd:
```ini
# /etc/systemd/system/cryptex.service
[Unit]
Description=Cryptex Matching Engine
After=network.target

[Service]
Type=simple
User=cryptex
WorkingDirectory=/opt/cryptex
Environment="TRADING_PAIR=BTC-USD"
Environment="REDIS_ADDR=localhost:6379"
Environment="NATS_URL=nats://localhost:4222"
ExecStart=/opt/cryptex/cryptex
Restart=always

[Install]
WantedBy=multi-user.target
```

5. Enable and start the service:
```bash
sudo systemctl enable cryptex
sudo systemctl start cryptex
sudo systemctl status cryptex
```

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

## Monitoring and Observability

### Health Checks
```bash
# Basic health check
curl http://localhost:8080/healthz

# Expected response: {"status":"healthy"}
```

### Metrics
If metrics are enabled (`ENABLE_METRICS=true`):
```bash
# Access Prometheus metrics
curl http://localhost:9090/metrics
```

Key metrics to monitor:
- `cryptex_orders_total`: Total number of orders processed
- `cryptex_trades_total`: Total number of trades executed
- `cryptex_errors_total`: Total number of errors
- `cryptex_orderbook_depth`: Current order book depth
- `cryptex_latency_seconds`: Order processing latency

### Logging
Cryptex uses structured JSON logging:
```bash
# View logs in production
journalctl -u cryptex -f

# Or with Docker
docker logs -f cryptex-container

# Kubernetes logs
kubectl logs -f deployment/cryptex
```

### Alerting
Recommended alerts:
- High error rate (>1%)
- High latency (>100ms)
- Low order book depth
- Redis connection failures
- NATS connection failures

## Scaling and Performance

### Horizontal Scaling
For high-throughput scenarios:
- Deploy multiple Cryptex instances behind a load balancer
- Use Redis Cluster for persistence
- Use NATS clustering for event streaming
- Consider sticky sessions if needed

### Vertical Scaling
For single-instance optimization:
- Increase CPU cores
- Add more RAM
- Optimize Go garbage collection
- Use faster storage for Redis

### Performance Tuning
```bash
# Increase Go maxprocs
export GOMAXPROCS=8

# Adjust garbage collection
export GOGC=50

# Increase timeouts for high-latency networks
export READ_TIMEOUT=60
export WRITE_TIMEOUT=60
```

## Backup and Recovery

### Redis Backup
```bash
# Manual Redis backup
redis-cli BGSAVE

# Scheduled backup (cron)
0 2 * * * redis-cli BGSAVE && cp /var/lib/redis/dump.rdb /backup/redis-$(date +%Y%m%d).rdb
```

### Cryptex State Recovery
Cryptex automatically restores order book state from Redis on startup. Ensure Redis persistence is configured:
```bash
# Redis configuration
save 900 1
save 300 10
save 60 10000
```

### Disaster Recovery
1. Regularly backup Redis data
2. Document disaster recovery procedures
3. Test recovery procedures regularly
4. Maintain infrastructure as code
5. Use multi-region deployment for critical systems

## Maintenance

### Rolling Updates
```bash
# Kubernetes rolling update
kubectl set image deployment/cryptex cryptex=cryptex:v2.0.0

# Docker rolling update
docker-compose up -d --no-deps --build cryptex
```

### Database Maintenance
```bash
# Redis memory cleanup
redis-cli MEMORY PURGE

# Redis key analysis
redis-cli --bigkeys

# Redis slow log
redis-cli SLOWLOG GET 10
```

### Security Updates
```bash
# Update Go dependencies
go get -u ./...
go mod tidy

# Update Docker base images
docker pull redis:7-alpine
docker pull nats:latest

# Rebuild and redeploy
docker build -t cryptex:latest .
```

## Troubleshooting

### Common Issues

#### Redis Connection Failed
```bash
# Check Redis status
redis-cli ping

# Check network connectivity
telnet localhost 6379

# Check Redis logs
docker logs redis-container
```

#### High Memory Usage
```bash
# Check Go memory stats
curl http://localhost:8080/debug/pprof/heap

# Check Redis memory
redis-cli INFO memory

# Reduce order book depth
export MAX_ORDER_BOOK_DEPTH=1000
```

#### Slow Performance
```bash
# Profile CPU usage
curl http://localhost:8080/debug/pprof/profile?seconds=30

# Check goroutine count
curl http://localhost:8080/debug/pprof/goroutine

# Review system resources
top
htop
```

### Support
For additional help:
- Check [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md)
- Review [FAQ.md](docs/FAQ.md)
- Open a GitHub issue
