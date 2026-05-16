# Clean Architecture Workshop (WS6): Integrating External Services

In this workshop, you will add "Slack notification on new posts" to the BBS (2channel-style bulletin board).
You will experience the **new Port/Adapter pattern** — defining a new Port and adding an Infra Adapter — with minimal impact on existing code.

## Prerequisites

This workshop uses the [BBS project](./assets/bbs/) as the subject code.

## Workshop Scenario

Add a feature to send a Slack notification whenever a new post is created in a thread.
This is a classic example of the new Port/Adapter pattern.

---

## Exercise: Adding Slack Notifications

### Change Overview

| Layer | Change | Role |
| ------- | -------- | ------ |
| **Domain** | Define `port.NotificationGateway` interface (new file) | Abstract definition of "notification is needed" |
| **UseCase** | Inject `NotificationGateway` into `CreatePostUseCase`, call after successful post | Control notification timing |
| **Infra** | Create `infra/notification/slack_gateway.go` (new file) | Concrete Slack API implementation |
| **Presentation** | **No changes** | — |
| **Entity** | **No changes** | — |

### Step 1: Domain Layer — Define a New Port

Define the "notification functionality" as an abstraction, not "how to notify."

```go
// domain/port/notification.go (new file)
package port

import "context"

// NotificationGateway is the abstract interface for sending notifications.
// Slack, Email, LINE — the Domain layer doesn't know the specific notification method.
type NotificationGateway interface {
    NotifyNewPost(ctx context.Context, threadTitle string, postAuthor string, postBody string) error
}
```

**Key observation**: The word "Slack" never appears in this interface. Domain decides "what to notify" and leaves "how to notify" to Infra.

### Step 2: UseCase Layer — Invoke Notification

Inject the notification gateway into `CreatePostUseCase` and call it after successful post creation.

```go
// usecase/post_usecase.go (modify existing file)
// NOTE: Add `"log/slog"` to the import list
type CreatePostUseCase struct {
    threadRepo port.ThreadRepository
    postRepo   port.PostRepository
    tm         port.TransactionManager
    notifier   port.NotificationGateway  // ← Added: notification gateway
}

// Add notifier to constructor
func NewCreatePostUseCase(
    threadRepo port.ThreadRepository,
    postRepo port.PostRepository,
    tm port.TransactionManager,
    notifier port.NotificationGateway,  // ← Added
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

### Step 3: Infra Layer — Slack Implementation

Create a Slack gateway in a new directory.

```go
// infra/notification/slack_gateway.go (new file)
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

### Step 4: Composition Root — Add to DI

Create the Slack Gateway and inject it into the UseCase in `main.go`.

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

### Step 5: Verify

```bash
# Set Slack Webhook URL
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T.../B.../xxx"

# Build and start
go build -o bbs ./cmd/bbs/
./bbs

# Post → Slack notification is sent
curl -X POST http://localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"gopher","body":"Slack notification test"}'
```

---

## Easy Notification Target Swap

Switching from Slack to Email requires only a Composition Root change.

```go
// infra/notification/email_gateway.go (new file)
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

```go
// cmd/bbs/main.go — swap (one line)
notifier := notification.NewEmailGateway("smtp.example.com:587", "bbs@example.com", "admin@example.com")
```

UseCase, Domain, and Presentation require **zero changes**.

---

## Handling Non-Notification Scenarios

For testing or CLI versions where notifications are not needed, inject a NoOp (no-operation) implementation.

```go
// infra/notification/noop_gateway.go
type NoOpGateway struct{}

func (n *NoOpGateway) NotifyNewPost(ctx context.Context, threadTitle, author, body string) error {
    return nil  // Does nothing
}
```

```go
// cmd/bbs/main.go — for scenarios where notification is not needed
notifier := notification.NewNoOpGateway()
createPost := usecase.NewCreatePostUseCase(threadRepo, postRepo, tm, notifier)
```

---

## Testing Benefits

By mocking the notification, UseCase tests can verify behavior without sending actual Slack messages.

```go
// usecase/create_post_test.go
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
    - Domain defines "what to notify" (`NotificationGateway` signature)
    - UseCase decides "when to notify" (after successful post)
    - Infra implements "how to notify" (Slack Webhook / Email SMTP)
3. **Testability**: Interfaces enable notification mocking, allowing tests independent of external services.
4. **Notification Target Swap**: Changing Slack → Email → LINE requires only a one-line change in Composition Root.
