# Clean Architecture WS1 — Swapping the Data Source

The runnable reference implementation for Workshop 1: swap a SQL `UserRepository` for an LDAP/Active Directory one and prove that only the wiring changes. Part of the [WS1 workshop](../../clean_arch_ws1_en.md).

## What this is

The same "veteran check" use case runs twice — once against a fake SQL repository, once against a fake Active Directory repository. Because the use case depends only on the `domain.UserRepository` interface, the swap happens entirely in `main.go`'s `checkWith` wiring; `internal/domain` and `internal/usecase` are untouched. This is the repository-swap resilience of Clean Architecture, made executable.

## Requirements

- Go 1.23 (`go.mod`)
- No containers or external services (SQL and LDAP are fakes)

## How to run

```bash
go mod tidy
go run .
```

Verified output:

```text
== data source: SQL database ==
  alice: veteran=true
  bob: veteran=false
== data source: Active Directory (LDAP) ==
  alice: veteran=true
  bob: veteran=false
```

`go test ./...` runs the domain and usecase unit tests.

## Layout

```text
.
├── main.go            # Runs the use case against both repository implementations
└── internal/
    ├── domain/        # User entity + UserRepository interface (+ tests)
    ├── usecase/       # CheckVeteranUseCase (+ tests)
    └── adapters/infra/ # Fake SQL repo, LDAP client, and AD UserRepository
```
