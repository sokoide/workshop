# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is a collection of hands-on technical workshops covering infrastructure and software architecture topics. The repository serves as educational material with practical, executable examples.

**Primary Language:** Go 1.25.5
**Documentation:** Bilingual (English/Japanese)
**Architecture Pattern:** Clean Architecture (4-layer structure)

## Repository Structure

```
/workshop/
├── infra/              # Infrastructure workshops (DNS, VLAN, K8s, TLS, etc.)
├── software/           # Software architecture workshops
│   ├── advent-of-calm-2025/    # Clean Architecture reference implementation
│   └── go_cache_patterns/      # Caching pattern examples (5 patterns)
├── conductor/          # Project management workflows and guidelines
└── Makefile            # Repository-level markdown formatting
```

## Clean Architecture (4-Layer Structure)

All Go projects in this repository follow Clean Architecture principles with dependencies pointing inward:

```
┌─────────────────────────────────────────┐
│  Framework Layer (Web/gRPC/CLI)         │
│  - Controllers, Handlers, Presenters    │
└──────────────┬──────────────────────────┘
               │ depends on
┌──────────────▼──────────────────────────┐
│  UseCase Layer                          │
│  - Application logic, orchestration     │
└──────────────┬──────────────────────────┘
               │ depends on
┌──────────────▼──────────────────────────┐
│  Domain Layer                           │
│  - Entities, Domain Services            │
│  - Repository Interfaces (Ports)        │
│  - NO external dependencies             │
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│  Infra Adapters Layer                   │
│  - Implements Domain interfaces         │
│  - Database, External APIs, MQ          │
└──────────────▲──────────────────────────┘
               │ implements
    Domain Interfaces (Ports)
```

**Key Principles:**

- **Domain Layer**: Pure Go with zero external dependencies. Contains entities, domain services, and repository interfaces.
- **UseCase Layer**: Orchestrates business logic using Domain interfaces. Unaware of external implementation details.
- **Infra Adapters Layer**: Concrete implementations of Domain interfaces (PostgreSQL, Redis, RabbitMQ, HTTP clients).
- **Framework Layer**: Entry points (CLI, HTTP handlers) that depend on UseCase interfaces.
- **Dependency Injection**: main.go wires all dependencies together.

**Directory Pattern per Project:**

```
project/
├── domain/         # Business entities and interfaces (no external deps)
├── usecase/        # Application logic orchestration
├── infra/          # External system implementations
├── cmd/            # CLI entry points (if applicable)
└── main.go         # Dependency injection
```

## Development Commands

### Repository Level

```bash
# Format all markdown files
make format
```

### Go Projects (Individual Workshops)

```bash
# Setup dependencies
go mod tidy

# Run application
go run main.go

# Run tests
go test ./...

# Run tests with coverage
go test ./... -cover

# Format code
go fmt ./...
```

### Infrastructure Workshops

```bash
# Redis leaderboard (infra/assets/redis_leaderboard/)
make redis-up        # Start Redis container (podman)
make redis-down      # Stop Redis container

# RabbitMQ crypto (infra/assets/rabbitmq_crypto/)
make mq-up           # Start RabbitMQ container
make mq-down         # Stop RabbitMQ container
```

## Testing Conventions

**Test Location:** Tests are co-located with implementation using Go's `*_test.go` convention.

**Testing Approach:**

- **Unit Tests:** Use mock implementations for domain interfaces
- **Integration Tests:** Test infrastructure adapters with real containers (Redis, RabbitMQ)
- **Test Structure:** Standard Go testing with table-driven tests where appropriate

**Example Test Pattern:**

```go
// Define mock struct implementing domain interface
type mockRepository struct { ... }

// Write test function
func TestFeature(t *testing.T) {
    repo := &mockRepository{...}
    uc := NewUsecase(repo)
    result, err := uc.Execute(context.Background(), input)

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    // Assertions...
}
```

## Go Code Style Conventions

- **Naming:** `MixedCaps` for exported names, `mixedCaps` for private (no underscores in names)
- **Formatting:** Use `gofmt` or `go fmt ./...`
- **Interfaces:** Defined in Domain layer, implemented in Infra layer
- **Context:** Always pass `context.Context` as first parameter for cancellation/timeout/tracing
- **Error Handling:** Explicit error checking, never ignore errors

## Project Management Workflow

The `/conductor/` directory contains structured development workflows for project-based work:

- **TDD (Test-Driven Development):** Red → Green → Refactor cycle
- **High Coverage:** Target >80% code coverage
- **Git Notes:** Attach task summaries to commits using `git notes add`
- **Checkpoints:** Create phase checkpoint commits with verification reports

## Serena MCP Integration

This repository uses Serena MCP with configuration in `/.serena/project.yml`:

- **Language:** Go (LSP enabled)
- **Gitignore:** Enabled for file filtering
- **Memories:** Project-specific memories available via `/sc:load` and `/sc:save`

## Important Architectural Decisions

**Why Clean Architecture?**

- **Resilience:** Swap databases/external systems without touching business logic
- **Testability:** Mock dependencies easily, test in isolation
- **Microservice Ready:** Clear domain boundaries enable future splitting

**Why Podman over Docker?**

- Daemonless architecture (no root daemon required)
- Better security model
- Drop-in Docker replacement

**Why Go?**

- Performance with compiled execution
- Built-in concurrency (goroutines, channels)
- Strong typing reduces bugs
- Rich standard library

## Working with This Repository

- Each workshop directory is self-contained
- Read workshop documentation (English or Japanese) before starting
- Follow Clean Architecture layering for any new code
- Use context.Context for all operations that may block or need cancellation
- Write tests before implementation (TDD approach)
- Run `go fmt ./...` before committing changes
- Container-based workshops use individual Makefiles for management
