# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is a collection of hands-on technical workshops covering infrastructure and software architecture topics. The repository serves as educational material with practical, executable examples.

**Primary Language:** Go 1.25.5
**Documentation:** Bilingual (English/Japanese)
**Architecture Pattern:** Clean Architecture (3-layer variant: Adapters / UseCases / Domain)

## Repository Structure

```text
/workshop/
├── infra/              # Infrastructure workshops (DNS, VLAN, K8s, TLS, etc.)
├── software/           # Software architecture workshops
│   ├── advent-of-calm-2025/    # Clean Architecture reference implementation
│   └── go_cache_patterns/      # Caching pattern examples (5 patterns)
├── conductor/          # Project management workflows and guidelines
└── Makefile            # Repository-level markdown formatting
```text

## Clean Architecture (3-Layer Variant: Adapters / UseCases / Domain)

All Go projects in this repository follow Clean Architecture principles with dependencies pointing inward:

```text
┌─────────────────────────────────────────────────────┐
│  Adapters Layer (side-effect boundary)              │
│  ┌──────────────────────┐ ┌───────────────────────┐ │
│  │ Presentation Adapters│ │ Infrastructure        │ │
│  │ (Inbound/Driving)    │ │ Adapters (Outbound/   │ │
│  │ - HTTP/gRPC/CLI      │ │  Driven)              │ │
│  │ - Controllers,       │ │ - Repository impls    │ │
│  │   Handlers,          │ │ - DB, External APIs,  │ │
│  │   Presenters         │ │   MQ, SDKs            │ │
│  └──────────┬───────────┘ └───────────────────────┘ │
└─────────────┼───────────────────────────────────────┘
              │ depends on
┌─────────────▼───────────────────────────────────────┐
│  UseCases Layer                                     │
│  - Application logic, orchestration                 │
│  - Input/output DTOs, UseCase-owned ports           │
└─────────────┬───────────────────────────────────────┘
              │ depends on
┌─────────────▼───────────────────────────────────────┐
│  Domain Layer                                       │
│  - Entities, Value Objects, Domain Services         │
│  - Domain Errors, Domain-owned Ports                │
│  - NO external dependencies                         │
└─────────────────────────────────────────────────────┘
```

**Key Principles:**

- **Domain Layer**: Pure Go with zero external dependencies. Contains entities, domain services, domain errors, and domain-owned ports.
- **UseCases Layer**: Orchestration only. Coordinates Domain objects and boundary interfaces. Defines input/output DTOs and UseCase-owned ports. Unaware of external implementation details.
- **Adapters Layer = side-effect boundary**: All I/O, external integrations, and framework interactions confined here.
  - **Presentation Adapters (Inbound)**: HTTP/gRPC/CLI handlers, controllers, presenters, request/response mapping.
  - **Infrastructure Adapters (Outbound)**: Repository implementations, external API gateways, DB models, error conversion.
- **Dependency Injection**: main.go wires all dependencies together.

**Directory Pattern per Project:**

```text
project/
├── internal/
│   ├── domain/                        # Entities, value objects, domain services, domain errors
│   ├── usecase/                       # Interactors, input/output DTOs, usecase-owned ports
│   ├── adapters/
│   │   ├── presentation/http/         # HTTP handlers/controllers/presenters
│   │   └── infra/persistence/         # Repository implementations, DB models
│   └── (presentation|infra)/          # Simplified layout for small projects
├── cmd/app/main.go                    # Composition root
└── main.go                            # Simplified entry point (small projects)
```

## Development Commands

### Repository Level

```bash
# Format all markdown files
make format
```text

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
```text

### Infrastructure Workshops

```bash
# Redis leaderboard (infra/assets/redis_leaderboard/)
make redis-up        # Start Redis container (podman)
make redis-down      # Stop Redis container

# RabbitMQ crypto (infra/assets/rabbitmq_crypto/)
make mq-up           # Start RabbitMQ container
make mq-down         # Stop RabbitMQ container
```text

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
```text

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
