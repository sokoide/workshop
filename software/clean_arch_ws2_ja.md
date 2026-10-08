# クリーンアーキテクチャ実習 (WS2): 拡張と最適化

> **⏱️ 所要時間**: 約 60 分
> **🎯 レベル**: Intermediate
> **💻 環境**: Go 1.23+、OS 不問（外部依存ゼロ — 通知とキャッシュはスタンドインを使用）
>
> > **💡 用語集**: [インターフェース](../infra/glossary_ja.md#go)や[キャッシュ](../infra/glossary_ja.md#cache)、[クリーンアーキテクチャ](../infra/glossary_ja.md#architecture)などの専門用語は [用語集](../infra/glossary_ja.md) を参照してください。

WS1 に続き、さらに実践的なシナリオを通じて、副作用の制御とパフォーマンス最適化におけるレイヤー分離の恩恵を学びます。

## ゴール

この実習を終えると、次のことができるようになります:

- 新しい副作用（通知）を UseCase が所有する Port として追加し、実装の差し替え（Email → Slack）を Composition Root の 1 行変更で行える
- Decorator パターンで透過的キャッシュを導入し、キャッシュの存在を Domain と UseCase に気づかせないままにできる

## 実習のシナリオ

1. **通知チャネルの多角化**: 通知手段を Email から Slack へ、ビジネスロジックを変えずに変更します。
2. **透過的キャッシュの導入**: ビジネスロジックを一切汚さずに、インフラ層の差し替えだけでキャッシュ（ここでは in-memory スタンドイン、本番では Redis）を導入します。

## 前提知識

- [ ] [WS1](./clean_arch_ws1_ja.md) を完了している（Entity / Domain Service / リポジトリの差し替え）— この実習は WS1 の完成コードを直接土台にします
- [ ] Go 1.23 以上がインストール済み（`go version` で確認）

### ワークスペースの準備

WS2 は WS1 の完成コード（ベテラン判定 + SQL/LDAP リポジトリ）を出発点とします。以下のどちらかの方法で進めてください:

```bash
# 方法 A: WS1 の参照実装を出発点としてコピーする
cp -r software/assets/clean_arch_ws1 ~/ws2 && cd ~/ws2

# 方法 B: 完成版の WS2 参照実装をコピーして中身を読む
cp -r software/assets/clean_arch_ws2 ~/ws2 && cd ~/ws2
```

> 以下のコード断片はモジュールパス `github.com/sokoide/workshop/software/assets/clean_arch_ws2` を使います。方法 A で WS1 のコピーから始めた場合は、`github.com/sokoide/workshop/software/assets/clean_arch_ws1`（または `go mod init` に渡した自分のモジュールパス）に読み替えてください。
>
> Redis 等の外部サービスは不要です: 通知実装はメッセージを表示し、キャッシュはメモリ上にエントリを保持します。本番で Redis に繋ぐ方法は、課題 2 末尾の補足を参照してください。

---

## 課題 1: 通知システムの抽象化

「メンバーシップが承認されたら通知を送る」というユースケースを考えます。

### 1-1. UseCases 層でのインターフェース定義

「どう送るか」ではなく「通知を送るという機能」を定義します。通知はアプリケーションワークフローに必要な道具であるため、ポートは UseCases 層に所属します。

```go
// internal/usecase/port/notification.go
// Package port holds the output ports owned by the UseCase layer.
package port

import "context"

// NotificationGateway is the port for sending notifications. The UseCase
// layer owns it because notification is a tool for the application workflow,
// not part of the core domain language. "How" to send (Email, Slack, ...) is
// an infrastructure detail implemented in the Infra Adapters layer.
type NotificationGateway interface {
	Send(ctx context.Context, message string) error
}
```

### 1-2. UseCase での利用

UseCase は、通知が Email なのか Slack なのかを気にしません。

```go
// internal/usecase/approve_membership.go
package usecase

import (
	"context"
	"fmt"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/domain"
	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/usecase/port"
)

// ApproveUseCase approves a membership and notifies the user.
type ApproveUseCase struct {
	repo     domain.UserRepository
	notifier port.NotificationGateway
}

func NewApproveUseCase(repo domain.UserRepository, notifier port.NotificationGateway) *ApproveUseCase {
	return &ApproveUseCase{repo: repo, notifier: notifier}
}

func (uc *ApproveUseCase) Execute(ctx context.Context, userID string) error {
	user, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user %s: %w", userID, err)
	}
	// The UseCase does not care whether this goes out as Email or Slack.
	message := fmt.Sprintf("Membership approved: welcome, %s!", user.Name)
	return uc.notifier.Send(ctx, message)
}
```

### 1-3. Infra 層での実装差し替え

両実装ともスタンドインです — SMTP や webhook を呼ぶ代わりにメッセージを表示するため、実習は外部依存ゼロで動きます。差し替えは `main` での配線で行います。

```go
// internal/adapters/infra/email_notifier.go
package infra

import (
	"context"
	"fmt"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/usecase/port"
)

// EmailNotifier is a stand-in for an SMTP client: it prints the message
// instead of sending mail, so the workshop runs with zero external
// dependencies.
type EmailNotifier struct {
	from string
}

func NewEmailNotifier(from string) *EmailNotifier {
	return &EmailNotifier{from: from}
}

// Compile-time check that EmailNotifier satisfies the UseCase-owned port.
var _ port.NotificationGateway = (*EmailNotifier)(nil)

func (n *EmailNotifier) Send(ctx context.Context, message string) error {
	fmt.Printf("[email from %s] %s\n", n.from, message)
	return nil
}
```

```go
// internal/adapters/infra/slack_notifier.go (後から追加)
package infra

import (
	"context"
	"fmt"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/usecase/port"
)

// SlackNotifier is a stand-in for the Slack API: it prints the message
// instead of calling a webhook, so the workshop runs with zero external
// dependencies.
type SlackNotifier struct {
	webhookURL string
}

func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{webhookURL: webhookURL}
}

// Compile-time check that SlackNotifier satisfies the same port.
var _ port.NotificationGateway = (*SlackNotifier)(nil)

func (n *SlackNotifier) Send(ctx context.Context, message string) error {
	fmt.Printf("[slack webhook %s] %s\n", n.webhookURL, message)
	return nil
}
```

`main.go` がポートと具象実装を配線します — Email から Slack への差し替えは 1 行変更だけです:

```go
// main.go (抜粋)
func approveVia(ctx context.Context, notifier port.NotificationGateway) {
	uc := usecase.NewApproveUseCase(infra.NewSQLUserRepository(), notifier)
	if err := uc.Execute(ctx, "alice"); err != nil {
		fmt.Printf("approve: error: %v\n", err)
	}
}

func main() {
	ctx := context.Background()
	fmt.Println("== Exercise 1: notification channel swap ==")
	approveVia(ctx, infra.NewEmailNotifier("noreply@example.com"))
	// 1 行で差し替え: Email → Slack。他は何も変わらない。
	approveVia(ctx, infra.NewSlackNotifier("https://hooks.slack.com/services/T000/B000/XXXX"))
}
```

```bash
go run .
# Expected output:
# == Exercise 1: notification channel swap ==
# [email from noreply@example.com] Membership approved: welcome, Alice!
# [slack webhook https://hooks.slack.com/services/T000/B000/XXXX] Membership approved: welcome, Alice!
```

### ✅ チェックポイント

- [ ] `NotificationGateway` が `usecase`（port）パッケージに定義されている — `domain` や `infra` ではない
- [ ] `ApproveUseCase` が `infra` パッケージを **一切 import せずに** コンパイルできる
- [ ] Email から Slack への差し替えが `main.go` の **1 行変更だけ** で完了する
- [ ] モック/フェイクの `NotificationGateway` を使い、Slack に一切接続せずに「Send が呼ばれたこと」だけを検証するテストを書ける

---

## 課題 2: 透過的キャッシュの追加 (Decorator パターン)

「ユーザー情報取得を高速化したい。ただし、既存の `FindByID` を呼び出している箇所（UseCase など）は 1 行も直したくない」という要件に対応します。

### 2-1. キャッシュストアとキャッシュ用リポジトリの実装

まず、Decorator が使うキャッシュ DB を定義します。これは Infra 層内部の抽象化であり、TTL 対応の in-memory 実装を用意します。本番では同じ 2 メソッドのインターフェースを Redis で実装します（下記の補足を参照）。

```go
// internal/adapters/infra/cache_store.go
package infra

import (
	"context"
	"errors"
	"sync"
	"time"
)

// errCacheMiss is returned by CacheStore.Get when the key is absent or
// expired. It never leaves the Infra layer: the decorator treats every cache
// error as a miss and falls back to the real repository.
var errCacheMiss = errors.New("cache miss")

// CacheStore is the key-value store used by CachedUserRepository. It is an
// infra-internal abstraction: in production you can implement it with Redis
// (e.g. github.com/redis/go-redis/v9) without touching any other layer.
type CacheStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// MemoryCacheStore is an in-memory CacheStore with TTL support. It plays the
// role of the cache database in this workshop.
type MemoryCacheStore struct {
	mu      sync.Mutex
	entries map[string]memoryCacheEntry
	now     func() time.Time // injectable clock for tests
}

type memoryCacheEntry struct {
	value     []byte
	expiresAt time.Time
}

func NewMemoryCacheStore() *MemoryCacheStore {
	return &MemoryCacheStore{
		entries: map[string]memoryCacheEntry{},
		now:     time.Now,
	}
}

func (s *MemoryCacheStore) Get(ctx context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[key]
	if !ok || !s.now().Before(entry.expiresAt) {
		// Absent or expired — indistinguishable from the caller's viewpoint.
		// Drop the stale entry so the map does not grow without bound
		// (a real cache like Redis evicts expired keys itself).
		delete(s.entries, key)
		return nil, errCacheMiss
	}
	return entry.value, nil
}

func (s *MemoryCacheStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = memoryCacheEntry{value: value, expiresAt: s.now().Add(ttl)}
	return nil
}
```

次に、元の `SQLUserRepository` をラップする新しい Infra 実装を作ります。

```go
// internal/adapters/infra/cached_user_repository.go
package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/domain"
)

// userCacheTTL is how long a cached user entry stays fresh.
const userCacheTTL = 10 * time.Minute

// CachedUserRepository decorates a real domain.UserRepository with a cache
// (Decorator pattern). It satisfies the same port, so neither the UseCase
// layer nor the wrapped repository knows that the cache exists.
type CachedUserRepository struct {
	origin domain.UserRepository // Real repository (DB, LDAP, ...)
	cache  CacheStore            // Cache database (in-memory here, Redis in production)
}

func NewCachedUserRepository(origin domain.UserRepository, cache CacheStore) *CachedUserRepository {
	return &CachedUserRepository{origin: origin, cache: cache}
}

// Compile-time check that the decorator satisfies the same port as the origin.
var _ domain.UserRepository = (*CachedUserRepository)(nil)

func (r *CachedUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	// 1. Return if in cache
	if user, err := r.getFromCache(ctx, id); err == nil {
		return user, nil
	}

	// 2. Otherwise, ask the real repository
	user, err := r.origin.FindByID(ctx, id)
	if err == nil {
		if cacheErr := r.saveToCache(ctx, user); cacheErr != nil {
			slog.Warn("failed to save user to cache", "error", cacheErr) // Log cache failures and proceed
		}
	}
	return user, err
}

// Cache helpers — implementation details of this Decorator.
// They are NOT part of the UserRepository interface, so neither the
// UseCase layer nor the wrapped repository knows about the cache store.

func (r *CachedUserRepository) getFromCache(ctx context.Context, id string) (*domain.User, error) {
	data, err := r.cache.Get(ctx, "user:"+id)
	if err != nil {
		return nil, err // Cache miss or cache failure → fall back to the real repository
	}
	var user domain.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("unmarshal cached user %s: %w", id, err)
	}
	return &user, nil
}

func (r *CachedUserRepository) saveToCache(ctx context.Context, user *domain.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	// Set a TTL so stale cache entries expire automatically
	return r.cache.Set(ctx, "user:"+user.ID, data, userCacheTTL)
}
```

> **💡 ロギングに関する補足（教育的簡略化）:**
> 本実習ではコードの簡略化のため、キャッシュ処理の警告ログ出力に標準ライブラリの `log/slog` を直接呼び出しています。実際のプロダクションコードでは、テストの容易性やロギングライブラリの抽象化のために、ロガーをインターフェース経由で注入（DI）するか、あるいはデコレーター等で横断的関心事（Cross-Cutting Concerns）として分離する設計が推奨されます。

### 2-2. DI コンテナでの構成変更

`main.go` で、本物のリポジトリをキャッシュ用リポジトリで「包んで」から UseCase に渡します。効果が見えるように、WS1 の SQL スタンドインには「これまでに発行したクエリ数」を数える `QueryCount()` メソッドを追加してあります:

```go
// main.go — 課題 2 の部分（課題 1 の配線の後に追加）
fmt.Println("\n== Exercise 2: transparent cache ==")
realRepo := infra.NewSQLUserRepository()
cachedRepo := infra.NewCachedUserRepository(realRepo, infra.NewMemoryCacheStore())

// The UseCase sees only domain.UserRepository — it cannot tell whether
// the repository it received is cached or not.
uc := usecase.NewCheckVeteranUseCase(cachedRepo, domain.VeteranService{})

for lookup := 1; lookup <= 3; lookup++ {
	veteran, err := uc.Execute(ctx, "alice")
	if err != nil {
		fmt.Printf("lookup #%d: error: %v\n", lookup, err)
		continue
	}
	fmt.Printf("lookup #%d: alice veteran=%v (SQL queries so far: %d)\n",
		lookup, veteran, realRepo.QueryCount())
}
```

```bash
go run .
# Expected output:
# == Exercise 1: notification channel swap ==
# [email from noreply@example.com] Membership approved: welcome, Alice!
# [slack webhook https://hooks.slack.com/services/T000/B000/XXXX] Membership approved: welcome, Alice!
#
# == Exercise 2: transparent cache ==
# lookup #1: alice veteran=true (SQL queries so far: 1)
# lookup #2: alice veteran=true (SQL queries so far: 1)
# lookup #3: alice veteran=true (SQL queries so far: 1)
```

検索 #2 と #3 はキャッシュから返されます: SQL クエリ数は 1 のままで、UseCase はそのことにまったく気づいていません。

### ✅ チェックポイント

- [ ] `CachedUserRepository` が `domain.UserRepository` を満たしている — `var _ domain.UserRepository = (*infra.CachedUserRepository)(nil)` がコンパイルできる
- [ ] キャッシュのロジックが `internal/adapters/infra/` に **のみ** 存在する — `git grep -i cache internal/domain internal/usecase` で何も見つからない
- [ ] キャッシュエントリが失効しても読み取りは成功する（Decorator が本物のリポジトリにフォールバックする）。キャッシュ保存の失敗は警告ログのみで処理が続行する
- [ ] Decorator を配線した状態で `go test ./...` が通る

> **💡 キャッシュを本番へ:** `MemoryCacheStore` はこの実習のためのものです。本番では、Redis クライアント（例: `github.com/redis/go-redis/v9` — セットアップは [Redis 実習](../infra/cache_ja.md) を参照）で `CacheStore` を実装し、それを `NewCachedUserRepository` に渡します。インターフェースが Infra 層内部のものであるため、この差し替えで変更されるのは **1 ファイルだけ** です — WS1 で体験した耐障害性が、もう一段深い層でも働きます。

---

## この実習のポイント

1. **副作用の抽象化 (課題 1)**:
   - 「通知を送る」という行為自体がアプリケーション要件であり、「Slack で送る」のは単なる手段（インフラの詳細）です。
   - `NotificationGateway` インターフェースは UseCase Port（Domain Port ではない）です。通知はコアドメイン言語の一部ではなく、アプリケーションワークフローの道具だからです。
   - 詳細を外に追い出すことで、テスト時に実際の通知を飛ばさず「通知が呼ばれたこと」だけを検証するのが容易になります。
2. **透過的な機能追加 (課題 2)**:
   - インターフェースが同じであれば、中身が「DB から取るもの」から「キャッシュ制御付きで取るもの」に変わっても、呼び出し元は一切気づきません。
   - これにより、ビジネスロジックの健全性を保ったまま、インフラ面での改善（パフォーマンス対策、ロギング、リトライ処理の追加など）を自由に行えます。
3. **ポートと実装の分離**:
   - 通知インターフェースは UseCase が所有する出力ポートであり、Slack/Email 実装は Infra Adapters (インフラアダプター層) に置きます。
   - キャッシュストアのような詳細は Infra Adapters (インフラアダプター層) に閉じ込めます。

---

## 理解度チェック

以下の問いに自分で答えてみてください。

### 問 1: Decorator パターンの限界

`CachedUserRepository` は「キャッシュにあれば返す、なければ DB に問い合わせる」というシンプルな戦略です。もし「キャッシュの有効期限（TTL）が切れたら非同期で DB を更新する（Write-Behind）」という戦略に変えた場合、`CachedUserRepository` の実装はどのように変わりますか？また、この変更は UseCase 層に影響しますか？

### 問 2: 通知の失敗

`NotificationGateway.Send` がエラーを返した場合、UseCase はどうすべきですか？「通知失敗＝ユースケース失敗」とすべきか、それとも「通知はベストエフォートで、ユースケースは継続すべき」か、どのような基準で判断しますか？

### 問 3: キャッシュと整合性

`CachedUserRepository` を使うと、DB のデータが更新されてもキャッシュは古い（stale）ままになる可能性があります。この問題に対して、クリーンアーキテクチャ的にはどの層でどのような対策を取るのが適切ですか？

---

## 学んだこと

- アプリケーション要件である副作用（通知）は **UseCase 所有の出力ポート** にし、手段（Slack/Email）は Infra Adapters 層に置くことで、`main.go` の 1 行変更で差し替えられる。
- Decorator パターンを使うと、キャッシュ・ロギング・リトライといった横断的な振る舞いを **透過的に** 追加できる。インターフェースが変わらない限り、Domain と UseCase は一切気づかない。
- キャッシュの失敗はあえて致命的にしない: ログ出力と本来のデータ源へのフォールバックにより、ビジネスロジックの健全性を保つ。
- ポートの所属は「能力を必要とする層」で決める: ドメインポリシーは Domain Port、アプリケーションのオーケストレーション（通知）は UseCase Port。ユーザーの永続化はリポジトリ契約がこのドメインの言語の一部であるため、Domain Port（`domain.UserRepository`）のままです — 基準は [Port 所有権のガイダンス](./clean_arch_ja.md#port-所有権のガイダンス) を参照してください。
- 外部サービス（Slack や Redis）は、Infra 層内部のインターフェースの背後にスタンドインとして置き換えられる。実習は外部依存ゼロで動きながら、本物との差は 1 ファイルに保てる。

**次**: [WS3: 通信プロトコルの差し替え](./clean_arch_ws3_ja.md) — BBS の REST API を Domain・UseCase・Infra に触れずに gRPC へ移行します。

---

## 片付け

この実習ではコンテナも外部サービスも使いません（通知とキャッシュは代替実装です）。作業ディレクトリが不要になったら削除してください:

```bash
rm -rf ~/ws2
```

---

## 参考文献

- [The Clean Architecture — Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Redis Documentation](https://redis.io/docs/latest/)
- [go-redis Documentation](https://redis.uptrace.dev/)
- [log/slog package — Go Documentation](https://pkg.go.dev/log/slog)
