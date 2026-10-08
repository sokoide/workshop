# Redis Game Leaderboard

A CLI that operates a real-time game leaderboard on Redis sorted sets (add score, top N, rank lookup, ban user), structured with Clean Architecture. Part of the [Redis Workshop](../../cache_en.md).

## What this is

`main.go` selects an action with the `-action` flag and drives it through a use case and a Redis repository. Scores accumulate with ZINCRBY, top rankers come back instantly with ZREVRANGE, and banned users are tracked in a plain set so they can be excluded from rankings.

## Requirements

- Go 1.25.6 (`go.mod`)
- Podman for Redis
- Makefile targets: `redis-up` / `redis-down` (Redis alpine on port 6379)

## How to run

```bash
make redis-up
go mod tidy

go run main.go -action add -user user1 -score 100
go run main.go -action add -user user2 -score 250
go run main.go -action add -user user3 -score 180

go run main.go -action top -n 3
# Output: user2(250), user3(180), user1(100)

go run main.go -action rank -user user2
go run main.go -action ban -user user3   # excluded from subsequent top listings
```

Redis address defaults to `localhost:6379`; override with `REDIS_ADDR`. `go test ./...` runs the usecase tests. The [workshop](../../cache_en.md) explains each Redis command and data structure choice.

## Layout

```text
.
├── main.go         # CLI entry point (add / top / rank / ban)
├── domain/         # Leaderboard entities
├── usecase/        # Leaderboard use cases
├── infra/redis/    # Redis repository (sorted sets + ban set)
└── Makefile        # redis-up / redis-down
```
