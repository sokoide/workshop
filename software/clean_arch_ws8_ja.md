# クリーンアーキテクチャの極意: 最小構成で本質を理解する (WS8)

この実習では、わずか 100 行程度の「ユーザー挨拶アプリ」を通じて、クリーンアーキテクチャ（3 層バリアント: Adapters / UseCases / Domain）の設計思想と「変更への強さ」を体験します。

## 1. 概要

クリーンアーキテクチャの本質は**「依存性の制御」**にあります。ビジネスルール（ドメイン）が外部の都合（データベース、Web フレームワークなど）に汚染されないように、依存関係を内側に向けることがゴールです。

このバリアントは 3 つの概念層を用います。**Adapters** 層は単一レイヤーの巨大化を防ぎ、副作用の境界を明確にするため、**Presentation Adapters**（入力/駆動側）と **Infrastructure Adapters**（出力/被駆動側）に分割しています。

### 本アプリの機能

- HTTP リクエストを受け取る (`GET /?id=1`)
- 指定された ID のユーザー名をデータベース（今回はメモリ）から取得する
- `Hello, [Name]!` という挨拶文を生成して返す

---

## 2. ディレクトリ構造と依存ルール

依存関係は常に **外側から内側へ** 向かいます。

```text
software/assets/greeting/
├── internal/domain/       (Layer 1: 最内) ビジネスルール、Entity、Port(Interface)、Domain Error
├── internal/usecase/      (Layer 2) ユースケース（シナリオ）— オーケストレーションのみ、アプリケーション DTO、UseCase Port
├── internal/adapters/infra/        (Layer 3: Adapters — Infrastructure) インフラ詳細の実装
├── internal/adapters/presentation/ (Layer 3: Adapters — Presentation) HTTP Handler、外部ライブラリ
└── main.go       (Composition Root) 全レイヤーの組み立て（依存注入）
```

> **補足:** `infra/` と `presentation/` はどちらも Adapters 層の一部であり、方向（出力側 vs 入力側）で分割されています。規模が大きくなるプロジェクトでは `internal/adapters/infra/` と `internal/adapters/presentation/` に配置するのが一般的です。

```text
Presentation Adapters ---→ UseCases ---→ Domain
                             ↑              ↑
                             +--------------+-- Infrastructure Adapters
        implements ports owned by UseCases or Domain
```

### 依存マトリクス

| 依存元 ↓ →                         | Domain | UseCases | Adapters |
|:----------------------------------:|:------:|:--------:|:--------:|
| Domain                             | yes    | no       | no       |
| UseCases                           | yes    | yes      | no       |
| Adapters（Presentation）           | limited| yes      | self     |
| Adapters（Infrastructure）         | yes    | yes      | self     |
| Composition Root                   | yes    | yes      | yes      |

- `Presentation → Domain` は `limited`: Presentation は UseCases から返された Domain の値をシリアライズのために **読み取ってもよい** が、ワークフローの判断のために Domain の振る舞いを **直接呼び出してはならない**。
- Presentation Adapters と Infrastructure Adapters は同じ概念層にありますが、互いに直接依存してはなりません。

---

## 3. レイヤーごとの解説

### 3-1. Domain 層: システムの「核心」

`domain/` には、このシステムにとって不変の知識を定義します。

- `user.go`: 「ユーザー」という概念（Entity）。
- `repository.go`: 「ユーザーを取得する方法」という窓口（Port）。インターフェースで定義し、具体的な DB 操作はここには書きません。
- `errors.go`: ドメイン層で定義されたエラー型。**Infra 層の技術的エラーをそのまま外に漏らさず、ドメインの意味を持つエラーに変換します。**

### 3-2. UseCases 層: シナリオの「手順」

`usecase/` には、「どういう順番で何をするか」というビジネスロジックを書きます。

- `GreetingUseCase` は `UserRepository` という「窓口」を通じ、データがどこから来るかを知らずに挨拶文を組み立てます。
- UseCase は Domain にのみ依存し、具象 Adapter の実装を知りません。

### 3-3. Infrastructure Adapters 層: 技術的な「詳細」（出力側）

`infra/` には、Domain 層で定義されたインターフェースの「中身」を実装します。

- 今回は `MemoryUserRepo` としてメモリ上にデータを持ちますが、ここを MySQL や API に差し替えても、他のレイヤーを壊さずに済みます。
- **エラーバウンダリ**: ユーザーが見つからない場合、技術的エラーではなくドメインエラー（`domain.ErrUserNotFound`）を返します。

### 3-4. Presentation Adapters 層: 外部との「接点」（入力側）

`presentation/` には、HTTP などの通信プロトコルに関する処理を書きます。

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
go test ./internal/usecase/... -v
```

`usecase/greeting_test.go` を読み、以下の点を確認してください。

- どうやってリポジトリを偽物（Mock）に差し替えているか
- ユーザー不在時のエラーケースがどうテストされているか

### ステップ 3: 【演習】ビジネスルールを変更する

`usecase/greeting.go` を開き、挨拶文を日本語（`こんにちは、[Name]さん！`）に変更してみてください。

- **ポイント**: この変更の際、`infra` や `presentation` のコードに触れる必要がないことを確認してください。

### ステップ 4: 【演習】インフラを差し替える

このステップが、Clean Architecture の最大の強みを体感するパートです。

1. `infra/` に新しいファイル `slice.go` を作成し、スライスでユーザーを管理するリポジトリを実装してください:

```go
package infra

import (
	"context"

	"workshop/greeting/domain"
)

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

func (r *SliceUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
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

- **ポイント**: この変更で触れたのは Infrastructure Adapter（`infra/`）と Composition Root（`main.go`）だけです。`domain/`、`usecase/`、`presentation/` は一切変更なしに動作します。これが「依存性の逆転」の実感です。

---

## 5. 発展トピック: エラーとデータの境界

Clean Architecture では、レイヤー間のエラーとデータにも明確な境界があります。

### エラーバウンダリ

```text
Infrastructure Adapters → ドライバエラーをドメインエラーに変換して返す
UseCases 層              → ドメインエラーをそのまま伝播
Presentation Adapters    → ドメインエラーを HTTP ステータスに変換（404, 500 など）
```

この設計により、データベース固有のエラー（例: `sql.ErrNoRows`）が UseCases や Presentation に漏れることを防ぎます。

### データバウンダリ（DTO）

実務では、UseCase の入出力を明示的な構造体（DTO: Data Transfer Object）で定義し、Entity（`domain.User`）と Presentation のリクエスト/レスポンスを分離します。今回は最小構成のためプリミティブ型を使用していますが、システムが大きくなるにつれて DTO の導入が推奨されます。

---

## 6. まとめ

1. **Domain は何にも依存しない**: ビジネスの核となる知識とエラーを独立させます。
2. **UseCases は Port (Interface) を通じて会話する**: 実装（DB 等）を知らなくてもロジックは完結できます。
3. **エラーバウンダリを守る**: 技術的エラーは Infrastructure Adapters でドメインエラーに変換し、Presentation Adapters で HTTP ステータスに変換します。
4. **Adapters = 副作用の境界**: すべての I/O と外部統合は Adapters 層に閉じ込めます。
5. **依存性の注入 (DI)**: `main.go` で各パーツをカチッとはめ込むことで、全体が動くようになります。

これが、メンテナンス性が高く、テストが容易で、変化に強い「クリーン」な設計の第一歩です。
