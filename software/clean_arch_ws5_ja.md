# クリーンアーキテクチャ実習 (WS5): 永続化層の差し替え

> **⏱️ 所要時間**: 約 60 分
> **🎯 レベル**: Intermediate
> **💻 環境**: Go 1.22+、PostgreSQL 16（podman または docker）
>
> > **💡 用語集**: [クリーンアーキテクチャ](../infra/glossary_ja.md#architecture)や[インターフェース](../infra/glossary_ja.md#go)などの専門用語は [用語集](../infra/glossary_ja.md) を参照してください。

この実習では、BBS（2 ちゃんねる風掲示板）のデータベースを SQLite から PostgreSQL に移行します。
**Infra Adapter 層と Composition Root の配線だけを変更**し、Domain・UseCase・Presentation が一切変更不要であることを体験します。

## 前提知識

本実習は [BBS プロジェクト](./assets/bbs/) のコードを題材にします。
共有のアーキテクチャ解説（3 層バリアント・依存ルール・エラー境界）は [clean_arch.md](./clean_arch_ja.md#レイヤー構造と依存関係) にまとめてあります。

開始前のチェックリスト:

- [ ] [WS4](./clean_arch_ws4_ja.md) を完了している — 本実習は BBS プロジェクトの WS4 完了時点の状態から始めます
- [ ] Go 1.22 以上がインストール済み（`go version` で確認）
- [ ] BBS プロジェクトがビルドできる: `cd software/assets/bbs && go build ./...` が何も出力せずに終了する
- [ ] PostgreSQL コンテナ用に `podman`（または `docker`）が使える（`podman --version` で確認）

## 実習のシナリオ

「トラフィック増加により SQLite から PostgreSQL に移行する」という要件に対応します。

---

## 課題: SQLite → PostgreSQL 移行

### 変更範囲の確認（やってはいけないこと）

以下の層は **1行も変更しません**。

| 層               | 理由                                                                                                                                           |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| **Domain**       | エンティティやドメインエラーに永続化の詳細はなく、DB の種類を一切知らない                                                                      |
| **UseCase**      | Repository の **Port Interface**（`internal/usecase/repository.go`）を所有し、それにのみ依存しているため、具象実装が何に置き換わっても影響なし |
| **Presentation** | Handler は UseCase を呼ぶだけで、DB の種類を知らない                                                                                           |

### Step 1: 現在の SQLite 実装を確認する

現在の Infra 実装が UseCase の Port Interface を満たしていることを確認します。

```go
// internal/usecase/repository.go（変更不要 — 見るだけ）
type BoardRepository interface {
    FindAll(ctx context.Context) ([]*entity.Board, error)
    FindByName(ctx context.Context, name string) (*entity.Board, error)
    Save(ctx context.Context, board *entity.Board) error
}

type ThreadRepository interface {
    FindByBoardID(ctx context.Context, boardID int64) ([]*entity.Thread, error)
    FindByID(ctx context.Context, id int64) (*entity.Thread, error)
    Save(ctx context.Context, thread *entity.Thread) error
}
```

```go
// internal/adapters/infra/persistence/sqlite/board_repo.go（現在の実装 — 確認のみ）
type BoardRepository struct {
    db *sql.DB
}

func (r *BoardRepository) FindByName(ctx context.Context, name string) (*entity.Board, error) {
    var m BoardModel
    err := executor(ctx, r.db).QueryRowContext(ctx,
        "SELECT id, name, description, created_at FROM boards WHERE name = ?", name,  // SQLite の ? プレースホルダ
    ).Scan(&m.ID, &m.Name, &m.Description, &m.CreatedAt)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, domain.ErrBoardNotFound  // DB エラー → ドメインエラー
    }
    if err != nil {
        return nil, fmt.Errorf("query board by name: %w", err)
    }
    return r.toEntity(&m), nil
}
```

**確認ポイント**: UseCase は `BoardRepository` **インターフェース** に依存しており、`sqlite.BoardRepository` という具象型を知りません。

### ✅ チェックポイント

- [ ] `internal/usecase/repository.go` が interface を定義しており、SQLite 固有のコードが含まれていないことを確認した
- [ ] クエリ内の SQLite の `?` プレースホルダを見つけ、PostgreSQL では `$1` になることを理解した
- [ ] `software/assets/bbs` で `go build ./...` が通る（まだ何も変更していない状態）

### Step 2: PostgreSQL 用リポジトリを作成する

新しいディレクトリに PostgreSQL 実装を作ります。**既存の SQLite ファイルは残したまま**新しい実装を追加します。

```text
internal/adapters/infra/persistence/
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
// internal/adapters/infra/persistence/postgres/board_repo.go（新規ファイル）
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
        "SELECT id, name, description, created_at FROM boards WHERE name = $1", name,  // PostgreSQL の $1 プレースホルダ
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
    // ... SQLite 版と同じスキャン処理
}
```

**2-2. PostgreSQL 用 Thread リポジトリ**

```go
// internal/adapters/infra/persistence/postgres/thread_repo.go（新規ファイル）
package postgres

type ThreadRepository struct {
    db *sql.DB
}

func (r *ThreadRepository) Save(ctx context.Context, thread *entity.Thread) error {
    // RETURNING で自動採番された ID を取得（PostgreSQL 固有）
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
    thread.ID = id  // 自動採番された ID を entity に反映
    return nil
}
```

**2-3. PostgreSQL 用 TransactionManager**

```go
// internal/adapters/infra/persistence/postgres/transaction.go（新規ファイル）
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

**2-4. リポジトリがトランザクションに参加する仕組み**

TransactionManager は `context.WithValue` で `*sql.Tx` を context に格納します。各リポジトリはヘルパー関数でトランザクションを取り出します:

```go
// internal/adapters/infra/persistence/postgres/executor.go
type txKey struct{}

func executor(ctx context.Context, db *sql.DB) DBTX {
    if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
        return tx  // トランザクション内 → *sql.Tx を使用
    }
    return db      // トランザクション外 → *sql.DB を使用
}
```

リポジトリは `r.db` を直接使わず `executor(ctx, r.db)` を呼び出します:

```go
func (r *BoardRepository) FindByName(ctx context.Context, name string) (*entity.Board, error) {
    err := executor(ctx, r.db).QueryRowContext(ctx,  // ← r.db ではなく executor
        "SELECT ... WHERE name = $1", name,
    ).Scan(...)
    // ...
}
```

`RunInTransaction` が context に `*sql.Tx` を格納すると、`fn(txCtx)` 内の全リポジトリ呼び出しが自動的に同じトランザクションに参加します。リポジトリのコード変更は不要です。

### ✅ チェックポイント

- [ ] `internal/adapters/infra/persistence/postgres/` 配下はすべて **新規ファイル** で、`sqlite/` ディレクトリは無変更（`git status` で確認）
- [ ] すべてのクエリが `$1` 形式のプレースホルダを使い、自動採番 ID は `RETURNING` で取得している
- [ ] すべてのクエリが `r.db` 直接ではなく `executor(ctx, r.db)` 経由で実行されている
- [ ] `go build ./...` が通る

### Step 3: Composition Root で差し替える

**変更箇所は `cmd/bbs/main.go` の数行だけ**です。

```go
// cmd/bbs/main.go
func main() {
    // 旧: SQLite
    // db, _ := sqlite.OpenDB("bbs.db")
    // boardRepo := sqlite.NewBoardRepository(db)
    // threadRepo := sqlite.NewThreadRepository(db)
    // postRepo := sqlite.NewPostRepository(db)
    // tm := sqlite.NewTransactionManager(db)

    // 新: PostgreSQL（ここだけ変更）
    db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
    if err != nil {
        slog.Error("failed to open db", "error", err)
        os.Exit(1)
    }
    boardRepo := postgres.NewBoardRepository(db)
    threadRepo := postgres.NewThreadRepository(db)
    postRepo := postgres.NewPostRepository(db)
    tm := postgres.NewTransactionManager(db)

    // ↓ UseCase の組み立ては一切変わらない！
    listBoards := usecase.NewListBoardsUseCase(boardRepo)
    listThreads := usecase.NewListThreadsUseCase(boardRepo, threadRepo)
    createThread := usecase.NewCreateThreadUseCase(boardRepo, threadRepo, postRepo, tm)
    listPosts := usecase.NewListPostsUseCase(postRepo)
    createPost := usecase.NewCreatePostUseCase(threadRepo, postRepo, tm)

    // Handler の組み立ても変わらない
    boardHandler := handler.NewBoardHandler(listBoards)
    threadHandler := handler.NewThreadHandler(listThreads, createThread)
    postHandler := handler.NewPostHandler(listPosts, createPost)
    router := httpPresentation.NewRouter(boardHandler, threadHandler, postHandler)
    // ...
}
```

### ✅ チェックポイント

- [ ] `cmd/bbs/main.go` で変更したのはリポジトリ生成の数行だけで、その下の UseCase・Handler の組み立ては無変更
- [ ] `postgres` パッケージに配線を替えた状態で `go build -o bbs ./cmd/bbs/` が通る

### Step 4: 動作確認

```bash
# PostgreSQL の起動（Podman を使用）
podman run -d --name bbs-postgres \
  -e POSTGRES_USER=bbs \
  -e POSTGRES_PASSWORD=bbs \
  -e POSTGRES_DB=bbs \
  -p 5432:5432 \
  postgres:16
# Expected output: 64 文字のコンテナ ID。`podman ps` で bbs-postgres が Up になっている

# マイグレーション（DDL は PostgreSQL 用に調整）
# SERIAL, RETURNING などの PostgreSQL 固有構文を使用

# アプリケーションのビルド・起動
export DATABASE_URL="postgres://bbs:bbs@localhost:5432/bbs?sslmode=disable"
go build -o bbs ./cmd/bbs/
# Expected output: （なし — bbs バイナリが作業ディレクトリに作成される）
./bbs
# Expected output: サーバーが起動し、:8080 で待ち受ける

# 動作確認（API は変わらない）
curl http://localhost:8080/api/boards
# Expected output: HTTP 200 で板一覧の JSON 配列が返る。例:
# [{"id":1,"name":"program","description":"Programming General",...}]
curl -X POST http://localhost:8080/api/boards/program/threads \
  -H 'Content-Type: application/json' \
  -d '{"title":"PostgreSQLスレ","author":"pg","body":"移行成功！"}'
# Expected output: 作成されたスレッドが JSON で返る（SQLite 版が同じリクエストに返したのと同じ形）
```

### ✅ チェックポイント

- [ ] `podman ps` で `bbs-postgres` が稼働しており、マイグレーション DDL がエラーなしで適用されている
- [ ] `curl http://localhost:8080/api/boards` が HTTP 200 で板一覧を返す
- [ ] `curl` でスレッド作成が成功し、レスポンスが SQLite 版の同一リクエストと同じ形である
- [ ] `software/assets/bbs` で `go test ./...` が通る

### Step 5: SQLite 版が残っていることを確認

```bash
# SQLite 版もそのまま動く（環境変数で切り替え）
# 先に PostgreSQL 版を停止してから実行する（Ctrl+C）。でないと :8080 が競合する
BBS_DB=/tmp/bbs.db ./bbs
# Expected output: サーバーが起動し、:8080 で待ち受ける（バックエンドは SQLite）
```

SQLite 用のコードは **1行も変更していない** ため、そのまま動作します。

### ✅ チェックポイント

- [ ] `BBS_DB=/tmp/bbs.db ./bbs` で SQLite バックエンドのサーバーが起動し、API が動くことを確認した
- [ ] `git diff` で `internal/adapters/infra/persistence/sqlite/` 配下に **変更がない** ことを確認した

---

## エラー境界の重要性

DB 移行を安全に行うための前提として、Infra Adapter が **DB エラーをドメインエラーに変換** していることが重要です。

```go
// OK: Infra Adapter 内で変換
func (r *BoardRepository) FindByName(ctx context.Context, name string) (*entity.Board, error) {
    // ...
    if errors.Is(err, sql.ErrNoRows) {
        return nil, domain.ErrBoardNotFound  // DB エラー → ドメインエラー
    }
}
```

```go
// NG: UseCase が DB エラーを知ってしまう
func (u *CreateThreadUseCase) Execute(...) {
    board, err := u.boardRepo.FindByName(ctx, name)
    if errors.Is(err, sql.ErrNoRows) {  // ← database/sql に依存！DB変更不可に
        // ...
    }
}
```

この変換を怠ると、UseCase が `database/sql` に依存してしまい、DB 変更ができなくなります。

---

## この実習のポイント

1. **影響範囲の局所化**: DB の変更は Infra 層だけで完結。Domain・UseCase・Presentation は一切触らない。
2. **Interface の役割**: UseCase が依存しているのは Port Interface（抽象）であり、具象実装（SQLite/PostgreSQL）ではない。これが差し替えを可能にしている。
3. **Composition Root が唯一の知識源**: 「どの DB を使うか」を知っているのは `main.go` だけ。各層は自分が何を使っているかを知らない。
4. **並行稼働**: SQLite 版と PostgreSQL 版が同時に存在でき、環境変数やビルドタグで切り替え可能。

---

## 学んだこと

- 永続化層の差し替えで触るのは **Infra Adapter** と **Composition Root の配線** だけ。SQLite と PostgreSQL の違い（`?` と `$1` のプレースホルダ、自動採番 ID の `RETURNING`）はすべて新しい Adapter の中で吸収された。
- 差し替えを安全にするのは **エラー境界**: `sql.ErrNoRows` を Infra 内で `domain.ErrBoardNotFound` に変換しておけば、UseCase はどの DB が動いているかを知る必要がない。
- トランザクションは `context` 経由で伝播する（`context.WithValue` + `executor` ヘルパー）。リポジトリはコード変更なしでトランザクションに参加できる。
- 同じ Port の背後に 2 つの実装が並立するため、ロールバックも `main.go` の 1 行変更で済む。

**次**: [WS6: 外部サービス統合](./clean_arch_ws6_ja.md) — 既存 Adapter の差し替えではなく、新しい Port を定義してそれに対応する Adapter を追加します。

---

## 片付け

```bash
# 先にサーバーを停止して（./bbs を動かしているターミナルで Ctrl+C）、以下を実行:
podman stop bbs-postgres
podman rm bbs-postgres
rm -f bbs /tmp/bbs.db
```

---

## 参考文献

- [PostgreSQL 16 ドキュメント](https://www.postgresql.org/docs/16/)
- [pgx — PostgreSQL Driver and Toolkit](https://github.com/jackc/pgx)
- [Go Databases: リレーショナルデータベースへのアクセス](https://go.dev/doc/database/)
- [Podman — コンテナの実行](https://docs.podman.io/en/latest/Introduction.html)
