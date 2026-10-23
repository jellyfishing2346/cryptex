# Benchmarks

Measured with Go's benchmark tooling on a single core of an Intel Xeon @ 2.80GHz, Go 1.26, Linux. Absolute numbers depend on hardware; the scaling with book size is the part that carries over.

## How to run

```bash
# Order book and matching engine
go test -run='^$' -bench=. ./internal/orderbook/ ./internal/matching/

# Full HTTP request path, in memory
go test -run='^$' -bench=HTTPPlaceOrder ./internal/api/

# Full HTTP request path with Redis persistence (flushes the given database)
BENCH_REDIS_URL=redis://localhost:6379/15 go test -run='^$' -bench=HTTPPlaceOrderRedis ./internal/api/
```

"Levels" is the number of price levels on each side of the book. The matching benchmarks rest a sell order and then send a buy that fully matches it, so the book is the same size on every iteration. Latency percentiles time the matching submission alone.

## Matching engine (with risk checks, as the server runs)

| Price levels per side | p50 latency | p99 latency | Throughput | Memory per order |
|---|---|---|---|---|
| 10 | 1.1 µs | 6.0 µs | 664,000 orders/s | 968 B |
| 100 | 1.4 µs | 9.3 µs | 310,000 orders/s | 968 B |
| 1,000 | 2.9 µs | 30.5 µs | 103,000 orders/s | 968 B |
| 10,000 | 24 µs | 93 µs | 12,100 orders/s | 968 B |

Latency grows with depth because price levels are kept in sorted slices and adding a new level re-sorts them. A tree or skip list keyed by price would make that logarithmic.

### Before and after the risk-check fix

These benchmarks found that each risk check copied every order in the book to compute one user's position, and scanned every price level for self-trades. Both checks now use a per-user index of resting orders.

| Price levels per side | p50 before | p50 after | Throughput before | Throughput after | Memory per order before |
|---|---|---|---|---|---|
| 10 | 13.8 µs | 1.1 µs | 24,100/s | 664,000/s | 247 KB |
| 1,000 | 249 µs | 2.9 µs | 2,600/s | 103,000/s | 849 KB |
| 10,000 | 6.4 ms | 24 µs | 162/s | 12,100/s | 8.3 MB |

## Order book

| Operation | 10 levels | 100 | 1,000 | 10,000 |
|---|---|---|---|---|
| Add an order at a new price level, then cancel it | 1.4 µs | 3.5 µs | 13.7 µs | 111 µs |
| Top-10 snapshot (served by `GET /orderbook`, broadcast after every order) | 0.8 µs | 0.8 µs | 1.1 µs | 1.5 µs |
| Market order sweeping 10 levels | 8.5 µs | 9.4 µs | 15 µs | 45 µs |

## Full HTTP request (`POST /orders`)

JSON decoding, validation, risk checks, matching and the WebSocket broadcast, measured in-process (no network).

| Resting orders | In memory | With Redis persistence (local) |
|---|---|---|
| 20 | 31 µs | 158 µs |
| 1,000 | 39 µs | 2.7 ms |

Persistence dominates: after every order the server rewrites the whole resting book to Redis as one value, so the cost grows with the number of resting orders (1.4 MB allocated per request at 1,000). Storing orders individually and writing only what changed would make it constant. Over a network, add the round trip to Redis.

## Not yet measured

- Concurrent WebSocket connections
- Throughput under concurrent clients (the book has a single write lock, so orders are matched one at a time)
