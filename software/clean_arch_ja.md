# クリーンアーキテクチャ（3層バリアント: Adapters / UseCases / Domain）

ビジネスロジックを中心に据え、外部の詳細（DB や通信）に依存しない疎結合なソフトウェアを構築するための構成案です。

このバリアントは 3 つの概念層を用います。**Adapters** 層は概念的には 1 層ですが、単一レイヤーの巨大化を防ぎ、副作用の境界を明確にするため、**Presentation Adapters**（入力/駆動側）と **Infrastructure Adapters**（出力/被駆動側）に分割しています。

## レイヤー構造と依存関係

依存関係は常に **内側（より高位のポリシー）** に向かいます。外部入力（Presentation Adapters）は UseCases を呼び出し、Infrastructure Adapters は内側のポートを実装します。

```text
Presentation Adapters ---→ UseCases ---→ Domain
                             ↑              ↑
                             +--------------+-- Infrastructure Adapters
        implements ports owned by UseCases or Domain
```

```mermaid
graph TD
    subgraph AdaptersLayer [Adapters]
        subgraph PresentationAdapters [Presentation Adapters - 入力側]
            Web[Web / gRPC / CLI]
            Controller[Controller / Handler]
            Presenter[Presenter]
        end
        subgraph InfraAdapters [Infrastructure Adapters - 出力側]
            RI_Impl[Repository Impl]
            GW_Impl[Gateway Impl]
            DB[(Database)]
        end
    end

    subgraph UseCaseLayer [UseCases]
        UC[UseCase]
        UC_Port[UseCase Port]
    end

    subgraph DomainLayer [Domain]
        DS[Domain Service]
        E[Entity]
        RI[Repository Interface]
        DE[Domain Error]
    end

    %% 入力側の依存
    Web --> Controller
    Controller --> UC
    UC --> DomainLayer

    %% 出力側の依存
    RI_Impl -- "implements" --> RI
    RI_Impl --> DB
    GW_Impl --> DB

    %% UseCase ポート
    UC --> UC_Port
    Presenter -- "implements" --> UC_Port
```

---

## オリジナルアーキテクチャとの対応

| オリジナル Clean Architecture |                 このバリアントでの位置づけ                |
|:-----------------------------:|:---------------------------------------------------------:|
|            Entities           |                           Domain                          |
|           Use Cases           |                          UseCases                         |
|       Interface Adapters      |         Adapters（Presentation + Infrastructure）         |
|      Frameworks & Drivers     | Adapters と Composition Root が使用する具体的なメカニズム |

| Hexagonal Architecture            | このバリアントでの位置づけ |
| --------------------------------- | -------------------------- |
| Driving（Inbound）Adapters        | Presentation Adapters      |
| Application（Ports & Use Cases）  | UseCases                   |
| Driven（Outbound）Adapters        | Infrastructure Adapters    |
| Domain                            | Domain                     |

名前は異なりますが、主なルールは同じです：**ソースコードの依存関係は常に内側の高位ポリシーに向かいます。**

---

## レイヤーの定義

### 1. Domain（ドメイン層）

ビジネスルールそのものを表現する、アプリケーションの心臓部です。

* **Entity / Aggregate / Value Object:** ビジネス上の「物」や「概念」。
* **Domain Service:** 複数のエンティティに跨る知識やロジック。
* **Domain Error:** ドメイン語彙で定義されたエラー。
* **Domain Port（Interface）:** 抽象がドメイン言語の一部である場合にのみ、リポジトリ等のポートを定義。
* **外部依存なし:** DB、HTTP、ORM、SDK、Web フレームワーク、生成されたトランスポート型への依存なし。

### 2. UseCases（ユースケース層）

アプリケーションとしての具体的な「機能」を実現するための手順を記述します。**オーケストレーションのみ。**

* **役割:** Domain 層のオブジェクトと境界インターフェースを調整し、直接的な SQL / HTTP / ファイル / SDK 呼び出しを行わない。
* **入出力境界**とアプリケーション DTO を定義。
* **ポート**を定義（アプリケーション固有の外部機能に対する UseCase Port）。
* **トランザクション境界**とアプリケーションポリシー（リトライ、冪等性、認可の判断）を制御。
* **依存先:** Domain のみ。具象 Adapter、Web フレームワーク、DB ドライバへの直接依存なし。

### 3. Adapters（アダプター層）

**Adapter = 副作用の境界。** すべての I/O、外部統合、フレームワークとの相互作用はここに閉じ込めます。

#### 3a. Presentation Adapters（入力 / 駆動側）

アプリケーションの動作をユーザーや外部呼び出し元に届けるアダプター。

* **Controller / Handler:** HTTP リクエストや CLI 引数を UseCase の入力形式に変換し、UseCase を呼び出します。
* **Presenter:** UseCase の実行結果を外部向けのレスポンス形式（JSON など）に整形します。
* **Request/Response DTO:** トランスポート固有のデータ構造。
* **認証ミドルウェア:** 認証・認可のエントリポイントチェックがデリバリー関心事である場合。

#### 3b. Infrastructure Adapters（出力 / 被駆動側）

UseCases または Domain が所有するポートを外部システムに接続するアダプター。

* **Repository Impl:** Domain 層や UseCases 層で定義されたインターフェースを実装します。データのマッピングやクエリ組み立て、DB エラーの変換を行います。
* **Gateway Impl:** 外部 API クライアント、通知、検索、決済などの実装。
* **エラー変換:** ドライバエラー（例: `sql.ErrNoRows`）を内側の層に渡す前にドメインエラーに変換。

---

## 依存マトリクス

|         依存元 ↓ →         |  Domain | UseCases | Adapters |
|:--------------------------:|:-------:|:--------:|:--------:|
|           Domain           |   yes   |    no    |    no    |
|          UseCases          |   yes   |    yes   |    no    |
|  Adapters（Presentation）  | limited |    yes   |   self   |
| Adapters（Infrastructure） |   yes   |    yes   |   self   |
|      Composition Root      |   yes   |    yes   |    yes   |

* `Presentation → Domain` は `limited`: Presentation は UseCases から返された Domain の値をシリアライズのために **読み取ってもよい** が、ワークフローの判断のために Domain の振る舞いを **直接呼び出してはならない**。
* Presentation Adapters と Infrastructure Adapters は同じ概念層にありますが、互いに直接依存してはなりません。

---

## 実装例（Go）

「ユーザーが特定のグループに所属しているか」を判定するシンプルな機能を例に、各レイヤーの実装イメージを示します。

### 1. Domain 層

ビジネスルール（インターフェース）を定義します。

```go
// internal/domain/membership.go
package domain

import "context"

// MembershipRepository は、データソースに対する抽象的な問い合わせを定義します。
type MembershipRepository interface {
	IsMember(ctx context.Context, userID, groupID string) (bool, error)
}
```

### 2. UseCases 層

ビジネスの「手順」を定義します。Domain のインターフェースを利用します。

```go
// internal/usecase/membership.go
package usecase

import (
	"context"

	"your-project/internal/domain"
)

// MembershipUseCase は、ユースケースの具体的な実行者です。
type MembershipUseCase struct {
	repo domain.MembershipRepository
}

func NewMembershipUseCase(r domain.MembershipRepository) *MembershipUseCase {
	return &MembershipUseCase{repo: r}
}

// Execute は「所属確認」というユースケースを実行します。
func (uc *MembershipUseCase) Execute(ctx context.Context, userID, groupID string) (bool, error) {
	// 必要に応じてここでドメイン固有のバリデーションなどを行う。
	return uc.repo.IsMember(ctx, userID, groupID)
}
```

### 3. Infrastructure Adapters 層

Domain 層で定義されたインターフェース（Port）を具体的に実装します。DB ドライバなどの詳細はここから分離し、上位レイヤーが特定の技術に依存しないようにします。

```go
// internal/adapters/infra/persistence/membership_repository.go
package persistence

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLMembershipRepository は、SQL データベースを使用したリポジトリの実装です。
type SQLMembershipRepository struct {
	db *sql.DB
}

func NewSQLMembershipRepository(db *sql.DB) *SQLMembershipRepository {
	return &SQLMembershipRepository{db: db}
}

// IsMember は実際のデータベースに対して SQL を発行します。
func (r *SQLMembershipRepository) IsMember(ctx context.Context, userID, groupID string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM memberships WHERE user_id = ? AND group_id = ?)"
	err := r.db.QueryRowContext(ctx, query, userID, groupID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("query membership: %w", err) // ドライバエラーを変換
	}
	return exists, nil
}
```

---

## 外部からの呼び出し (REST / gRPC)

Web フレームワークや gRPC サーバーは最外周に位置し、`UseCase` を呼び出す役割のみを担います。ハンドラは **UseCase のインターフェース（入力ポート）** に依存させ、実装を差し替え可能にします。

```go
// internal/usecase/membership_port.go
package usecase

import "context"

// MembershipChecker は入力ポート（UseCaseの公開API）です。
type MembershipChecker interface {
	Execute(ctx context.Context, userID, groupID string) (bool, error)
}
```

```go
// internal/adapters/presentation/http/handler.go
package http

import (
	"encoding/json"
	"net/http"

	"your-project/internal/usecase"
)

// Web ハンドラでの使用例（Composition Root が useCase を注入）
func HandleCheckMembership(w http.ResponseWriter, r *http.Request, useCase usecase.MembershipChecker) {
	userID := r.URL.Query().Get("id")
	groupID := r.URL.Query().Get("group")

	// UseCase を実行
	isMember, err := useCase.Execute(r.Context(), userID, groupID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 結果をレスポンスとして返す
	json.NewEncoder(w).Encode(map[string]bool{"is_member": isMember})
}
```

> **注意**: DB インスタンスと UseCase の構築は Composition Root（`main.go`）で行います。ハンドラは UseCase を依存として受け取り、リクエスト/レスポンスの変換のみを担当します。

### context.Context の役割

Go の実装例で登場する `ctx context.Context` は、主に以下の目的で各レイヤーをバケツリレーします。

1. **キャンセルの伝搬:** ユーザーがブラウザを閉じた際などに、実行中の DB クエリなどを即座に中断させ、リソースの無駄を防ぎます。
2. **タイムアウト管理:** 「API 全体で 5 秒以内」といった制限時間を、末端の DB 操作まで伝えます。
3. **トレーシング:** リクエスト ID などを運び、ログから一連の処理を追跡可能にします。

**キャンセルの実装例:**

`context` のキャンセルシグナルは `Done()` チャネルを通じて伝搬します。長い処理を行う場合は、定期的にこのチャネルを確認する必要があります。

```go
func LongRunningProcess(ctx context.Context) error {
	select {
	case <-ctx.Done():
		// キャンセルされた場合（タイムアウト含む）
		return ctx.Err()
	default:
		// 処理を続行
	}
	// ... 重い処理 ...
	return nil
}
```

#### ctx と引数の使い分け

* **引数で渡すもの:** `userID` や `groupID` などの **ビジネスロジックに不可欠なデータ** です。型安全性を保ち、関数の依存関係を明確にするために、明示的に引数として渡します。
* **ctx に含めるもの:** `Request ID` や `認証トークン` などの **横断的（付加的）な情報** です。ビジネスロジックの本質ではないが、ログ出力やインフラ層での認可などに必要な情報を運びます。

**context でのリソースの密輸禁止:** `sql.Tx`、DB ハンドル、リクエストオブジェクト、SDK クライアントなどを `context.Context` に隠してアーキテクチャ境界を越えてはなりません。明示的な関数パラメータや依存注入を使用してください。

---

## Port 所有権のガイダンス

* **Domain Port:** 抽象が「ドメイン言語」の一部であり、ドメインモデルがコアビジネスルールを満たすために不可欠な場合。
  * *例:* `UserRepository.FindByID`（エンティティの再構成に不可欠）。
  * *判断基準:* 「この能力なしではドメインモデルが不完全か、不変条件を強制できないか？」
* **UseCase Port:** 抽象がアプリケーション固有の手順を完了するための「道具」である場合。
  * *例:* `NotificationGateway.SendWelcomeEmail`（ワークフローの副作用）、`IdentityGateway.IsMember`（外部プロバイダに対する認可チェック）。
  * *判断基準:* 「これはコアビジネスロジックではなく、アプリケーションワークフロー要件か？」
* 具象実装は、どの内側レイヤーがインターフェースを所有していても **Infrastructure Adapters** に置きます。

## ポート設計とリポジトリ境界

* **入力ポート（Input Port）:** Presentation Adapters（Web/CLI/Batch）から呼ばれる UseCase のインターフェースです。Controller はこのポートに依存します。
* **出力ポート（Output Port）:** Domain / UseCases が外側に要求する契約です（例: Repository）。インターフェースは内側に置き、実装は Infrastructure Adapters 側に置きます。
* **リポジトリ境界:** Repository は「永続化の契約」です。トランザクションやリトライなどの制御は UseCases、データ変換とクエリ組み立ては Infrastructure Adapters が担当します。

## エラー境界ルール

* **Infrastructure Adapters** はドライバエラー（例: `sql.ErrNoRows`）を内側の層に渡す前にドメインエラーに変換します。
* **Domain / UseCases** のエラーはビジネスやアプリケーションの意味を持ち、HTTP ステータスコードや SQL 固有のエラーを含みません。
* **Presentation Adapters** はアプリケーションエラーをトランスポートエラー（HTTP ステータス、gRPC ステータス、CLI 終了コード）に変換します。

## データ境界ルール

* **UseCase の入出力** は、内側のポリシーをトランスポートや永続化の詳細から保護するために明示的に定義すべきです。
* **Domain オブジェクト** は Presentation DTO や ORM レコードではありません。トランスポートアノテーション、ORM タグ、生成された API 型をドメインオブジェクトに含めないでください。
* **マッピングの責務** は一貫させます: Presentation はリクエスト/レスポンスを変換、Infrastructure は永続化/外部データを変換、UseCases はアプリケーション DTO を変換（任意）。

## アンチパターン

* Domain が DB / HTTP / ORM / SDK / フレームワークの型に依存している。
* UseCases が境界インターフェースに依存せず、直接 SQL / HTTP / ファイル I/O を実行している。
* Presentation が UseCases をバイパスしてビジネスワークフローや永続化の判断を直接実行している。
* Infrastructure Adapter のコードが Domain や UseCases に属するビジネス判断を所有している。
* トランスポート DTO、ORM レコード、生成された API モデルが Domain オブジェクトとして再利用されている。
* Presentation と Infrastructure Adapters がマージされ、それぞれの責務が見つけにくくなっている。

### Go 実装の補足

* **SQL プレースホルダ:** ドライバによって `?` や `$1` のように異なります。実装環境に合わせて選択します。
* **命名:** Go では `SQL` のようなイニシャリズムは大文字にまとめるのが慣例です。
* **コンテキスト:** `context` の値は型付きキーで管理し、ビジネスデータは入れません。

## 典型的なディレクトリレイアウト（Go）

```text
cmd/app/main.go                                  // composition root
internal/domain/...                               // entities, value objects, domain services, domain errors
internal/usecase/...                              // interactors, input/output DTOs, usecase-owned ports
internal/adapters/presentation/http/...          // HTTP handlers/controllers/presenters
internal/adapters/presentation/grpc/...          // gRPC handlers and transport mapping
internal/adapters/presentation/cli/...           // CLI commands and output formatting
internal/adapters/infra/persistence/...          // repository implementations, DB models, mapping
internal/adapters/infra/external/...             // external API gateway implementations
```

各プロジェクトは、依存ルールと責務の境界が保たれる限り、このレイアウトに適応しても構いません。
