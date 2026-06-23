# Docker Deployment Guide

This guide covers deploying Cryptex using Docker containers, from local development to production environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Docker Compose](#docker-compose)
- [Building Images](#building-images)
- [Production Deployment](#production-deployment)
- [Multi-Stage Builds](#multi-stage-builds)
- [Docker Optimization](#docker-optimization)
- [Security Best Practices](#security-best-practices)
- [Troubleshooting](#troubleshooting)

## Prerequisites

- Docker 20.10+
- Docker Compose 2.0+
- At least 2GB RAM available
- 10GB free disk space

## Quick Start

### Using Docker Compose (Recommended)

The fastest way to get started is using Docker Compose:

```bash
# Start all services
docker-compose -f docker/docker-compose.yml up

# Run in detached mode
docker-compose -f docker/docker-compose.yml up -d

# View logs
docker-compose -f docker/docker-compose.yml logs -f

# Stop services
docker-compose -f docker/docker-compose.yml down
```

This will start:
- Redis on port 6379
- NATS on port 4222
- Monitoring ports 8222 (NATS monitoring)

### Manual Docker Build

```bash
# Build the Cryptex image
docker build -t cryptex:latest .

# Run with environment variables
docker run -d \
  --name cryptex \
  -p 8080:8080 \
  -e TRADING_PAIR=BTC-USD \
  -e REDIS_ADDR=redis:6379 \
  -e NATS_URL=nats://nats:4222 \
  --network cryptex-network \
  cryptex:latest
```

## Docker Compose

### Basic Configuration

The current `docker/docker-compose.yml` includes:

```yaml
version: '3.8'

services:
  nats:
    image: nats:latest
    ports:
      - "4222:4222"
      - "8222:8222"
    command: "-js"
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8222/varz"]
      interval: 5s
      timeout: 3s
      retries: 3

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 3
```

### Extended Development Configuration

Create `docker/docker-compose.dev.yml` for development:

```yaml
version: '3.8'

services:
  cryptex:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
      - "9090:9090"  # Metrics
    environment:
      - TRADING_PAIR=BTC-USD
      - REDIS_ADDR=redis:6379
      - NATS_URL=nats://nats:4222
      - ENABLE_METRICS=true
      - METRICS_PORT=9090
      - GOMAXPROCS=4
    depends_on:
      redis:
        condition: service_healthy
      nats:
        condition: service_healthy
    volumes:
      - ./web:/app/web
    restart: unless-stopped

  redis:
    volumes:
      - redis-data:/data
    command: redis-server --appendonly yes

  nats:
    volumes:
      - nats-data:/data

volumes:
  redis-data:
  nats-data:
```

### Production Configuration

Create `docker/docker-compose.prod.yml` for production:

```yaml
version: '3.8'

services:
  cryptex:
    image: cryptex:latest
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - TRADING_PAIR=BTC-USD
      - REDIS_ADDR=redis:6379
      - REDIS_PASSWORD=${REDIS_PASSWORD}
      - NATS_URL=nats://nats:4222
      - ENABLE_METRICS=true
      - METRICS_PORT=9090
      - GOMAXPROCS=8
      - GOGC=50
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '2'
          memory: 1G
        reservations:
          cpus: '1'
          memory: 512M
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
    depends_on:
      redis:
        condition: service_healthy
      nats:
        condition: service_healthy
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD} --appendonly yes
    volumes:
      - redis-data:/data
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M

  nats:
    image: nats:2.10-alpine
    command: "-js"
    volumes:
      - nats-data:/data
    deploy:
      resources:
        limits:
          cpus: '0.5'
          memory: 256M
        reservations:
          cpus: '0.25'
          memory: 128M

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'

volumes:
  redis-data:
  nats-data:
  prometheus-data:
```

### Using Multiple Compose Files

```bash
# Base + development
docker-compose -f docker/docker-compose.yml -f docker/docker-compose.dev.yml up

# Base + production
docker-compose -f docker/docker-compose.yml -f docker/docker-compose.prod.yml up -d

# Override specific values
docker-compose -f docker/docker-compose.yml -f docker/docker-compose.prod.yml \
  --env-file .env.prod up -d
```

## Building Images

### Standard Build

```bash
# Build with default tag
docker build -t cryptex:latest .

# Build with custom tag
docker build -t cryptex:v1.0.0 .

# Build without cache
docker build --no-cache -t cryptex:latest .

# Build with build arguments
docker build --build-arg VERSION=1.0.0 -t cryptex:latest .
```

### Platform-Specific Builds

```bash
# Build for Linux AMD64
docker build --platform linux/amd64 -t cryptex:latest .

# Build for Linux ARM64
docker build --platform linux/arm64 -t cryptex:latest .

# Build for multiple platforms
docker buildx build --platform linux/amd64,linux/arm64 -t cryptex:latest .
```

### Optimized Build

```bash
# Build with Go build optimizations
docker build \
  --build-arg CGO_ENABLED=0 \
  --build-arg GOOS=linux \
  --build-arg GOARCH=amd64 \
  -t cryptex:latest .
```

## Production Deployment

### Registry Setup

```bash
# Login to Docker Hub
docker login

# Tag for registry
docker tag cryptex:latest your-registry/cryptex:latest

# Push to registry
docker push your-registry/cryptex:latest

# Push with version tag
docker tag cryptex:latest your-registry/cryptex:v1.0.0
docker push your-registry/cryptex:v1.0.0
```

### Environment Variables

Create `.env` file for production:

```bash
# Trading Configuration
TRADING_PAIR=BTC-USD
MAX_POSITION_SIZE=1000.0
MIN_PRICE=0.01
MAX_PRICE=1000000.0

# Infrastructure
REDIS_ADDR=redis:6379
REDIS_PASSWORD=your-secure-password
REDIS_DB=0
NATS_URL=nats://nats:4222
PORT=8080

# Performance
GOMAXPROCS=8
GOGC=50
READ_TIMEOUT=30
WRITE_TIMEOUT=30

# Security
ENABLE_METRICS=true
METRICS_PORT=9090
```

### Running in Production

```bash
# Load environment variables
export $(cat .env | xargs)

# Run with production settings
docker run -d \
  --name cryptex \
  --restart unless-stopped \
  -p 8080:8080 \
  -p 9090:9090 \
  --env-file .env \
  --health-cmd="curl -f http://localhost:8080/healthz || exit 1" \
  --health-interval=30s \
  --health-timeout=10s \
  --health-retries=3 \
  --memory=1g \
  --cpus=2 \
  your-registry/cryptex:latest
```

### Docker Swarm Deployment

```bash
# Initialize swarm
docker swarm init

# Deploy stack
docker stack deploy -c docker/docker-compose.prod.yml cryptex

# Scale services
docker service scale cryptex_cryptex=5

# View services
docker service ls
docker service ps cryptex_cryptex

# Update service
docker service update --image cryptex:v2.0.0 cryptex_cryptex

# Remove stack
docker stack rm cryptex
```

## Multi-Stage Builds

### Optimized Dockerfile

Create an optimized `Dockerfile` for production:

```dockerfile
# Build stage
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags='-w -s' \
    -o cryptex ./cmd/server

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates curl

# Create non-root user
RUN addgroup -g 1000 cryptex && \
    adduser -D -u 1000 -G cryptex cryptex

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/cryptex .
COPY --from=builder /app/web ./web

# Change ownership
RUN chown -R cryptex:cryptex /app

# Switch to non-root user
USER cryptex

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/healthz || exit 1

# Run the application
CMD ["./cryptex"]
```

### Build with Multi-Stage Dockerfile

```bash
# Build using the optimized Dockerfile
docker build -f Dockerfile.optimized -t cryptex:latest .

# Compare image sizes
docker images cryptex
```

## Docker Optimization

### Image Size Reduction

```bash
# Use Alpine-based images
FROM alpine:latest

# Remove build dependencies
RUN apk add --no-cache --virtual .build-deps gcc musl-dev && \
    go build ... && \
    apk del .build-deps

# Use .dockerignore
cat > .dockerignore << EOF
.git
.gitignore
*.md
docs/
.env
vendor/
EOF
```

### Layer Caching

```dockerfile
# Copy dependencies first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build application
RUN go build ...
```

### BuildKit Features

```bash
# Enable BuildKit
export DOCKER_BUILDKIT=1

# Use build cache mounts
docker build \
  --mount=type=cache,target=/go/pkg/mod \
  -t cryptex:latest .

# Use secret mounts
docker build \
  --secret=id=secrets,src=secrets.txt \
  -t cryptex:latest .
```

## Security Best Practices

### Non-Root User

```dockerfile
# Create non-root user
RUN addgroup -g 1000 cryptex && \
    adduser -D -u 1000 -G cryptex cryptex

# Use non-root user
USER cryptex
```

### Minimal Base Images

```dockerfile
# Use minimal base image
FROM alpine:latest

# Or use distroless
FROM gcr.io/distroless/static
```

### Secrets Management

```bash
# Use Docker secrets
docker secret create redis_password redis_password.txt

# Reference in compose file
services:
  cryptex:
    secrets:
      - redis_password
    environment:
      - REDIS_PASSWORD_FILE=/run/secrets/redis_password

secrets:
  redis_password:
    external: true
```

### Image Scanning

```bash
# Scan with Docker
docker scan cryptex:latest

# Scan with Trivy
trivy image cryptex:latest

# Scan with Clair
clairctl scan cryptex:latest
```

### Network Isolation

```bash
# Create isolated network
docker network create cryptex-network

# Run containers in isolated network
docker run --network cryptex-network cryptex:latest

# Don't expose unnecessary ports
docker run -p 127.0.0.1:8080:8080 cryptex:latest
```

## Troubleshooting

### Container Won't Start

```bash
# Check container logs
docker logs cryptex

# Check container status
docker ps -a

# Inspect container
docker inspect cryptex

# Run in interactive mode for debugging
docker run -it --entrypoint /bin/sh cryptex:latest
```

### Network Issues

```bash
# List networks
docker network ls

# Inspect network
docker network inspect cryptex-network

# Test connectivity
docker exec cryptex ping redis
docker exec cryptex curl http://redis:6379
```

### Performance Issues

```bash
# Check container stats
docker stats cryptex

# Check resource limits
docker inspect cryptex | grep -A 10 Memory
docker inspect cryptex | grep -A 10 Cpu

# Increase resources
docker update --memory=2g --cpus=4 cryptex
```

### Volume Issues

```bash
# List volumes
docker volume ls

# Inspect volume
docker volume inspect redis-data

# Clean up volumes
docker volume prune
```

### Build Issues

```bash
# Clean build cache
docker builder prune

# Clean everything
docker system prune -a

# Rebuild without cache
docker build --no-cache -t cryptex:latest .
```

## Maintenance

### Updating Containers

```bash
# Pull new image
docker pull your-registry/cryptex:latest

# Stop and remove old container
docker stop cryptex
docker rm cryptex

# Run new container
docker run -d --name cryptex your-registry/cryptex:latest

# Or use rolling update
docker run -d --name cryptex-new your-registry/cryptex:latest
docker stop cryptex
docker rm cryptex
docker rename cryptex-new cryptex
```

### Backup and Restore

```bash
# Backup volumes
docker run --rm -v redis-data:/data -v $(pwd):/backup \
  alpine tar czf /backup/redis-backup.tar.gz /data

# Restore volumes
docker run --rm -v redis-data:/data -v $(pwd):/backup \
  alpine tar xzf /backup/redis-backup.tar.gz -C /
```

### Monitoring

```bash
# Container logs
docker logs -f cryptex

# Resource usage
docker stats cryptex

# Health status
docker inspect --format='{{.State.Health.Status}}' cryptex
```

## Additional Resources

- [Docker Documentation](https://docs.docker.com/)
- [Docker Compose Reference](https://docs.docker.com/compose/)
- [Docker Best Practices](https://docs.docker.com/develop/dev-best-practices/)
- [Go Docker Best Practices](https://github.com/golang/go/wiki/Docker)

---

For general deployment information, see [DEPLOYMENT.md](../DEPLOYMENT.md).
