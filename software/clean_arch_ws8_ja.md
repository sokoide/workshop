# クリーンアーキテクチャの極意: 最小構成で本質を理解する (WS8)

この実習では、わずか 100 行程度の「ユーザー挨拶アプリ」を通じて、クリーンアーキテクチャ（4 レイヤー）の設計思想と「変更への強さ」を体験します。

## 1. 概要

クリーンアーキテクチャの本質は**「依存性の制御」**にあります。ビジネスルール（ドメイン）が外部の都合（データベース、Web フレームワークなど）に汚染されないように、依存関係を内側に向けることがゴールです。

### 本アプリの機能

- HTTP リクエストを受け取る (`GET /?id=1`)
- 指定された ID のユーザー名をデータベース（今回はメモリ）から取得する
- `Hello, [Name]!` という挨拶文を生成して返す

---

## 2. ディレクトリ構造と依存ルール

依存関係は常に **外側から内側へ** 向かいます。

```text
software/assets/greeting/
├── domain/       (Layer 1: 最内) ビジネスルール、Entity、Port(Interface)、Domain Error
├── usecase/      (Layer 2) ユースケース（シナリオ）
├── infra/        (Layer 3) インフラ詳細の実装（Adapter）
├── framework/    (Layer 4: 最外) HTTP Handler、外部ライブラリ
└── main.go       (Composition Root) 全レイヤーの組み立て（依存注入）
```

### 依存マトリクス

| 依存元 ↓ → | Domain | UseCase | Infra | Framework |
|------------|--------|---------|-------|-----------|
| Domain     |   ✓    |    ✗    |   ✗   |     ✗     |
| UseCase    |   ✓    |    ✓    |   ✗   |     ✗     |
| Infra      |   ✓    |    ✗    |   ✓   |     ✗     |
| Framework  |   ✗    |    ✓    |   ✗   |     ✓     |

✓ = 許可 | ✗ = 禁止

---

## 3. レイヤーごとの解説

### 3-1. Domain 層: システムの「核心」

`domain/` には、このシステムにとって不変の知識を定義します。

- `user.go`: 「ユーザー」という概念（Entity）。
- `repository.go`: 「ユーザーを取得する方法」という窓口（Port）。インターフェースで定義し、具体的な DB 操作はここには書きません。
- `errors.go`: ドメイン層で定義されたエラー型。**Infra層の技術的エラーをそのまま外に漏らさず、ドメインの意味を持つエラーに変換します。**

### 3-2. UseCase 層: シナリオの「手順」

`usecase/` には、「どういう順番で何をするか」というビジネスロジックを書きます。

- `GreetingUseCase` は `UserRepository` という「窓口」を通じ、データがどこから来るかを知らずに挨拶文を組み立てます。

### 3-3. Infra Adapter 層: 技術的な「詳細」

`infra/` には、Domain 層で定義されたインターフェースの「中身」を実装します。

- 今回は `MemoryUserRepo` としてメモリ上にデータを持ちますが、ここを MySQL や API に差し替えても、他のレイヤーを壊さずに済みます。
- **エラーバウンダリ**: ユーザーが見つからない場合、技術的エラーではなくドメインエラー（`domain.ErrUserNotFound`）を返します。

### 3-4. Framework 層: 外部との「接点」

`framework/` には、HTTP などの通信プロトコルに関する処理を書きます。

- リクエストから ID を取り出し、UseCase に渡し、結果をレスポンスとして返す役割に専念します。
- **エラーの変換**: UseCase/Domain から返されたドメインエラーを HTTP ステータスコードに変換します（例: `ErrUserNotFound` → 404）。

---

## 4. 実習: 「変更への強さ」を体験する

### ステップ 1: アプリを起動する

ターミナルで以下を実行し、サーバーを起動します。

```bash
cd software/assets/greeting
go run main.go
```

別のターミナルからリクエストを送り、動作を確認します。

```bash
curl "http://localhost:8080?id=1"
# 出力: Hello, Alice!

curl "http://localhost:8080?id=999"
# 出力: user not found (HTTP 404)
```

### ステップ 2: ユニットテストを実行する

Clean Architecture の利点は、外部環境（DB など）をモックに差し替えて、ビジネスロジックだけを高速にテストできることです。

```bash
go test ./usecase/... -v
```

`usecase/greeting_test.go` を読み、以下の点を確認してください。

- どうやってリポジトリを偽物（Mock）に差し替えているか
- ユーザー不在時のエラーケースがどうテストされているか

### ステップ 3: 【演習】ビジネスルールを変更する

`usecase/greeting.go` を開き、挨拶文を日本語（`こんにちは、[Name]さん！`）に変更してみてください。

- **ポイント**: この変更の際、`infra` や `framework` のコードに触れる必要がないことを確認してください。

### ステップ 4: 【演習】インフラを差し替える

このステップが、Clean Architecture の最大の強みを体感するパートです。

1. `infra/` に新しいファイル `slice.go` を作成し、スライスでユーザーを管理するリポジトリを実装してください:

```go
package infra

import "workshop/greeting/domain"

type SliceUserRepo struct {
	users []*domain.User
}

func NewSliceUserRepo() *SliceUserRepo {
	return &SliceUserRepo{
		users: []*domain.User{
			{ID: "1", Name: "Alice"},
			{ID: "2", Name: "Bob"},
		},
	}
}

func (r *SliceUserRepo) FindByID(id string) (*domain.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, domain.ErrUserNotFound
}
```

1. `main.go` の 1 行を書き換えます:

```go
// 変更前
repo := infra.NewMemoryUserRepo()
// 変更後
repo := infra.NewSliceUserRepo()
```

1. サーバーを再起動し、同じように動くことを確認します。

- **ポイント**: この変更で触れたのは `infra/` と `main.go` だけです。`domain/`、`usecase/`、`framework/` は一切変更なしに動作します。これが「依存性の逆転」の実感です。

---

## 5. 発展トピック: エラーとデータの境界

Clean Architecture では、レイヤー間のエラーとデータにも明確な境界があります。

### エラーバウンダリ

```text
Infra層 → ドメインエラーに変換して返す（driver errorは外に漏らさない）
UseCase層 → ドメインエラーをそのまま伝播
Framework層 → ドメインエラーをHTTPステータスに変換（404, 500など）
```

この設計により、データベース固有のエラー（例: `sql.ErrNoRows`）が UseCase や Framework に漏れることを防ぎます。

### データバウンダリ（DTO）

実務では、UseCase の入出力を明示的な構造体（DTO: Data Transfer Object）で定義し、Entity（`domain.User`）と Framework のリクエスト/レスポンスを分離します。今回は最小構成のためプリミティブ型を使用していますが、システムが大きくなるにつれて DTO の導入が推奨されます。

---

## 6. まとめ

1. **Domain は何にも依存しない**: ビジネスの核となる知識とエラーを独立させます。
2. **UseCase は Port (Interface) を通じて会話する**: 実装（DB 等）を知らなくてもロジックは完結できます。
3. **エラーバウンダリを守る**: 技術的エラーは Infra 層でドメインエラーに変換し、Framework 層で HTTP ステータスに変換します。
4. **依存性の注入 (DI)**: `main.go` で各パーツをカチッとはめ込むことで、全体が動くようになります。

これが、メンテナンス性が高く、テストが容易で、変化に強い「クリーン」な設計の第一歩です。
