# クリーンアーキテクチャ実習 (WS2): 拡張と最適化

> **⏱️ 所要時間**: 約 60 分
> **🎯 レベル**: Intermediate
> **💻 環境**: Go 1.22+、OS 不問（課題 2 でのみ Redis を使用）
>
> > **💡 用語集**: [インターフェース](../infra/glossary_ja.md#go)や[キャッシュ](../infra/glossary_ja.md#cache)、[クリーンアーキテクチャ](../infra/glossary_ja.md#architecture)などの専門用語は [用語集](../infra/glossary_ja.md) を参照してください。

WS1 に続き、さらに実践的なシナリオを通じて、副作用の制御とパフォーマンス最適化におけるレイヤー分離の恩恵を学びます。

## 実習のシナリオ

1. **通知チャネルの多角化**: 通知手段を Email から Slack へ、ビジネスロジックを変えずに変更します。
2. **透過的キャッシュの導入**: ビジネスロジックを一切汚さずに、インフラ層の差し替えだけで Redis キャッシュを導入します。

## 前提知識

- [ ] [WS1](./clean_arch_ws1_ja.md) を完了している（Entity / Domain Service / リポジトリの差し替え）
- [ ] Go 1.22 以上がインストール済み（`go version` で確認）
- [ ] 課題 2 のみ: 起動済みの Redis（セットアップは [Redis 実習](../infra/cache_ja.md) を参照）と `go.mod` への `github.com/redis/go-redis/v9` 追加

---

## 課題 1: 通知システムの抽象化

「メンバーシップが承認されたら通知を送る」というユースケースを考えます。

### 1-1. UseCases 層でのインターフェース定義

「どう送るか」ではなく「通知を送るという機能」を定義します。通知はアプリケーションワークフローに必要な道具であるため、ポートは UseCases 層に所属します。

```go
// internal/usecase/port/notification.go
package port

import "context"

type NotificationGateway interface {
	Send(ctx context.Context, message string) error
}
```

### 1-2. UseCase での利用

UseCase は、通知が Email なのか Slack なのかを気にしません。

```go
// internal/usecase/approve_membership.go
package usecase

import "context"

func (uc *ApproveUseCase) Execute(ctx context.Context, userID string) error {
	// ... 承認ロジック ...
	return uc.notifier.Send(ctx, "メンバーシップが承認されました！")
}
```

### 1-3. Infra 層での実装差し替え

最初は Email 実装を使いますが、後で Slack 実装を作成し、DI (Dependency Injection) で差し替えるだけです。

```go
// internal/adapters/infra/slack_notifier.go (後から追加)
package infra

import "context"

type SlackNotifier struct {
	webhookURL string
}

func (n *SlackNotifier) Send(ctx context.Context, msg string) error {
	// Slack API を叩く実装
	return nil
}
```

### ✅ チェックポイント

- [ ] `NotificationGateway` が `usecase`（port）パッケージに定義されている — `domain` や `infra` ではない
- [ ] `ApproveUseCase` が `infra` パッケージを **一切 import せずに** コンパイルできる
- [ ] Email から Slack への差し替えが `main.go` の **1 行変更だけ** で完了する
- [ ] モック/フェイクの `NotificationGateway` を使い、Slack に一切接続せずに「Send が呼ばれたこと」だけを検証するテストを書ける

---

## 課題 2: 透過的キャッシュの追加 (Decorator パターン)

「ユーザー情報取得を高速化したい。ただし、既存の `FindByID` を呼び出している箇所（UseCase など）は 1 行も直したくない」という要件に対応します。

### 2-1. キャッシュ用リポジトリの実装

元の `SQLUserRepository` をラップする、新しい Infra 実装を作ります。

```go
// internal/adapters/infra/cached_user_repository.go
package infra

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"your-project/internal/domain"
	"github.com/redis/go-redis/v9"
)

type CachedUserRepository struct {
	origin domain.UserRepository // 本物のDBリポジトリ
	cache  *redis.Client         // キャッシュDB
}

func (r *CachedUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	// 1. キャッシュにあればそれを返す
	if user, err := r.getFromCache(ctx, id); err == nil {
		return user, nil
	}

	// 2. なければ本物のDBに聞きに行く
	user, err := r.origin.FindByID(ctx, id)
	if err == nil {
		if cacheErr := r.saveToCache(ctx, user); cacheErr != nil {
			slog.Warn("failed to save user to cache", "error", cacheErr) // キャッシュ保存の失敗はログ出力に留め、メイン処理は続行
		}
	}
	return user, err
}

// キャッシュ用ヘルパー — この Decorator の実装詳細。
// UserRepository interface の一部ではないため、UseCase 層も
// ラップされる側のリポジトリも Redis の存在を知らない。

func (r *CachedUserRepository) getFromCache(ctx context.Context, id string) (*domain.User, error) {
	data, err := r.cache.Get(ctx, "user:"+id).Bytes()
	if err != nil {
		return nil, err // redis.Nil（キャッシュミス）やキャッシュ障害 → 本物のDBへフォールバック
	}
	var user domain.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *CachedUserRepository) saveToCache(ctx context.Context, user *domain.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	// TTL を設定し、古いキャッシュエントリが自動的に失効するようにする
	return r.cache.Set(ctx, "user:"+user.ID, data, 10*time.Minute).Err()
}
```

> **💡 ロギングに関する補足（教育的簡略化）:**  
> 本実習ではコードの簡略化のため、キャッシュ処理の警告ログ出力に標準ライブラリの `log/slog` を直接呼び出しています。実際のプロダクションコードでは、テストの容易性やロギングライブラリの抽象化のために、ロガーをインターフェース経由で注入（DI）するか、あるいはデコレーター等で横断的関心事（Cross-Cutting Concerns）として分離する設計が推奨されます。

### 2-2. DI コンテナでの構成変更

`main.go` 等で、本物のリポジトリをキャッシュ用リポジトリで「包んで」から UseCase に渡します。

```go
func main() {
	realRepo := infra.NewSQLUserRepository(db)
	// 本物をキャッシュ機能でラッピングする
	cachedRepo := infra.NewCachedUserRepository(realRepo, redisClient)

	// UseCase は interface を見ているので、ラップされた cachedRepo もそのまま受け取れる
	useCase := usecase.NewCheckVeteranUseCase(cachedRepo, domain.VeteranService{})
}
```

### ✅ チェックポイント

- [ ] `CachedUserRepository` が `domain.UserRepository` を満たしている — `var _ domain.UserRepository = (*infra.CachedUserRepository)(nil)` がコンパイルできる
- [ ] Redis が `internal/adapters/infra/` にのみ存在する — `git grep redis internal/domain internal/usecase` で何も見つからない
- [ ] Redis を止めても読み取りは成功する（Decorator が本物の DB にフォールバックする）。キャッシュ保存の失敗は警告ログのみで処理が続行する
- [ ] Decorator を配線した状態で `go test ./...` が通る

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
   - Redis クライアントのような詳細は Infra Adapters (インフラアダプター層) に閉じ込めます。

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
- ポートの所属は「能力を必要とする層」で決める: ドメインポリシーは Domain Port、アプリケーションのオーケストレーション（通知・永続化）は UseCase Port。

**次**: [WS3: 通信プロトコルの差し替え](./clean_arch_ws3_ja.md) — BBS の REST API を Domain・UseCase・Infra に触れずに gRPC へ移行します。

---

## 参考文献

- [The Clean Architecture — Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Redis Documentation](https://redis.io/docs/latest/)
- [go-redis Documentation](https://redis.uptrace.dev/)
- [log/slog package — Go Documentation](https://pkg.go.dev/log/slog)
