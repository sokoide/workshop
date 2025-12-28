# Clean Architecture Workshop (WS2): Extension and Optimization

Following WS1, we will learn about the benefits of layer separation in side-effect control and performance optimization through further practical scenarios.

## Workshop Scenario

1. **Notification Channel Diversification:** Change the notification method from Email to Slack without altering business logic.
2. **Transparent Cache Implementation:** Introduce Redis caching just by swapping the infrastructure layer, without tainting the business logic at all.

---

## Exercise 1: Abstracting the Notification System

Consider a use case: "Send a notification when membership is approved."

### 1-1. Interface Definition in the Domain Layer

Define the "notification functionality" rather than "how to send it."

```go
// domain/notification.go
package domain

type NotificationService interface {
 Send(ctx context.Context, message string) error
}
```

### 1-2. Usage in UseCase

The UseCase does not care whether the notification is Email or Slack.

```go
// usecase/approve_membership.go
package usecase

import "context"

func (uc *ApproveUseCase) Execute(ctx context.Context, userID string) error {
 // ... Approval logic ...
 return uc.notifier.Send(ctx, "Membership has been approved!")
}
```

### 1-3. Implementation Swap in the Infra Layer

Initially, use an Email implementation, but later create a Slack implementation and swap it via Dependency Injection (DI).

```go
// infra/slack_notifier.go (Added later)
type SlackNotifier struct {
 webhookURL string
}

func (n *SlackNotifier) Send(ctx context.Context, msg string) error {
 // Implementation to call Slack API
 return nil
}
```

---

## Exercise 2: Adding Transparent Cache (Decorator Pattern)

Address the requirement: "Speed up user information retrieval. However, do not change a single line of existing code (such as UseCase) that calls `FindByID`."

### 2-1. Implementing a Caching Repository

Create a new Infra implementation that wraps the original `SqlUserRepository`.

```go
// infra/cached_user_repository.go
package infra

type CachedUserRepository struct {
 origin domain.UserRepository // Real DB repository
 cache  *redis.Client         // Cache DB
}

func (r *CachedUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
 // 1. Return if in cache
 if user, err := r.getFromCache(id); err == nil {
  return user, nil
 }

 // 2. Otherwise, ask the real DB
 user, err := r.origin.FindByID(ctx, id)
 if err == nil {
  r.saveToCache(user) // Save for next time
 }
 return user, err
}
```

### 2-2. Configuration Change in DI Container

In `main.go`, "wrap" the real repository with the caching repository before passing it to the UseCase.

```go
func main() {
 realRepo := infra.NewSqlUserRepository(db)
 // Wrap the real one with caching functionality
 cachedRepo := infra.NewCachedUserRepository(realRepo, redisClient)

 // UseCase looks at the interface, so it can accept the wrapped cachedRepo as-is
 useCase := usecase.NewCheckVeteranUseCase(cachedRepo)
}
```

---

## Key Points of This Workshop

1. **Abstraction of Side Effects (Exercise 1):**
    - The act of "sending a notification" itself is a business rule, while "sending via Slack" is merely a means (infrastructure detail).
    - By pushing details outward, it becomes easy to verify that "a notification was called" during testing without actually sending a notification.
2. **Transparent Feature Addition (Exercise 2):**
    - As long as the interface remains the same, the caller won't notice if the internal logic changes from "retrieving from DB" to "retrieving with cache control."
    - This allows for infrastructure-level improvements (performance tuning, logging, retry logic, etc.) while keeping the business logic healthy.
