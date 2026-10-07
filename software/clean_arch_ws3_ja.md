# クリーンアーキテクチャ実習 (WS3): 通信プロトコルの差し替え

> **⏱️ 所要時間**: 約 90 分
> **🎯 レベル**: Intermediate
> **💻 環境**: Go 1.22+、OS 不問（grpcurl 必須、Step 2 では protoc ツールチェーンを使用）
>
> > **💡 用語集**: [クリーンアーキテクチャ](../infra/glossary_ja.md#architecture)や[インターフェース](../infra/glossary_ja.md#go)などの専門用語は [用語集](../infra/glossary_ja.md) を参照してください。

この実習では、BBS（2 ちゃんねる風掲示板）の REST API を gRPC に移行します。
**Presentation Adapter** を差し替え、それを組み立てる **Composition Root** を更新します。Domain・UseCase・Infrastructure Adapters は変更しません。

## BBS アプリの概要

題材とする BBS は、掲示板（Board）→ スレッド（Thread）→ 投稿（Post）の 3 階層を持つシンプルな掲示板アプリです。REST API の詳細や操作方法は [BBS README](assets/bbs/README_ja.md) を参照してください。

> **本実習の焦点:** 既存の REST API を **gRPC** に移行します。Presentation Adapter（Handler / Router）と Composition Root の配線を変更し、Domain・UseCase・Infrastructure Adapters は変更しません。

### sage と age について

BBS 特有の「スレッドの浮上」仕様です。
API 仕様と `sage` パラメータの詳細は [BBS README](assets/bbs/README_ja.md#レスの投稿) を参照してください。

> **概要**: `age`（上げる）はスレッドを一覧の最上部に浮上させ、`sage`（下げる）は浮上させない仕様です。

---

## BBS プロジェクトの構造と依存関係

BBS は [clean_arch.md](./clean_arch_ja.md) で説明する 3 層バリアント（Adapters / UseCases / Domain）に従います。依存関係は常に内側向きで、Presentation Adapters が UseCases を呼び、Infrastructure Adapters が UseCases / Domain 所有の Port を実装します。Presentation と Infra Adapters が直接依存し合うことはなく、全体像を知って具象実装を組み立てるのは Composition Root（`cmd/bbs/main.go`）だけです。レイヤー図・依存マトリクス・各レイヤーの定義は [レイヤー構造と依存関係](./clean_arch_ja.md#レイヤー構造と依存関係) を、BBS 固有の全体構成図は [BBS README](assets/bbs/README_ja.md#アーキテクチャ構成図) をそれぞれ参照してください。

### BBS プロジェクトの Port 構成

Domain 層には Port がありません。Entity と Domain Error のみで不変条件を強制できるためです。BBS の Port はすべて UseCase 層が所有します。永続化もトランザクション境界も、コアドメイン言語の一部ではなくアプリケーションワークフローを完遂するために必要な機能だからです。配置基準の詳細は [Port 所有権のガイダンス](./clean_arch_ja.md#port-所有権のガイダンス) を参照してください。

| ファイル                          | Interface            | 役割                                             |
| :-------------------------------- | :------------------- | :----------------------------------------------- |
| `internal/usecase/repository.go`  | `BoardRepository`    | 掲示板の永続化（ドメインエンティティの再構成）   |
|                                   | `ThreadRepository`   | スレッドの永続化（ドメインエンティティの再構成） |
|                                   | `PostRepository`     | 投稿の永続化（ドメインエンティティの再構成）     |
| `internal/usecase/transaction.go` | `TransactionManager` | トランザクション境界の制御                       |

### エラー境界

エラーもレイヤー境界を跨ぐ際に変換されます。Infra Adapter はドライバエラー（`sql.ErrNoRows`）をドメインエラー（`domain.ErrThreadNotFound`）に変換し、UseCase はドメインエラーをそのまま上位へ伝播し、Presentation Adapter はトランスポートエラー（HTTP 404 / gRPC `NotFound`）に変換します。ルールの詳細は [エラー境界ルール](./clean_arch_ja.md#エラー境界ルール) を参照してください。

---

## 前提知識

本実習は [BBS プロジェクト](./assets/bbs/) のコードを題材にします。
3 層バリアント（Adapters / UseCases / Domain）を理解していることが前提です — [レイヤー構造と依存関係](./clean_arch_ja.md#レイヤー構造と依存関係) を参照してください。

開始前のチェックリスト:

- [ ] [WS1](./clean_arch_ws1_ja.md) と [WS2](./clean_arch_ws2_ja.md) を完了している（または Port と Composition Root の考え方に慣れている）
- [ ] Go 1.22 以上がインストール済み（`go version` で確認）
- [ ] BBS プロジェクトがビルドできる: `cd software/assets/bbs && go build ./...` が何も出力せずに終了する
- [ ] `grpcurl` をインストール済み（`grpcurl --version`）— [grpcurl releases](https://github.com/fullstorydev/grpcurl/releases) を参照
- [ ] Step 2 のみ: `protoc`、`protoc-gen-go`、`protoc-gen-go-grpc` をインストール済み — [gRPC Go クイックスタート](https://grpc.io/docs/languages/go/quickstart/) を参照

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

### ✅ チェックポイント

- [ ] HTTP Handler 内の「入力変換 → UseCase 呼び出し → 出力変換」の構造を確認できた
- [ ] `Execute` の呼び出し行に HTTP 固有の要素がない（`r.Context()` 以外）ことを確認した
- [ ] `software/assets/bbs` で `go build ./...` が通る（まだ何も変更していない状態）

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

### ✅ チェックポイント

- [ ] `api/proto/bbs.proto` は **新規ファイル** である — 既存ファイルは変更していない（`git status` で新しい .proto だけが見える）
- [ ] コード生成が成功する: `protoc --go_out=. --go-grpc_out=. api/proto/bbs.proto` が何も出力せずに終了する
- [ ] 生成された `*.pb.go` が出力先ディレクトリに存在し、`go build ./...` が通る

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

### ✅ チェックポイント

- [ ] `bbs_server.go` は `internal/adapters/presentation/grpc/` 配下の **新規ファイル** である — Domain・UseCase・Infra は無変更（`git status` で確認）
- [ ] `BBSServer` が具象 UseCase 構造体ではなく `usecase.ThreadCreator` / `usecase.ThreadLister` という **インターフェース** に依存している
- [ ] エラー変換がここだけで完結している: `domain.ErrBoardNotFound → codes.NotFound`、未知のエラー → 汎用メッセージで `codes.Internal`
- [ ] `go build ./...` が通る

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

### ✅ チェックポイント

- [ ] `main.go` で変更したのはサーバー起動ブロックだけ — DI の組み立て（リポジトリ・UseCase）は無変更
- [ ] 切り替え後も `go build -o bbs ./cmd/bbs/` が通る

### Step 5: 動作確認

```bash
# ビルド
go build -o bbs ./cmd/bbs/
# Expected output: （なし — bbs バイナリが作業ディレクトリに作成される）

# 起動（別ターミナルで起動したままにする）
./bbs
# Expected output: サーバーが起動し、:9090 で待ち受ける

# gRPC クライアントで確認（grpcurl を使用）
# reflection.Register を有効にしているため -plaintext だけでサービス一覧を表示可能
grpcurl -plaintext localhost:9090 bbs.BBSService/ListBoards
# Expected output: 初期登録済みの板一覧が JSON で返る。例:
# {
#   "board": [
#     { "id": "1", "name": "programming", "description": "Programming General" },
#     ...
#   ]
# }

grpcurl -plaintext -d '{"board_name":"program","title":"Go言語スレ","author":"gopher","body":"Go最高！"}' \
    localhost:9090 bbs.BBSService/CreateThread
# Expected output: 作成された "thread"（postCount: "1"）と最初の "post"（number: "1"）
# を含む JSON が返る
```

### ✅ チェックポイント

- [ ] `grpcurl -plaintext localhost:9090 bbs.BBSService/ListBoards` で板一覧が JSON で返る
- [ ] gRPC 経由の `CreateThread` でスレッドが作成され、スレッドと最初の投稿の両方が返る
- [ ] Presentation Adapter と `main.go` 以外のファイルが変わっていないことを確認した: `git diff --stat` に presentation/grpc 配下・生成 .pb.go・`cmd/bbs/main.go` しか表示されない
- [ ] `software/assets/bbs` で `go test ./...` が通る

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

---

## 学んだこと

- 通信プロトコルの差し替えで触るのは **Presentation Adapter** と **Composition Root の配線** だけ。HTTP と gRPC のハンドラで `Execute(ctx, Input) (Output, error)` の呼び出し行は完全に同一だった。
- プロトコルの違いは純粋な **変換** の違い: 入力は JSON か protobuf か、エラーは HTTP ステータスか gRPC ステータスコードか。
- エラーは層の境界ごとに翻訳される: ドライバエラー → ドメインエラー（Infra）、ドメインエラー → トランスポートエラー（Presentation）。
- protobuf 生成の構造体は Domain 層に入れない。Presentation アダプタが UseCase の DTO と相互変換する。

**次**: [WS4: ビジネスルールの追加](./clean_arch_ws4_ja.md) — 「スレ主のみ書き込み可能」ルールを追加し、変更が Domain から外側へ波及する様子を体験します。

---

## 片付け

```bash
# gRPC サーバーを停止して（ターミナルで Ctrl+C）、以下を実行:
cd software/assets/bbs
make clean   # bbs バイナリと bbs.db を削除
```

手動で行う場合:

```bash
rm -f software/assets/bbs/bbs software/assets/bbs/bbs.db
```

---

## 参考文献

- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers Language Specification](https://protobuf.dev/programming-guides/proto3/)
- [grpcurl — GitHub](https://github.com/fullstorydev/grpcurl)
- [gRPC Go Quick Start](https://grpc.io/docs/languages/go/quickstart/)
