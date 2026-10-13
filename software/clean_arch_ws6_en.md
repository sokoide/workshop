# Clean Architecture Workshop (WS6): Integrating External Services

> **⏱️ Estimated Time**: Approx. 45 minutes
> **🎯 Level**: Intermediate
> **💻 Environment**: Go 1.26.2+; a Slack Incoming Webhook URL is optional (a NoOp gateway works without one)
>
> > **💡 Glossary**: For technical terms such as [Clean Architecture](../infra/glossary_en.md#architecture), [Interface](../infra/glossary_en.md#go), and [Context](../infra/glossary_en.md#go), please refer to the [Glossary](../infra/glossary_en.md).

In this workshop, you will add "Slack notification on new posts" to the BBS (2channel-style bulletin board).
You will experience the **new Port/Adapter pattern** — defining a new Port and adding an Infra Adapter — with minimal impact on existing code.

## Goal

By the end of this workshop, you can:

- Add a new external integration (Slack notification on new posts) by defining a new Port and implementing its Infra Adapter
- Keep the system runnable and testable without Slack, using a NoOp gateway wired at the Composition Root

## Prerequisites

This workshop uses the [BBS project](./assets/bbs/) as the subject code.
The shared architecture foundations (3-layer variant, dependency rule, error boundaries) are described once in [clean_arch.md](./clean_arch_en.md#layer-structure-and-dependencies).

Checklist before starting:

- [ ] Completed [WS5](./clean_arch_ws5_en.md) — this workshop builds on the WS5 state of the BBS project
- [ ] Go 1.26.2+ installed — verify with `go version`
- [ ] After preparing the start copy below, BBS builds: `cd /tmp/bbs-ws6 && go build ./...` exits with no output
- [ ] (Optional) A Slack Incoming Webhook URL for the final verification — without one, use the NoOp gateway described below

## Start and Finish States

Start from a fresh HTTP copy; do not carry WS3 gRPC changes forward. WS6 starts at the completed WS5 PostgreSQL state and finishes with post-commit notifications.

```bash
# From the repository root
python3 tools/prepare_bbs.py ws6 /tmp/bbs-ws6
cd /tmp/bbs-ws6
go test ./...
# Expected output: existing tests pass
```

Start a dedicated DB instead of carrying over WS5 resources. Verify the baseline before adding notifications.

```bash
podman run -d --name bbs-ws6-postgres -p 127.0.0.1:55434:5432 -e POSTGRES_USER=bbs -e POSTGRES_PASSWORD=bbs -e POSTGRES_DB=bbs postgres:16
for attempt in $(seq 1 30); do
  podman exec bbs-ws6-postgres pg_isready -U bbs -d bbs && break
  sleep 1
done
podman exec bbs-ws6-postgres pg_isready -U bbs -d bbs
# Expected output: accepting connections; do not continue unless ready
export DATABASE_URL='postgres://bbs:bbs@127.0.0.1:55434/bbs?sslmode=disable'
```

Go blocks below are labeled by type. Do not paste ellipses from excerpts; preserve the omitted parts of existing files. Work only inside `/tmp/bbs-ws6` and build/test each stage.

## Workshop Scenario

Add a feature to send a Slack notification whenever a new post is created in a thread.
This is a classic example of the new Port/Adapter pattern.

---

## Exercise: Adding Slack Notifications

### Change Overview

| Layer            | Change                                                                                                                                               | Role                                                                          |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------- |
| **Domain**       | **No changes**                                                                                                                                       | —                                                                             |
| **UseCase**      | Define `NotificationGateway` interface in `internal/usecase/notification.go` (new file), inject into `CreatePostUseCase`, call after successful post | Abstract definition of "notification is needed" + control notification timing |
| **Infra**        | Create `infra/notification/slack_gateway.go` (new file)                                                                                              | Concrete Slack API implementation                                             |
| **Presentation** | **No changes**                                                                                                                                       | —                                                                             |
| **Entity**       | **No changes**                                                                                                                                       | —                                                                             |

### Step 1: UseCases Layer — Define a New Port

Define the "notification functionality" as an abstraction, not "how to notify." Notification is a tool needed by the application workflow, so the port belongs to the UseCases layer.

**Edit or explanatory excerpt (integrate into the named file):**

```go
// internal/usecase/notification.go (new file)
package usecase

import "context"

// NotificationGateway is the abstract interface for sending notifications.
// Slack, Email, LINE — the UseCase layer doesn't know the specific notification method.
type NotificationGateway interface {
    NotifyNewPost(ctx context.Context, threadTitle string, postAuthor string, postBody string) error
}
```

The new file sits directly under `internal/usecase/`, next to `repository.go` and `transaction.go`, following the same UseCase-owned port placement as the other ports.

Because the constructor now takes another argument, update calls in existing `internal/usecase/post_usecase_test.go` and `internal/adapters/infra/persistence/postgres/repository_test.go` to pass a test NotificationGateway. Its NotifyNewPost can simply return nil; add notification assertions with the mock later in this workshop.

**Key observation**: The word "Slack" never appears in this interface. The UseCase decides "what to notify" and leaves "how to notify" to Infra.

### ✅ Checkpoint

- [ ] `internal/usecase/notification.go` is a **new file** — no existing file was modified (`git status` confirms)
- [ ] The interface name and signature contain nothing Slack-specific (no URLs, no Slack types)
- [ ] `go build ./...` passes

### Step 2: UseCase Layer — Invoke Notification

Inject the notification gateway into `CreatePostUseCase` and call it after successful post creation.

**Edit or explanatory excerpt (integrate into the named file):**

```go
// internal/usecase/post_usecase.go (modify existing file)
// NOTE: Add `"log/slog"` to the import list
type CreatePostUseCase struct {
    threadRepo ThreadRepository
    postRepo   PostRepository
    tm         TransactionManager
    notifier   NotificationGateway  // ← Added: notification gateway
}

// Add notifier to constructor
func NewCreatePostUseCase(
    threadRepo ThreadRepository,
    postRepo PostRepository,
    tm TransactionManager,
    notifier NotificationGateway,  // ← Added
) *CreatePostUseCase {
    return &CreatePostUseCase{
        threadRepo: threadRepo,
        postRepo:   postRepo,
        tm:         tm,
        notifier:   notifier,
    }
}

func (u *CreatePostUseCase) Execute(ctx context.Context, in CreatePostInput) (*CreatePostOutput, error) {
    var (
        out     *CreatePostOutput
        thread  *entity.Thread
        post    *entity.Post
    )
    // ──────────────────────────────────────────────
    // Transaction boundary: all DB operations inside
    // ──────────────────────────────────────────────
    if err := u.tm.RunInTransaction(ctx, func(txCtx context.Context) error {
        var err error
        thread, err = u.threadRepo.FindByID(txCtx, in.ThreadID)
        if err != nil {
            return err
        }
        if thread == nil {
            return domain.ErrThreadNotFound
        }
        // The business rule added in WS4 remains in place (not silently dropped)
        if !thread.CanPost(in.Author) {
            return domain.ErrNotThreadOwner
        }
        count, err := u.postRepo.CountByThreadID(txCtx, thread.ID)
        if err != nil {
            return err
        }
        post, err = entity.NewPost(thread.ID, count+1, in.Author, in.Body, in.Sage)
        if err != nil {
            return err
        }
        if err := u.postRepo.Save(txCtx, post); err != nil {
            return err
        }
        thread.Bump(post.CreatedAt, in.Sage)
        if err := u.threadRepo.Save(txCtx, thread); err != nil {
            return err
        }
        out = &CreatePostOutput{Post: toPostDTO(post)}
        return nil
    }); err != nil {
        return nil, err
    }

    // ──────────────────────────────────────────────
    // Outside transaction: notification
    // Notification failure must NOT roll back the post
    // ──────────────────────────────────────────────
    if err := u.notifier.NotifyNewPost(ctx, thread.Title, post.Author, post.Body); err != nil {
        slog.Warn("notification failed", "error", err)  // Log only, post still succeeds
    }

    return out, nil
}
```

**Key observation**: The notification is called **outside** the transaction. A Slack notification failure should not roll back the post — this is a deliberate design decision.

### ✅ Checkpoint

- [ ] `CreatePostUseCase` now holds a `notifier NotificationGateway` field injected via the constructor
- [ ] `NotifyNewPost` is called **after** `RunInTransaction` returns `nil`, and its error is only logged (`slog.Warn`), never propagated
- [ ] The WS4 `CanPost` check is still in place inside the transaction
- [ ] `go test ./internal/usecase` passes; full build is checked after Step 4 wiring

### Step 3: Infra Layer — Slack Implementation

Create a Slack gateway in a new directory.

**Edit or explanatory excerpt (integrate into the named file):**

```go
// internal/adapters/infra/notification/slack_gateway.go (new file)
package notification

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type SlackGateway struct {
    webhookURL string
    client     *http.Client
}

func NewSlackGateway(webhookURL string) *SlackGateway {
    return &SlackGateway{
        webhookURL: webhookURL,
        client:     &http.Client{},
    }
}

func (g *SlackGateway) NotifyNewPost(ctx context.Context, threadTitle, author, body string) error {
    payload := map[string]string{
        "text": fmt.Sprintf("[%s] %s: %s", threadTitle, author, body),
    }
    jsonPayload, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("marshal slack payload: %w", err)
    }
    req, err := http.NewRequestWithContext(ctx, "POST", g.webhookURL, bytes.NewReader(jsonPayload))
    if err != nil {
        return fmt.Errorf("create slack request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := g.client.Do(req)
    if err != nil {
        return fmt.Errorf("send slack notification: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("slack returned status %d", resp.StatusCode)
    }
    return nil
}
```

### ✅ Checkpoint

- [ ] `SlackGateway` is a **new file** under `internal/adapters/infra/notification/` — Domain, UseCase (other files), and Presentation are untouched
- [ ] `SlackGateway` satisfies the `NotificationGateway` interface without importing anything from the `usecase` package
- [ ] The HTTP request uses `http.NewRequestWithContext`, so cancellation propagates
- [ ] `go test ./internal/adapters/infra/notification` passes; full build is checked after Step 4 wiring

### Step 4: Composition Root — Add to DI

Create the Slack Gateway and inject it into the UseCase in `main.go`.

**Edit or explanatory excerpt (integrate into the named file):**

```go
// cmd/bbs/main.go
func main() {
    // ...existing DI wiring

    // ↓ Added: notification gateway
    notifier := notification.NewSlackGateway(os.Getenv("SLACK_WEBHOOK_URL"))

    // Add notifier to UseCase constructor
    createPost := usecase.NewCreatePostUseCase(threadRepo, postRepo, tm, notifier)
    // Other UseCases need no changes (no notification needed)
}
```

### ✅ Checkpoint

- [ ] Only `main.go` changed in this step — the notifier is created in exactly one place
- [ ] `go build -o bbs ./cmd/bbs/` passes

### Step 5: Verify

```bash
# Set Slack Webhook URL
# (If you don't have one, wire NoOpGateway in main.go instead — see "Handling Non-Notification Scenarios")
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T.../B.../xxx"

# Build and start
go build -o bbs ./cmd/bbs/
# Expected output: (none — the bbs binary appears in the working directory)
DATABASE_URL='postgres://bbs:bbs@127.0.0.1:55434/bbs?sslmode=disable' ./bbs
# Expected output: the server starts and listens on :8080

# Post → Slack notification is sent
curl -X POST http://localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"gopher","body":"Slack notification test"}'
# Expected output: HTTP 201 with the created post as JSON
# (the same response as before this workshop — notification happens in the background)
```

### ✅ Checkpoint

- [ ] Posting via `curl` returns the same response as before the workshop
- [ ] The Slack message arrives in the channel (or, with `NoOpGateway`, the post still succeeds with nothing sent)
- [ ] Temporarily breaking the webhook URL (e.g., an invalid host) shows the warning log `notification failed` while the post still succeeds with HTTP 201
- [ ] `go test ./...` passes in `software/assets/bbs`

---

## Easy Notification Target Swap

Switching from Slack to Email requires only a Composition Root change.

**Edit or explanatory excerpt (integrate into the named file):**

```go
// internal/adapters/infra/notification/email_gateway.go (new file)
type EmailGateway struct {
    smtpHost string
    from     string
    to       string
}

func (g *EmailGateway) NotifyNewPost(ctx context.Context, threadTitle, author, body string) error {
    // Send email via SMTP
    return nil
}
```

**Edit or explanatory excerpt (integrate into the named file):**

```go
// cmd/bbs/main.go — swap (one line)
notifier := notification.NewEmailGateway("smtp.example.com:587", "bbs@example.com", "admin@example.com")
```

UseCase, Domain, and Presentation require **zero changes**.

---

## Handling Non-Notification Scenarios

For testing or CLI versions where notifications are not needed, inject a NoOp (no-operation) implementation.

**Edit or explanatory excerpt (integrate into the named file):**

```go
// internal/adapters/infra/notification/noop_gateway.go
package notification

import "context"

type NoOpGateway struct{}

func NewNoOpGateway() *NoOpGateway {
    return &NoOpGateway{}
}

func (n *NoOpGateway) NotifyNewPost(ctx context.Context, threadTitle, author, body string) error {
    return nil  // Does nothing
}
```

**Edit or explanatory excerpt (integrate into the named file):**

```go
// cmd/bbs/main.go — for scenarios where notification is not needed
notifier := notification.NewNoOpGateway()
createPost := usecase.NewCreatePostUseCase(threadRepo, postRepo, tm, notifier)
```

---

## Testing Benefits

By mocking the notification, UseCase tests can verify behavior without sending actual Slack messages.

**Edit or explanatory excerpt (integrate into the named file):**

```go
// internal/usecase/create_post_test.go
type mockNotifier struct {
    called bool
    thread string
    author string
    body   string
}

func (m *mockNotifier) NotifyNewPost(ctx context.Context, threadTitle, author, body string) error {
    m.called = true
    m.thread = threadTitle
    m.author = author
    m.body = body
    return nil
}

func TestCreatePost_SendsNotification(t *testing.T) {
    notifier := &mockNotifier{}
    uc := usecase.NewCreatePostUseCase(mockThreadRepo, mockPostRepo, mockTM, notifier)

    uc.Execute(ctx, input)

    if !notifier.called {
        t.Error("notification should have been sent")
    }
    if notifier.thread != "Test Thread" {
        t.Errorf("thread title = %q, want %q", notifier.thread, "Test Thread")
    }
}
```

---

## Comparison: Without Layer Separation

**Edit or explanatory excerpt (integrate into the named file):**

```go
// NG: Notification buried in Handler
func CreatePost(w http.ResponseWriter, r *http.Request) {
    // ...save post to DB
    // Hardcoded Slack notification
    http.Post("https://hooks.slack.com/...", "application/json", bytes.NewReader(payload))
    w.WriteHeader(201)
}
```

In this case:

- Changing Slack → Email requires **rewriting the Handler**
- Retry/error handling for notification failures gets mixed into the Handler
- Tests send actual Slack notifications every time (cannot mock)

---

## Key Points

1. **New Port/Adapter Pattern**: New features are added by defining a new Port (interface) and adding a corresponding Adapter. Existing code is minimally affected.
2. **Clear Responsibility Separation**:
   - UseCase defines "what to notify" (`NotificationGateway` signature) — this is a UseCase Port because notification is an application workflow tool, not part of the core domain language
   - UseCase decides "when to notify" (after successful post)
   - Infra implements "how to notify" (Slack Webhook / Email SMTP)
3. **Testability**: Interfaces enable notification mocking, allowing tests independent of external services.
4. **Notification Target Swap**: Changing Slack → Email → LINE requires only a one-line change in Composition Root.

---

## What you learned

- Adding a new external integration means **defining a new Port in the UseCases layer and adding one Adapter** — Domain and Presentation are untouched.
- The Port signature describes **what** the workflow needs (`NotifyNewPost`), never **how** it is delivered — the word "Slack" never crossed into the UseCase layer.
- Side-effecting calls that must not undo the main operation run **outside the transaction**, and their failures degrade gracefully (log-only) instead of failing the request.
- NoOp and mock implementations of the same Port make testing and CLI scenarios free of external dependencies.
- Swapping notification targets (Slack → Email) is a one-line change in the Composition Root.

**Next**: [WS7: Adding Authentication](./clean_arch_ws7_en.md) — the same "keep the core unaware" idea applied to the Presentation layer with middleware.

---

## Cleanup

```bash
# Stop the server first (Ctrl+C in the terminal running ./bbs), then:
unset SLACK_WEBHOOK_URL DATABASE_URL
# Dedicated container for this workshop:
podman rm -f bbs-ws6-postgres
cd /tmp
rm -r /tmp/bbs-ws6
```

---

## References

- [Slack API: Incoming Webhooks](https://api.slack.com/messaging/webhooks)
- [Go net/http documentation](https://pkg.go.dev/net/http)
- [Go Log/slog (structured logging)](https://go.dev/blog/slog)
