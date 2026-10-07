# Clean Architecture Workshop (WS1): Learning Resilient Design

> **⏱️ Estimated Time**: Approx. 60 minutes
> **🎯 Level**: Beginner
> **💻 Environment**: Go 1.23+, any OS (zero external dependencies)
>
> > **💡 Glossary**: For technical terms such as [Clean Architecture](../infra/glossary_en.md#architecture), [Interface](../infra/glossary_en.md#go), and [Context](../infra/glossary_en.md#go), please refer to the [Glossary](../infra/glossary_en.md).

In this workshop, we will extend the existing example by adding business rules and replacing the infrastructure, experiencing the benefits of Clean Architecture firsthand.

## Workshop Scenario

1. **Add Business Rules:** Implement the rule "Employees with 5 or more years of service are recognized as veterans" using Entities and Domain Services.
2. **Infrastructure Swap:** Change the data storage from a SQL database to Active Directory (LDAP). Confirm that no changes are required in the UseCase or Domain layers.

## Prerequisites

- [ ] Go 1.23+ installed — verify with `go version`
- [ ] You have read the [Clean Architecture guide](./clean_arch.md) (3-layer variant: Adapters / UseCases / Domain)
- [ ] A workspace ready (see the next section)

### Setting Up Your Workspace

A complete, runnable reference implementation of this workshop lives at [`software/assets/clean_arch_ws1/`](./assets/clean_arch_ws1/). You can run it (`go run .`) at any time to compare with your own code or to see the finished result.

Pick one of two ways to follow along:

```bash
# Option A: copy the reference implementation and modify it (quick start)
cp -r software/assets/clean_arch_ws1 ~/ws1 && cd ~/ws1

# Option B: start from an empty module and type everything yourself
mkdir ws1 && cd ws1
go mod init example.com/ws1
mkdir -p internal/domain internal/usecase internal/adapters/infra
```

> The code snippets below use the reference module path `github.com/sokoide/workshop/software/assets/clean_arch_ws1`. If you chose Option B, replace it with the module path you passed to `go mod init`.

---

## Exercise 1: Implementing Entity and Domain Service

Encapsulate the business knowledge of "Veteran Determination" within the Domain layer.

### 1-1. Creating the Entity

Give the User attributes and the knowledge to calculate years of service.

```go
// internal/domain/user.go
package domain

import "time"

// User is the user entity.
type User struct {
	ID       string
	Name     string
	JoinedAt time.Time // Joining date
}

// GetTenureYears returns the number of years of service as of now.
func (u *User) GetTenureYears() int {
	now := time.Now()
	years := now.Year() - u.JoinedAt.Year()
	if now.YearDay() < u.JoinedAt.YearDay() {
		years--
	}
	return years
}
```

### 1-2. Creating the Domain Service

Criteria like "What defines a veteran?" are better defined as a "Service" rather than within the Entity itself.

```go
// internal/domain/veteran_service.go
package domain

// VeteranService decides who counts as a veteran.
type VeteranService struct{}

// IsVeteran determines if a user is a veteran (5 or more years of service).
func (s VeteranService) IsVeteran(user *User) bool {
	return user.GetTenureYears() >= 5
}
```

### 1-3. Defining the Repository Port

The UseCase will fetch users through a repository, so the Domain layer defines the port (interface) first. Implementations will live in the Infra Adapters layer — this is the seam that makes the infrastructure swap in Exercise 2 possible. (Repository ports may also live in the UseCase layer when persistence is treated as application orchestration — the BBS workshops do it that way. See the [Port Ownership Guidance](./clean_arch_en.md#port-ownership-guidance); this workshop keeps the contract in the Domain because `UserRepository` is part of this domain's vocabulary.)

```go
// internal/domain/user_repository.go
package domain

import "context"

// UserRepository is the output port for user persistence. The UseCase layer
// depends on this interface — never on SQL drivers, LDAP clients, or any
// other infrastructure detail.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
}
```

```go
// internal/domain/errors.go
package domain

import "errors"

// ErrUserNotFound is returned when the repository cannot find the user.
// It belongs to the Domain layer so every caller can test for it with
// errors.Is, no matter which infrastructure produced it.
var ErrUserNotFound = errors.New("user not found")
```

### 1-4. UseCase Implementation

Combine Domain objects to realize the use case.

```go
// internal/usecase/check_veteran.go
package usecase

import (
	"context"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
)

// CheckVeteranUseCase answers "is this user a veteran?".
type CheckVeteranUseCase struct {
	repo       domain.UserRepository
	veteranSvc domain.VeteranService
}

func NewCheckVeteranUseCase(repo domain.UserRepository, veteranSvc domain.VeteranService) *CheckVeteranUseCase {
	return &CheckVeteranUseCase{repo: repo, veteranSvc: veteranSvc}
}

func (uc *CheckVeteranUseCase) Execute(ctx context.Context, id string) (bool, error) {
	user, err := uc.repo.FindByID(ctx, id) // Retrieved via repository
	if err != nil {
		return false, err
	}
	return uc.veteranSvc.IsVeteran(user), nil
}
```

> **Note on interface design:** The UseCase depends on the concrete `domain.VeteranService` directly rather than defining a separate `VeteranChecker` interface. This is a **simplicity vs. decoupling trade-off**: since `VeteranService` is a pure Domain Service with no external dependencies, the risk of it gaining unwanted side effects is low, and an extra interface layer adds ceremony without practical benefit. The downside is that if `VeteranService`'s method signature changes, the UseCase must also change. For Domain Services that may later acquire external dependencies (e.g., a lookup service that might call an API), defining an interface upfront is the safer choice.

### ✅ Checkpoint

- [ ] `internal/domain/user.go` and `internal/domain/veteran_service.go` compile — `go build ./...` exits with no output
- [ ] `CheckVeteranUseCase` compiles with **zero imports from the `infra` package**
- [ ] You can write a unit test for `VeteranService.IsVeteran` without any DB or HTTP setup

Verify — the Domain layer is testable on its own, with no database or HTTP setup:

```bash
go test ./internal/domain/
# Expected output: ok  github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain  0.007s
```

---

## Exercise 2: Infrastructure Swap (SQL → Active Directory)

Due to a sudden policy change, user information is now retrieved from Active Directory (AD) instead of a SQL database.

### 2-1. Baseline: The SQL Repository (Stand-In)

First you need the _current_ implementation: a repository backed by a SQL database. Instead of a real database, the reference implementation uses an in-memory stand-in — from the outside it behaves exactly like a SQL repository (query, reconstitute the entity, return a domain error on miss), so the workshop runs with **zero external dependencies**.

```go
// internal/adapters/infra/sql_user_repository.go
package infra

import (
	"context"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
)

// userRow mimics one row of the users table.
type userRow struct {
	ID       string
	Name     string
	JoinedAt time.Time
}

// memoryDB is a stand-in for a real SQL database. It keeps rows in a map so
// the workshop runs with zero external dependencies.
type memoryDB struct {
	users map[string]userRow
}

// seedUsers keeps the sample dataset in one place. Joining dates are relative
// to "now" so the veteran verdicts never go stale: alice has 10 years of
// service (veteran), bob has 1 year (not yet a veteran).
func seedUsers() map[string]userRow {
	now := time.Now()
	return map[string]userRow{
		"alice": {ID: "alice", Name: "Alice", JoinedAt: now.AddDate(-10, 0, 0)},
		"bob":   {ID: "bob", Name: "Bob", JoinedAt: now.AddDate(-1, 0, 0)},
	}
}

// SQLUserRepository behaves like a repository backed by a SQL database: it
// "queries" the in-memory users table and reconstitutes domain entities.
type SQLUserRepository struct {
	db *memoryDB
}

func NewSQLUserRepository() *SQLUserRepository {
	return &SQLUserRepository{db: &memoryDB{users: seedUsers()}}
}

// Compile-time check that SQLUserRepository satisfies the domain port.
var _ domain.UserRepository = (*SQLUserRepository)(nil)

// FindByID pretends to run:
//
//	SELECT id, name, joined_at FROM users WHERE id = ?
func (r *SQLUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	row, ok := r.db.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return &domain.User{
		ID:       row.ID,
		Name:     row.Name,
		JoinedAt: row.JoinedAt,
	}, nil
}
```

### 2-2. The LDAP Client Stub

The new infrastructure talks to Active Directory through an LDAP client library. The reference implementation ships a **stub** instead of a real one: it behaves like the real thing (`Search` resolves a UID to a directory entry, or returns a driver-style error) but keeps its entries in memory. Swapping in a real library later only touches this file.

```go
// internal/adapters/infra/ldap_client.go
package infra

import (
	"context"
	"errors"
	"time"
)

// LDAPEntry mimics an entry returned by an LDAP search.
type LDAPEntry struct {
	UID          string
	DisplayName  string
	CreationDate time.Time
}

// errNoSuchObject mimics the error a real LDAP library returns when the
// search base holds no matching entry.
var errNoSuchObject = errors.New("ldap: no such object")

// LDAPClient is a stub of a real LDAP client library (such as go-ldap/ldap).
// It keeps its directory entries in memory so the workshop has zero external
// dependencies, while still behaving like the real thing: Search resolves a
// UID to an entry or returns a driver-style error.
type LDAPClient struct {
	entries map[string]LDAPEntry
}

func NewLDAPClient(entries map[string]LDAPEntry) *LDAPClient {
	return &LDAPClient{entries: entries}
}

// Search resolves a UID to a directory entry.
func (c *LDAPClient) Search(ctx context.Context, uid string) (LDAPEntry, error) {
	entry, ok := c.entries[uid]
	if !ok {
		return LDAPEntry{}, errNoSuchObject
	}
	return entry, nil
}
```

### 2-3. Adding the New Infrastructure Implementation

Create an AD repository that satisfies the same `domain.UserRepository` interface.

```go
// internal/adapters/infra/ad_user_repository.go
package infra

import (
	"context"
	"errors"
	"fmt"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
)

// ADUserRepository reads users from Active Directory via an LDAP client.
type ADUserRepository struct {
	ldapClient *LDAPClient // stubbed LDAP library
}

func NewADUserRepository(ldapClient *LDAPClient) *ADUserRepository {
	return &ADUserRepository{ldapClient: ldapClient}
}

// Compile-time check that ADUserRepository satisfies the same domain port as
// the SQL implementation.
var _ domain.UserRepository = (*ADUserRepository)(nil)

func (r *ADUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	// Issue the LDAP query to retrieve the entry
	entry, err := r.ldapClient.Search(ctx, id)
	if err != nil {
		if errors.Is(err, errNoSuchObject) {
			// Translate the driver-specific "not found" into a domain error
			// so the UseCase layer never sees LDAP details.
			return nil, fmt.Errorf("ldap search user %s: %w", id, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("ldap search user %s: %w", id, err) // Convert driver error
	}
	// Reconstruct the entity from the directory entry (bare struct literal
	// is allowed here because we restore persisted state, not create new).
	return &domain.User{
		ID:       entry.UID,
		Name:     entry.DisplayName,
		JoinedAt: entry.CreationDate,
	}, nil
}
```

> **Note on Entity Reconstruction (Allowing Bare Struct Literals):**
> In Clean Architecture, creating entities directly with bare struct literals (like `&domain.User{}`) is generally restricted in layers like UseCases to protect domain invariants (which should be enforced by constructors like `NewUser`).
> However, in Infrastructure Adapters, when restoring existing entities from a database or external service ("Reconstruction"), it is exceptionally allowed to use bare struct literals to reconstruct the state without re-triggering new entity validation.
>
> \* Note: If you encapsulate entity state by using unexported (lowercase) fields in your domain design, you cannot use bare struct literals from outer packages like Infrastructure Adapters. In such cases, apply the **Reconstructor pattern** by defining a dedicated reconstitution function inside the domain package (e.g., `domain.ReconstructUser(...)`) to restore the entities safely.

### 2-4. Switching via Dependency Injection (DI)

`main.go` is the Composition Root — the only place that knows concrete implementations. It runs the same veteran check against both data sources so you can see the swap in action:

```go
// main.go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/adapters/infra"
	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/usecase"
)

func main() {
	ctx := context.Background()

	// Phase 1: users are served by the SQL database.
	checkWith(ctx, "SQL database", infra.NewSQLUserRepository())

	// Phase 2: policy change — the user directory moved to Active Directory.
	// Only this wiring changes; internal/domain and internal/usecase are untouched.
	now := time.Now()
	directory := infra.NewLDAPClient(map[string]infra.LDAPEntry{
		"alice": {UID: "alice", DisplayName: "Alice", CreationDate: now.AddDate(-10, 0, 0)},
		"bob":   {UID: "bob", DisplayName: "Bob", CreationDate: now.AddDate(-1, 0, 0)},
	})
	checkWith(ctx, "Active Directory (LDAP)", infra.NewADUserRepository(directory))
}

// checkWith wires the given repository into the use case and prints the
// veteran verdicts. It works with ANY implementation of domain.UserRepository.
func checkWith(ctx context.Context, source string, repo domain.UserRepository) {
	fmt.Printf("== data source: %s ==\n", source)
	uc := usecase.NewCheckVeteranUseCase(repo, domain.VeteranService{})
	for _, id := range []string{"alice", "bob"} {
		veteran, err := uc.Execute(ctx, id)
		if err != nil {
			fmt.Printf("  %s: error: %v\n", id, err)
			continue
		}
		fmt.Printf("  %s: veteran=%v\n", id, veteran)
	}
}
```

Since the UseCase takes an interface as its argument, it accepts `adRepo` as-is — simply swap the concrete class being injected at the entry point. Run it:

```bash
go run .
# Expected output:
# == data source: SQL database ==
#   alice: veteran=true
#   bob: veteran=false
# == data source: Active Directory (LDAP) ==
#   alice: veteran=true
#   bob: veteran=false
```

The verdicts are identical for both data sources — the business logic did not notice the swap.

### ✅ Checkpoint

- [ ] `ADUserRepository` satisfies `domain.UserRepository` — this compiles: `var _ domain.UserRepository = (*infra.ADUserRepository)(nil)`
- [ ] Only `main.go` (the Composition Root) was modified to swap the data source
- [ ] `go run .` prints identical veteran verdicts for the SQL and LDAP data sources
- [ ] `go build ./...` still passes with the AD repository wired in, and `internal/domain` / `internal/usecase` are untouched (`git diff` shows no changes there)

---

## Key Points of This Workshop

1. **Location of Knowledge:**
   - Calculation method for tenure = **Entity**
   - Definition of a veteran (5 years) = **Domain Service**
   - These can be tested independently of "Database" or "Web".
2. **Localization of Change:**
   - Even when the data source changed from DB to AD, we only had to add new code to the `infra` layer and change the injection target in `main`.
   - **The core business logic (Domain/UseCase) remains untouched with zero lines of modification.** This is the true value of Clean Architecture.
3. **Ports and Boundaries:**
   - `domain.UserRepository` is an **output port**; implementations live in Infra Adapters Layer.
   - Details like the `LDAPClient` stub or DB drivers stay in the Infra Adapters Layer and never leak into Domain/UseCase.

---

## Comprehension Check

Try answering these questions yourself. There isn't always one correct answer, but the goal is to be able to explain your reasoning.

### Question 1: Interface for Domain Service

`VeteranService` is used as a concrete type rather than an interface. Why didn't we create an interface for it? Conversely, in what situations **must** we create an interface?

### Question 2: Impact Scope of Changes

If "the definition of a veteran changed from 5 years to 3 years," which layer would need to be modified? Conversely, which layer would require **zero** changes?

### Question 3: Risks of Bare Struct Literals

`ADUserRepository.FindByID` uses a bare struct literal like `&domain.User{...}`. To protect domain invariants (e.g., ID must not be empty), what kind of design could be added?

---

## What you learned

- Business knowledge ("who is a veteran") lives in the Domain layer: the calculation as an **Entity** method, the threshold as a **Domain Service** — both testable with no DB, HTTP, or mocks.
- The UseCase depends only on the `domain.UserRepository` **output port**; swapping SQL for Active Directory required **zero changes** to Domain and UseCase — only `main.go` changed.
- Infra Adapters may use bare struct literals only for **reconstructing** persisted entities; new entities should go through constructors to protect invariants.
- Define an interface when a collaborator has (or may gain) external dependencies — not for every pure domain service.
- External systems (SQL databases, LDAP directories) can be replaced with in-memory stand-ins and stubs, keeping workshops runnable with zero external dependencies.

**Next**: [WS2: Extension and Optimization](./clean_arch_ws2_en.md) — abstract side effects (notification) and add a transparent cache with the Decorator pattern.

---

## References

- [The Clean Architecture — Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Effective Go: Interfaces](https://go.dev/doc/effective_go#interfaces_and_types)
- [Go Documentation](https://go.dev/doc/)
