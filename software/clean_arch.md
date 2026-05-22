# Clean Architecture Structure

This document serves as the single source of truth for the architecture pattern used across this repository.

## 3-Layer Variant: Adapters / UseCases / Domain

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

### Key Principles

- **Domain Layer**: Pure Go with zero external dependencies. Contains entities, domain services, domain errors, and domain-owned ports.
- **UseCases Layer**: Orchestration only. Coordinates Domain objects and boundary interfaces. Defines input/output DTOs and UseCase-owned ports. Unaware of external implementation details.
- **Adapters Layer = side-effect boundary**: All I/O, external integrations, and framework interactions confined here.
  - **Presentation Adapters (Inbound)**: HTTP/gRPC/CLI handlers, controllers, presenters, request/response mapping.
  - **Infrastructure Adapters (Outbound)**: Repository implementations, external API gateways, DB models, error conversion.
- **Dependency Injection**: `main.go` acts as the Composition Root to wire all dependencies together.

---

## Error Boundary Flow

Error handling follows a layered transformation pattern to maintain boundary integrity:

1. **Infrastructure Adapter**: Converts technical driver errors (e.g., `sql.ErrNoRows`) into domain-specific errors (e.g., `domain.ErrNotFound`).
2. **UseCase Layer**: Propagates domain errors as-is, remaining agnostic to technical implementations.
3. **Presentation Layer**: Converts domain errors into transport-specific errors (e.g., HTTP 404, gRPC `codes.NotFound`).
