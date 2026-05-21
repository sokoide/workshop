# Clean Architecture Workshop (WS1): Learning Resilient Design

In this workshop, we will extend the existing example by adding business rules and replacing the infrastructure, experiencing the benefits of Clean Architecture firsthand.

## Workshop Scenario

1. **Add Business Rules:** Implement the rule "Employees with 5 or more years of service are recognized as veterans" using Entities and Domain Services.
2. **Infrastructure Swap:** Change the data storage from a SQL database to Active Directory (LDAP). Confirm that no changes are required in the UseCase or Domain layers.

---

## Exercise 1: Implementing Entity and Domain Service

Encapsulate the business knowledge of "Veteran Determination" within the Domain layer.

### 1-1. Creating the Entity

Give the User attributes and the knowledge to calculate years of service.

```go
// internal/domain/user.go
package domain

import "time"

type User struct {
	ID       string
	Name     string
	JoinedAt time.Time // Joining date
}

// GetTenureYears returns the number of years of service.
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

type VeteranService struct{}

// IsVeteran determines if a user is a veteran (5 or more years of service).
func (s VeteranService) IsVeteran(user *User) bool {
	return user.GetTenureYears() >= 5
}
```

### 1-3. UseCase Implementation

Combine Domain objects to realize the use case.

```go
// internal/usecase/check_veteran.go
package usecase

import (
	"context"

	"your-project/internal/domain"
)

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

---

## Exercise 2: Infrastructure Swap (SQL → Active Directory)

Due to a sudden policy change, user information is now retrieved from Active Directory (AD) instead of a SQL database.

### 2-1. Adding a New Infrastructure Implementation

Create an AD repository that satisfies the `domain.UserRepository` interface.

```go
// internal/adapters/infra/ad_user_repository.go
package infra

import (
	"context"
	"fmt"

	"your-project/internal/domain"
)

type ADUserRepository struct {
	ldapClient *LDAPClient // Pseudo library
}

func (r *ADUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	// Issue LDAP query to retrieve info
	entry, err := r.ldapClient.Search(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ldap search user %s: %w", id, err) // Convert driver error
	}
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

### 2-2. Switching via Dependency Injection (DI)

Simply swap the concrete class being injected in the main process (entry point).

```go
func main() {
	// Old: sqlRepo := infra.NewSQLUserRepository(db)
	// New:
	adRepo := infra.NewADUserRepository(ldapClient)

	// Since UseCase takes an interface as an argument, it can accept adRepo as-is
	useCase := usecase.NewCheckVeteranUseCase(adRepo, domain.VeteranService{})

	// After this, the code calling useCase.Execute() requires ZERO changes!
}
```

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
    - Details like `LDAPClient` or DB drivers stay in the Infra Adapters Layer and never leak into Domain/UseCase.
