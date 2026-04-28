# クリーンアーキテクチャ実習 (WS3): 通信プロトコルの差し替え

この実習では、BBS（2 ちゃんねる風掲示板）の REST API を gRPC に移行します。
**Framework 層だけを変更**し、Domain・UseCase・Infra が一切変更不要であることを体験します。

## BBS アプリの概要

題材とする BBS は、掲示板（Board）→ スレッド（Thread）→ 投稿（Post）の 3 階層を持つシンプルな掲示板アプリです。

| リソース | 説明 |
| -------- | ---- |
| **Board** | 掲示板。`name`（例: `programming`）で識別 |
| **Thread** | スレッド。特定の Board に属し、`threadID`（数値）で識別 |
| **Post** | 投稿（レス）。特定の Thread に属し、通し番号を持つ。`sage` フラグでスレッドを浮上させない |

### REST API 一覧

現在の HTTP エンドポイントは以下の 5 つです。

| メソッド | パス | 内容 |
| -------- | ---- | ---- |
| GET | `/api/boards` | 掲示板一覧 |
| GET | `/api/boards/{name}/threads` | スレッド一覧 |
| POST | `/api/boards/{name}/threads` | スレッド作成 |
| GET | `/api/threads/{threadID}/posts` | 投稿一覧 |
| POST | `/api/threads/{threadID}/posts` | レス投稿 |

### curl での使用例

```bash
# 掲示板一覧
curl localhost:8080/api/boards

# スレッド一覧
curl localhost:8080/api/boards/programming/threads

# スレッド作成
curl -X POST localhost:8080/api/boards/programming/threads \
  -H 'Content-Type: application/json' \
  -d '{"title":"テスト","author":"anonymous","body":"最初の投稿"}'

# 投稿一覧
curl localhost:8080/api/threads/1/posts

# レス投稿（sage 付き）
curl -X POST localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"名無し","body":"sage","sage":true}'
```

### 一連の操作例（スレ立て → レス）

```bash
# 1. 掲示板を確認
curl localhost:8080/api/boards
# → {"boards":[{"id":1,"name":"programming","name":"Programming General","created_at":"2025-01-01T00:00:00Z"}]}

# 2. programming 板にスレッドを立てる
curl -s -X POST localhost:8080/api/boards/programming/threads \
  -H 'Content-Type: application/json' \
  -d '{"title":"Go言語スレ","author":"gopher","body":"Go最高！"}' | jq .
# → { "thread": { "id": 1, "title": "Go言語スレ", ... }, "post": { "id": 1, "number": 1, ... } }

# 3. レスを投稿する（2 で返った thread.id を使う）
curl -s -X POST localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"名無し","body":"確かに"}' | jq .
# → { "id": 2, "number": 2, "author": "名無し", "body": "確かに", ... }

# 4. sage 付きでレスを投稿
curl -s -X POST localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"sage","body":"sage","sage":true}' | jq .
# → { "id": 3, "number": 3, "sage": true, ... }

# 5. 投稿一覧で結果を確認
curl -s localhost:8080/api/threads/1/posts | jq .
# → [ { "number": 1, "author": "gopher", "body": "Go最高！" },
#     { "number": 2, "author": "名無し", "body": "確かに" },
#     { "number": 3, "author": "sage", "body": "sage", "sage": true } ]
```

### sage と age について

2 ちゃんねる風掲示板特有の「スレッドの浮上」仕様です。

| 用語 | 意味 | 挙動 |
| :--- | :--- | :--- |
| **age** | 上げる | `LastPostedAt` を更新し、一覧の最上部に浮上させる |
| **sage** | 下げる | `LastPostedAt` を更新せず、スレッドの位置を変えない |

**デフォルト動作:**

- `sage: false` または省略 → **age**（スレッドが浮上）
- `sage: true` → **sage**（浮上しない）

**スレッド一覧の並び順:**

```sql
ORDER BY last_posted_at DESC  -- 最近レスがあったスレが上に来る
```

**具体例:**

```bash
# 1. Go スレ、Python スレ、Ruby スレが作成された状態
#    一覧: Ruby(10:00) → Python(10:05) → Go(10:10)

# 2. Ruby スレに age レス（sage: false）
curl -X POST localhost:8080/api/threads/3/posts -d '{"author":"A","body":"hello"}'
#    Ruby の LastPostedAt が現在時刻に更新 → 先頭へ
#    一覧: Ruby(10:15) ← 浮上！ → Go(10:10) → Python(10:05)

# 3. Python スレに sage レス（sage: true）
curl -X POST localhost:8080/api/threads/2/posts -d '{"author":"B","body":"sage","sage":true}'
#    Python の LastPostedAt は古いまま → 順位変わらず
#    一覧: Ruby(10:15) → Go(10:10) → Python(10:05) ← そのまま
```

**使い分けの目安:**

- **age**: 「会話が続いています」を目立たせたいとき
- **sage**: 補足回答や、古いスレッドを無理に目立たせたくないとき

**実装ロジック:**

```go
// Thread.Bump() - スレッドの最終投稿日時を更新
func (t *Thread) Bump(postedAt time.Time, sage bool) {
    t.PostCount++
    if !sage {  // sage=false の場合のみ更新
        t.LastPostedAt = postedAt
    }
}
```

---

## BBS プロジェクトの構造と依存関係

### Domain Interface（Port）の一覧

Clean Architecture の中核となる「抽象」がここにあります。

| ファイル | Interface | 役割 |
| :--- | :--- | :--- |
| `internal/domain/port/repository.go` | `BoardRepository` | 掲示板の永続化 |
| | `ThreadRepository` | スレッドの永続化 |
| | `PostRepository` | 投稿の永続化 |
| `internal/domain/port/transaction.go` | `TransactionManager` | トランザクション境界の制御 |

```go
// internal/domain/port/repository.go
type BoardRepository interface {
    FindAll(ctx context.Context) ([]*entity.Board, error)
    FindByName(ctx context.Context, name string) (*entity.Board, error)
    Save(ctx context.Context, board *entity.Board) error
}
```

### 完全な依存関係図

```text
                   ┌─────────────────────────────────────────┐
                   │  cmd/bbs/main.go (Composition Root)     │
                   │  ─────────────────────────────────────  │
                   │  func main() {                          │
                   │      // 1. Infra を構築                 │
                   │      boardRepo := sqlite.NewBoardRepo() │
                   │      // 2. UseCase を構築               │
                   │      listThreads := usecase.New(...)    │
                   │         └── boardRepo を注入            │
                   │      // 3. Framework を構築             │
                   │      handler := NewHandler(listThreads) │
                   │         └── listThreads を注入          │
                   │      // 4. サーバー起動                 │
                   │      http.Serve(router)                 │
                   │  }                                      │
                   └─────────────────────────────────────────┘
                              │
                 ┌────────────┼────────────┐
                 │            │            │
                 ↓            ↓            ↓
        ┌────────────┐  ┌────────────┐  ┌────────────┐
        │   Infra    │  │  UseCase   │  │ Framework  │
        │  Adapter   │  │   Layer    │  │   Layer    │
        └──────┬─────┘  └──────┬─────┘  └──────┬─────┘
               │               │               │
               │               │               │
┌──────────────┴───────────────┴───────────────┴─────────────────────┐
│                         INTERNAL/DOMAIN                            │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  internal/domain/entity/board.go, thread.go, post.go         │  │
│  │  ─────────────────────────────────────────────────────────   │  │
│  │  type Board struct { ID, Name, Description, CreatedAt }      │  │
│  │  type Thread struct { ... }                                  │  │
│  │  type Post struct { ... }                                    │  │
│  │                                                              │  │
│  │  // Domain Logic の例                                        │  │
│  │  func (t *Thread) Bump(postedAt time.Time, sage bool) {      │  │
│  │      t.PostCount++                                           │  │
│  │      if !sage { t.LastPostedAt = postedAt }                  │  │
│  │  }                                                           │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  internal/domain/port/repository.go  ← Domain Interface      │  │
│  │  type BoardRepository interface { ... }                      │  │
│  └──────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────┘
               ↑ implements
┌──────────────┴───────────────┐
│  internal/infra/persistence/ │
│  sqlite/board_repo.go        │  ← Infra Adapter（具象実装）
│  func (r *BoardRepository)   │     (Domain Interface を実装)
│  FindByName(...) {           │
│     SELECT ... WHERE name = ?│
│  }                           │
└──────────────────────────────┘
```

### 各レイヤーの依存方向

```text
           ┌─────────────┐
           │  Framework  │  ← 最外側（HTTP/gRPC/CLI）
           └──────┬──────┘
                  │ depends on
                  ↓
           ┌─────────────┐
           │   UseCase   │  ← アプリケーションロジック
           └──────┬──────┘
                  │ depends on
                  ↓
           ┌────────────────────────────────────┐
           │         Domain Layer               │
           │  Entity + Port Interface (抽象)    │  ← 最内側
           └────────────────────────────────────┘
                  ↑ implements
           ┌─────────────┐
           │   Infra     │  ← 最外側（DB/外部API）
           │   Adapter   │     (Domain Interface を実装)
           └─────────────┘
```

### エラー境界の流れ

Clean Architecture では、エラーもレイヤー境界を跨ぐ際に変換されます。

```text
┌─────────────────────────────────────────────────────────────────────┐
│ Infra Adapter Layer                                                │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ func (r *ThreadRepo) FindByID(...) {                         │  │
│  │     err := db.QueryRow(...)                                  │  │
│  │     if errors.Is(err, sql.ErrNoRows) {                       │  │
│  │         return nil, domain.ErrThreadNotFound  // ← 変換     │  │
│  │     }                                                        │  │
│  │ }                                                            │  │
│  └──────────────────────────────────────────────────────────────┘  │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ returns domain error
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ UseCase Layer                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ func (u *CreatePostUseCase) Execute(...) {                  │  │
│  │     thread, err := u.threadRepo.FindByID(...)               │  │
│  │     // err は domain.ErrThreadNotFound などのドメインエラー  │  │
│  │     return nil, err  // ← そのまま上位へ                    │  │
│  │ }                                                            │  │
│  └──────────────────────────────────────────────────────────────┘  │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ returns domain error
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ Framework Layer (HTTP)                                              │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ if errors.Is(err, domain.ErrThreadNotFound) {                │  │
│  │     writeError(w, http.StatusNotFound, err.Error())  // ← 変換│  │
│  │ }                                                            │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

**重要なポイント**:

- **Infra Adapter**: ドライバエラー（`sql.ErrNoRows`）をドメインエラー（`domain.ErrThreadNotFound`）に変換
- **UseCase**: ドメインエラーをそのまま上位へ（技術詳細を知らない）
- **Framework**: ドメインエラーをトランスポートエラー（HTTP 404, gRPC NotFound）に変換

このように各レイヤーがエラー変換の責務を持つことで、上位のレイヤーは下位の技術詳細を知る必要がなくなります。

### キーポイント

1. **UseCase は具象実装を知らない**: `sqlite.BoardRepository` ではなく `port.BoardRepository`（Interface）に依存
2. **Infra は Domain Interface を実装する**: `internal/infra/persistence/sqlite/` が `internal/domain/port/` の Interface を実装
3. **Framework と Infra は直接関係しない**: どちらも「最外側」だが、UseCase を介して間接的に繋がる
4. **Composition Root（main.go）だけが全体を知っている**: どの Infra 実装を使うか、どの Framework を使うかをここで決定

```go
// UseCase は Interface に依存（具象を知らない）
type ListThreadsUseCase struct {
    boardRepo port.BoardRepository  // ← sqlite.* ではなく port.*
}

// Infra は Interface を実装
type BoardRepository struct {
    db *sql.DB
}

func (r *BoardRepository) FindByName(...) {  // ← Interface を実装
    // SQLite 固有の SQL
}
```

---

## 前提知識

本実習は [BBS プロジェクト](./assets/bbs/) のコードを題材にします。
以下の 4 層構造を理解していることが前提です。

```text
Framework (HTTP/gRPC)  →  UseCase (アプリケーション手順)  →  Domain (ビジネスルール)
                                                          →  Port Interface (抽象)
                             ↑                                   ↑
                         Infra Adapter (DB) ─────────────────────┘  (DIP: 具象が抽象に依存)
```

## 実習のシナリオ

「REST API を gRPC に移行したい」という要件に対応します。

---

## 課題: HTTP → gRPC 移行

### 変更範囲の確認（やってはいけないこと）

以下の層は **1行も変更しません**。

| 層 | 理由 |
| ---- | ------ |
| **Domain** | Entity（Board, Thread, Post）、Port Interface は通信プロトコルに依存しないため |
| **UseCase** | `Execute(ctx, Input) (Output, error)` のシグネチャが不変。DTO もプロトコル非依存 |
| **Infra** | Repository 実装（SQLクエリ、エラー変換）は通信方式と無関係 |

### Step 1: 現在の HTTP Handler を確認する

Framework 層の現状を確認します。Handler は「入力変換 → UseCase 呼び出し → 出力変換」の構造です。

```go
// internal/framework/http/handler/thread_handler.go
func (h *ThreadHandler) CreateThread(w http.ResponseWriter, r *http.Request) {
    // HTTP 固有の入力変換
    name := r.PathValue("name")
    var req createThreadRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid json")
        return
    }

    // UseCase 呼び出し（← 通信方式に依存しない）
    out, err := h.createThread.Execute(r.Context(), usecase.CreateThreadInput{
        BoardName: name, Title: req.Title, Author: req.Author, Body: req.Body,
    })

    // HTTP 固有の出力変換
    if errors.Is(err, domain.ErrBoardNotFound) {
        writeError(w, http.StatusNotFound, err.Error())
        return
    }
    writeJSON(w, http.StatusCreated, out)
}
```

**確認ポイント**: `Execute` の呼び出し部分は、HTTP にも gRPC にも依存していません。

### Step 2: Protobuf 定義を作成する

gRPC 用の型定義を作ります。これは新規ファイルであり、既存コードの変更ではありません。

```protobuf
// api/proto/bbs.proto
syntax = "proto3";
package bbs;

service BBSService {
    rpc ListBoards(ListBoardsRequest) returns (ListBoardsResponse);
    rpc ListThreads(ListThreadsRequest) returns (ListThreadsResponse);
    rpc CreateThread(CreateThreadRequest) returns (CreateThreadResponse);
    rpc ListPosts(ListPostsRequest) returns (ListPostsResponse);
    rpc CreatePost(CreatePostRequest) returns (CreatePostResponse);
}

message CreateThreadRequest {
    string board_name = 1;
    string title = 2;
    string author = 3;
    string body = 4;
}

message CreateThreadResponse {
    Thread thread = 1;
    Post post = 2;
}

message Thread {
    int64 id = 1;
    int64 board_id = 2;
    string title = 3;
    int32 post_count = 4;
    string created_at = 5;
    string last_posted_at = 6;
}

message Post {
    int64 id = 1;
    int64 thread_id = 2;
    int32 number = 3;
    string author = 4;
    string body = 5;
    bool sage = 6;
    string created_at = 7;
}
```

### Step 3: gRPC Handler を実装する

新しいファイルに gRPC 用のハンドラを作ります。**UseCase の呼び出し方が HTTP 版と同一**であることを確認してください。

```go
// internal/framework/grpc/bbs_server.go（新規ファイル）
type BBSServer struct {
    pb.UnimplementedBBSServiceServer
    createThread *usecase.CreateThreadUseCase
    listThreads  *usecase.ListThreadsUseCase
    // ...
}

func (s *BBSServer) CreateThread(ctx context.Context, req *pb.CreateThreadRequest) (*pb.CreateThreadResponse, error) {
    // gRPC 固有の入力変換（ここだけが違う）
    out, err := s.createThread.Execute(ctx, usecase.CreateThreadInput{
        BoardName: req.BoardName,
        Title:     req.Title,
        Author:    req.Author,
        Body:      req.Body,
    })

    // gRPC 固有の出力変換（HTTP status → gRPC status）
    if errors.Is(err, domain.ErrBoardNotFound) {
        return nil, status.Errorf(codes.NotFound, err.Error())
    }
    if err != nil {
        // ドメインエラーでない未知のエラーは、詳細を隠蔽して Internal Server Error に
        return nil, status.Errorf(codes.Internal, "internal server error")
    }

    return &pb.CreateThreadResponse{
        Thread: toProtoThread(out.Thread),
        Post:   toProtoPost(out.Post),
    }, nil
}
```

**比較**: HTTP 版と gRPC 版の `Execute` 呼び出し行を並べてみてください。

```go
// HTTP 版
out, err := h.createThread.Execute(r.Context(), usecase.CreateThreadInput{
    BoardName: name, Title: req.Title, Author: req.Author, Body: req.Body,
})

// gRPC 版（全く同じ！）
out, err := s.createThread.Execute(ctx, usecase.CreateThreadInput{
    BoardName: req.BoardName, Title: req.Title, Author: req.Author, Body: req.Body,
})
```

UseCase の呼び出しが完全に同一です。変わるのは変換先（JSON → protobuf）とエラー変換先（HTTP status → gRPC status）だけです。

### Step 4: エントリポイントを gRPC に切り替える

Composition Root で、HTTP から gRPC へ起動方法を変えます。

```go
// cmd/bbs/main.go
func main() {
    // DI の組み立て（UseCase への注入）は変わらない
    boardRepo := sqlite.NewBoardRepository(db)
    threadRepo := sqlite.NewThreadRepository(db)
    postRepo := sqlite.NewPostRepository(db)
    tm := sqlite.NewTransactionManager(db)

    createThread := usecase.NewCreateThreadUseCase(boardRepo, threadRepo, postRepo, tm)
    listThreads := usecase.NewListThreadsUseCase(boardRepo, threadRepo)
    // ...

    // 旧: HTTP サーバー起動
    // threadHandler := handler.NewThreadHandler(listThreads, createThread)
    // http.ListenAndServe(":8080", router)

    // 新: gRPC サーバー起動（ここだけ変更）
    grpcServer := grpc.NewServer()
    reflection.Register(grpcServer)  // grpcurl でサービス一覧を表示するために必要
    bbsServer := framework.NewBBSServer(createThread, listThreads, ...)
    pb.RegisterBBSServiceServer(grpcServer, bbsServer)

    lis, err := net.Listen("tcp", ":9090")
    if err != nil {
        slog.Error("failed to listen", "error", err)
        os.Exit(1)
    }
    if err := grpcServer.Serve(lis); err != nil {
        slog.Error("grpc server failed", "error", err)
        os.Exit(1)
    }
}
```

### Step 5: 動作確認

```bash
# ビルド
go build -o bbs ./cmd/bbs/

# 起動
./bbs

# gRPC クライアントで確認（grpcurl を使用）
# reflection.Register を有効にしているため -plaintext だけでサービス一覧を表示可能
grpcurl -plaintext localhost:9090 bbs.BBSService/ListBoards

grpcurl -plaintext -d '{"board_name":"program","title":"Go言語スレ","author":"gopher","body":"Go最高！"}' \
    localhost:9090 bbs.BBSService/CreateThread
```

---

## レイヤー分離がない場合との比較

```go
// 全てが1関数に混在している例
func CreateThread(w http.ResponseWriter, r *http.Request) {
    name := r.PathValue("name")           // HTTP 依存
    db, _ := sql.Open("sqlite3", "bbs.db") // DB 依存
    tx, _ := db.Begin()                    // 技術詳細
    res, _ := tx.Exec("INSERT INTO ...")   // SQL
    w.WriteHeader(201)                     // HTTP 依存
    json.NewEncoder(w).Encode(res)         // JSON 依存
}
```

この場合、gRPC に変えるには **関数全体を書き直す** 必要があります。

---

## この実習のポイント

1. **影響範囲の局所化**: Framework 層だけで完結。UseCase の `Execute` 呼び出しは一切変わらない。
2. **プロトコルの違いは「変換」の違い**: HTTP も gRPC も「入力を DTO に詰め替えて UseCase を呼ぶ」構造は同じ。
3. **差し替えの容易さ**: 本番は gRPC、社内ツル用は HTTP、テスト用は CLI —— どれも UseCase を共有可能。
