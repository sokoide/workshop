# Clean Architecture Essentials: Understanding the Core with a Minimum Example (WS8)

> **⏱️ Estimated Time**: Approx. 30 minutes
> **🎯 Level**: Beginner
> **💻 Environment**: Go 1.23+, curl, Linux / macOS / WSL2

In this workshop, you will experience the design philosophy of Clean Architecture (3-layer variant: Adapters / UseCases / Domain) and its "resilience to change" through a "User Greeting App" consisting of only about 100 lines of code.

## Goal

By the end of this workshop, you can:

- Explain the three layers (Adapters / UseCases / Domain) and the inward dependency rule of this Clean Architecture variant
- Experience "resilience to change" in the ~100-line greeting app: swap the user storage by editing only the Infra Adapter and the Composition Root

## Prerequisites

- [ ] Go 1.23 or later (`go version`)
- [ ] Python 3 (`python3 --version`)
- [ ] curl (`curl --version`)
- [ ] No prior Clean Architecture knowledge required — this is the recommended entry point before `clean_arch.md` and WS1

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
├── domain/         (Layer 1: Innermost) Business rules, Entities, Ports (Interfaces), Domain Errors
├── usecase/        (Layer 2) Use cases — orchestration only, application DTOs, UseCase Ports
├── infra/          (Layer 3: Adapters — Infrastructure) Infrastructure implementation details
├── presentation/   (Layer 3: Adapters — Presentation) HTTP Handlers, external libraries
├── go.mod
└── main.go         (Composition Root) Assembling all layers (Dependency Injection)
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

## 4. Exercise: Experience "Resilience to Change"

### Step 1: Copy and Start the App

Start at the repository root. Python 3 is required. If `/tmp/greeting-ws8` already exists, choose a different name in every command below; the copy operation refuses to overwrite it.

```bash
# Terminal A — repository root
python3 -c 'import shutil; shutil.copytree("software/assets/greeting", "/tmp/greeting-ws8")'
cd /tmp/greeting-ws8
go run main.go
# Expected output: Starting server on http://localhost:8080 ...
```

Keep terminal A running; edit and verify in terminal B.

```bash
# Terminal B
cd /tmp/greeting-ws8
curl -i 'http://localhost:8080?id=1'
# Expected output: HTTP/1.1 200 OK, body Hello, Alice!
curl -i 'http://localhost:8080?id=999'
# Expected output: HTTP/1.1 404 Not Found, body user not found
```

### ✅ Checkpoint: Baseline

- [ ] Working in `/tmp/greeting-ws8`, leaving the repository original untouched
- [ ] Existing ID returns 200; missing ID returns 404

### Step 2: Run Unit Tests

```bash
# Terminal B — /tmp/greeting-ws8
go test ./usecase/... -v
# Expected output: PASS for both existing-user and missing-user cases
```

Read `usecase/greeting_test.go` to see how a fake repository allows tests without a database.

### Step 3: Change the Business Rule and Test

**Edit to an existing file:** change the return value in `usecase/greeting.go`:

```go
return "こんにちは、" + user.Name + "さん！", nil
```

Also change `expected := "Hello, TestUser!"` in `usecase/greeting_test.go` to `expected := "こんにちは、TestUserさん！"`. The old expectation fails because the behavior specification changed from English to Japanese.

Stop terminal A with Ctrl+C and restart `go run main.go`. In terminal B:

```bash
go test ./usecase/...
# Expected output: ok
curl -i 'http://localhost:8080?id=1'
# Expected output: 200 OK, body こんにちは、Aliceさん！
```

### ✅ Checkpoint: Business Rule Change

- [ ] Japanese output and the test expecting Japanese both succeed
- [ ] Only `usecase/greeting.go` and `usecase/greeting_test.go` were edited at this stage

### Step 4: Swap Storage

**Predict:** which files must change when replacing map storage with a slice? Answer before checking the diff below.

Save the Japanese version as the baseline for this second exercise:

```bash
# Terminal B — /tmp/greeting-ws8
mkdir .baseline
cp main.go .baseline/
cp -R domain usecase presentation infra .baseline/
# Expected output: none; .baseline stores the state after the Japanese change
```

**Complete new file:** save the following as `infra/slice.go`:

```go
package infra

import (
	"context"

	"github.com/sokoide/workshop/software/assets/greeting/domain"
)

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

func (r *SliceUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}
```

**Edit to an existing file:** replace `repo := infra.NewMemoryUserRepo()` in `main.go` with `repo := infra.NewSliceUserRepo()`.

Restart terminal A, then verify in terminal B:

```bash
go test ./...
# Expected output: all tests succeed
curl -i 'http://localhost:8080?id=1'
# Expected output: 200 OK, body こんにちは、Aliceさん！
curl -i 'http://localhost:8080?id=999'
# Expected output: 404 Not Found, body user not found
diff -ru .baseline/domain domain
diff -ru .baseline/usecase usecase
diff -ru .baseline/presentation presentation
# Expected output: none from all three commands (unchanged during the storage swap)
diff -u .baseline/main.go main.go
diff -ru .baseline/infra infra
# Expected output: one wiring-line change and new slice.go; diff exit code 1 is expected
```

### ✅ Checkpoint: Storage Change

- [ ] Japanese output and 404 remain the same after replacing storage
- [ ] Compared with the post-Japanese baseline, only `infra/` and `main.go` changed
- [ ] Tests pass without a DB, and the diff can be compared with the initial prediction

<details>
<summary>Hint and sample answer</summary>

`infra/slice.go` implements the interface and `main.go` selects the concrete type. The use case depends on `UserRepository`, so it does not know the concrete storage type.

</details>

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

**Next**: read [`clean_arch.md`](./clean_arch_en.md) for the full layer definitions, then proceed to [WS1](./clean_arch_ws1_en.md) to build layers yourself.

---

## Troubleshooting

| Symptom                       | Cause                                            | Fix                                                                                              |
| ----------------------------- | ------------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| `curl` cannot connect         | Server not running or another process uses :8080 | Check `go run main.go` output; free the port or change it in `main.go`                           |
| `go test ./usecase/...` fails | Module not built / wrong directory               | Run from `software/assets/greeting/` after `go mod tidy`                                         |
| Compile error after Step 4    | `slice.go` package path typo                     | Package must be `infra` and import `github.com/sokoide/workshop/software/assets/greeting/domain` |

---

## Cleanup

Stop terminal A with Ctrl+C. Review the changes, then remove the exercise copy in terminal B. No restoration of the original is needed.

```bash
cd /tmp
rm -r /tmp/greeting-ws8
# Expected output: exercise copy and .baseline removed; repository original remains intact
```

---

## References

- [The Clean Architecture — Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Go and Clean Architecture reference in this repo](./clean_arch_en.md)
