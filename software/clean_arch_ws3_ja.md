# クリーンアーキテクチャ実習: advent-calm-2025

このワークショップでは、Go 言語を用いて「Clean Architecture」に基づいた堅牢でテスト容易なアプリケーションを構築する方法を学びます。
完全なプロジェクトファイルは [advent-of-calm-2025](./advent-of-calm-2025/) ディレクトリにあります。

## 1. Clean Architecture とは？

Clean Architecture（クリーンアーキテクチャ）は、ソフトウェアの関心事を分離し、ビジネスロジックをフレームワークや外部ツールから独立させるための設計指針です。

### 4層構造（The 4 Layers）

本ワークショップでは、シンプルで実用的な **4層構造** を採用します。

1. **ドメイン層 (Domain Layer)** - `domain/`
    * **役割**: ビジネスの中核となるルールとデータ構造。
    * **特徴**: **他のどの層にも依存しません**。純粋な Go のコードのみで記述されます。
    * **構成要素**: エンティティ (Entity), ポートインターフェース (Port), ドメインサービス (Domain Service: 複雑なロジック用)。

2. **ユースケース層 (Usecase Layer)** - `usecase/`
    * **役割**: アプリケーション固有のビジネスルール（ユーザーが何をしたいか）。
    * **特徴**: ドメイン層にのみ依存します。DB や HTTP の詳細を知りません。
    * **構成要素**: ユースケース (Interactor), 入力/出力データ構造 (DTO)。

3. **インフラアダプター層 (Infra Adapters)** - `infra/`
    * **役割**: ドメインの契約（Port）を具体的に実装し、外部 I/O（DB 等）との橋渡しを行う。
    * **特徴**: **ドメイン層のインターフェースに依存して実装する層**です（依存は内向き）。
    * **構成要素**: リポジトリの実装 (Repository Impl), 外部クライアント (Client Impl), DB 連携。

4. **フレームワーク層 (Framework Layer)**
    * **役割**: Web フレームワーク、gRPC、CLI、およびそれらのハンドラー。
    * **特徴**: 最外周の I/O を制御し、入力を UseCase 向けに変換して呼び出します。
    * **構成要素**: Web ハンドラー, Router, DTO 変換。

### 依存性のルール (The Dependency Rule)

**「依存は常に内側（ドメイン側）に向かう」**
ソースコードの依存関係は、常に低レベル（詳細）から高レベル（抽象）へ向かいます。

```mermaid
graph TD
    Customer[Customer / Admin]
    Framework[Framework<br>API Handler]
    Usecase[Usecase<br>CreateOrder / CheckInventory]
    Domain["Domain<br>Entities + Ports"]
    Infra["Infra Adapters<br>Repo / REST Client"]
    OrderDB[(Order DB)]
    InvDB[(Inventory DB)]
    ExternalInvAPI["Inventory Service API (External)"]

    Customer --> Framework
    Framework --> Usecase
    Usecase --> Domain
    Infra --> Domain
    Infra --> OrderDB
    Infra --> InvDB
    Infra --> ExternalInvAPI

    style Framework fill: #555, stroke-width:2px
    style Usecase fill: #555, stroke-width:2px
    style Domain fill: #555, stroke-width:2px
    style Infra fill: #555, stroke-width:2px
```

> **注記: 外部インターフェースの集約**
> `Customer`（注文者）と `Admin`（在庫管理者）は、それぞれ適切な API エンドポイントを叩きます。`Order Service` 内の `Inventory REST Client` が叩く先は、**同一プロセスの Framework ではなく外部 Inventory Service の API** として扱います（依存方向の誤解を防ぐため）。
>
> **Ports とは？**
> Ports は「内側のルールが外側に求める契約（インターフェース）」です。DBや外部APIの詳細は Ports の背後に隠れ、ユースケースは Ports に依存して振る舞いだけを定義します。外側（Infra Adapters）は Ports を実装することで依存方向を内向きに保ちます。

### ポート設計とリポジトリ境界

* **入力ポート:** UseCase が公開するインターフェース。Web/CLI などの Controller はこのポートに依存します。
* **出力ポート:** Domain/UseCase が外側に要求する契約（例: Repository, Client）。インターフェースは内側に置き、実装は Adapter 側に置きます。
* **リポジトリ境界:** 永続化の契約。トランザクション/リトライなどの制御は UseCase 側、データ変換やクエリ組み立ては Adapter 側の責務です。

---

## ワークショップ: 注文システムの構築

架空の「注文作成システム」を題材に、内側から外側へと実装を進めていきます。

### Step 1: ドメイン層の設計 (`domain/`)

ドメイン層はアプリケーションの**心臓部**であり、以下の要素で構成されます。これらは外部（DB や Web）の都合に一切依存しません。

1. **Entity**: ビジネスデータとルール（例: `Order`, `Inventory`）。
2. **Interface (Port)**: データの永続化や外部連携のための契約（例: `OrderRepository`, `InventoryClient`）。
3. **Domain Service**: 複数のエンティティにまたがる複雑な計算やロジック（※単純な I/O ラップは避ける）。

まずはビジネスのコアとなる「注文 (Order)」と、外界と対話するための契約「インターフェース」を定義します。

**1. エンティティの定義 (`domain/entity/models.go`)**
注文の状態や構造を定義します。

```go
type Order struct {
	ID         string
	CustomerID string
	Amount     float64
	Status     OrderStatus
	CreatedAt  time.Time
}
```

**2. インターフェース（Ports）の定義 (`domain/repository/interfaces.go`)**
データの保存や外部サービスへのアクセス方法を**抽象化**します。ここで定義したインターフェースの実装は、Step 3 で行います。

```go
// 依存性逆転の原則 (DIP): 上位モジュールがインターフェースを所有する
type OrderRepository interface {
	Save(ctx context.Context, order *entity.Order) error
	FindByID(ctx context.Context, id string) (*entity.Order, error)
}

type InventoryClient interface {
	CheckAndReserve(ctx context.Context, productID string, quantity int) (bool, error)
}

type PaymentPublisher interface {
	PublishPaymentTask(ctx context.Context, order *entity.Order) error
}
```

### Step 2: ユースケース層の実装 (`usecase/`)

ドメイン層の部品（Entity や Port）を組み合わせて、「注文を作成する」というアプリケーションの機能を実装します。

**実装 (`usecase/create_order.go`)**

```go
type CreateOrderUsecase struct {
	orderRepo repository.OrderRepository // 抽象に依存
	invClient repository.InventoryClient // 外部 API 連携も Port 経由
	// ...
}

func (u *CreateOrderUsecase) Execute(ctx context.Context, input CreateOrderInput) error {
	// 1. バリデーションと在庫確保 (Port利用)
	// 2. 注文エンティティ作成
	// 3. データベース保存 (Repository利用)
	// 4. イベント発行
}
```

ここでのポイントは、`CreateOrderUsecase` が具体的なデータベース（Postgres など）や通信プロトコル（REST/gRPC）を知らないことです。知っているのは Ports（`OrderRepository` など）のみです。

### Step 3: インフラアダプター層の実装 (`infra/`)

ここで初めて「PostgreSQL」や「REST API」といった具体的な技術が登場します。**Step 1で定義したドメイン層のインターフェースを実装**します。

* `PostgresOrderRepository` は `domain.OrderRepository` を実装。
* `RestInventoryClient` は `domain.InventoryClient` を実装。

**リポジトリの実装 (`infra/repository/postgres_order_repository.go`)**

```go
type PostgresOrderRepository struct {
	// DB接続インスタンスなど
}

// domain/repository.OrderRepository インターフェースを満たす
func (r *PostgresOrderRepository) Save(ctx context.Context, order *entity.Order) error {
	fmt.Printf("Saving order %s to Postgres\n", order.ID)
	// 実際のSQL実行処理...
	return nil
}
```

**エラー境界の注意**

* Infra Adapter は `sql.ErrNoRows` などの driver error を上位にそのまま返さず、`entity.ErrOrderNotFound` のような Domain Error に変換して返します。
* Framework は最終的に Domain/UseCase Error を HTTP status / gRPC status に変換します。

### Step 4: アプリケーションの組み立て (`main.go`)

最後に、`main.go` で全てのパーツを組み立てます（Dependency Injection）。

```go
func main() {
	// 1. 依存オブジェクト（Infra Adapters）の生成
	orderRepo := &repository.PostgresOrderRepository{}
	inventoryClient := &client.RestInventoryClient{}
	paymentPub := &messaging.RabbitMQPaymentPublisher{}
	idGen := &util.UUIDGenerator{}
	inventoryRepo := &repository.PostgresInventoryRepository{}

	// 2. ユースケースへの直接注入 (Domain Port を実装した Adapter を渡す)
	createOrderUsecase := usecase.NewCreateOrderUsecase(orderRepo, inventoryClient, paymentPub, idGen)
	checkInventoryUsecase := usecase.NewCheckInventoryUsecase(inventoryRepo)
	updateInventoryUsecase := usecase.NewUpdateInventoryUsecase(inventoryRepo)

	// 3. 実行
	createOrderUsecase.Execute(ctx, input)
}
```

> **補足: Composition Root と Framework の分離**
> このサンプルは説明簡略化のため `main.go` に組み立てと実行を同居させています。規約をより厳密に適用する場合は、`main.go` を Composition Root 専用にし、CLI/Web の入出力処理は `framework/...` に分離します。

---

## 設計分析と品質 (Clean Architecture 分析)

本プロジェクトは以下の観点で高品質な設計が維持されています。

1. **疎結合な設計**: 注文 (Order) と在庫 (Inventory) がドメインレベルで分離されており、将来的なマイクロサービス化が容易です。
2. **ビジネスロジックの純粋性**: `domain` パッケージには外部ライブラリへの依存が一切なく、ビジネスルールのみが記述されています。
3. **適切な責務分割**: 単なる I/O ラップを「ドメインサービス」と呼ぶ誤用を避け、UseCase がオーケストレーションを担うことで、真にビジネス価値のあるロジックだけを Domain に集中させています。

---

## 実行方法

プロジェクトのルートディレクトリで以下のコマンドを実行し、依存関係を解決してから実行してください。

```bash
# 依存関係の整理
go mod tidy

# アプリケーションの実行
go run main.go
```

## まとめ

* **変更に強い**: DB を MySQL に変えても、`domain` や `usecase` のコードは 1 行も変わりません。
* **テストしやすい**: `usecase` のテストでは、`repository` のモックを作るだけで済みます。DB は不要です。
* **関心の分離**: ビジネスロジックと技術的詳細が明確に分かれています。
