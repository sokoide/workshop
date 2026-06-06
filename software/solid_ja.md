# SOLID principles (Goで学ぶ設計原則)

変更に強く、テストしやすいコードを作るための 5 つの設計原則です。Go ではクラス継承よりも、**小さなインターフェース** と **明確な責務分離** で SOLID を実現します。

---

## S: Single Responsibility Principle (単一責任の原則)

1 つの型・関数は、1 つの理由でだけ変更されるべきです。

### なぜ守らないと困るのか

責務が混ざると、1 つの変更が別の関心事を壊します。

例: `ProcessOrder` が「金額計算」「DB 保存」「メール通知」を同時に持つ場合。

- 税計算ルールを変えたいだけでも、同じ関数内の通知処理まで影響確認が必要になる
- DB 保存の失敗を再現したいテストで、SMTP モックまで必要になる
- メール文面変更のリリースで、注文計算に不具合を混入させるリスクが上がる

つまり、**変更の影響範囲が読めなくなり、改修コストと障害率が上がる** のが本質です。

言い換えると S の狙いは、**1 つの変更を 1 か所に閉じ込める** ことです。これにより他の機能が壊れにくくなります。

### 悪い例

```go
// 注文計算・保存・通知を1つで担当している
func ProcessOrder(o Order, db *sql.DB, smtp SMTPClient) error {
	// 金額計算
	// DB保存
	// メール送信
	return nil
}
```

```go
// 税率変更時に、同じ関数内の通知文面ロジックとの整合まで見る必要がある例
func ProcessOrder(o Order, db *sql.DB, smtp SMTPClient) error {
	taxRate := 0.10
	taxLabel := "標準税率"
	if o.HasReducedTaxItem {
		taxRate = 0.08
		taxLabel = "軽減税率"
	}
	o.Total = int(float64(o.Subtotal) * (1 + taxRate))

	// DB保存
	if _, err := db.Exec("INSERT INTO orders(total) VALUES(?)", o.Total); err != nil {
		return err
	}

	// 通知文面が税計算の詳細に依存している
	body := fmt.Sprintf("注文合計: %d (%s)", o.Total, taxLabel)
	return smtp.Send(o.Email, body)
}
```

この形だと税ルール変更時に、通知文面（`taxLabel` の扱い）まで同時に確認・調整が必要になります。
通知コードを「必ず書き換える」わけではないですが、同居しているため変更影響の調査範囲が広がります。

### 改善例

```go
// domain/order_pricer.go
type PriceResult struct {
	Total    int
	TaxLabel string
}

type OrderPricer interface {
	Price(o Order) PriceResult
}

// domain/order_repository.go
type OrderRepository interface {
	Save(ctx context.Context, o Order) error
}

// domain/notifier.go (Output Port)
type Notifier interface {
	SendOrderCreated(ctx context.Context, to, body string) error
}

// usecase/create_order.go
type OrderUseCase struct {
	pricer   OrderPricer
	repo     OrderRepository
	notifier Notifier
}

func NewOrderUseCase(
	pricer OrderPricer,
	repo OrderRepository,
	notifier Notifier,
) *OrderUseCase {
	return &OrderUseCase{
		pricer:   pricer,
		repo:     repo,
		notifier: notifier,
	}
}

func (uc *OrderUseCase) Execute(ctx context.Context, o Order) error {
	price := uc.pricer.Price(o) // Domain ルール
	o.Total = price.Total
	if err := uc.repo.Save(ctx, o); err != nil {
		return err
	}
	// 悪い例と同じ通知仕様: 「注文合計: <total> (<taxLabel>)」
	body := fmt.Sprintf("注文合計: %d (%s)", price.Total, price.TaxLabel)
	return uc.notifier.SendOrderCreated(ctx, o.Email, body) // Domain Port 経由
}

// infra/notifier/mail_notifier.go
type MailNotifier struct {
	client SMTPClient
}

func (n MailNotifier) SendOrderCreated(ctx context.Context, to, body string) error {
	return n.client.Send(ctx, to, body)
}
```

責務を分離すると、計算ロジックだけのテスト、通知だけの差し替えが可能になります。
税ルール変更（10% -> 12%、軽減税率条件変更など）は `OrderPricer` 実装だけを直せばよく、`MailNotifier` は変更不要です。

### クリーンアーキテクチャで見る S

- **Domain:** 金額計算のルール（Entity/Domain Service）
- **UseCase:** 注文作成の手順（計算 -> 保存 -> 通知）
- **Infra Adapters:** DB 保存実装、メール通知実装
- **Framework:** HTTP Handler/CLI から UseCase を呼ぶ

ポイントは、変更理由ごとに修正レイヤーが分かれることです。

- 税計算変更: Domain だけ
- DB 製品変更: Infra Adapters だけ
- API 入出力形式変更: Framework だけ
- メール送信方式変更（SMTP -> SES など）: Infra Adapters だけ

---

## O: Open/Closed Principle (開放/閉鎖の原則)

拡張には開き、既存コードの修正には閉じる設計にします。

### なぜ守らないと困るのか

`switch paymentType` のような分岐を既存コードに増やし続けると、支払い方法追加のたびに既存の決済経路へ手を入れることになります。

- 新規追加のたびに既存の回帰テスト範囲が肥大化
- 既存分岐の修正ミスで別決済が壊れる

### 悪い例

```go
func Checkout(ctx context.Context, paymentType string, amount int) error {
	switch paymentType {
	case "card":
		return chargeCard(ctx, amount)
	case "paypay":
		return chargePayPay(ctx, amount)
	// ApplePay などの追加のたびに、この既存関数を編集する必要がある
	default:
		return errors.New("unsupported payment type")
	}
}
```

### 例: 支払い方法の追加

```go
type PaymentMethod interface {
	Pay(ctx context.Context, amount int) error
}

type CardPayment struct{}

func (CardPayment) Pay(ctx context.Context, amount int) error { return nil }

type PayPayPayment struct{}

func (PayPayPayment) Pay(ctx context.Context, amount int) error { return nil }

func Checkout(ctx context.Context, pm PaymentMethod, amount int) error {
	return pm.Pay(ctx, amount)
}
```

新しい支払い方法は `PaymentMethod` 実装を追加するだけで対応できます。

クリーンアーキテクチャ的には、`PaymentMethod` は UseCase 側の Port として置き、`CardPayment` などは Infra Adapter として追加します。既存 UseCase を編集せずに拡張できます。

---

## L: Liskov Substitution Principle (リスコフの置換原則)

実装は、期待される契約を壊さずに置き換え可能であるべきです。

### なぜ守らないと困るのか

同じインターフェースでも実装ごとに挙動が違うと、環境差（local/test/prod）で不具合が発生します。

- in-memory では `nil, nil`、RDB では `ErrNotFound` など契約不一致
- 呼び出し側に実装分岐が増え、抽象化の価値が消える

### 悪い例: 同じ interface でも契約を壊すと置換できない

```go
type PaymentGateway interface {
	Charge(ctx context.Context, amount int) error
}

type SafeGateway struct{}

func (SafeGateway) Charge(ctx context.Context, amount int) error {
	if amount <= 0 {
		return errors.New("invalid amount")
	}
	return nil
}

type PanicGateway struct{}

func (PanicGateway) Charge(ctx context.Context, amount int) error {
	if amount <= 0 {
		panic("invalid amount") // 契約違反: error で返す想定を壊す
	}
	return nil
}

func Checkout(ctx context.Context, gw PaymentGateway, amount int) error {
	// 呼び出し側は「失敗は error で返る」と期待している
	return gw.Charge(ctx, amount)
}
```

`Checkout` は `error` を処理する前提で書かれています。`PanicGateway` に差し替えると前提が崩れ、同じ `PaymentGateway` でも安全に置換できません。これが L 違反です。

クリーンアーキテクチャ的には、同じ Output Port を実装する全 Adapter が同じ契約（戻り値、エラー意味、タイムアウト方針）を守る必要があります。

---

## I: Interface Segregation Principle (インターフェース分離の原則)

大きすぎるインターフェースではなく、利用側に必要最小限のインターフェースを定義します。

### なぜ守らないと困るのか

巨大インターフェースは、不要な依存とモック実装の負担を増やします。

- 読み取り処理のテストなのに `Create/Update/Delete` のダミー実装が必要
- 変更の影響が広がり、コンパイルエラーが連鎖しやすい

### 悪い例

```go
type UserService interface {
	Create(ctx context.Context, u User) error
	Update(ctx context.Context, u User) error
	Delete(ctx context.Context, id string) error
	Find(ctx context.Context, id string) (User, error)
	List(ctx context.Context) ([]User, error)
}
```

### 改善例

```go
type UserFinder interface {
	Find(ctx context.Context, id string) (User, error)
}

type UserCreator interface {
	Create(ctx context.Context, u User) error
}
```

読み取り専用ユースケースは `UserFinder` のみを要求し、依存を最小化できます。

クリーンアーキテクチャでは Input Port も UseCase ごとに細かく分けると分かりやすくなります。例: `CreateUserUseCase`, `GetUserUseCase`。

---

## D: Dependency Inversion Principle (依存性逆転の原則)

上位モジュール（UseCase）は下位モジュール（DB, 外部 API）に直接依存せず、抽象に依存します。

### なぜ守らないと困るのか

UseCase が DB クライアントや外部 SDK に直接依存すると、技術選定の変更がビジネスロジック改修に直結します。

- MySQL -> PostgreSQL 移行で UseCase まで大量修正
- ユースケース単体テストが困難（本物の DB/API が必要）

### 例

```go
// usecase 側で抽象を定義

type InventoryGateway interface {
	Reserve(ctx context.Context, sku string, qty int) error
}

type CreateOrderUseCase struct {
	inv InventoryGateway
}

func (uc *CreateOrderUseCase) Execute(ctx context.Context, sku string, qty int) error {
	return uc.inv.Reserve(ctx, sku, qty)
}
```

`infra` 側で `InventoryGateway` を実装すれば、UseCase は外部技術の詳細を知らずに済みます。

クリーンアーキテクチャの依存方向（外 -> 内）を守ると、ビジネスルールは技術変更の影響を受けにくくなります。

---

## まとめ: SOLID を使う目的

SOLID は「きれいに書くため」ではなく、以下を下げるための実務ルールです。

- 変更時の影響範囲
- テスト準備コスト
- 実装差による本番障害リスク

---

## Go で SOLID を実践するコツ

- インターフェースは実装側ではなく **利用側（consumer）** に置く
- `context.Context` は I/O を伴う関数の第 1 引数で受ける
- エラー契約（例: `ErrNotFound`）を実装間で統一する
- まずはシンプルに作り、重複や変更圧が出た時に抽象化する
