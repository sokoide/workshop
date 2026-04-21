# クリーンアーキテクチャ実習 (WS3): 通信プロトコルの差し替え

この実習では、BBS（2ちゃんねる風掲示板）の REST API を gRPC に移行します。
**Framework 層だけを変更**し、Domain・UseCase・Infra が一切変更不要であることを体験します。

## 前提知識

本実習は [BBS プロジェクト](./assets/bbs/) のコードを題材にします。
以下の4層構造を理解していることが前提です。

```
Framework (HTTP/gRPC)  →  UseCase (アプリケーション手順)  →  Domain (ビジネスルール)
                              ↓                                   ↑
                         Infra Adapter (DB) ─────────────────────┘
```

## 実習のシナリオ

「REST API を gRPC に移行したい」という要件に対応します。

---

## 課題: HTTP → gRPC 移行

### 変更範囲の確認（やってはいけないこと）

以下の層は **1行も変更しません**。

| 層 | 理由 |
|----|------|
| **Domain** | Entity（Board, Thread, Post）、Port Interface は通信プロトコルに依存しないため |
| **UseCase** | `Execute(ctx, Input) (Output, error)` のシグネチャが不変。DTO もプロトコル非依存 |
| **Infra** | Repository 実装（SQLクエリ）は通信方式と無関係 |

### Step 1: 現在の HTTP Handler を確認する

Framework 層の現状を確認します。Handler は「入力変換 → UseCase 呼び出し → 出力変換」の構造です。

```go
// framework/handler/thread_handler.go
func (h *ThreadHandler) CreateThread(w http.ResponseWriter, r *http.Request) {
    // HTTP 固有の入力変換
    slug := r.PathValue("slug")
    var req createThreadRequest
    json.NewDecoder(r.Body).Decode(&req)

    // UseCase 呼び出し（← 通信方式に依存しない）
    out, err := h.createThread.Execute(r.Context(), usecase.CreateThreadInput{
        BoardSlug: slug, Title: req.Title, Author: req.Author, Body: req.Body,
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
    string board_slug = 1;
    string title = 2;
    string author = 3;
    string body = 4;
}

message CreateThreadResponse {
    Thread thread = 1;
}

message Thread {
    int64 id = 1;
    string title = 2;
    string author = 3;
    int32 response_count = 4;
    string created_at = 5;
    string last_posted_at = 6;
}
```

### Step 3: gRPC Handler を実装する

新しいファイルに gRPC 用のハンドラを作ります。**UseCase の呼び出し方が HTTP 版と同一**であることを確認してください。

```go
// framework/grpc/bbs_server.go（新規ファイル）
type BBSServer struct {
    pb.UnimplementedBBSServiceServer
    createThread usecase.CreateThreadUseCase
    listThreads  usecase.ListThreadsUseCase
    // ...
}

func (s *BBSServer) CreateThread(ctx context.Context, req *pb.CreateThreadRequest) (*pb.CreateThreadResponse, error) {
    // gRPC 固有の入力変換（ここだけが違う）
    out, err := s.createThread.Execute(ctx, usecase.CreateThreadInput{
        BoardSlug: req.BoardSlug,
        Title:     req.Title,
        Author:    req.Author,
        Body:      req.Body,
    })

    // gRPC 固有の出力変換（HTTP status → gRPC status）
    if errors.Is(err, domain.ErrBoardNotFound) {
        return nil, status.Errorf(codes.NotFound, err.Error())
    }
    if err != nil {
        return nil, status.Errorf(codes.Internal, err.Error())
    }

    return &pb.CreateThreadResponse{Thread: toProtoThread(out)}, nil
}
```

**比較**: HTTP 版と gRPC 版の `Execute` 呼び出し行を並べてみてください。

```go
// HTTP 版
out, err := h.createThread.Execute(r.Context(), usecase.CreateThreadInput{
    BoardSlug: slug, Title: req.Title, Author: req.Author, Body: req.Body,
})

// gRPC 版（全く同じ！）
out, err := s.createThread.Execute(ctx, usecase.CreateThreadInput{
    BoardSlug: req.BoardSlug, Title: req.Title, Author: req.Author, Body: req.Body,
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

    createThread := usecase.NewCreateThreadUseCase(boardRepo, threadRepo, tm)
    listThreads := usecase.NewListThreadsUseCase(threadRepo)
    // ...

    // 旧: HTTP サーバー起動
    // handler := framework.NewThreadHandler(createThread, listThreads)
    // http.ListenAndServe(":8080", router)

    // 新: gRPC サーバー起動（ここだけ変更）
    grpcServer := grpc.NewServer()
    bbsServer := framework.NewBBSServer(createThread, listThreads, ...)
    pb.RegisterBBSServiceServer(grpcServer, bbsServer)

    lis, _ := net.Listen("tcp", ":9090")
    grpcServer.Serve(lis)
}
```

### Step 5: 動作確認

```bash
# ビルド
go build -o bbs ./cmd/bbs/

# 起動
./bbs

# gRPC クライアントで確認（grpcurl を使用）
grpcurl -plaintext localhost:9090 bbs.BBSService/ListBoards

grpcurl -plaintext -d '{"board_slug":"program","title":"Go言語スレ","author":"gopher","body":"Go最高！"}' \
    localhost:9090 bbs.BBSService/CreateThread
```

---

## レイヤー分離がない場合との比較

```go
// 全てが1関数に混在している例
func CreateThread(w http.ResponseWriter, r *http.Request) {
    slug := r.PathValue("slug")           // HTTP 依存
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
