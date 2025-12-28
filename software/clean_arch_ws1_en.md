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
// domain/user.go
package domain

import "time"

type User struct {
 ID       string
 Name     string
 JoinedAt time.Time // Joining date
}

// GetTenureYears returns the number of years of service
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
// domain/veteran_service.go
package domain

type VeteranService struct{}

// IsVeteran determines if a user is a veteran (5 or more years of service)
func (s *VeteranService) IsVeteran(user *User) bool {
 return user.GetTenureYears() >= 5
}
```

### 1-3. UseCase Implementation

Combine Domain objects to realize the use case.

```go
// usecase/check_veteran.go
package usecase

import (
 "context"
 "your-project/domain"
)

type CheckVeteranUseCase struct {
 repo       domain.UserRepository
 veteranSvc domain.VeteranService
}

func (uc *CheckVeteranUseCase) Execute(ctx context.Context, id string) (bool, error) {
 user, err := uc.repo.FindByID(ctx, id) // Retrieved via repository
 if err != nil {
  return false, err
 }
 return uc.veteranSvc.IsVeteran(user), nil
}
```

---

## Exercise 2: Infrastructure Swap (SQL → Active Directory)

Due to a sudden policy change, user information is now retrieved from Active Directory (AD) instead of a SQL database.

### 2-1. Adding a New Infrastructure Implementation

Create an AD repository that satisfies the `domain.UserRepository` interface.

```go
// infra/ad_user_repository.go
package infra

type AdUserRepository struct {
 ldapClient *LdapClient // Pseudo library
}

func (r *AdUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
 // Issue LDAP query to retrieve info
 entry, _ := r.ldapClient.Search(id)
 return &domain.User{
  ID:       entry.UID,
  Name:     entry.DisplayName,
  JoinedAt: entry.CreationDate,
 }, nil
}
```

### 2-2. Switching via Dependency Injection (DI)

Simply swap the concrete class being injected in the main process (entry point).

```go
func main() {
 // Old: sqlRepo := infra.NewSqlUserRepository(db)
 // New:
 adRepo := infra.NewAdUserRepository(ldapClient)

 // Since UseCase takes an interface as an argument, it can accept adRepo as-is
 useCase := usecase.NewCheckVeteranUseCase(adRepo)

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
