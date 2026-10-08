# Idempotent Charge CLI

A CLI that executes a charge with an idempotency key: retries and duplicates must never charge twice, and a key reused for a different request is rejected. Part of the [Idempotency Workshop](../../idempotency_en.md).

## What this is

`main.go` parses a user, amount, and idempotency key, then drives `ChargeUseCase` through the full idempotency protocol: validate the request, check for an existing result, acquire an ownership lock (`SetNX`), re-check under the lock, update the balance, and store both request and response so future retries return the original result. Redis is the backing store (`infra/`), and balances default to 1000 on first use.

## Requirements

- Go 1.23 (`go.mod`)
- Redis on `localhost:6379` (e.g. `podman run -d --name workshop-redis -p 6379:6379 docker.io/library/redis:alpine`)

## How to run

```bash
go mod tidy
go run main.go -user u1 -amount 100 -idempotency-key order-123
# Output: the charge result; the balance drops by 100

# Same key again: the original result is replayed, no second charge
go run main.go -user u1 -amount 100 -idempotency-key order-123

# Same key with a different amount: rejected
go run main.go -user u1 -amount 200 -idempotency-key order-123
```

`go test ./...` runs the usecase, lock-adapter, and account-repository unit tests (using in-memory miniredis, so no server is needed). The [workshop](../../idempotency_en.md) explains each protocol step and its failure modes.

## Layout

```text
.
├── main.go         # CLI: flags + Redis-backed account repository
├── main_test.go    # Tests using an in-memory Redis (miniredis)
├── domain/         # Account repository interface
├── usecase/        # ChargeUseCase (idempotency protocol)
└── infra/          # Redis idempotency store (lock + stored results)
```
