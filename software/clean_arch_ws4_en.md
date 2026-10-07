# Clean Architecture Workshop (WS4): Adding Business Rules

> **⏱️ Estimated Time**: Approx. 90 minutes
> **🎯 Level**: Intermediate
> **💻 Environment**: Go 1.22+, any OS (curl required for verification)
>
> > **💡 Glossary**: For technical terms such as [Clean Architecture](../infra/glossary_en.md#architecture), [Interface](../infra/glossary_en.md#go), and [Context](../infra/glossary_en.md#go), please refer to the [Glossary](../infra/glossary_en.md).

In this workshop, you will add a "only the thread owner can post" business rule to the BBS (2channel-style bulletin board).
You will observe how **changes propagate from inside to outside**, with each layer modifying only its own responsibility.

## Prerequisites

This workshop uses the [BBS project](./assets/bbs/) as the subject code.

Checklist before starting:

- [ ] Completed [WS1](./clean_arch_ws1_en.md)–[WS3](./clean_arch_ws3_en.md) (or you are comfortable with entities, ports, and the Composition Root)
- [ ] Go 1.22+ installed — verify with `go version`
- [ ] The BBS project builds and tests pass: `cd software/assets/bbs && go build ./... && go test ./...`
- [ ] `curl` available — verify with `curl --version`

## Workshop Scenario

Add a restriction mode where "only the thread owner (first post author) can write" in certain threads.

---

## Exercise: Owner-Only Posting Mode

### Change Overview

Business rule additions propagate from inside to outside, but each layer's change is **minimal and tied to its own responsibility**.

| Layer            | Change                                                                                  | Role                          |
| ---------------- | --------------------------------------------------------------------------------------- | ----------------------------- |
| **Domain**       | Add `Thread.OwnerOnly` flag, `CanPost()` method, `ErrNotThreadOwner` error              | Define the rule               |
| **UseCase**      | Add `thread.CanPost(in.Author)` call, set `thread.Owner = in.Author` on thread creation | Apply the rule, persist owner |
| **Infra**        | Add `owner_only` and `owner` columns to `threads` table, update read/write              | Persist the change            |
| **Presentation** | Add one case: `ErrNotThreadOwner → 403 Forbidden`                                       | Display the change            |

### Step 1: Domain Layer — Define the Rule

Encapsulate the business rule in the Entity.

**1-1. Add flag and validation method to Thread Entity**

> **Note:** The current `Thread` struct does not have `OwnerOnly` or `Owner` fields. These are added in this step.

```go
// internal/domain/entity/thread.go
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

// Added: Encapsulated owner-only mode setup to protect invariants
func (t *Thread) EnableOwnerOnlyMode(owner string) error {
    if owner == "" {
        return errors.New("owner must not be empty when owner-only mode is enabled")
    }
    t.OwnerOnly = true
    t.Owner = owner
    return nil
}
```

**Key observation**: The validation logic exists in **one place only**: `CanPost()`. It knows nothing about DB or HTTP.

**1-2. Add domain error**

```go
// internal/domain/error.go
var (
    ErrBoardNotFound  = errors.New("board not found")
    ErrThreadNotFound = errors.New("thread not found")
    // ↓ Added
    ErrNotThreadOwner = errors.New("only thread owner can post")
)
```

### ✅ Checkpoint

- [ ] `CanPost` and `EnableOwnerOnlyMode` live **only** in the Domain layer — no HTTP, DB, or gRPC references in `internal/domain/entity/thread.go`
- [ ] An empty owner cannot enable owner-only mode: `EnableOwnerOnlyMode("")` returns an error (add a quick table-driven test)
- [ ] `go build ./...` and `go test ./...` pass in `software/assets/bbs`

### Step 2: UseCase Layer — Apply the Rule

The UseCase layer knows "when to apply the rule." Add **one line** to the post creation process.

**2-1. Add the rule check to CreatePostUseCase**

```go
// internal/usecase/post_usecase.go
func (u *CreatePostUseCase) Execute(ctx context.Context, in CreatePostInput) (*CreatePostOutput, error) {
    var out *CreatePostOutput
    if err := u.tm.RunInTransaction(ctx, func(txCtx context.Context) error {
        thread, err := u.threadRepo.FindByID(txCtx, in.ThreadID)
        if err != nil {
            return err
        }

        // ↓ Add this one line
        if !thread.CanPost(in.Author) {
            return domain.ErrNotThreadOwner
        }

        count, err := u.postRepo.CountByThreadID(txCtx, thread.ID)
        if err != nil {
            return err
        }
        post, err := entity.NewPost(thread.ID, count+1, in.Author, in.Body, in.Sage)
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

    return out, nil
}
```

**2-2. Transaction Management**

The UseCase layer also has the responsibility of controlling transaction boundaries.

- **TransactionManager**: Defined as a UseCase Port (`usecase.TransactionManager`)
- **Infra Adapter**: Implements concretely using SQLite's `sql.Tx`, etc.
- **UseCase**: Groups multiple repository operations into a single transaction

```go
// internal/usecase/transaction.go
type TransactionManager interface {
    RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

In post creation, the entire sequence — fetch thread, check permission, count existing posts, assign post number, save the post, and bump (update) the thread — must execute atomically. If any step fails, all changes must roll back. The UseCase controls this transaction boundary to prevent partial states (e.g., post saved but thread not bumped).

**Key observation**: Transactions are not "technical details" but "application policies." The UseCase decides "this operation set should be atomic," and the Infra Adapter provides the concrete implementation (`BEGIN`/`COMMIT`/`ROLLBACK`).

**2-3. Add `OwnerOnly` to the DTO**

Add the `OwnerOnly` field to `CreateThreadInput` in `usecase/dto.go` so the Presentation layer can pass the flag.

```go
// internal/usecase/dto.go
type CreateThreadInput struct {
    BoardName string
    Title     string
    Author    string
    Body      string
    OwnerOnly bool   // Added: owner-only mode
}
```

**2-4. Set Owner and OwnerOnly in CreateThreadUseCase**

`Owner` (thread owner = first post author) is set during thread creation in `CreateThreadUseCase`.

```go
// internal/usecase/thread_usecase.go (inside CreateThreadUseCase.Execute)
thread, err := entity.NewThread(board.ID, in.Title, in.Author)
if err != nil {
    return nil, err
}
if in.OwnerOnly {
    if err := thread.EnableOwnerOnlyMode(in.Author); err != nil {
        return nil, err  // Protects invariant: Owner must not be empty
    }
}
```

### ✅ Checkpoint

- [ ] The rule check is the **one added line** — `if !thread.CanPost(in.Author) { return domain.ErrNotThreadOwner }` — inside the transaction, before any post is created
- [ ] The UseCase still imports no driver or HTTP packages (`git grep -e "database/sql" -e "net/http" internal/usecase` finds nothing)
- [ ] `CreateThreadInput` carries `OwnerOnly`, and the owner is set only via `EnableOwnerOnlyMode` (never by bare assignment)
- [ ] Existing usecase tests still pass: `go test ./internal/usecase/...`

### Step 3: Infra Layer — Persist the Change

Add a new column to the DB and update read/write operations. **Only SQL and model conversion changes**.

**3-1. Migration**

```sql
ALTER TABLE threads ADD COLUMN owner_only BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE threads ADD COLUMN owner TEXT NOT NULL DEFAULT '';
UPDATE threads SET owner_only = FALSE, owner = '';
```

**3-2. Update DB model**

```go
// internal/adapters/infra/persistence/model.go
type ThreadModel struct {
    ID            int64  `db:"id"`
    BoardID       int64  `db:"board_id"`
    Title         string `db:"title"`
    PostCount     int    `db:"post_count"`
    OwnerOnly     bool   `db:"owner_only"`  // Added
    Owner         string `db:"owner"`       // Added
    // ...
}
```

**3-3. Update repository conversion**

```go
// internal/adapters/infra/persistence/thread_repo.go
func (r *ThreadRepository) toEntity(m *ThreadModel) *entity.Thread {
    return &entity.Thread{
        ID:           m.ID,
        BoardID:      m.BoardID,
        Title:        m.Title,
        PostCount:    m.PostCount,
        OwnerOnly:    m.OwnerOnly,  // Added
        Owner:        m.Owner,      // Added
        // ...
    }
}
```

> **Note on Entity Reconstruction (Allowing Bare Struct Literals):**  
> As a domain rule, creating or initializing entity structures directly using bare struct literals (such as `&entity.Thread{}`) in layers like UseCases should be avoided to protect domain invariants (ensuring they pass validation).  
> On the other hand, in the Infra Adapters layer, which restores the entity state based on existing data retrieved from the database ("Reconstruction"), it is exceptionally acceptable to use bare struct literals to map fields directly.

**3-4. Convert driver errors to domain errors**

Infra Adapter has the **responsibility of converting driver errors to domain errors**. This ensures the UseCase layer remains unaware of database details.

```go
// internal/adapters/infra/persistence/sqlite/thread_repo.go
import (
    "database/sql"
    "errors"
    "your-project/internal/domain"
)

func (r *ThreadRepository) FindByID(ctx context.Context, id int64) (*entity.Thread, error) {
    var m ThreadModel
    err := executor(ctx, r.db).QueryRowContext(ctx, "SELECT ... FROM threads WHERE id = ?", id).Scan(
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

### ✅ Checkpoint

- [ ] The migration adds only the two new columns; existing rows get `owner_only = FALSE, owner = ''`
- [ ] `ThreadModel` and `toEntity` map `OwnerOnly` / `Owner` in both directions
- [ ] `sql.ErrNoRows` never escapes the Infra layer: `git grep sql.ErrNoRows internal/adapters/infra` finds it only there, converted to `domain.ErrThreadNotFound`
- [ ] `go build ./...` and `go test ./...` pass

### Step 4: Presentation Layer — Display the Change

Add one case to the error handling in `PostHandler`, and add the `owner_only` field to the thread creation request DTO.

**4-1. Error handling for PostHandler**

```go
// internal/adapters/presentation/http/handler/post_handler.go — error handling in CreatePost
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
    out, err := h.createPost.Execute(r.Context(), input)
    if err != nil {
        switch {
        case errors.Is(err, domain.ErrThreadNotFound):
            writeError(w, http.StatusNotFound, err.Error())
        // ↓ Add one case
        case errors.Is(err, domain.ErrNotThreadOwner):
            writeError(w, http.StatusForbidden, err.Error())
        case errors.Is(err, domain.ErrEmptyBody):
            writeError(w, http.StatusBadRequest, err.Error())
        default:
            writeError(w, http.StatusInternalServerError, "internal server error")
        }
        return
    }
    // ...
}
```

**4-2. Add `owner_only` to thread creation request**

```go
// internal/adapters/presentation/http/handler/thread_handler.go
type createThreadRequest struct {
    Title     string `json:"title"`
    Author    string `json:"author"`
    Body      string `json:"body"`
    OwnerOnly bool   `json:"owner_only"`  // Added
}
```

### ✅ Checkpoint

- [ ] `ErrNotThreadOwner` maps to exactly **one added case** (`403 Forbidden`) in `PostHandler` — no other handler changed
- [ ] Non-domain errors still return a generic 500 without leaking internals
- [ ] `go build ./...` passes

### Step 5: Verify

```bash
# Build and start first (if not already running):
go build -o bbs ./cmd/bbs/
# Expected output: (none — the bbs binary appears in the working directory)
./bbs
# Expected output: the server starts and listens on :8080

# Create an owner-only thread
curl -X POST http://localhost:8080/api/boards/program/threads \
  -H 'Content-Type: application/json' \
  -d '{"title":"Owner-only thread","author":"gopher","body":"Restricted mode ON","owner_only":true}'
# Expected output: HTTP 201 — JSON of the created thread (with owner-only enabled) and the first post

# Non-owner tries to post → 403 Forbidden
curl -X POST http://localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"other","body":"Can I post?"}'
# Expected output: HTTP 403 — {"error":"only thread owner can post"}

# Owner posts → success
curl -X POST http://localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"gopher","body":"My own thread, so OK"}'
# Expected output: HTTP 201 — JSON of the created post
```

### ✅ Checkpoint

- [ ] Non-owner post returns `403` with `only thread owner can post`
- [ ] Owner post succeeds (`201`)
- [ ] Posting to a normal (non owner-only) thread from any author still succeeds — the rule did not leak outside restricted threads
- [ ] `go test ./...` passes in `software/assets/bbs`

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

If invited users are managed within the `Thread` entity (e.g., an `InvitedUsers` field), only Domain changes are needed — UseCase, Infra, and Presentation require **zero changes**. If invited users require a new data source (e.g., a separate table or external service), Infra and potentially UseCase would also need updates.

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
   - Presentation knows "how to display rule violations" (403 Forbidden)
   - Infra knows "how to persist rule data" (owner_only + owner columns)
2. **Inside→Outside Propagation**: Business rule changes start in Domain and propagate outward, but each layer's changes are limited to its own responsibility.
3. **Minimal Changes**: Adding "invited users also OK" only requires changing `CanPost()` internals, provided invited users are managed within the Thread entity.

---

## What you learned

- A business rule has a **single home**: Domain defines it (`CanPost`), UseCase applies it (call timing), Infra persists its data (`owner_only` / `owner` columns), Presentation displays violations (`403 Forbidden`).
- Rule changes propagate **inside → outside**, and each layer's diff stays tied to its own responsibility — nothing else.
- Transaction boundaries are application policy owned by the UseCase; Infra only supplies `BEGIN`/`COMMIT`/`ROLLBACK`.
- Guarding invariants through methods (`EnableOwnerOnlyMode`) instead of bare field writes keeps invalid states unrepresentable.
- Driver errors are converted to domain errors at the Infra boundary, so the UseCase reasons in domain concepts only.

**Next**: [WS5: Swapping the Persistence Layer](./clean_arch_ws5_en.md) — the next step in the Clean Architecture workshop series.

---

## Cleanup

```bash
# Stop the BBS server (Ctrl+C in its terminal), then:
cd software/assets/bbs
make clean   # removes the bbs binary and bbs.db
```

Or manually:

```bash
rm -f software/assets/bbs/bbs software/assets/bbs/bbs.db
```

---

## References

- [The Clean Architecture — Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [SQLite: ALTER TABLE](https://www.sqlite.org/lang_altertable.html)
- [net/http package — Go Documentation](https://pkg.go.dev/net/http)
- [errors package — Go Documentation](https://pkg.go.dev/errors)
