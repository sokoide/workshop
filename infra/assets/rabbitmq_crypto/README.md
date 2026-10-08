# RabbitMQ Crypto Market Simulator

A publish/subscribe crypto price ticker over a RabbitMQ topic exchange, with topic-keyed consumers that each receive a different slice of the market. Part of the [RabbitMQ Workshop](../../rabbitmq_en.md).

## What this is

`cmd/ticker` publishes fake trades every 0.5 seconds with routing keys like `market.btc.usd` to the `crypto_market` topic exchange. Three consumers subscribe with different binding keys: `cmd/logger` (everything, `market.#`), `cmd/japandesk` (JPY pairs only, `market.*.jpy`), and `cmd/alert` (BTC trades, `market.btc.#`). Messaging wiring lives in `internal/infra/rabbitmq`; domain logic and trade simulation in `internal/domain` and `internal/usecase`.

## Requirements

- Go 1.25.5 (`go.mod`)
- Podman (or Docker) for RabbitMQ
- Makefile targets: `mq-up` / `mq-down` (RabbitMQ 3-management on ports 5672 / 15672)

## How to run

```bash
make mq-up   # start RabbitMQ (guest/guest)
go mod tidy
```

In separate terminals:

```bash
go run cmd/ticker/main.go
# Output: "Ticker starting... (Ctrl+C to stop)", publishing every 0.5s

go run cmd/logger/main.go
# Output: "Logger starting (market.#)..." followed by a JSON trade every 0.5s
# e.g. {"id":"...","symbol":"btc","target_currency":"usd",...}

go run cmd/japandesk/main.go   # JPY pairs only (market.*.jpy)
go run cmd/alert/main.go       # large BTC trades only (market.btc.#)
```

Open <http://localhost:15672> (guest/guest) to inspect the `crypto_market` exchange, its queues, and bindings. The [workshop](../../rabbitmq_en.md) steps through the routing exercises; `go test ./...` covers the domain and observer logic with unit tests.

## Layout

```text
.
├── cmd/
│   ├── ticker/        # Publisher: fake trades every 0.5s
│   ├── logger/        # Consumer: market.# (all trades)
│   ├── japandesk/     # Consumer: market.*.jpy (JPY pairs)
│   └── alert/         # Consumer: market.btc.# (BTC trades)
├── internal/
│   ├── domain/        # Trade model + repository interface
│   ├── usecase/       # Observer pattern + market simulation
│   └── infra/rabbitmq/ # Connection, publisher, subscriber
└── Makefile           # mq-up / mq-down
```
