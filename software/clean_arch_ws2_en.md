# Clean Architecture Workshop (WS2): Extension and Optimization

> **⏱️ Estimated Time**: Approx. 60 minutes
> **🎯 Level**: Intermediate
> **💻 Environment**: Go 1.23+, any OS (zero external dependencies — notifiers and cache use stand-ins)
>
> > **💡 Glossary**: For technical terms such as [Interface](../infra/glossary_en.md#go), [Write-Through / Cache-Aside](../infra/glossary_en.md#cache), and [Clean Architecture](../infra/glossary_en.md#architecture), please refer to the [Glossary](../infra/glossary_en.md).

Following WS1, we will learn about the benefits of layer separation in side-effect control and performance optimization through further practical scenarios.

## Workshop Scenario

1. **Notification Channel Diversification:** Change the notification method from Email to Slack without altering business logic.
2. **Transparent Cache Implementation:** Introduce a cache (an in-memory stand-in here; Redis in production) just by swapping the infrastructure layer, without tainting the business logic at all.

## Prerequisites

- [ ] Completed [WS1](./clean_arch_ws1_en.md) (Entity / Domain Service / repository swap) — this workshop builds directly on its finished code
- [ ] Go 1.23+ installed — verify with `go version`

### Setting Up Your Workspace

WS2 starts from the WS1 finished code (veteran check + SQL/LDAP repositories). Pick one of two ways to follow along:

```bash
# Option A: copy the WS1 reference implementation as your starting point
cp -r software/assets/clean_arch_ws1 ~/ws2 && cd ~/ws2

# Option B: copy the completed WS2 reference implementation and explore it
cp -r software/assets/clean_arch_ws2 ~/ws2 && cd ~/ws2
```

> The code snippets below use the module path `github.com/sokoide/workshop/software/assets/clean_arch_ws2`. If you started from the WS1 copy, replace it with `github.com/sokoide/workshop/software/assets/clean_arch_ws1` (or your own module path from `go mod init`).
>
> No Redis or any other external service is required: the notifiers print their messages and the cache keeps entries in memory. See the note at the end of Exercise 2 for wiring a real Redis.

---

## Exercise 1: Abstracting the Notification System

Consider a use case: "Send a notification when membership is approved."

### 1-1. Interface Definition in the UseCases Layer

Define the "notification functionality" rather than "how to send it." Notification is a tool needed by the application workflow, so the port belongs to the UseCases layer.

```go
// internal/usecase/port/notification.go
// Package port holds the output ports owned by the UseCase layer.
package port

import "context"

// NotificationGateway is the port for sending notifications. The UseCase
// layer owns it because notification is a tool for the application workflow,
// not part of the core domain language. "How" to send (Email, Slack, ...) is
// an infrastructure detail implemented in the Infra Adapters layer.
type NotificationGateway interface {
	Send(ctx context.Context, message string) error
}
```

### 1-2. Usage in UseCase

The UseCase does not care whether the notification is Email or Slack.

```go
// internal/usecase/approve_membership.go
package usecase

import (
	"context"
	"fmt"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/domain"
	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/usecase/port"
)

// ApproveUseCase approves a membership and notifies the user.
type ApproveUseCase struct {
	repo     domain.UserRepository
	notifier port.NotificationGateway
}

func NewApproveUseCase(repo domain.UserRepository, notifier port.NotificationGateway) *ApproveUseCase {
	return &ApproveUseCase{repo: repo, notifier: notifier}
}

func (uc *ApproveUseCase) Execute(ctx context.Context, userID string) error {
	user, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user %s: %w", userID, err)
	}
	// The UseCase does not care whether this goes out as Email or Slack.
	message := fmt.Sprintf("Membership approved: welcome, %s!", user.Name)
	return uc.notifier.Send(ctx, message)
}
```

### 1-3. Implementation Swap in the Infra Layer

Both implementations are stand-ins — they print the message instead of calling SMTP or a webhook, so the workshop runs with zero external dependencies. The wiring in `main` is where the swap happens.

```go
// internal/adapters/infra/email_notifier.go
package infra

import (
	"context"
	"fmt"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/usecase/port"
)

// EmailNotifier is a stand-in for an SMTP client: it prints the message
// instead of sending mail, so the workshop runs with zero external
// dependencies.
type EmailNotifier struct {
	from string
}

func NewEmailNotifier(from string) *EmailNotifier {
	return &EmailNotifier{from: from}
}

// Compile-time check that EmailNotifier satisfies the UseCase-owned port.
var _ port.NotificationGateway = (*EmailNotifier)(nil)

func (n *EmailNotifier) Send(ctx context.Context, message string) error {
	fmt.Printf("[email from %s] %s\n", n.from, message)
	return nil
}
```

```go
// internal/adapters/infra/slack_notifier.go (Added later)
package infra

import (
	"context"
	"fmt"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/usecase/port"
)

// SlackNotifier is a stand-in for the Slack API: it prints the message
// instead of calling a webhook, so the workshop runs with zero external
// dependencies.
type SlackNotifier struct {
	webhookURL string
}

func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{webhookURL: webhookURL}
}

// Compile-time check that SlackNotifier satisfies the same port.
var _ port.NotificationGateway = (*SlackNotifier)(nil)

func (n *SlackNotifier) Send(ctx context.Context, message string) error {
	fmt.Printf("[slack webhook %s] %s\n", n.webhookURL, message)
	return nil
}
```

`main.go` wires the port to a concrete implementation — swapping Email for Slack is a one-line change:

```go
// main.go (excerpt)
func approveVia(ctx context.Context, notifier port.NotificationGateway) {
	uc := usecase.NewApproveUseCase(infra.NewSQLUserRepository(), notifier)
	if err := uc.Execute(ctx, "alice"); err != nil {
		fmt.Printf("approve: error: %v\n", err)
	}
}

func main() {
	ctx := context.Background()
	fmt.Println("== Exercise 1: notification channel swap ==")
	approveVia(ctx, infra.NewEmailNotifier("noreply@example.com"))
	// One-line swap: Email → Slack. Nothing else changes.
	approveVia(ctx, infra.NewSlackNotifier("https://hooks.slack.com/services/T000/B000/XXXX"))
}
```

```bash
go run .
# Expected output:
# == Exercise 1: notification channel swap ==
# [email from noreply@example.com] Membership approved: welcome, Alice!
# [slack webhook https://hooks.slack.com/services/T000/B000/XXXX] Membership approved: welcome, Alice!
```

### ✅ Checkpoint

- [ ] `NotificationGateway` is defined in the `usecase` (port) package — not in `domain` or `infra`
- [ ] `ApproveUseCase` compiles with **zero imports from the `infra` package**
- [ ] You can swap Email for Slack by changing **one line in `main.go`** and nothing else
- [ ] You can write a test that asserts "Send was called" using a mock/fake `NotificationGateway`, without hitting Slack

---

## Exercise 2: Adding Transparent Cache (Decorator Pattern)

Address the requirement: "Speed up user information retrieval. However, do not change a single line of existing code (such as UseCase) that calls `FindByID`."

### 2-1. The Cache Store and the Caching Repository

First define the cache database the decorator will use. It is an infra-internal abstraction with an in-memory, TTL-aware implementation. In production you would implement the same two-method interface with Redis (see the note below).

```go
// internal/adapters/infra/cache_store.go
package infra

import (
	"context"
	"errors"
	"sync"
	"time"
)

// errCacheMiss is returned by CacheStore.Get when the key is absent or
// expired. It never leaves the Infra layer: the decorator treats every cache
// error as a miss and falls back to the real repository.
var errCacheMiss = errors.New("cache miss")

// CacheStore is the key-value store used by CachedUserRepository. It is an
// infra-internal abstraction: in production you can implement it with Redis
// (e.g. github.com/redis/go-redis/v9) without touching any other layer.
type CacheStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// MemoryCacheStore is an in-memory CacheStore with TTL support. It plays the
// role of the cache database in this workshop.
type MemoryCacheStore struct {
	mu      sync.Mutex
	entries map[string]memoryCacheEntry
	now     func() time.Time // injectable clock for tests
}

type memoryCacheEntry struct {
	value     []byte
	expiresAt time.Time
}

func NewMemoryCacheStore() *MemoryCacheStore {
	return &MemoryCacheStore{
		entries: map[string]memoryCacheEntry{},
		now:     time.Now,
	}
}

func (s *MemoryCacheStore) Get(ctx context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[key]
	if !ok || !s.now().Before(entry.expiresAt) {
		// Absent or expired — indistinguishable from the caller's viewpoint.
		// Drop the stale entry so the map does not grow without bound
		// (a real cache like Redis evicts expired keys itself).
		delete(s.entries, key)
		return nil, errCacheMiss
	}
	return entry.value, nil
}

func (s *MemoryCacheStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = memoryCacheEntry{value: value, expiresAt: s.now().Add(ttl)}
	return nil
}
```

Then create a new Infra implementation that wraps the original `SQLUserRepository`.

```go
// internal/adapters/infra/cached_user_repository.go
package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/domain"
)

// userCacheTTL is how long a cached user entry stays fresh.
const userCacheTTL = 10 * time.Minute

// CachedUserRepository decorates a real domain.UserRepository with a cache
// (Decorator pattern). It satisfies the same port, so neither the UseCase
// layer nor the wrapped repository knows that the cache exists.
type CachedUserRepository struct {
	origin domain.UserRepository // Real repository (DB, LDAP, ...)
	cache  CacheStore            // Cache database (in-memory here, Redis in production)
}

func NewCachedUserRepository(origin domain.UserRepository, cache CacheStore) *CachedUserRepository {
	return &CachedUserRepository{origin: origin, cache: cache}
}

// Compile-time check that the decorator satisfies the same port as the origin.
var _ domain.UserRepository = (*CachedUserRepository)(nil)

func (r *CachedUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	// 1. Return if in cache
	if user, err := r.getFromCache(ctx, id); err == nil {
		return user, nil
	}

	// 2. Otherwise, ask the real repository
	user, err := r.origin.FindByID(ctx, id)
	if err == nil {
		if cacheErr := r.saveToCache(ctx, user); cacheErr != nil {
			slog.Warn("failed to save user to cache", "error", cacheErr) // Log cache failures and proceed
		}
	}
	return user, err
}

// Cache helpers — implementation details of this Decorator.
// They are NOT part of the UserRepository interface, so neither the
// UseCase layer nor the wrapped repository knows about the cache store.

func (r *CachedUserRepository) getFromCache(ctx context.Context, id string) (*domain.User, error) {
	data, err := r.cache.Get(ctx, "user:"+id)
	if err != nil {
		return nil, err // Cache miss or cache failure → fall back to the real repository
	}
	var user domain.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("unmarshal cached user %s: %w", id, err)
	}
	return &user, nil
}

func (r *CachedUserRepository) saveToCache(ctx context.Context, user *domain.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	// Set a TTL so stale cache entries expire automatically
	return r.cache.Set(ctx, "user:"+user.ID, data, userCacheTTL)
}
```

> **💡 Educational Note on Logging (Trade-offs for Simplicity):**
> For simplicity, this workshop code directly calls the standard library `log/slog` to record cache warnings. In production-grade systems, it is highly recommended to define a logging interface and inject it via Dependency Injection (DI) to improve testability and decouple code from specific logging libraries, or handle logging as a cross-cutting concern using middleware or decorators.

### 2-2. Configuration Change in DI Container

In `main.go`, "wrap" the real repository with the caching repository before passing it to the UseCase. To make the effect visible, the SQL stand-in from WS1 gained a `QueryCount()` method that counts the queries it has issued:

```go
// main.go — Exercise 2 part (added after the Exercise 1 wiring)
fmt.Println("\n== Exercise 2: transparent cache ==")
realRepo := infra.NewSQLUserRepository()
cachedRepo := infra.NewCachedUserRepository(realRepo, infra.NewMemoryCacheStore())

// The UseCase sees only domain.UserRepository — it cannot tell whether
// the repository it received is cached or not.
uc := usecase.NewCheckVeteranUseCase(cachedRepo, domain.VeteranService{})

for lookup := 1; lookup <= 3; lookup++ {
	veteran, err := uc.Execute(ctx, "alice")
	if err != nil {
		fmt.Printf("lookup #%d: error: %v\n", lookup, err)
		continue
	}
	fmt.Printf("lookup #%d: alice veteran=%v (SQL queries so far: %d)\n",
		lookup, veteran, realRepo.QueryCount())
}
```

```bash
go run .
# Expected output:
# == Exercise 1: notification channel swap ==
# [email from noreply@example.com] Membership approved: welcome, Alice!
# [slack webhook https://hooks.slack.com/services/T000/B000/XXXX] Membership approved: welcome, Alice!
#
# == Exercise 2: transparent cache ==
# lookup #1: alice veteran=true (SQL queries so far: 1)
# lookup #2: alice veteran=true (SQL queries so far: 1)
# lookup #3: alice veteran=true (SQL queries so far: 1)
```

Lookups #2 and #3 are served from the cache: the SQL query count stays at 1, and the UseCase is completely unaware of it.

### ✅ Checkpoint

- [ ] `CachedUserRepository` satisfies `domain.UserRepository` — this compiles: `var _ domain.UserRepository = (*infra.CachedUserRepository)(nil)`
- [ ] Cache logic exists **only** inside `internal/adapters/infra/` — `git grep -i cache internal/domain internal/usecase` finds nothing
- [ ] When a cache entry expires, reads still succeed (the decorator falls back to the real repository); cache-save failures only produce a warning log
- [ ] `go test ./...` passes with the decorator wired in

> **💡 Taking the cache to production:** `MemoryCacheStore` lives only in this workshop. In production, implement `CacheStore` with a Redis client (e.g. `github.com/redis/go-redis/v9` — see the [Redis workshop](../infra/cache_en.md) for setup) and pass that implementation to `NewCachedUserRepository`. Because the interface is infra-internal, this swap changes **one file** and nothing else — the same resilience you experienced in WS1, one layer deeper.

---

## Key Points of This Workshop

1. **Abstraction of Side Effects (Exercise 1):**
   - The act of "sending a notification" itself is an application requirement, while "sending via Slack" is merely a means (infrastructure detail).
   - The `NotificationGateway` interface is a UseCase Port (not a Domain Port) because notification is a tool for the application workflow, not part of the core domain language.
   - By pushing details outward, it becomes easy to verify that "a notification was called" during testing without actually sending a notification.
2. **Transparent Feature Addition (Exercise 2):**
   - As long as the interface remains the same, the caller won't notice if the internal logic changes from "retrieving from DB" to "retrieving with cache control."
   - This allows for infrastructure-level improvements (performance tuning, logging, retry logic, etc.) while keeping the business logic healthy.
3. **Ports and Implementation Separation:**
   - The notification interface is a UseCase-owned output port; Slack/Email implementations live in Infra Adapters Layer.
   - Details like the cache store stay in the Infra Adapters Layer.

---

## Comprehension Check

Try answering these questions yourself.

### Question 1: Limits of the Decorator Pattern

`CachedUserRepository` uses a simple strategy: "return from cache if available, otherwise query the DB." If you changed to a "Write-Behind" strategy where "when cache TTL expires, asynchronously update the DB," how would the `CachedUserRepository` implementation change? Would this change affect the UseCase layer?

### Question 2: Notification Failures

If `NotificationGateway.Send` returns an error, what should the UseCase do? Should "notification failure = use case failure," or should "notifications be best-effort, and the use case should continue"? What criteria would you use to decide?

### Question 3: Cache and Consistency

When using `CachedUserRepository`, even if the DB data is updated, the cache may remain stale. To address this problem, which layer should implement the countermeasure according to Clean Architecture, and what kind of measures would be appropriate?

---

## What you learned

- Application-required side effects (notifications) become **UseCase-owned output ports**; the means (Slack/Email) stay in the Infra Adapters layer, so implementations swap with a one-line change in `main.go`.
- The Decorator pattern adds cross-cutting behavior (caching, logging, retries) **transparently** — as long as the interface is unchanged, Domain and UseCase never notice.
- Cache failures are deliberately non-fatal: logging and falling back to the source of truth keeps the business logic healthy.
- Ports belong to the layer that needs the capability: domain policies → Domain ports, application orchestration (notification) → UseCase ports. User persistence stayed a Domain port (`domain.UserRepository`) because the repository contract is part of this domain's language — see the [Port Ownership Guidance](./clean_arch_en.md#port-ownership-guidance) for the criterion.
- External services (Slack, Redis) can be replaced with stand-ins behind infra-internal interfaces, keeping the workshop runnable with zero external dependencies while remaining one file away from the real thing.

**Next**: [WS3: Swapping Communication Protocols](./clean_arch_ws3_en.md) — migrate the BBS REST API to gRPC without touching Domain, UseCase, or Infra.

---

## References

- [The Clean Architecture — Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Redis Documentation](https://redis.io/docs/latest/)
- [go-redis Documentation](https://redis.uptrace.dev/)
- [log/slog package — Go Documentation](https://pkg.go.dev/log/slog)
