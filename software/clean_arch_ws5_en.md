# Clean Architecture Workshop (WS5): Swapping the Persistence Layer

In this workshop, you will migrate the BBS (2channel-style bulletin board) database from SQLite to PostgreSQL.
You will modify **only the Infra layer**, confirming that Domain, UseCase, and Framework remain completely untouched.

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
| **Framework** | Handlers call UseCases and know nothing about the DB type |

### Step 1: Review the Current SQLite Implementation

Verify that the current Infra implementation satisfies the Domain's Port Interface.

```go
// domain/port/repository.go (no changes — read only)
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
// infra/persistence/sqlite/board_repo.go (current — review only)
type BoardRepository struct {
    db *sql.DB
}

func (r *BoardRepository) FindByName(ctx context.Context, name string) (*entity.Board, error) {
    var m BoardModel
    err := r.db.QueryRowContext(ctx,
        "SELECT id, name, name, created_at FROM boards WHERE name = ?", name,  // SQLite ? placeholder
    ).Scan(&m.ID, &m.Slug, &m.Name, &m.CreatedAt)
    if err == sql.ErrNoRows {
        return nil, domain.ErrBoardNotFound  // DB error → domain error
    }
    return r.toEntity(&m), nil
}
```

**Key observation**: UseCase depends on the `BoardRepository` **interface**, not the `sqlite.BoardRepository` concrete type.

### Step 2: Create PostgreSQL Repositories

Create PostgreSQL implementations in a new directory. **Leave existing SQLite files intact**.

```text
infra/persistence/
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
// infra/persistence/postgres/board_repo.go (new file)
package postgres

type BoardRepository struct {
    db *sql.DB
}

func NewBoardRepository(db *sql.DB) *BoardRepository {
    return &BoardRepository{db: db}
}

func (r *BoardRepository) FindByName(ctx context.Context, name string) (*entity.Board, error) {
    var m BoardModel
    err := r.db.QueryRowContext(ctx,
        "SELECT id, name, name, created_at FROM boards WHERE name = $1", name,  // PostgreSQL $1 placeholder
    ).Scan(&m.ID, &m.Slug, &m.Name, &m.CreatedAt)
    if err == sql.ErrNoRows {
        return nil, domain.ErrBoardNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("query board by name: %w", err)
    }
    return r.toEntity(&m), nil
}

func (r *BoardRepository) FindAll(ctx context.Context) ([]*entity.Board, error) {
    rows, err := r.db.QueryContext(ctx, "SELECT id, name, name, created_at FROM boards ORDER BY id")
    if err != nil {
        return nil, fmt.Errorf("list boards: %w", err)
    }
    defer rows.Close()
    // ... same scanning logic as SQLite version
}
```

**2-2. PostgreSQL Thread Repository**

```go
// infra/persistence/postgres/thread_repo.go (new file)
package postgres

type ThreadRepository struct {
    db *sql.DB
}

func (r *ThreadRepository) Save(ctx context.Context, thread *entity.Thread) error {
    _, err := r.db.ExecContext(ctx,
        `INSERT INTO threads (board_id, title, owner, post_count, created_at, last_posted_at)
         VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,  // RETURNING = PostgreSQL-specific
        thread.BoardID, thread.Title, thread.Owner,
        thread.PostCount, thread.CreatedAt, thread.LastPostedAt,
    )
    return err
}
```

**2-3. PostgreSQL TransactionManager**

```go
// infra/persistence/postgres/transaction.go (new file)
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
    db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    boardRepo := postgres.NewBoardRepository(db)
    threadRepo := postgres.NewThreadRepository(db)
    postRepo := postgres.NewPostRepository(db)
    tm := postgres.NewTransactionManager(db)

    // ↓ UseCase wiring is completely unchanged!
    createThread := usecase.NewCreateThreadUseCase(boardRepo, threadRepo, postRepo, tm)
    createPost := usecase.NewCreatePostUseCase(threadRepo, postRepo, tm)
    listBoards := usecase.NewListBoardsUseCase(boardRepo)
    listThreads := usecase.NewListThreadsUseCase(boardRepo, threadRepo)
    listPosts := usecase.NewListPostsUseCase(postRepo)

    // Handler wiring is also unchanged
    handler := framework.NewHandler(createThread, createPost, listBoards, listThreads, listPosts)
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
    if err == sql.ErrNoRows {
        return nil, domain.ErrBoardNotFound  // DB error → domain error
    }
}
```

```go
// NG: UseCase learns about DB errors
func (u *CreateThreadUseCase) Execute(...) {
    board, err := u.boardRepo.FindByName(ctx, name)
    if err == sql.ErrNoRows {  // ← depends on database/sql! Cannot change DB
        // ...
    }
}
```

Without this conversion, UseCase depends on `database/sql`, making DB changes impossible.

---

## Key Points

1. **Localized Impact**: DB changes are confined to the Infra layer. Domain, UseCase, and Framework are untouched.
2. **Role of Interfaces**: UseCase depends on Port Interfaces (abstractions), not concrete implementations (SQLite/PostgreSQL). This enables swapping.
3. **Composition Root Is the Single Source of Truth**: Only `main.go` knows "which DB to use." Each layer is unaware of what it's using.
4. **Coexistence**: SQLite and PostgreSQL versions can coexist, switchable via environment variables or build tags.
