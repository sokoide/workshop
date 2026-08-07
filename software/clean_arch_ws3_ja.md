# クリーンアーキテクチャ実習 (WS3): 通信プロトコルの差し替え

この実習では、BBS（2 ちゃんねる風掲示板）の REST API を gRPC に移行します。
**Presentation Adapter** を差し替え、それを組み立てる **Composition Root** を更新します。Domain・UseCase・Infrastructure Adapters は変更しません。

## BBS アプリの概要

題材とする BBS は、掲示板（Board）→ スレッド（Thread）→ 投稿（Post）の 3 階層を持つシンプルな掲示板アプリです。REST API の詳細や操作方法は [BBS README](assets/bbs/README_ja.md) を参照してください。

> **本実習の焦点:** 既存の REST API を **gRPC** に移行します。Presentation Adapter（Handler / Router）と Composition Root の配線を変更し、Domain・UseCase・Infrastructure Adapters は変更しません。

### sage と age について

BBS 特有の「スレッドの浮上」仕様です。
詳細な仕様、具体例、実装ロジックは [BBS README](assets/bbs/README_ja.md#sage-と-age-について) を参照してください。

> **概要**: `age`（上げる）はスレッドを一覧の最上部に浮上させ、`sage`（下げる）は浮上させない仕様です。

---

## BBS プロジェクトの構造と依存関係

### Interface（Port）の一覧と配置基準

クリーンアーキテクチャの中核となる「抽象」（Port）の一覧です。
インターフェースを除去したとき、ドメインモデルが不変条件（ビジネスルール）を強制できなくなるものは **Domain Port**（Domain 層帰属）とし、そうでないものは **UseCase Port**（UseCase 層帰属）とします（P1 配置基準）。

#### Domain Port（Domain 層帰属）

これらはドメインモデルの不変条件の強制や、エンティティ・集約の再構成に不可欠なため、Domain 層に定義します。

| ファイル                             | Interface          | 役割                                             |
| :----------------------------------- | :----------------- | :----------------------------------------------- |
| `internal/domain/port/repository.go` | `BoardRepository`  | 掲示板の永続化（ドメインエンティティの再構成）   |
|                                      | `ThreadRepository` | スレッドの永続化（ドメインエンティティの再構成） |
|                                      | `PostRepository`   | 投稿の永続化（ドメインエンティティの再構成）     |

#### UseCase Port（UseCase 層帰属）

トランザクション境界の制御は、ビジネスルールそのものではなく、ユースケースのオーケストレーション（アプリケーション制御）の関心事であるため、UseCase 層に定義します。

| ファイル                          | Interface            | 役割                       |
| :-------------------------------- | :------------------- | :------------------------- |
| `internal/usecase/transaction.go` | `TransactionManager` | トランザクション境界の制御 |

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
                   ┌───────────────────────────────────────────┐
                   │  cmd/bbs/main.go (Composition Root)       │
                   │  ───────────────────────────────────────  │
                   │  func main() {                            │
                   │      // 1. Infra を構築                   │
                   │      boardRepo := sqlite.NewBoardRepo()   │
                   │      // 2. UseCase を構築                 │
                   │      listThreads := usecase.New(...)      │
                   │         └── boardRepo を注入              │
                   │      // 3. Presentation を構築            │
                   │      handler := NewHandler(listThreads)   │
                   │         └── listThreads を注入            │
                   │      // 4. サーバー起動                   │
                   │      http.Serve(router)                   │
                   │  }                                        │
                   └───────────────────────────────────────────┘
                              │
                 ┌────────────┼────────────┐
                 │            │            │
                 ↓            ↓            ↓
        ┌────────────┐  ┌────────────┐  ┌─────────────┐
        │   Infra    │  │  UseCase   │  │ Presentation│
        │  Adapter   │  │   Layer    │  │    Layer    │
        └──────┬─────┘  └──────┬─────┘  └──────┬──────┘
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
│  internal/adapters/infra/persistence/ │
│  sqlite/board_repo.go        │  ← Infra Adapter（具象実装）
│  func (r *BoardRepository)   │     (Domain Interface を実装)
│  FindByName(...) {           │
│     SELECT ... WHERE name = ?│
│  }                           │
└──────────────────────────────┘
```

### 各レイヤーの依存方向

```text
           ┌─────────────────────────────────────┐
           │         Adapters 層                 │
           │  ┌────────────┐ ┌────────────────┐  │
           │  │Presentation│ │ Infrastructure │  │
           │  │ (入力側)   │ │  (出力側)      │  │
           │  │HTTP/gRPC/  │ │  DB/外部API    │  │
           │  │  CLI       │ │                │  │
           │  └──────┬─────┘ └────────────────┘  │
           └─────────┼───────────────────────────┘
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
           Infrastructure Adapters（Domain/UseCase Port を実装）
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
│ Presentation Layer (HTTP)                                              │
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
- **Presentation**: ドメインエラーをトランスポートエラー（HTTP 404, gRPC NotFound）に変換

このように各レイヤーがエラー変換の責務を持つことで、上位のレイヤーは下位の技術詳細を知る必要がなくなります。

### キーポイント

1. **UseCase は具象実装を知らない**: `sqlite.BoardRepository` ではなく `port.BoardRepository`（Interface）に依存
2. **Infra は Domain Interface を実装する**: `internal/adapters/infra/persistence/sqlite/` が `internal/domain/port/` の Interface を実装
3. **Presentation と Infra Adapters は直接関係しない**: どちらも Adapters 層の一部だが方向が異なり、UseCase を介して間接的に繋がる
4. **Composition Root（main.go）だけが全体を知っている**: どの Infra 実装を使うか、どの Presentation を使うかをここで決定

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
以下の 3 層バリアント（Adapters / UseCases / Domain）を理解していることが前提です。

```text
Presentation Adapters ──→ UseCases ──→ Domain
                            ↑              ↑
                        Infrastructure Adapters
        implements ports owned by UseCases or Domain
```

## 実習のシナリオ

「REST API を gRPC に移行したい」という要件に対応します。

---

## 課題: HTTP → gRPC 移行

### 変更範囲の確認（やってはいけないこと）

以下の層は **1行も変更しません**。

| 層          | 理由                                                                             |
| ----------- | -------------------------------------------------------------------------------- |
| **Domain**  | Entity（Board, Thread, Post）と Domain Error は通信プロトコルに依存しないため    |
| **UseCase** | `Execute(ctx, Input) (Output, error)` のシグネチャが不変。DTO もプロトコル非依存 |
| **Infra**   | Repository 実装（SQLクエリ、エラー変換）は通信方式と無関係                       |

### Step 1: 現在の HTTP Handler を確認する

Adapters (Presentation) 層の現状を確認します。Handler は「入力変換 → UseCase 呼び出し → 出力変換」の構造です。

```go
// internal/adapters/presentation/http/handler/thread_handler.go
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
    if err != nil {
        // ドメインエラーでない未知のエラーは、詳細を隠蔽して Internal Server Error に
        writeError(w, http.StatusInternalServerError, "internal server error")
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

import "google/protobuf/timestamp.proto";

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
    google.protobuf.Timestamp created_at = 5;
    google.protobuf.Timestamp last_posted_at = 6;
}

message Post {
    int64 id = 1;
    int64 thread_id = 2;
    int32 number = 3;
    string author = 4;
    string body = 5;
    bool sage = 6;
    google.protobuf.Timestamp created_at = 7;
}
```

### Step 3: gRPC Handler を実装する

新しいファイルに gRPC 用のハンドラを作ります。**UseCase の呼び出し方が HTTP 版と同一**であることを確認してください。

まず、UseCases 層に Input Port interface を定義します（まだ定義されていない場合）:

```go
// internal/usecase/port.go
package usecase

import "context"

type ThreadCreator interface {
    Execute(ctx context.Context, in CreateThreadInput) (*CreateThreadOutput, error)
}

type ThreadLister interface {
    Execute(ctx context.Context, in ListThreadsInput) (*ListThreadsOutput, error)
}

// 他の UseCase についても同様に interface を定義
```

次に、これらの interface に依存する gRPC Handler を実装します:

```go
// internal/adapters/presentation/grpc/bbs_server.go（新規ファイル）
type BBSServer struct {
    pb.UnimplementedBBSServiceServer
    createThread usecase.ThreadCreator  // ← interface（Input Port）
    listThreads  usecase.ThreadLister   // ← interface（Input Port）
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
    bbsServer := presentation.NewBBSServer(createThread, listThreads, ...)  // 具象型は interface を満たす
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

1. **影響範囲の局所化**: Presentation Adapter と Composition Root の配線だけで完結。UseCase の `Execute` 呼び出しは一切変わらない。
2. **プロトコルの違いは「変換」の違い**: HTTP も gRPC も「入力を DTO に詰め替えて UseCase を呼ぶ」構造は同じ。
3. **差し替えの容易さ**: 本番は gRPC、社内ツール用は HTTP、テスト用は CLI —— どれも UseCase を共有可能。

---

## 理解度チェック

以下の問いに自分で答えてみてください。

### 問 1: Protobuf と Domain 層

.proto ファイルから生成された Go の struct（例：`pb.Post`）を、Domain 層のエンティティとして直接使用するとどのような問題が生じますか？クリーンアーキテクチャ的には、どの層でどのように変換すべきですか？

### 問 2: エラーハンドリングの違い

HTTP では `404 Not Found` を返すのが自然ですが、gRPC では `status.NotFound` を使います。エラーの変換はどの層で行うべきでしょうか？もし UseCase 層で「HTTP 用」「gRPC 用」の分岐を書くと、どのような問題が起きますか？

### 問 3: トランザクション境界

`CreateThreadUseCase` は「スレッド作成」と「最初の投稿」を同じトランザクションで行います。このトランザクション制御はどの層に属するべきですか？もし Presentation 層（HTTP Handler）でトランザクションを開始・終了すると、クリーンアーキテクチャのどの原則に違反しますか？
