# Heroku Deployment Guide

## Prerequisites

1. Install Heroku CLI:
   ```bash
   # macOS
   brew tap heroku/brew && brew install heroku
   ```

2. Login to Heroku:
   ```bash
   heroku login
   ```

## Deployment Steps

### 1. Create Heroku App
```bash
heroku create cryptex-app
```

### 2. Add Redis
```bash
heroku addons:create heroku-redis:mini -a cryptex-app
```

### 3. Set Environment Variables
```bash
# Basic configuration
heroku config:set TRADING_PAIR=BTC-USD -a cryptex-app
heroku config:set PORT=8080 -a cryptex-app
heroku config:set MAX_POSITION_SIZE=1000.0 -a cryptex-app
heroku config:set MIN_PRICE=0.01 -a cryptex-app
heroku config:set MAX_PRICE=1000000.0 -a cryptex-app

# Redis configuration (auto-set by addon, but verify)
heroku config:set REDIS_DB=0 -a cryptex-app

# NATS configuration
# Option 1: Use NATS cloud service (recommended)
heroku config:set NATS_URL=your-nats-cloud-url -a cryptex-app

# Option 2: Use embedded NATS (testing only)
heroku config:set NATS_URL=nats://localhost:4222 -a cryptex-app
```

### 4. Deploy Using Docker
```bash
# Login to Heroku Container Registry
heroku container:login

# Build and push the image
heroku container:push web -a cryptex-app

# Release the image
heroku container:release web -a cryptex-app
```

### 5. Scale the Application
```bash
heroku ps:scale web=1 -a cryptex-app
```

### 6. View Logs
```bash
heroku logs --tail -a cryptex-app
```

### 7. Open the Application
```bash
heroku open -a cryptex-app
```

## NATS Setup Options

### Option 1: NATS Cloud Service (Recommended)
1. Sign up for NATS cloud at https://synadia.com
2. Create a NATS account
3. Get your NATS URL
4. Set the NATS_URL environment variable

### Option 2: Self-Hosted NATS
1. Deploy NATS to a separate server
2. Configure network access
3. Set NATS_URL to your server address

### Option 3: Embedded NATS (Testing Only)
For development/testing, you can run NATS embedded in the same dyno, but this is not recommended for production.

## Database Configuration

Redis is automatically configured by the Heroku Redis addon. The `REDIS_URL` environment variable will be set automatically.

## Troubleshooting

### Application Crashes
Check the logs:
```bash
heroku logs --tail -a cryptex-app
```

### Redis Connection Issues
Verify Redis addon is provisioned:
```bash
heroku addons -a cryptex-app
```

### NATS Connection Issues
Verify NATS_URL is set correctly:
```bash
heroku config:get NATS_URL -a cryptex-app
```

## Scaling

### Add More Web Dynos
```bash
heroku ps:scale web=3 -a cryptex-app
```

### Upgrade Redis
```bash
heroku addons:upgrade heroku-redis:premium-0 -a cryptex-app
```

## Monitoring

### View Metrics
```bash
heroku ps -a cryptex-app
```

### Set Up Monitoring
Consider adding New Relic or Librato for advanced monitoring.

## Cost Estimation

- Eco dyno: $5/month
- Basic dyno: $7/month
- Redis Mini: $15/month
- Redis Premium: $50+/month

Estimated minimum: ~$20/month for basic setup.
