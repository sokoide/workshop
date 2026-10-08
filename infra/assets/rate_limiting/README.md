# Rate Limiting Algorithms on Redis

A CLI that executes a single allow/deny decision using one of three rate-limiting algorithms (fixed window, sliding window, token bucket), each implemented as an atomic Lua script on Redis. Part of the [Rate Limiting Workshop](../../rate_limiting_en.md).

## What this is

Each invocation parses the algorithm flag, calls the matching limiter in `usecase/limiters.go` against `localhost:6379`, and prints whether the request was allowed or denied. The Lua scripts make the check-and-update step atomic, use the Redis server clock for token refill, and attach expiry to counters. Run the same command repeatedly to watch the limit engage.

## Requirements

- Go 1.23 (`go.mod`)
- Redis running on `localhost:6379` — start one with podman, e.g. `podman run -d --name workshop-redis -p 6379:6379 docker.io/library/redis:alpine` (the [redis_leaderboard asset](../redis_leaderboard/) has a matching `make redis-up`)

## How to run

```bash
go mod tidy

# Fixed window: 5 requests per 10s
go run main.go -algorithm fixed-window -user u1
# Output: [fixed-window] Allowed: User u1
# After 5 calls within the window:
# Output: [fixed-window] Denied: User u1 (Rate limit exceeded)

# Sliding window approximation
go run main.go -algorithm sliding-window -user u1

# Token bucket: capacity 10, refill 1 token/s
go run main.go -algorithm token-bucket -user u1
```

Flags: `-algorithm` (fixed-window | sliding-window | token-bucket), `-user`, `-limit`, `-window`, `-capacity`, `-rate`. The [workshop](../../rate_limiting_en.md) compares the three algorithms and their trade-offs.

## Layout

```text
.
├── main.go          # CLI: flag parsing + one decision per invocation
├── usecase/         # Limiters (Lua scripts on Redis)
└── domain/          # Rate-limit domain types and errors
```
