# Clean Architecture (4-Layer Structure)

A proposal for building loosely coupled software centered on business logic, independent of external details such as databases or communication protocols.

## Layer Structure and Dependencies

Dependencies always point **inwards (towards the Domain)**. External inputs (Framework) call the UseCase, and Infra Adapters depend on the Domain through interfaces.

```mermaid
graph TD
    subgraph FrameworkLayer [Framework]
        Web[Web / gRPC / CLI]
        Controller[Controller / Handler]
        Presenter[Presenter]
    end

    subgraph UseCaseLayer [UseCase]
        UC[UseCase]
    end

    subgraph DomainLayer [Domain]
        DS[Domain Service]
        E[Entity]
        RI[Repository Interface]
    end

    subgraph InfraLayer [Infra Adapters]
        RI_Impl[Repository Impl]
        DB[(Database)]
    end

    %% Dependencies
    Web --> Controller
    Controller --> UC
    UC --> DomainLayer
    RI_Impl -- "implements" --> RI
    RI_Impl --> DB
    UC --> Presenter
```

---

## 1. Domain Layer

The heart of the application, representing the business rules themselves.

* **Entity:** Business "objects" or "concepts".
* **Domain Service:** Knowledge or logic that spans multiple entities.
* **Repository Interface:** An "abstract contract (Port)" regarding data persistence. Implementation is not included here.

## 2. UseCase Layer

Describes the steps to realize specific "features" of the application.

* **Role:** Manipulates objects from the Domain layer and defines the flow of processing (orchestration).
* **Dependencies:** Depends only on the Domain layer. It is unaware of what the external database actually is.

## 3. Infra Adapters Layer

Specifically implements the interfaces (Ports) defined in the Domain layer and bridges external systems.

* **Repository Impl:** Implements the interface defined in the Domain layer. Mapping and query construction live here.
* **Gateway Impl:** Implementation of external API clients, etc.

## 4. Framework Layer

The outermost I/O layer, such as Web frameworks or CLI.

* **Controller / Handler:** Converts external requests (HTTP, CLI) to UseCase inputs and calls the UseCase.
* **Presenter:** Formats the UseCase output for external consumption (e.g., JSON).

---

## Implementation Example (Go)

A simple example determining if a user belongs to a specific group shows the implementation image of each layer.

### 1. Domain Layer (Implementation)

Defines business rules (interfaces).

```go
// domain/membership.go
package domain

import "context"

// MembershipRepository defines abstract queries against a data source.
type MembershipRepository interface {
	IsMember(ctx context.Context, userID, groupID string) (bool, error)
}
```

### 2. UseCase Layer (Implementation)

Defines business "procedures". Uses Domain interfaces.

```go
// usecase/membership.go
package usecase

import (
	"context"

	"your-project/domain"
)

// MembershipUseCase is the concrete executor of the use case.
type MembershipUseCase struct {
	repo domain.MembershipRepository
}

func NewMembershipUseCase(r domain.MembershipRepository) *MembershipUseCase {
	return &MembershipUseCase{repo: r}
}

// Execute performs the "membership check" use case.
func (uc *MembershipUseCase) Execute(ctx context.Context, userID, groupID string) (bool, error) {
	// Domain-specific validation can be performed here if necessary.
	return uc.repo.IsMember(ctx, userID, groupID)
}
```

### 3. Infra Adapters Layer (Implementation)

Specifically implements the interfaces (Ports) defined in the Domain layer. Details like DB drivers are isolated here to keep upper layers independent of specific technologies.

```go
// infra/membership_repository.go
package infra

import (
	"context"
	"database/sql"
)

// SQLMembershipRepository is a repository implementation using a SQL database.
type SQLMembershipRepository struct {
	db *sql.DB
}

func NewSQLMembershipRepository(db *sql.DB) *SQLMembershipRepository {
	return &SQLMembershipRepository{db: db}
}

// IsMember issues actual SQL against the database.
func (r *SQLMembershipRepository) IsMember(ctx context.Context, userID, groupID string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM memberships WHERE user_id = ? AND group_id = ?)"
	err := r.db.QueryRowContext(ctx, query, userID, groupID).Scan(&exists)
	return exists, err
}
```

---

## Calling from External (REST / gRPC)

Web frameworks or gRPC servers reside at the outermost edge and are only responsible for calling the `UseCase`. Handlers depend on **UseCase interfaces (input ports)** to keep implementations swappable.

```go
// usecase/membership_port.go
package usecase

import "context"

// MembershipChecker is the input port (public UseCase API).
type MembershipChecker interface {
	Execute(ctx context.Context, userID, groupID string) (bool, error)
}
```

```go
// Example usage in a Web handler
func HandleCheckMembership(w http.ResponseWriter, r *http.Request) {
	// 1. Create real DB instance (usually done at startup)
	dbRepo := infra.NewSQLMembershipRepository(sqlDB)

	// 2. Inject repository into the UseCase (Dependency Injection)
	useCase := usecase.NewMembershipUseCase(dbRepo)

	// 3. Execute the UseCase
	isMember, err := useCase.Execute(r.Context(), "user123", "groupA")

	// 4. Return result as response
	json.NewEncoder(w).Encode(map[string]bool{"is_member": isMember})
}
```

### Role of context.Context

In the Go implementation example, `ctx context.Context` is passed through each layer for the following primary purposes:

1. **Cancellation Propagation:** If a user closes their browser, the signal is propagated down to the DB query, immediately stopping the execution and saving resources.

2. **Timeout Management:** It allows enforcing deadlines (e.g., "the whole request must finish within 5 seconds") across all operations, including database calls.

3. **Tracing:** It carries request-scoped metadata like Request IDs, enabling you to trace a single request's journey through multiple layers and services in logs.

#### 💡 ctx vs. Arguments

* **Use Arguments for:** **Essential business data** such as `userID` or `groupID`. Passing these explicitly as arguments ensures type safety and makes the function's dependencies clear.

* **Use ctx for:** **Cross-cutting (supplementary) information** such as `Request ID` or `Auth Tokens`. These are not core to the business logic but are necessary for logging, authorization at the infra layer, or distributed tracing.

## Ports and Repository Boundary

* **Input Port:** The UseCase interface called by external adapters (Web/CLI/Batch). Controllers depend on this port.
* **Output Port:** Contracts the Domain/UseCase require from the outside (e.g., repositories). Interfaces live inside, implementations live in adapters.
* **Repository Boundary:** Repositories define persistence contracts. Transactions and retries sit in UseCase; mapping and query construction belong in adapters.

### Go Implementation Notes

* **SQL placeholders:** Drivers differ (`?`, `$1`, etc.). Choose what matches your driver.
* **Initialisms:** In Go, initialisms like `SQL` are typically all-caps.
* **Context values:** Use typed keys and avoid storing business data in `context`.
