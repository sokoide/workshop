# クリーンアーキテクチャ実習 (WS6): 外部サービス統合

> **⏱️ 所要時間**: 約 45 分
> **🎯 レベル**: Intermediate
> **💻 環境**: Go 1.22+。Slack Incoming Webhook URL は任意（なければ NoOp Gateway で進められます）
>
> > **💡 用語集**: [クリーンアーキテクチャ](../infra/glossary_ja.md#architecture)や[インターフェース](../infra/glossary_ja.md#go)などの専門用語は [用語集](../infra/glossary_ja.md) を参照してください。

この実習では、BBS（2 ちゃんねる風掲示板）に「新規投稿時に Slack 通知を送る」機能を追加します。
**新しい Port を定義して Infra Adapter を増やすパターン**を体験し、既存コードへの影響が最小限であることを確認します。

## 前提知識

本実習は [BBS プロジェクト](./assets/bbs/) のコードを題材にします。
共有のアーキテクチャ解説（3 層バリアント・依存ルール・エラー境界）は [clean_arch.md](./clean_arch_ja.md#レイヤー構造と依存関係) にまとめてあります。

開始前のチェックリスト:

- [ ] [WS5](./clean_arch_ws5_ja.md) を完了している — 本実習は BBS プロジェクトの WS5 完了時点の状態から始めます
- [ ] Go 1.22 以上がインストール済み（`go version` で確認）
- [ ] BBS プロジェクトがビルドできる: `cd software/assets/bbs && go build ./...` が何も出力せずに終了する
- [ ] （任意）最終確認用の Slack Incoming Webhook URL — なければ後述の NoOp Gateway を使用する

## 実習のシナリオ

「スレに新しい投稿があったら Slack に通知する」という機能を追加します。
これは新しい Port を定義し、Infra Adapter を増やすパターンの典型的な例です。

---

## 課題: Slack 通知の追加

### 変更範囲の全体像

| 層               | 変更内容                                                                                                                           | 役割                                                        |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------- |
| **Domain**       | **変更なし**                                                                                                                       | —                                                           |
| **UseCase**      | `internal/usecase/notification.go` に `NotificationGateway` interface を新規定義、`CreatePostUseCase` に注入、投稿成功後に呼び出し | 「通知が必要である」という抽象の定義 + 通知のタイミング制御 |
| **Infra**        | `infra/notification/slack_gateway.go` を新規作成                                                                                   | Slack API の具体実装                                        |
| **Presentation** | **変更なし**                                                                                                                       | —                                                           |
| **Entity**       | **変更なし**                                                                                                                       | —                                                           |

### Step 1: UseCases 層 — 新しい Port の定義

「どう通知するか」ではなく「通知を送るという機能」を抽象として定義します。通知はアプリケーションワークフローに必要な道具であるため、ポートは UseCases 層に所属します。

```go
// internal/usecase/notification.go（新規ファイル）
package usecase

import "context"

// NotificationGateway は、通知送信の抽象インターフェースです。
// Slack、Email、LINE など、具体的な通知手段は UseCase 層では知りません。
type NotificationGateway interface {
    NotifyNewPost(ctx context.Context, threadTitle string, postAuthor string, postBody string) error
}
```

新規ファイルは `internal/usecase/` の直下、`repository.go` や `transaction.go` と同じ位置に置きます。他の Port と同様、UseCase 層が所有します。

**確認ポイント**: この interface には Slack という言葉が一切出てきません。UseCase は「何を通知するか」を決め、「どう通知するか」は Infra に任せます。

### ✅ チェックポイント

- [ ] `internal/usecase/notification.go` は **新規ファイル** で、既存ファイルは変更していない（`git status` で確認）
- [ ] interface 名とシグネチャに Slack 固有の要素がない（URL も Slack 型も出てこない）
- [ ] `go build ./...` が通る

### Step 2: UseCase 層 — 通知の呼び出し

`CreatePostUseCase` に通知ゲートウェイを注入し、投稿成功後に呼び出します。

```go
// internal/usecase/post_usecase.go（既存ファイルを一部変更）
// ※ `"log/slog"` を import に追加してください
type CreatePostUseCase struct {
    threadRepo ThreadRepository
    postRepo   PostRepository
    tm         TransactionManager
    notifier   NotificationGateway  // ← 追加: 通知ゲートウェイ
}

// コンストラクタ引数に notifier を追加
func NewCreatePostUseCase(
    threadRepo ThreadRepository,
    postRepo PostRepository,
    tm TransactionManager,
    notifier NotificationGateway,  // ← 追加
) *CreatePostUseCase {
    return &CreatePostUseCase{
        threadRepo: threadRepo,
        postRepo:   postRepo,
        tm:         tm,
        notifier:   notifier,
    }
}

func (u *CreatePostUseCase) Execute(ctx context.Context, in CreatePostInput) (*CreatePostOutput, error) {
    var (
        out    *CreatePostOutput
        thread *entity.Thread
        post   *entity.Post
    )
    // ──────────────────────────────────────────────
    // トランザクション境界: DB操作はすべて内部で実行
    // ──────────────────────────────────────────────
    if err := u.tm.RunInTransaction(ctx, func(txCtx context.Context) error {
        var err error
        thread, err = u.threadRepo.FindByID(txCtx, in.ThreadID)
        if err != nil {
            return err
        }
        if thread == nil {
            return domain.ErrThreadNotFound
        }
        // WS4 で追加した CanPost チェックはそのまま残る（省略せず維持する）
        if !thread.CanPost(in.Author) {
            return domain.ErrNotThreadOwner
        }
        count, err := u.postRepo.CountByThreadID(txCtx, thread.ID)
        if err != nil {
            return err
        }
        post, err = entity.NewPost(thread.ID, count+1, in.Author, in.Body, in.Sage)
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

    // ──────────────────────────────────────────────
    // トランザクション外: 通知
    // 通知失敗で投稿を巻き戻さない設計判断
    // ──────────────────────────────────────────────
    if err := u.notifier.NotifyNewPost(ctx, thread.Title, post.Author, post.Body); err != nil {
        slog.Warn("notification failed", "error", err)  // ログだけ出す、投稿は成功させる
    }

    return out, nil
}
```

**確認ポイント**: 通知はトランザクションの **外** で呼び出しています。Slack 通知の失敗で投稿が巻き戻るべきではないという設計判断です。エラーはログに記録しつつ、投稿処理自体は成功として扱います。

### ✅ チェックポイント

- [ ] `CreatePostUseCase` が `notifier NotificationGateway` フィールドを持ち、コンストラクタで注入されている
- [ ] `NotifyNewPost` は `RunInTransaction` が `nil` を返した **後** に呼ばれ、そのエラーは伝播させずログ出力（`slog.Warn`）のみ
- [ ] WS4 で追加した `CanPost` チェックがトランザクション内にそのまま残っている
- [ ] `go build ./...` が通る

### Step 3: Infra 層 — Slack 実装

新しいディレクトリに Slack 用の Gateway を作ります。

```go
// internal/adapters/infra/notification/slack_gateway.go（新規ファイル）
package notification

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type SlackGateway struct {
    webhookURL string
    client     *http.Client
}

func NewSlackGateway(webhookURL string) *SlackGateway {
    return &SlackGateway{
        webhookURL: webhookURL,
        client:     &http.Client{},
    }
}

func (g *SlackGateway) NotifyNewPost(ctx context.Context, threadTitle, author, body string) error {
    payload := map[string]string{
        "text": fmt.Sprintf("[%s] %s: %s", threadTitle, author, body),
    }
    jsonPayload, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("marshal slack payload: %w", err)
    }
    req, err := http.NewRequestWithContext(ctx, "POST", g.webhookURL, bytes.NewReader(jsonPayload))
    if err != nil {
        return fmt.Errorf("create slack request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := g.client.Do(req)
    if err != nil {
        return fmt.Errorf("send slack notification: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("slack returned status %d", resp.StatusCode)
    }
    return nil
}
```

### ✅ チェックポイント

- [ ] `SlackGateway` は `internal/adapters/infra/notification/` 配下の **新規ファイル** で、Domain・UseCase（他ファイル）・Presentation は無変更
- [ ] `SlackGateway` が `usecase` パッケージを import せずに `NotificationGateway` interface を満たしている
- [ ] HTTP リクエストが `http.NewRequestWithContext` を使い、キャンセルが伝播する
- [ ] `go build ./...` が通る

### Step 4: Composition Root — DI に追加

`main.go` で Slack Gateway を生成し、UseCase に注入します。

```go
// cmd/bbs/main.go
func main() {
    // ...既存の DI 組み立て

    // ↓ 追加: 通知ゲートウェイ
    notifier := notification.NewSlackGateway(os.Getenv("SLACK_WEBHOOK_URL"))

    // UseCase のコンストラクタ引数に notifier を追加
    createPost := usecase.NewCreatePostUseCase(threadRepo, postRepo, tm, notifier)
    // 他の UseCase は変更不要（通知不要なため）
}
```

### ✅ チェックポイント

- [ ] このステップで変更したのは `main.go` だけで、notifier を生成している箇所が 1 か所のみ
- [ ] `go build -o bbs ./cmd/bbs/` が通る

### Step 5: 動作確認

```bash
# Slack Webhook URL を設定
# （ない場合は main.go で NoOpGateway に配線する — 「通知が不要な場面での対応」を参照）
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/T.../B.../xxx"

# ビルド・起動
go build -o bbs ./cmd/bbs/
# Expected output: （なし — bbs バイナリが作業ディレクトリに作成される）
./bbs
# Expected output: サーバーが起動し、:8080 で待ち受ける

# 投稿 → Slack に通知が飛ぶ
curl -X POST http://localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"gopher","body":"Slack通知テスト"}'
# Expected output: HTTP 201 で作成された投稿が JSON で返る
# （本実習前に投稿したときと同じレスポンス — 通知はバックグラウンドで行われる）
```

### ✅ チェックポイント

- [ ] `curl` での投稿が、本実習前に投稿したときと同じレスポンスを返す
- [ ] Slack チャンネルにメッセージが届く（`NoOpGateway` の場合は何も飛ばず投稿だけが成功する）
- [ ] Webhook URL を一時的に壊す（存在しないホストにする等）と、警告ログ `notification failed` が出つつ投稿は HTTP 201 で成功する
- [ ] `software/assets/bbs` で `go test ./...` が通る

---

## 通知先の差し替えが容易

Slack → Email に変更する場合、Composition Root だけの変更で済みます。

```go
// internal/adapters/infra/notification/email_gateway.go（新規ファイル）
type EmailGateway struct {
    smtpHost string
    from     string
    to       string
}

func (g *EmailGateway) NotifyNewPost(ctx context.Context, threadTitle, author, body string) error {
    // SMTP でメール送信
    return nil
}
```

```go
// cmd/bbs/main.go — 差し替え（1行だけ）
notifier := notification.NewEmailGateway("smtp.example.com:587", "bbs@example.com", "admin@example.com")
```

UseCase、Domain、Presentation は **一切変更不要** です。

---

## 通知が不要な場面での対応

テストや CLI 版など、通知が不要な場面では NoOp（何もしない）実装を注入します。

```go
// internal/adapters/infra/notification/noop_gateway.go
type NoOpGateway struct{}

func (n *NoOpGateway) NotifyNewPost(ctx context.Context, threadTitle, author, body string) error {
    return nil  // 何もしない
}
```

```go
// cmd/bbs/main.go — 通知不要な場面
notifier := notification.NewNoOpGateway()
createPost := usecase.NewCreatePostUseCase(threadRepo, postRepo, tm, notifier)
```

---

## テストでの利点

通知をモックすることで、UseCase のテストで実際の Slack にメッセージを飛ばさずに検証できます。

```go
// internal/usecase/create_post_test.go
type mockNotifier struct {
    called bool
    thread string
    author string
    body   string
}

func (m *mockNotifier) NotifyNewPost(ctx context.Context, threadTitle, author, body string) error {
    m.called = true
    m.thread = threadTitle
    m.author = author
    m.body = body
    return nil
}

func TestCreatePost_SendsNotification(t *testing.T) {
    notifier := &mockNotifier{}
    uc := usecase.NewCreatePostUseCase(mockThreadRepo, mockPostRepo, mockTM, notifier)

    uc.Execute(ctx, input)

    if !notifier.called {
        t.Error("notification should have been sent")
    }
    if notifier.thread != "テストスレ" {
        t.Errorf("thread title = %q, want %q", notifier.thread, "テストスレ")
    }
}
```

---

## レイヤー分離がない場合との比較

```go
// NG: 通知処理が Handler に直接埋まっている
func CreatePost(w http.ResponseWriter, r *http.Request) {
    // ...DB に投稿保存
    // Slack 通知をハードコード
    http.Post("https://hooks.slack.com/...", "application/json", bytes.NewReader(payload))
    w.WriteHeader(201)
}
```

この場合:

- Slack → Email に変えるのに **Handler を書き直す** 必要がある
- 通知失敗時のリトライやエラーハンドリングも Handler に混ざる
- テスト時に毎回 Slack に通知が飛ぶ（モックできない）

---

## この実習のポイント

1. **新規 Port/Adapter パターン**: 新機能は新しい Port（interface）を定義し、対応する Adapter を増やすことで追加できる。既存コードへの影響は最小限。
2. **責務の明確化**:
   - UseCase は「何を通知するか」を定義（`NotificationGateway` のシグネチャ）— 通知はコアドメイン言語の一部ではなくアプリケーションワークフローの道具であるため、UseCase Port に分類される
   - UseCase は「いつ通知するか」を決定（投稿成功後）
   - Infra は「どう通知するか」を実装（Slack Webhook / Email SMTP）
3. **テスト容易性**: interface によって通知をモックでき、外部サービスに依存しないテストが書ける。
4. **通知先の差し替え**: Slack → Email → LINE の変更は Composition Root の 1 行変更だけで完了。

---

## 学んだこと

- 新しい外部サービス統合とは、**UseCases 層に新しい Port を定義して Adapter を 1 本追加すること** — Domain と Presentation は一切触らない。
- Port のシグネチャはワークフローが必要とする「**何を**」（`NotifyNewPost`）だけを表し、「どう届けるか」は含まない — "Slack" という語は UseCase 層を超えなかった。
- 本操作を巻き戻してはならない副作用呼び出しは**トランザクションの外**で実行し、失敗時はリクエストを失敗させずログ記録のみで優雅に劣化させる。
- 同じ Port の NoOp 実装やモックにより、外部依存のないテストと CLI シナリオが書ける。
- 通知先の差し替え（Slack → Email）は Composition Root の 1 行変更で完了する。

**次のステップ**: [WS7: 認証の追加](./clean_arch_ws7_ja.md) — 「コアに知らせない」同じ考えを、Presentation 層のミドルウェアに適用します。

---

## 片付け

```bash
# まずサーバーを停止（./bbs を実行中のターミナルで Ctrl+C）してから:
unset SLACK_WEBHOOK_URL
# WS5 の PostgreSQL コンテナを起動したまま不要になった場合:
podman stop bbs-postgres && podman rm bbs-postgres
rm -f bbs
```

---

## 参考文献

- [Slack API: Incoming Webhooks](https://api.slack.com/messaging/webhooks)
- [Go net/http ドキュメント](https://pkg.go.dev/net/http)
- [Go Log/slog（構造化ログ）](https://go.dev/blog/slog)
