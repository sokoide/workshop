# Greeting App — Minimal 3-Layer Clean Architecture

The smallest complete example of the 3-layer Clean Architecture (Adapters / UseCases / Domain): an HTTP greeting service in about 100 lines. Part of [Workshop 8](../../clean_arch_ws8_en.md), the recommended entry point before the [Clean Architecture Guide](../../clean_arch.md).

## What this is

A single HTTP handler looks up a user in an in-memory repository via a use case. Each concern lives in its own package: `domain/` (the `User` entity, `UserRepository` interface, and errors), `usecase/` (the greeting workflow), `presentation/` (the HTTP handler), and `infra/` (the memory repository). `main.go` is the only place that knows all the concrete types — it wires them together.

## Requirements

- Go 1.23 (`go.mod`)
- curl (or a browser); no containers needed

## How to run

```bash
go mod tidy
go run main.go
# Output: "Starting server on http://localhost:8080 (try ?id=1)"
```

From another terminal:

```bash
curl "http://localhost:8080?id=1"
# Output: Hello, Alice!

curl "http://localhost:8080?id=999"
# Output: user not found (HTTP 404)
```

`go test ./usecase/...` runs the usecase unit tests — with no database, thanks to the repository interface.

## Layout

```text
.
├── main.go            # Composition root: wiring + HTTP server (:8080)
├── domain/            # User entity, UserRepository interface, errors
├── usecase/           # GreetingUseCase (+ tests)
├── presentation/      # HTTP handler
└── infra/             # In-memory UserRepository
```
