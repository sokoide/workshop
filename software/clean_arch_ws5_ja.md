# クリーンアーキテクチャ実習 (WS5): 永続化層の差し替え

この実習では、BBS（2ちゃんねる風掲示板）のデータベースを SQLite から PostgreSQL に移行します。
**Infra 層だけを変更**し、Domain・UseCase・Framework が一切変更不要であることを体験します。

## 前提知識

本実習は [BBS プロジェクト](./assets/bbs/) のコードを題材にします。

## 実習のシナリオ

「トラフィック増加により SQLite から PostgreSQL に移行する」という要件に対応します。

---

## 課題: SQLite → PostgreSQL 移行

### 変更範囲の確認（やってはいけないこと）

以下の層は **1行も変更しません**。

| 層 | 理由 |
|----|------|
| **Domain** | Port Interface（`BoardRepository`, `ThreadRepository` 等）が抽象的なため、実装が SQLite でも PostgreSQL でも同じように呼び出せる |
| **UseCase** | Repository の **Interface** に依存しているため、具象実装が何に置き換わっても影響なし |
| **Framework** | Handler は UseCase を呼ぶだけで、DB の種類を知らない |

### Step 1: 現在の SQLite 実装を確認する

現在の Infra 実装が Domain の Port Interface を満たしていることを確認します。

```go
// domain/port/repository.go（変更不要 — 見るだけ）
type BoardRepository interface {
    FindBySlug(ctx context.Context, slug string) (*entity.Board, error)
    ListAll(ctx context.Context) ([]*entity.Board, error)
}

type ThreadRepository interface {
    FindByID(ctx context.Context, id int64) (*entity.Thread, error)
    FindByBoardSlug(ctx context.Context, boardSlug string) ([]*entity.Thread, error)
    Save(ctx context.Context, thread *entity.Thread) error
}
```

```go
// infra/persistence/sqlite/board_repo.go（現在の実装 — 確認のみ）
type BoardRepository struct {
    db *sql.DB
}

func (r *BoardRepository) FindBySlug(ctx context.Context, slug string) (*entity.Board, error) {
    var m BoardModel
    err := r.db.QueryRowContext(ctx,
        "SELECT id, slug, name FROM boards WHERE slug = ?", slug,  // SQLite の ? プレースホルダ
    ).Scan(&m.ID, &m.Slug, &m.Name)
    if err == sql.ErrNoRows {
        return nil, domain.ErrBoardNotFound  // DB エラー → ドメインエラー
    }
    return r.toEntity(&m), nil
}
```

**確認ポイント**: UseCase は `BoardRepository` **インターフェース** に依存しており、`sqlite.BoardRepository` という具象型を知りません。

### Step 2: PostgreSQL 用リポジトリを作成する

新しいディレクトリに PostgreSQL 実装を作ります。**既存の SQLite ファイルは残したまま**新しい実装を追加します。

```text
infra/persistence/
├── sqlite/              # 既存（そのまま残す）
│   ├── board_repo.go
│   ├── thread_repo.go
│   ├── post_repo.go
│   └── transaction.go
└── postgres/            # 新規追加
    ├── board_repo.go
    ├── thread_repo.go
    ├── post_repo.go
    └── transaction.go
```

**2-1. PostgreSQL 用 Board リポジトリ**

```go
// infra/persistence/postgres/board_repo.go（新規ファイル）
package postgres

type BoardRepository struct {
    db *sql.DB
}

func NewBoardRepository(db *sql.DB) *BoardRepository {
    return &BoardRepository{db: db}
}

func (r *BoardRepository) FindBySlug(ctx context.Context, slug string) (*entity.Board, error) {
    var m BoardModel
    err := r.db.QueryRowContext(ctx,
        "SELECT id, slug, name FROM boards WHERE slug = $1", slug,  // PostgreSQL の $1 プレースホルダ
    ).Scan(&m.ID, &m.Slug, &m.Name)
    if err == sql.ErrNoRows {
        return nil, domain.ErrBoardNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("query board by slug: %w", err)
    }
    return r.toEntity(&m), nil
}

func (r *BoardRepository) ListAll(ctx context.Context) ([]*entity.Board, error) {
    rows, err := r.db.QueryContext(ctx, "SELECT id, slug, name FROM boards ORDER BY slug")
    if err != nil {
        return nil, fmt.Errorf("list boards: %w", err)
    }
    defer rows.Close()
    // ... SQLite 版と同じスキャン処理
}
```

**2-2. PostgreSQL 用 Thread リポジトリ**

```go
// infra/persistence/postgres/thread_repo.go（新規ファイル）
package postgres

type ThreadRepository struct {
    db *sql.DB
}

func (r *ThreadRepository) Save(ctx context.Context, thread *entity.Thread) error {
    _, err := r.db.ExecContext(ctx,
        `INSERT INTO threads (board_slug, title, owner, response_count, created_at, last_posted_at)
         VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,  // RETURNING = PostgreSQL 固有
        thread.BoardSlug, thread.Title, thread.Owner,
        thread.ResponseCount, thread.CreatedAt, thread.LastPostedAt,
    )
    return err
}
```

**2-3. PostgreSQL 用 TransactionManager**

```go
// infra/persistence/postgres/transaction.go（新規ファイル）
package postgres

type TransactionManager struct {
    db *sql.DB
}

func (tm *TransactionManager) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
    tx, err := tm.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    // ctx に tx を格納してリポジトリに伝播
    txCtx := context.WithValue(ctx, txKey{}, tx)
    if err := fn(txCtx); err != nil {
        tx.Rollback()
        return err
    }
    return tx.Commit()
}
```

### Step 3: Composition Root で差し替える

**変更箇所は `cmd/bbs/main.go` の数行だけ**です。

```go
// cmd/bbs/main.go
func main() {
    // 旧: SQLite
    // db, _ := sql.Open("sqlite3", "bbs.db")
    // boardRepo := sqlite.NewBoardRepository(db)
    // threadRepo := sqlite.NewThreadRepository(db)
    // postRepo := sqlite.NewPostRepository(db)
    // tm := sqlite.NewTransactionManager(db)

    // 新: PostgreSQL（ここだけ変更）
    db, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    boardRepo := postgres.NewBoardRepository(db)
    threadRepo := postgres.NewThreadRepository(db)
    postRepo := postgres.NewPostRepository(db)
    tm := postgres.NewTransactionManager(db)

    // ↓ UseCase の組み立ては一切変わらない！
    createThread := usecase.NewCreateThreadUseCase(boardRepo, threadRepo, tm)
    createPost := usecase.NewCreatePostUseCase(threadRepo, postRepo, tm)
    listBoards := usecase.NewListBoardsUseCase(boardRepo)
    listThreads := usecase.NewListThreadsUseCase(threadRepo)
    listPosts := usecase.NewListPostsUseCase(postRepo)

    // Handler の組み立ても変わらない
    handler := framework.NewHandler(createThread, createPost, listBoards, listThreads, listPosts)
    // ...
}
```

### Step 4: 動作確認

```bash
# PostgreSQL の起動（Podman を使用）
podman run -d --name bbs-postgres \
  -e POSTGRES_USER=bbs \
  -e POSTGRES_PASSWORD=bbs \
  -e POSTGRES_DB=bbs \
  -p 5432:5432 \
  postgres:16

# マイグレーション（DDL は PostgreSQL 用に調整）
# SERIAL, RETURNING などの PostgreSQL 固有構文を使用

# アプリケーションのビルド・起動
export DATABASE_URL="postgres://bbs:bbs@localhost:5432/bbs?sslmode=disable"
go build -o bbs ./cmd/bbs/
./bbs

# 動作確認（API は変わらない）
curl http://localhost:8080/api/boards
curl -X POST http://localhost:8080/api/boards/program/threads \
  -H 'Content-Type: application/json' \
  -d '{"title":"PostgreSQLスレ","author":"pg","body":"移行成功！"}'
```

### Step 5: SQLite 版が残っていることを確認

```bash
# SQLite 版もそのまま動く（環境変数で切り替え）
BBS_DB=/tmp/bbs.db ./bbs
```

SQLite 用のコードは **1行も変更していない** ため、そのまま動作します。

---

## エラー境界の重要性

DB 移行を安全に行うための前提として、Infra Adapter が **DB エラーをドメインエラーに変換** していることが重要です。

```go
// OK: Infra Adapter 内で変換
func (r *BoardRepository) FindBySlug(ctx context.Context, slug string) (*entity.Board, error) {
    // ...
    if err == sql.ErrNoRows {
        return nil, domain.ErrBoardNotFound  // DB エラー → ドメインエラー
    }
}
```

```go
// NG: UseCase が DB エラーを知ってしまう
func (u *CreateThreadUseCase) Execute(...) {
    board, err := u.boardRepo.FindBySlug(ctx, slug)
    if err == sql.ErrNoRows {  // ← database/sql に依存！DB変更不可に
        // ...
    }
}
```

この変換を怠ると、UseCase が `database/sql` に依存してしまい、DB 変更ができなくなります。

---

## この実習のポイント

1. **影響範囲の局所化**: DB の変更は Infra 層だけで完結。Domain・UseCase・Framework は一切触らない。
2. **Interface の役割**: UseCase が依存しているのは Port Interface（抽象）であり、具象実装（SQLite/PostgreSQL）ではない。これが差し替えを可能にしている。
3. **Composition Root が唯一の知識源**: 「どの DB を使うか」を知っているのは `main.go` だけ。各層は自分が何を使っているかを知らない。
4. **並行稼働**: SQLite 版と PostgreSQL 版が同時に存在でき、環境変数やビルドタグで切り替え可能。
