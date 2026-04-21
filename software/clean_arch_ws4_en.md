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

```go
// usecase/create_post.go
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
    post := entity.NewPost(thread.ID, in.Author, in.Body, in.Sage)
    // ...
}
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

### Step 4: Framework Layer — Display the Change

Add one case to the error handling.

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
