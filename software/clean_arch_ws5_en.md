# Clean Architecture Workshop (WS5): Swapping the Persistence Layer

> **⏱️ Estimated Time**: Approx. 60 minutes
> **🎯 Level**: Intermediate
> **💻 Environment**: Go 1.22+, PostgreSQL 16 (podman or docker)
>
> > **💡 Glossary**: For technical terms such as [Clean Architecture](../infra/glossary_en.md#architecture), [Interface](../infra/glossary_en.md#go), and [Context](../infra/glossary_en.md#go), please refer to the [Glossary](../infra/glossary_en.md).

In this workshop, you will migrate the BBS (2channel-style bulletin board) database from SQLite to PostgreSQL.
You will modify **the Infra Adapter layer and Composition Root wiring**, confirming that Domain, UseCase, and Presentation remain completely untouched.

## Prerequisites

This workshop uses the [BBS project](./assets/bbs/) as the subject code.

Checklist before starting:

- [ ] Completed [WS4](./clean_arch_ws4_en.md) — this workshop starts from the WS4 state of the BBS project
- [ ] Go 1.22+ installed — verify with `go version`
- [ ] The BBS project builds: `cd software/assets/bbs && go build ./...` exits with no output
- [ ] `podman` (or `docker`) installed for the PostgreSQL container — verify with `podman --version`

## Workshop Scenario

Due to increased traffic, migrate the database from SQLite to PostgreSQL.

---

## Exercise: SQLite → PostgreSQL Migration

### Identify the Scope (What NOT to Change)

The following layers require **zero modifications**:

| Layer            | Reason                                                                                                                                                  |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Domain**       | Entities and domain errors contain no persistence details; nothing here knows about the DB type                                                         |
| **UseCase**      | Owns the Repository **Port Interfaces** (`internal/usecase/repository.go`) and depends only on them, so swapping concrete implementations has no impact |
| **Presentation** | Handlers call UseCases and know nothing about the DB type                                                                                               |

### Step 1: Review the Current SQLite Implementation

Verify that the current Infra implementation satisfies the UseCase's Port Interface.

```go
// internal/usecase/repository.go (no changes — read only)
type BoardRepository interface {
    FindAll(ctx context.Context) ([]*entity.Board, error)
    FindByName(ctx context.Context, name string) (*entity.Board, error)
    Save(ctx context.Context, board *entity.Board) error
}

type ThreadRepository interface {
    FindByID(ctx context.Context, id int64) (*entity.Thread, error)
    FindByBoardID(ctx context.Context, boardID int64) ([]*entity.Thread, error)
    Save(ctx context.Context, thread *entity.Thread) error
}
```

```go
// internal/adapters/infra/persistence/sqlite/board_repo.go (current — review only)
type BoardRepository struct {
    db *sql.DB
}

func (r *BoardRepository) FindByName(ctx context.Context, name string) (*entity.Board, error) {
    var m BoardModel
    err := executor(ctx, r.db).QueryRowContext(ctx,
        "SELECT id, name, description, created_at FROM boards WHERE name = ?", name,  // SQLite ? placeholder
    ).Scan(&m.ID, &m.Name, &m.Description, &m.CreatedAt)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, domain.ErrBoardNotFound  // DB error → domain error
    }
    return r.toEntity(&m), nil
}
```

**Key observation**: UseCase depends on the `BoardRepository` **interface**, not the `sqlite.BoardRepository` concrete type.

### ✅ Checkpoint

- [ ] You confirmed `internal/usecase/repository.go` defines the interfaces and contains no SQLite-specific code
- [ ] You located the SQLite `?` placeholder in the query and know it becomes `$1` in PostgreSQL
- [ ] `go build ./...` passes in `software/assets/bbs` (nothing changed yet)

### Step 2: Create PostgreSQL Repositories

Create PostgreSQL implementations in a new directory. **Leave existing SQLite files intact**.

```text
internal/adapters/infra/persistence/
├── sqlite/              # Existing (keep as-is)
│   ├── board_repo.go
│   ├── thread_repo.go
│   ├── post_repo.go
│   └── transaction.go
└── postgres/            # New
    ├── board_repo.go
    ├── thread_repo.go
    ├── post_repo.go
    └── transaction.go
```

**2-1. PostgreSQL Board Repository**

```go
// internal/adapters/infra/persistence/postgres/board_repo.go (new file)
package postgres

type BoardRepository struct {
    db *sql.DB
}

func NewBoardRepository(db *sql.DB) *BoardRepository {
    return &BoardRepository{db: db}
}

func (r *BoardRepository) FindByName(ctx context.Context, name string) (*entity.Board, error) {
    var m BoardModel
    err := executor(ctx, r.db).QueryRowContext(ctx,
        "SELECT id, name, description, created_at FROM boards WHERE name = $1", name,  // PostgreSQL $1 placeholder
    ).Scan(&m.ID, &m.Name, &m.Description, &m.CreatedAt)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, domain.ErrBoardNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("query board by name: %w", err)
    }
    return r.toEntity(&m), nil
}

func (r *BoardRepository) FindAll(ctx context.Context) ([]*entity.Board, error) {
    rows, err := executor(ctx, r.db).QueryContext(ctx, "SELECT id, name, description, created_at FROM boards ORDER BY id")
    if err != nil {
        return nil, fmt.Errorf("list boards: %w", err)
    }
    defer rows.Close()
    // ... same scanning logic as SQLite version
}
```

**2-2. PostgreSQL Thread Repository**

```go
// internal/adapters/infra/persistence/postgres/thread_repo.go (new file)
package postgres

type ThreadRepository struct {
    db *sql.DB
}

func (r *ThreadRepository) Save(ctx context.Context, thread *entity.Thread) error {
    // RETURNING retrieves the auto-generated ID (PostgreSQL-specific)
    var id int64
    err := executor(ctx, r.db).QueryRowContext(ctx,
        `INSERT INTO threads (board_id, title, post_count, created_at, last_posted_at)
         VALUES ($1, $2, $3, $4, $5) RETURNING id`,
        thread.BoardID, thread.Title,
        thread.PostCount, thread.CreatedAt, thread.LastPostedAt,
    ).Scan(&id)
    if err != nil {
        return fmt.Errorf("insert thread: %w", err)
    }
    thread.ID = id  // Set auto-generated ID on entity
    return nil
}
```

**2-3. PostgreSQL TransactionManager**

```go
// internal/adapters/infra/persistence/postgres/transaction.go (new file)
package postgres

type TransactionManager struct {
    db *sql.DB
}

func (tm *TransactionManager) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
    tx, err := tm.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    txCtx := context.WithValue(ctx, txKey{}, tx)
    if err := fn(txCtx); err != nil {
        tx.Rollback()
        return err
    }
    return tx.Commit()
}
```

**2-4. How Repositories Participate in Transactions**

The TransactionManager stores `*sql.Tx` in the context via `context.WithValue`. Each repository extracts the transaction from the context using a helper:

```go
// internal/adapters/infra/persistence/postgres/executor.go
type txKey struct{}

func executor(ctx context.Context, db *sql.DB) DBTX {
    if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
        return tx  // Inside transaction → use *sql.Tx
    }
    return db      // Outside transaction → use *sql.DB
}
```

Repositories call `executor(ctx, db)` instead of using `db` directly:

```go
func (r *BoardRepository) FindByName(ctx context.Context, name string) (*entity.Board, error) {
    err := executor(ctx, r.db).QueryRowContext(ctx,  // ← executor, not r.db
        "SELECT ... WHERE name = $1", name,
    ).Scan(...)
    // ...
}
```

When `RunInTransaction` wraps `ctx` with `*sql.Tx`, all repository calls within `fn(txCtx)` automatically participate in the same transaction. No repository code changes are needed.

### ✅ Checkpoint

- [ ] Everything under `internal/adapters/infra/persistence/postgres/` is **new files** — the `sqlite/` directory is untouched (`git status` confirms)
- [ ] All queries use `$1`-style placeholders, and generated IDs are fetched with `RETURNING`
- [ ] Every query goes through `executor(ctx, r.db)`, not `r.db` directly
- [ ] `go build ./...` passes

### Step 3: Swap in Composition Root

**Only a few lines in `cmd/bbs/main.go` change**.

```go
// cmd/bbs/main.go
func main() {
    // Old: SQLite
    // db, _ := sql.Open("sqlite3", "bbs.db")
    // boardRepo := sqlite.NewBoardRepository(db)
    // threadRepo := sqlite.NewThreadRepository(db)
    // postRepo := sqlite.NewPostRepository(db)
    // tm := sqlite.NewTransactionManager(db)

    // New: PostgreSQL (only this changes)
    db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    if err != nil {
        slog.Error("failed to open db", "error", err)
        os.Exit(1)
    }
    boardRepo := postgres.NewBoardRepository(db)
    threadRepo := postgres.NewThreadRepository(db)
    postRepo := postgres.NewPostRepository(db)
    tm := postgres.NewTransactionManager(db)

    // ↓ UseCase wiring is completely unchanged!
    listBoards := usecase.NewListBoardsUseCase(boardRepo)
    listThreads := usecase.NewListThreadsUseCase(boardRepo, threadRepo)
    createThread := usecase.NewCreateThreadUseCase(boardRepo, threadRepo, postRepo, tm)
    listPosts := usecase.NewListPostsUseCase(postRepo)
    createPost := usecase.NewCreatePostUseCase(threadRepo, postRepo, tm)

    // Handler wiring is also unchanged
    boardHandler := handler.NewBoardHandler(listBoards)
    threadHandler := handler.NewThreadHandler(listThreads, createThread)
    postHandler := handler.NewPostHandler(listPosts, createPost)
    router := httpPresentation.NewRouter(boardHandler, threadHandler, postHandler)
    // ...
}
```

### ✅ Checkpoint

- [ ] In `cmd/bbs/main.go` only the repository construction lines changed — the UseCase and handler wiring below is untouched
- [ ] `go build -o bbs ./cmd/bbs/` passes with the `postgres` package wired in

### Step 4: Verify

```bash
# Start PostgreSQL (using Podman)
podman run -d --name bbs-postgres \
  -e POSTGRES_USER=bbs \
  -e POSTGRES_PASSWORD=bbs \
  -e POSTGRES_DB=bbs \
  -p 5432:5432 \
  postgres:16
# Expected output: a 64-character container ID; `podman ps` shows bbs-postgres as Up

# Run migration (DDL adjusted for PostgreSQL)
# Use SERIAL, RETURNING, and other PostgreSQL-specific syntax

# Build and start
export DATABASE_URL="postgres://bbs:bbs@localhost:5432/bbs?sslmode=disable"
go build -o bbs ./cmd/bbs/
# Expected output: (none — the bbs binary appears in the working directory)
./bbs
# Expected output: the server starts and listens on :8080

# Verify (API is unchanged)
curl http://localhost:8080/api/boards
# Expected output: HTTP 200 with a JSON array of boards, e.g.
# [{"id":1,"name":"program","description":"Programming General",...}]
curl -X POST http://localhost:8080/api/boards/program/threads \
  -H 'Content-Type: application/json' \
  -d '{"title":"PostgreSQL thread","author":"pg","body":"Migration successful!"}'
# Expected output: the created thread as JSON (the same shape the SQLite version returned)
```

### ✅ Checkpoint

- [ ] `podman ps` shows `bbs-postgres` running and the migration DDL applied without errors
- [ ] `curl http://localhost:8080/api/boards` returns HTTP 200 with the board list
- [ ] Creating a thread via `curl` succeeds and the response matches what the SQLite version returned for the same request
- [ ] `go test ./...` passes in `software/assets/bbs`

### Step 5: Confirm SQLite Version Still Works

```bash
# SQLite version still runs (switch via environment variable)
# Stop the PostgreSQL-backed instance first (Ctrl+C), or the two will fight over :8080
BBS_DB=/tmp/bbs.db ./bbs
# Expected output: the server starts and listens on :8080, now backed by SQLite
```

SQLite code has **zero modifications**, so it works as-is.

### ✅ Checkpoint

- [ ] The SQLite-backed server starts with `BBS_DB=/tmp/bbs.db ./bbs` and the API works
- [ ] `git diff` shows **no changes** under `internal/adapters/infra/persistence/sqlite/`

---

## Importance of Error Boundaries

A prerequisite for safe DB migration is that Infra Adapters **convert DB errors to domain errors**.

```go
// OK: Conversion inside Infra Adapter
func (r *BoardRepository) FindByName(ctx context.Context, name string) (*entity.Board, error) {
    // ...
    if errors.Is(err, sql.ErrNoRows) {
        return nil, domain.ErrBoardNotFound  // DB error → domain error
    }
}
```

```go
// NG: UseCase learns about DB errors
func (u *CreateThreadUseCase) Execute(...) {
    board, err := u.boardRepo.FindByName(ctx, name)
    if errors.Is(err, sql.ErrNoRows) {  // ← depends on database/sql! Cannot change DB
        // ...
    }
}
```

Without this conversion, UseCase depends on `database/sql`, making DB changes impossible.

---

## Key Points

1. **Localized Impact**: DB changes are confined to the Infra layer. Domain, UseCase, and Presentation are untouched.
2. **Role of Interfaces**: UseCase depends on Port Interfaces (abstractions), not concrete implementations (SQLite/PostgreSQL). This enables swapping.
3. **Composition Root Is the Single Source of Truth**: Only `main.go` knows "which DB to use." Each layer is unaware of what it's using.
4. **Coexistence**: SQLite and PostgreSQL versions can coexist, switchable via environment variables or build tags.

---

## What you learned

- Swapping the persistence layer touches only the **Infra Adapter** and the **Composition Root** wiring — every difference between SQLite and PostgreSQL (`?` vs `$1` placeholders, `RETURNING` for generated IDs) is absorbed inside the new adapter.
- The **error boundary** is what makes the swap safe: as long as `sql.ErrNoRows` is converted to `domain.ErrBoardNotFound` inside Infra, the UseCase never learns which database is running.
- Transactions propagate through `context` (`context.WithValue` + the `executor` helper), so repositories join a transaction without any code change.
- Coexistence of two implementations behind the same port makes rollback trivial: switching back is a one-line change in `main.go`.

**Next**: [WS6: Integrating External Services](./clean_arch_ws6_en.md) — instead of swapping an existing adapter, you will define a brand-new Port and add an adapter for it.

---

## Cleanup

```bash
# Stop the server first (Ctrl+C in the terminal running ./bbs), then:
podman stop bbs-postgres
podman rm bbs-postgres
rm -f bbs /tmp/bbs.db
```

---

## References

- [PostgreSQL 16 Documentation](https://www.postgresql.org/docs/16/)
- [pgx — PostgreSQL Driver and Toolkit](https://github.com/jackc/pgx)
- [Go Databases: Accessing relational databases](https://go.dev/doc/database/)
- [Podman — Running containers](https://docs.podman.io/en/latest/Introduction.html)
