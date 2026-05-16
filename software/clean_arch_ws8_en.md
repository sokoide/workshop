# Clean Architecture Essentials: Understanding the Core with a Minimum Example (WS8)

In this workshop, you will experience the design philosophy of Clean Architecture (3-layer variant: Adapters / UseCases / Domain) and its "resilience to change" through a "User Greeting App" consisting of only about 100 lines of code.

## 1. Overview

The essence of Clean Architecture lies in **"Control of Dependencies."** The goal is to point dependencies inward so that business rules (Domain) are not contaminated by external concerns (databases, web frameworks, etc.).

This variant uses three conceptual layers. The **Adapters** layer is split into **Presentation Adapters** (inbound/driving) and **Infrastructure Adapters** (outbound/driven) to prevent monolithic growth and clarify the side-effect boundary.

### App Features

- Receives an HTTP request (`GET /?id=1`)
- Retrieves the username for the specified ID from a database (in-memory this time)
- Generates and returns a greeting: `Hello, [Name]!`

---

## 2. Directory Structure and Dependency Rules

Dependencies always point **from the outside in**.

```text
software/assets/greeting/
├── internal/domain/       (Layer 1: Innermost) Business rules, Entities, Ports (Interfaces), Domain Errors
├── internal/usecase/      (Layer 2) Use cases — orchestration only, application DTOs, UseCase Ports
├── internal/adapters/infra/        (Layer 3: Adapters — Infrastructure) Infrastructure implementation details
├── internal/adapters/presentation/ (Layer 3: Adapters — Presentation) HTTP Handlers, external libraries
└── main.go       (Composition Root) Assembling all layers (Dependency Injection)
```

> **Note:** `infra/` and `presentation/` are both part of the Adapters layer, split by direction (outbound vs inbound). In larger projects, they would typically live under `internal/adapters/infra/` and `internal/adapters/presentation/`.

```text
Presentation Adapters ---→ UseCases ---→ Domain
                             ↑              ↑
                             +--------------+-- Infrastructure Adapters
        implements ports owned by UseCases or Domain
```

### Dependency Matrix

| From / To                 | Domain  | UseCases | Adapters |
| ------------------------- | ------- | -------- | -------- |
| Domain                    | yes     | no       | no       |
| UseCases                  | yes     | yes      | no       |
| Adapters (Presentation)   | limited | yes      | self     |
| Adapters (Infrastructure) | yes     | yes      | self     |
| Composition Root          | yes     | yes      | yes      |

- `Presentation → Domain` is `limited`: Presentation MAY read Domain values returned by UseCases for serialization, but MUST NOT invoke Domain behavior directly for workflow decisions.
- Presentation Adapters and Infrastructure Adapters are in the same conceptual layer but must not depend on each other directly.

---

## 3. Explanation of Each Layer

### 3-1. Domain Layer: The "Core" of the System

The `domain/` directory defines knowledge that is invariant to this system.

- `user.go`: The concept of a "User" (Entity).
- `repository.go`: The "window" (Port) for how to retrieve a user. Defined as an interface; concrete database operations are not written here.
- `errors.go`: Error types defined in the Domain layer. **Technical errors from the Infra layer are not leaked outward; instead, they are converted to semantically meaningful domain errors.**

### 3-2. UseCases Layer: The "Steps" of a Scenario

The `usecase/` directory contains business logic that defines "what to do and in what order."

- `GreetingUseCase` assembles the greeting string through the `UserRepository` "window" without knowing where the data comes from.
- UseCase depends only on Domain. It is unaware of concrete Adapter implementations.

### 3-3. Infrastructure Adapters: Technical "Details" (Outbound)

The `infra/` directory implements the "contents" of the interfaces defined in the Domain layer.

- This time we use `MemoryUserRepo` to hold data in memory, but you can swap this with MySQL or an external API without breaking other layers.
- **Error Boundary**: When a user is not found, a domain error (`domain.ErrUserNotFound`) is returned instead of a technical error.

### 3-4. Presentation Adapters: "Contact Point" with the Outside (Inbound)

The `presentation/` directory handles communication protocols like HTTP.

- It focuses solely on extracting the ID from the request, passing it to the UseCase, and returning the result as a response.
- **Error Conversion**: Domain errors returned from UseCase/Domain are converted to HTTP status codes (e.g., `ErrUserNotFound` → 404).

---

## 4. Workshop: Experience "Resilience to Change"

### Step 1: Start the App

Run the following in your terminal to start the server:

```bash
cd software/assets/greeting
go run main.go
```

Send a request from another terminal to verify the behavior:

```bash
curl "http://localhost:8080?id=1"
# Output: Hello, Alice!

curl "http://localhost:8080?id=999"
# Output: user not found (HTTP 404)
```

### Step 2: Run Unit Tests

A benefit of Clean Architecture is the ability to swap external environments (like DBs) with Mocks and test business logic at high speed.

```bash
go test ./internal/usecase/... -v
```

Read `usecase/greeting_test.go` and check the following:

- How the repository is swapped with a fake (Mock)
- How the error case for a missing user is tested

### Step 3: [Exercise] Change Business Rules

Open `usecase/greeting.go` and try changing the greeting message (e.g., to Japanese: `こんにちは、[Name]さん！`).

- **Key Point**: Observe that you don't need to touch any code in `infra` or `presentation` for this change.

### Step 4: [Exercise] Swap the Infrastructure

This step lets you experience the greatest strength of Clean Architecture.

1. Create a new file `slice.go` in `infra/` implementing a slice-based repository:

```go
package infra

import "workshop/greeting/domain"

type SliceUserRepo struct {
	users []*domain.User
}

func NewSliceUserRepo() *SliceUserRepo {
	return &SliceUserRepo{
		users: []*domain.User{
			{ID: "1", Name: "Alice"},
			{ID: "2", Name: "Bob"},
		},
	}
}

func (r *SliceUserRepo) FindByID(id string) (*domain.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}
```

1. Change one line in `main.go`:

```go
// Before
repo := infra.NewMemoryUserRepo()
// After
repo := infra.NewSliceUserRepo()
```

1. Restart the server and verify it works the same way.

- **Key Point**: You only touched the Infrastructure Adapter (`infra/`) and the Composition Root (`main.go`). `domain/`, `usecase/`, and `presentation/` remain completely unchanged. This is the practical experience of "Dependency Inversion."

---

## 5. Advanced Topics: Error and Data Boundaries

In Clean Architecture, there are clear boundaries for errors and data between layers.

### Error Boundary

```text
Infrastructure Adapters → Converts driver errors to domain errors before returning
UseCases Layer           → Propagates domain errors as-is
Presentation Adapters    → Converts domain errors to HTTP statuses (404, 500, etc.)
```

This design prevents database-specific errors (e.g., `sql.ErrNoRows`) from leaking into the UseCases or Presentation layers.

### Data Boundary (DTO)

In production, UseCase inputs and outputs are defined as explicit structs (DTOs: Data Transfer Objects), separating Entities (`domain.User`) from Presentation request/response types. This minimal example uses primitive types, but DTO introduction is recommended as the system grows.

---

## 6. Summary

1. **Domain depends on nothing**: Isolate the core knowledge and errors of the business.
2. **UseCases talk through Ports (Interfaces)**: Logic can be completed without knowing implementation details (DB, etc.).
3. **Respect Error Boundaries**: Technical errors are converted to domain errors in the Infrastructure Adapters, then to HTTP statuses in the Presentation Adapters.
4. **Adapters = side-effect boundary**: All I/O and external integrations are confined to the Adapters layer.
5. **Dependency Injection (DI)**: By snapping each part together in `main.go`, the entire system becomes operational.

This is the first step toward a "Clean" design that is highly maintainable, easy to test, and resilient to change.
