# Advent of Calm 2025 — Clean Architecture Sample

A small e-commerce flow (create order, check inventory, update inventory) used as the runnable reference implementation of the 3-layer Clean Architecture style. Part of the [Clean Architecture Guide](../../clean_arch_en.md).

## What this is

`main.go` wires a `CreateOrderUsecase`, a `CheckInventoryUsecase`, and an `UpdateInventoryUsecase` and runs a customer flow followed by an admin flow. The infrastructure adapters (Postgres repositories, a REST inventory client, a RabbitMQ payment publisher) are stubs that log what they would do, so the whole scenario runs with no external systems — showing that the core logic is independent of them.

## Requirements

- Go 1.25.5 (`go.mod`)
- No containers or external services (adapters are logging stubs)

## How to run

```bash
go mod tidy
go run main.go
```

Verified output (order ID and timestamps vary):

```text
Calling Inventory Service REST API for product product-456
Saving order d6348328-... to Postgres
Publishing payment task for order d6348328-... to RabbitMQ
Fetching stock for product product-456 from Postgres
2026/10/08 07:35:14 Current stock: 100
Updating stock for product product-456 to 150 in Postgres
```

`go test ./...` runs the usecase unit tests.

## Layout

```text
.
├── main.go                    # Composition root + demo scenario
├── internal/
│   ├── domain/                # Entities and repository interfaces
│   ├── usecase/               # Create order / check / update inventory use cases (+ tests)
│   └── adapters/infra/        # Stub adapters: repository, client, messaging, util
└── docs/                      # Workshop diagrams
```
