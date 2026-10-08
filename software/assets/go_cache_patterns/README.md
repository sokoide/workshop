# Go Cache Patterns — 5 Executable Examples

Five standalone, runnable programs covering the standard caching patterns in Go, from a thread-safe TTL map to a distributed Redis cache. Part of the [Cache Patterns catalog](../../go_cache_patterns_en.md).

## What this is

Each `patternN_*` directory is an independent `main` package demonstrating one pattern with in-memory fakes (or real Redis where the pattern demands it). The programs finish by themselves and print cache behavior — hits, misses, evictions — so each pattern can be verified by running it.

## Requirements

- Go 1.26.0 (`go.mod`)
- Redis on `localhost:6379` for `pattern4_redis_cache` only (e.g. `podman run -d --name workshop-redis -p 6379:6379 docker.io/library/redis:alpine`)

## How to run

```bash
go mod tidy
cd pattern1_ttl_cache && go run .   # then pattern2_*, pattern3_*, pattern4_*, pattern5_*
```

Verified outputs (abridged):

```text
pattern1_ttl_cache:               hit: alice / miss: expired
pattern2_lru_cache:               hit a: alpha / b evicted
pattern3_cache_aside_singleflight: worker 0..4 got value-for-user:42 / store hits: 1
pattern4_redis_cache:             set/get round-trip against Redis (needs a server)
pattern5_write_through:           read: paid (write-through update served from cache)
```

`go test ./...` covers the patterns that ship tests.

## Layout

```text
.
├── pattern1_ttl_cache/                 # Thread-safe in-memory TTL cache
├── pattern2_lru_cache/                 # Bounded LRU eviction
├── pattern3_cache_aside_singleflight/  # Cache-aside + singleflight collapse
├── pattern4_redis_cache/               # Distributed cache with go-redis
└── pattern5_write_through/             # Write-through cache
```
