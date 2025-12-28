# クリーンアーキテクチャ実習 (WS2): 拡張と最適化

WS1 に続き、さらに実践的なシナリオを通じて、副作用の制御とパフォーマンス最適化におけるレイヤー分離の恩恵を学びます。

## 実習のシナリオ

1. **通知チャネルの多角化**: 通知手段を Email から Slack へ、ビジネスロジックを変えずに変更します。
2. **透過的キャッシュの導入**: ビジネスロジックを一切汚さずに、インフラ層の差し替えだけで Redis キャッシュを導入します。

---

## 課題 1: 通知システムの抽象化

「メンバーシップが承認されたら通知を送る」というユースケースを考えます。

### 1-1. Domain 層でのインターフェース定義

「どう送るか」ではなく「通知を送るという機能」を定義します。

```go
// domain/notification.go
package domain

type NotificationService interface {
 Send(ctx context.Context, message string) error
}
```

### 1-2. UseCase での利用

UseCase は、通知が Email なのか Slack なのかを気にしません。

```go
// usecase/approve_membership.go
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
// infra/slack_notifier.go (後から追加)
type SlackNotifier struct {
 webhookURL string
}

func (n *SlackNotifier) Send(ctx context.Context, msg string) error {
 // Slack API を叩く実装
 return nil
}
```

---

## 課題 2: 透過的キャッシュの追加 (Decorator パターン)

「ユーザー情報取得を高速化したい。ただし、既存の `FindByID` を呼び出している箇所（UseCase など）は 1 行も直したくない」という要件に対応します。

### 2-1. キャッシュ用リポジトリの実装

元の `SqlUserRepository` をラップする、新しい Infra 実装を作ります。

```go
// infra/cached_user_repository.go
package infra

type CachedUserRepository struct {
 origin domain.UserRepository // 本物のDBリポジトリ
 cache  *redis.Client         // キャッシュDB
}

func (r *CachedUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
 // 1. キャッシュにあればそれを返す
 if user, err := r.getFromCache(id); err == nil {
  return user, nil
 }

 // 2. なければ本物のDBに聞きに行く
 user, err := r.origin.FindByID(ctx, id)
 if err == nil {
  r.saveToCache(user) // 次のために保存
 }
 return user, err
}
```

### 2-2. DI コンテナでの構成変更

`main.go` 等で、本物のリポジトリをキャッシュ用リポジトリで「包んで」から UseCase に渡します。

```go
func main() {
 realRepo := infra.NewSqlUserRepository(db)
 // 本物をキャッシュ機能でラッピングする
 cachedRepo := infra.NewCachedUserRepository(realRepo, redisClient)

 // UseCase は interface を見ているので、ラップされた cachedRepo もそのまま受け取れる
 useCase := usecase.NewCheckVeteranUseCase(cachedRepo)
}
```

---

## この実習のポイント

1. **副作用の抽象化 (課題 1)**:
    - 「通知を送る」という行為自体がビジネスルールであり、「Slack で送る」のは単なる手段（インフラの詳細）です。
    - 詳細を外に追い出すことで、テスト時に実際の通知を飛ばさず「通知が呼ばれたこと」だけを検証するのが容易になります。
2. **透過的な機能追加 (課題 2)**:
    - インターフェースが同じであれば、中身が「DB から取るもの」から「キャッシュ制御付きで取るもの」に変わっても、呼び出し元は一切気づきません。
    - これにより、ビジネスロジックの健全性を保ったまま、インフラ面での改善（パフォーマンス対策、ロギング、リトライ処理の追加など）を自由に行えます。
