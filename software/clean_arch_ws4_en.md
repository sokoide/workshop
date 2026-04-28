# Clean Architecture Workshop (WS4): Adding Business Rules

In this workshop, you will add a "only the thread owner can post" business rule to the BBS (2channel-style bulletin board).
You will observe how **changes propagate from inside to outside**, with each layer modifying only its own responsibility.

## Prerequisites

This workshop uses the [BBS project](./assets/bbs/) as the subject code.

## Workshop Scenario

Add a restriction mode where "only the thread owner (first post author) can write" in certain threads.

---

## Exercise: Owner-Only Posting Mode

### Change Overview

Business rule additions propagate from inside to outside, but each layer's change is **minimal and tied to its own responsibility**.

| Layer | Change | Role |
| ------- | -------- | ------ |
| **Domain** | Add `Thread.OwnerOnly` flag, `CanPost()` method, `ErrNotThreadOwner` error | Define the rule |
| **UseCase** | Add one line: `thread.CanPost(in.Author)` call | Apply the rule |
| **Infra** | Add `owner_only` column to `threads` table, update read/write | Persist the change |
| **Framework** | Add one case: `ErrNotThreadOwner → 403 Forbidden` | Display the change |

### Step 1: Domain Layer — Define the Rule

Encapsulate the business rule in the Entity.

**1-1. Add flag and validation method to Thread Entity**

```go
// domain/entity/thread.go
type Thread struct {
    ID           int64
    BoardID      int64
    Title        string
    PostCount    int
    CreatedAt    time.Time
    LastPostedAt time.Time
    // ↓ Added
    OwnerOnly bool   // Whether only the owner can post
    Owner     string // Thread owner (first post author)
}

// Added: Business rule for posting permission
func (t *Thread) CanPost(author string) bool {
    if !t.OwnerOnly {
        return true // Anyone can post if not in restriction mode
    }
    return t.Owner == author
}
```

**Key observation**: The validation logic exists in **one place only**: `CanPost()`. It knows nothing about DB or HTTP.

**1-2. Add domain error**

```go
// domain/error.go
var (
    ErrBoardNotFound  = errors.New("board not found")
    ErrThreadNotFound = errors.New("thread not found")
    // ↓ Added
    ErrNotThreadOwner = errors.New("only thread owner can post")
)
```

### Step 2: UseCase Layer — Apply the Rule

The UseCase layer knows "when to apply the rule." Add **one line** to the post creation process.

**2-1. Add the rule check to CreatePostUseCase**

```go
// usecase/post_usecase.go
func (u *CreatePostUseCase) Execute(ctx context.Context, in CreatePostInput) (*CreatePostOutput, error) {
    thread, err := u.threadRepo.FindByID(ctx, in.ThreadID)
    if err != nil {
        return nil, err
    }

    // ↓ Add this one line
    if !thread.CanPost(in.Author) {
        return nil, domain.ErrNotThreadOwner
    }

    // ...rest of the logic is unchanged
    count, err := u.postRepo.CountByThreadID(ctx, thread.ID)
    if err != nil {
        return nil, err
    }
    post, err := entity.NewPost(thread.ID, count+1, in.Author, in.Body, in.Sage)
    // ...
}
```

**2-2. Add `OwnerOnly` to the DTO**

Add the `OwnerOnly` field to `CreateThreadInput` in `usecase/dto.go` so the Framework layer can pass the flag.

```go
// usecase/dto.go
type CreateThreadInput struct {
    BoardSlug string
    Title     string
    Author    string
    Body      string
    OwnerOnly bool   // Added: owner-only mode
}
```

**2-3. Set Owner and OwnerOnly in CreateThreadUseCase**

`Owner` (thread owner = first post author) is set during thread creation in `CreateThreadUseCase`.

```go
// usecase/thread_usecase.go (inside CreateThreadUseCase.Execute)
thread, err := entity.NewThread(board.ID, in.Title)
if err != nil {
    return nil, err
}
thread.OwnerOnly = in.OwnerOnly  // Flag from Framework
thread.Owner = in.Author         // Record first post author as owner
```

### Step 3: Infra Layer — Persist the Change

Add a new column to the DB and update read/write operations. **Only SQL and model conversion changes**.

**3-1. Migration**

```sql
ALTER TABLE threads ADD COLUMN owner_only BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE threads SET owner_only = FALSE;
```

**3-2. Update DB model**

```go
// infra/persistence/sqlite/model.go
type ThreadModel struct {
    ID           int64  `db:"id"`
    BoardID      int64  `db:"board_id"`
    Title        string `db:"title"`
    OwnerOnly    bool   `db:"owner_only"`  // Added
    // ...
}
```

**3-3. Update repository conversion**

```go
// infra/persistence/sqlite/thread_repo.go
func (r *ThreadRepository) toEntity(m *ThreadModel) *entity.Thread {
    return &entity.Thread{
        ID:           m.ID,
        BoardID:      m.BoardID,
        Title:        m.Title,
        OwnerOnly:    m.OwnerOnly,  // Added
        Owner:        m.Owner,       // Leverage existing field
        // ...
    }
}
```

**3-4. Convert driver errors to domain errors**

Infra Adapter has the **responsibility of converting driver errors to domain errors**. This ensures the UseCase layer remains unaware of database details.

```go
// infra/persistence/sqlite/thread_repo.go
import (
    "database/sql"
    "errors"
    "yourproject/domain"
)

func (r *ThreadRepository) FindByID(ctx context.Context, id int64) (*entity.Thread, error) {
    var m ThreadModel
    err := r.db.QueryRowContext(ctx, "SELECT ... FROM threads WHERE id = ?", id).Scan(
        &m.ID, &m.BoardID, &m.Title, &m.OwnerOnly, &m.Owner, /* ... */,
    )
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            // Convert driver error to domain error
            return nil, domain.ErrThreadNotFound
        }
        return nil, err
    }
    return r.toEntity(&m), nil
}
```

**Key observation**: The technical error `sql.ErrNoRows` is converted to the domain concept `domain.ErrThreadNotFound` before reaching the UseCase layer. The UseCase doesn't need to know "SQL returned no rows" — it only needs to know "thread not found."

### Step 4: Framework Layer — Display the Change

Add one case to the error handling in `PostHandler`, and add the `owner_only` field to the thread creation request DTO.

**4-1. Error handling for PostHandler**

```go
// framework/handler/post_handler.go — error handling in CreatePost
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
    out, err := h.createPost.Execute(r.Context(), input)
    // ...existing error handling
    // ↓ Add one case
    case errors.Is(err, domain.ErrNotThreadOwner):
        writeError(w, http.StatusForbidden, err.Error())
    }
}
```

**4-2. Add `owner_only` to thread creation request**

```go
// internal/framework/http/handler/thread_handler.go
type createThreadRequest struct {
    Title     string `json:"title"`
    Author    string `json:"author"`
    Body      string `json:"body"`
    OwnerOnly bool   `json:"owner_only"`  // Added
}
```

### Step 5: Verify

```bash
# Create an owner-only thread
curl -X POST http://localhost:8080/api/boards/program/threads \
  -H 'Content-Type: application/json' \
  -d '{"title":"Owner-only thread","author":"gopher","body":"Restricted mode ON","owner_only":true}'

# Non-owner tries to post → 403 Forbidden
curl -X POST http://localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"other","body":"Can I post?"}'

# Owner posts → success
curl -X POST http://localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"gopher","body":"My own thread, so OK"}'
```

---

## Resilient to Further Specification Changes

If the rule changes to "owner + invited users can post":

```go
// Only Domain changes
func (t *Thread) CanPost(author string) bool {
    if !t.OwnerOnly {
        return true
    }
    return t.Owner == author || t.IsInvited(author)
}
```

UseCase, Infra, and Framework require **zero changes**. Callers don't know the internals of `CanPost()`, so they're unaffected.

---

## Comparison: Without Layer Separation

```go
// NG: Business rule buried in Handler
func CreatePost(w http.ResponseWriter, r *http.Request) {
    row := db.QueryRow(
        "SELECT author FROM posts WHERE thread_id=? AND number=1", threadID,
    )
    var owner string
    row.Scan(&owner)
    author := r.FormValue("author")
    if owner != author {                    // ← Business rule buried here
        w.WriteHeader(403)                  // ← HTTP dependency
        json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
        return
    }
    // INSERT ...
}
```

- "Thread owner = first post author" rule is buried in **SQL**
- "Only thread owner can post" rule is buried in **HTTP handler**
- "Invited users also OK" change: **unclear where to fix**

---

## Key Points

1. **Clear Rule Location**:
    - Domain knows "what the rule is" (`CanPost` implementation)
    - UseCase knows "when to apply the rule" (call timing)
    - Framework knows "how to display rule violations" (403 Forbidden)
    - Infra knows "how to persist rule data" (owner_only column)
2. **Inside→Outside Propagation**: Business rule changes start in Domain and propagate outward, but each layer's changes are limited to its own responsibility.
3. **Minimal Changes**: Adding "invited users also OK" only requires changing `CanPost()` internals.
