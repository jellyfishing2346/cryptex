# Cryptex Troubleshooting Guide

This guide helps you diagnose and resolve common issues with Cryptex.

## 🔧 Common Issues

### Connection Issues

#### Redis Connection Failed

**Problem:**
```
connect to redis: connection refused
```

**Causes:**
- Redis server is not running
- Wrong Redis address configured
- Firewall blocking connection
- Redis running on different port

**Solutions:**

1. **Check if Redis is running:**
```bash
redis-cli ping
# Should return: PONG
```

2. **Start Redis if not running:**
```bash
# Local installation
redis-server

# Docker
docker run -d -p 6379:6379 redis:latest

# Docker Compose
docker-compose up redis
```

3. **Verify Redis address:**
```bash
echo $REDIS_ADDR
# Should be: localhost:6379 or your Redis server address
```

4. **Check firewall:**
```bash
# Check if port 6379 is accessible
telnet localhost 6379
# or
nc -zv localhost 6379
```

#### NATS Connection Failed

**Problem:**
```
connect to nats: connection refused
```

**Causes:**
- NATS server is not running
- Wrong NATS URL configured
- NATS running on different port

**Solutions:**

1. **Check if NATS is running:**
```bash
# Check NATS server status
nats-server -js
```

2. **Start NATS if not running:**
```bash
# Local installation
nats-server -js

# Docker
docker run -d -p 4222:4222 nats:latest -js

# Docker Compose
docker-compose up nats
```

3. **Run without NATS (optional):**
```bash
unset NATS_URL
./cryptex
```

#### Port Already in Use

**Problem:**
```
bind: address already in use
```

**Causes:**
- Another process using the port
- Previous Cryptex instance still running
- Port conflict with other services

**Solutions:**

1. **Find process using the port:**
```bash
# macOS/Linux
lsof -i :8080

# Linux alternative
netstat -tulpn | grep :8080
```

2. **Kill the process:**
```bash
kill -9 <PID>
```

3. **Use a different port:**
```bash
export PORT=8081
./cryptex
```

### Build Issues

#### Go Version Incompatible

**Problem:**
```
go.mod requires go 1.26 but you have go 1.25
```

**Solutions:**

1. **Update Go:**
```bash
# Download latest Go from https://golang.org/dl/
# Or use Homebrew (macOS)
brew install go
```

2. **Verify Go version:**
```bash
go version
```

#### Module Download Failed

**Problem:**
```
go: github.com/module@v1.0.0: Get "https://proxy.golang.org/...": dial tcp: connection refused
```

**Solutions:**

1. **Check internet connection**
2. **Use Go proxy:**
```bash
export GOPROXY=https://proxy.golang.org,direct
go mod download
```

3. **Clear module cache:**
```bash
go clean -modcache
go mod download
```

#### Build Failures

**Problem:**
```
# github.com/jellyfishing2346/cryptex/internal/api
./router.go:123: undefined: SomeFunction
```

**Solutions:**

1. **Check for missing imports**
2. **Verify all dependencies are installed:**
```bash
go mod tidy
go mod download
```

3. **Clean and rebuild:**
```bash
go clean -cache
go build -o cryptex ./cmd/server
```

### Runtime Issues

#### Order Not Matching

**Problem:**
Orders are placed but not executing trades.

**Causes:**
- Prices don't cross
- Order on same side
- Risk checks blocking
- Order book empty

**Solutions:**

1. **Check order book state:**
```bash
curl http://localhost:8080/orderbook
```

2. **Verify order details:**
```bash
# Check if buy order price >= sell order price
# For matching to occur
```

3. **Check risk check logs:**
```bash
# Look for risk check errors in server logs
```

4. **Place crossing orders:**
```bash
# Place buy order at 50000
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"trading_pair":"BTC-USD","side":"buy","type":"limit","price":50000.0,"quantity":1.0,"user_id":"user1"}'

# Place sell order at 50000 (will cross)
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"trading_pair":"BTC-USD","side":"sell","type":"limit","price":50000.0,"quantity":1.0,"user_id":"user2"}'
```

#### Risk Check Errors

**Problem:**
```
position limit exceeded
self-trade prevention triggered
price below minimum collar
```

**Solutions:**

1. **Position limit exceeded:**
```bash
# Check current position
# Reduce order quantity
# Increase MAX_POSITION_SIZE
export MAX_POSITION_SIZE=2000.0
```

2. **Self-trade prevention:**
```bash
# Use different user_id for orders on opposite sides
# Cancel existing orders before placing crossing orders
```

3. **Price collar violation:**
```bash
# Adjust price within bounds
# Update MIN_PRICE/MAX_PRICE
export MIN_PRICE=0.001
export MAX_PRICE=2000000.0
```

#### WebSocket Connection Issues

**Problem:**
WebSocket connection fails or disconnects frequently.

**Causes:**
- Server not running
- Wrong WebSocket URL
- Network issues
- Firewall blocking

**Solutions:**

1. **Verify server is running:**
```bash
curl http://localhost:8080/healthz
```

2. **Check WebSocket URL:**
```javascript
// Should be: ws://localhost:8080/ws
const ws = new WebSocket('ws://localhost:8080/ws');
```

3. **Check browser console for errors:**
```javascript
ws.onerror = (error) => {
  console.error('WebSocket error:', error);
};
```

4. **Test with simple client:**
```bash
# Use wscat to test WebSocket connection
wscat -c ws://localhost:8080/ws
```

### Performance Issues

#### High Memory Usage

**Problem:**
Cryptex consuming excessive memory.

**Causes:**
- Too many orders in book
- Memory leak
- Insufficient resources

**Solutions:**

1. **Monitor order book size:**
```bash
curl http://localhost:8080/orderbook | jq '.bids | length'
curl http://localhost:8080/orderbook | jq '.asks | length'
```

2. **Cancel old orders:**
```bash
# Clean up old orders periodically
```

3. **Profile memory usage:**
```bash
go tool pprof http://localhost:8080/debug/pprof/heap
```

#### Slow Response Times

**Problem:**
API responses are slow.

**Causes:**
- Too many orders
- Lock contention
- Redis latency
- System overload

**Solutions:**

1. **Check system resources:**
```bash
# CPU usage
top

# Memory usage
free -h

# Disk I/O
iostat
```

2. **Profile CPU usage:**
```bash
go tool pprof http://localhost:8080/debug/pprof/profile
```

3. **Check Redis latency:**
```bash
redis-cli --latency
```

4. **Reduce order book depth:**
```bash
# Request fewer levels
curl http://localhost:8080/orderbook?depth=5
```

### Docker Issues

#### Container Won't Start

**Problem:**
Docker container exits immediately.

**Solutions:**

1. **Check container logs:**
```bash
docker logs cryptex
```

2. **Run in interactive mode:**
```bash
docker run -it cryptex:latest /bin/bash
```

3. **Check environment variables:**
```bash
docker run -e TRADING_PAIR=BTC-USD -e REDIS_ADDR=redis:6379 cryptex:latest
```

#### Docker Network Issues

**Problem:**
Container can't connect to Redis/NATS.

**Solutions:**

1. **Check Docker network:**
```bash
docker network ls
docker network inspect cryptex_default
```

2. **Use service names:**
```yaml
# In docker-compose.yml
REDIS_ADDR=redis:6379  # Not localhost
```

3. **Verify containers are on same network:**
```bash
docker-compose ps
```

### Persistence Issues

#### Orders Not Persisting

**Problem:**
Orders are lost after restart.

**Solutions:**

1. **Check Redis connection:**
```bash
redis-cli ping
```

2. **Verify Redis key:**
```bash
redis-cli
> KEYS cryptex:*
> GET cryptex:orderbook:BTC-USD:orders
```

3. **Check persistence logs:**
```bash
# Look for Redis save errors in server logs
```

#### Wrong Orders Restored

**Problem:**
After restart, wrong orders appear in book.

**Solutions:**

1. **Clear Redis data:**
```bash
redis-cli
> DEL cryptex:orderbook:BTC-USD:orders
```

2. **Verify trading pair:**
```bash
echo $TRADING_PAIR
```

3. **Check order validation:**
```bash
# Ensure orders match current trading pair
```

## 🔍 Debugging Tips

### Enable Debug Logging

Add logging to diagnose issues:

```go
log.Printf("DEBUG: Processing order: %+v", order)
log.Printf("DEBUG: Order book state: bids=%d, asks=%d", len(bids), len(asks))
```

### Use Delve Debugger

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug main program
dlv debug ./cmd/server

# Debug test
dlv test ./internal/matching/
```

### Monitor System Resources

```bash
# System monitoring
htop

# Go-specific monitoring
go tool pprof http://localhost:8080/debug/pprof/
```

### Check API Health

```bash
# Health check
curl http://localhost:8080/healthz

# Check order book
curl http://localhost:8080/orderbook

# Test API endpoint
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"trading_pair":"BTC-USD","side":"buy","type":"limit","price":50000.0,"quantity":1.0,"user_id":"test"}'
```

## 📞 Getting Help

If you can't resolve your issue:

1. **Check documentation:**
   - [README](../README.md)
   - [API Documentation](API.md)
   - [Development Guide](DEVELOPMENT.md)

2. **Search existing issues:**
   - GitHub Issues

3. **Create a new issue:**
   - Include error messages
   - Provide environment details
   - Share steps to reproduce
   - Include logs if available

4. **Ask for help:**
   - GitHub Discussions
   - Community forums

## 🛠️ Maintenance

### Regular Maintenance Tasks

1. **Monitor log files**
2. **Check disk space**
3. **Verify Redis persistence**
4. **Test backup recovery**
5. **Review error logs**
6. **Monitor performance metrics**

### Health Checks

```bash
# API health
curl http://localhost:8080/healthz

# Redis health
redis-cli ping

# NATS health (if configured)
# Check NATS monitoring
```

### Log Rotation

Configure log rotation to prevent disk space issues:

```bash
# Linux logrotate configuration
/var/log/cryptex/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
}
```

---

**Still having issues?** Please open a GitHub issue with detailed information about your problem.
