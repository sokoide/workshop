# Clean Architecture Workshop (WS5): Swapping the Persistence Layer

In this workshop, you will migrate the BBS (2channel-style bulletin board) database from SQLite to PostgreSQL.
You will modify **the Infra Adapter layer and Composition Root wiring**, confirming that Domain, UseCase, and Presentation remain completely untouched.

## Prerequisites

This workshop uses the [BBS project](./assets/bbs/) as the subject code.

## Workshop Scenario

Due to increased traffic, migrate the database from SQLite to PostgreSQL.

---

## Exercise: SQLite → PostgreSQL Migration

### Identify the Scope (What NOT to Change)

The following layers require **zero modifications**:

| Layer | Reason |
| ------- | -------- |
| **Domain** | Port Interfaces (`BoardRepository`, `ThreadRepository`, etc.) are abstract, so calls work identically regardless of SQLite or PostgreSQL |
| **UseCase** | Depends on Repository **Interfaces**, so swapping concrete implementations has no impact |
| **Presentation** | Handlers call UseCases and know nothing about the DB type |

### Step 1: Review the Current SQLite Implementation

Verify that the current Infra implementation satisfies the Domain's Port Interface.

```go
// internal/domain/port/repository.go (no changes — read only)
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

### Step 4: Verify

```bash
# Start PostgreSQL (using Podman)
podman run -d --name bbs-postgres \
  -e POSTGRES_USER=bbs \
  -e POSTGRES_PASSWORD=bbs \
  -e POSTGRES_DB=bbs \
  -p 5432:5432 \
  postgres:16

# Run migration (DDL adjusted for PostgreSQL)
# Use SERIAL, RETURNING, and other PostgreSQL-specific syntax

# Build and start
export DATABASE_URL="postgres://bbs:bbs@localhost:5432/bbs?sslmode=disable"
go build -o bbs ./cmd/bbs/
./bbs

# Verify (API is unchanged)
curl http://localhost:8080/api/boards
curl -X POST http://localhost:8080/api/boards/program/threads \
  -H 'Content-Type: application/json' \
  -d '{"title":"PostgreSQL thread","author":"pg","body":"Migration successful!"}'
```

### Step 5: Confirm SQLite Version Still Works

```bash
# SQLite version still runs (switch via environment variable)
BBS_DB=/tmp/bbs.db ./bbs
```

SQLite code has **zero modifications**, so it works as-is.

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
