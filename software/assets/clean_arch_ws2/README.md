# Clean Architecture WS2 — Ports and Transparent Decorators

The runnable reference implementation for Workshop 2: swap a notification gateway behind a port, then wrap a repository in a caching decorator without the use case noticing. Part of the [WS2 workshop](../../clean_arch_ws2_en.md).

## What this is

`main.go` contains two exercises. Exercise 1 approves a membership twice, once with an `EmailNotifier` and once with a `SlackNotifier` — both implement the `port.NotificationGateway` interface, so this is a one-line wiring change. Exercise 2 passes the use case a plain SQL repository and then the same repository wrapped in `CachedUserRepository` (Decorator pattern); the repeated lookups hit the cache, which is visible because the fake SQL store counts its queries.

## Requirements

- Go 1.23 (`go.mod`)
- No containers or external services (notifiers and SQL are fakes)

## How to run

```bash
go mod tidy
go run .
```

Verified output:

```text
== Exercise 1: notification channel swap ==
[email from noreply@example.com] Membership approved: welcome, Alice!
[slack webhook https://hooks.slack.com/services/T000/B000/XXXX] Membership approved: welcome, Alice!

== Exercise 2: transparent cache ==
lookup #1: alice veteran=true (SQL queries so far: 1)
lookup #2: alice veteran=true (SQL queries so far: 1)
lookup #3: alice veteran=true (SQL queries so far: 1)
```

`go test ./...` runs the domain and usecase unit tests.

## Layout

```text
.
├── main.go            # Exercises 1 and 2 (wiring shown side by side)
└── internal/
    ├── domain/        # User entity + UserRepository interface
    ├── usecase/       # Approve / CheckVeteran use cases; port/ holds NotificationGateway
    └── adapters/infra/ # Email/Slack notifiers, SQL repo, cached repo decorator, cache store
```
