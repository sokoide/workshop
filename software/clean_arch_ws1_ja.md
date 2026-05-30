# クリーンアーキテクチャ実習 (WS1): 変更に強い設計を学ぶ

この実習では、既存のサンプルを拡張しながら、ビジネスルールの追加とインフラの差し替えを行い、クリーンアーキテクチャの利点を体験します。

## 実習のシナリオ

1. **ビジネスルールの追加**: 「勤続 5 年以上の社員をベテランと認定する」というルールを、Entity と Domain Service を使って実装します。
2. **インフラの差し替え**: データの保存先を SQL データベースから Active Directory (LDAP) に変更します。この際、UseCase や Domain に一切手を加えないことを確認します。

---

## 課題 1: Entity と Domain Service の実装

「ベテラン判定」というビジネス知識を Domain 層に閉じ込めます。

### 1-1. Entity の作成

ユーザーの属性と、勤続年数を計算する知識を持たせます。

```go
// internal/domain/user.go
package domain

import "time"

type User struct {
	ID       string
	Name     string
	JoinedAt time.Time // 入社日
}

// GetTenureYears は勤続年数を返します。
func (u *User) GetTenureYears() int {
	now := time.Now()
	years := now.Year() - u.JoinedAt.Year()
	if now.YearDay() < u.JoinedAt.YearDay() {
		years--
	}
	return years
}
```

### 1-2. Domain Service の作成

「ベテランとは何か？」という判定基準は、Entity 自身よりも「サービス」として定義するのが適切です。

```go
// internal/domain/veteran_service.go
package domain

type VeteranService struct{}

// IsVeteran は、ユーザーがベテラン（勤続5年以上）かどうかを判定します。
func (s VeteranService) IsVeteran(user *User) bool {
	return user.GetTenureYears() >= 5
}
```

### 1-3. UseCase の実装

Domain オブジェクトを組み合わせてユースケースを実現します。

```go
// internal/usecase/check_veteran.go
package usecase

import (
	"context"

	"your-project/internal/domain"
)

type CheckVeteranUseCase struct {
	repo       domain.UserRepository
	veteranSvc domain.VeteranService
}

func NewCheckVeteranUseCase(repo domain.UserRepository, veteranSvc domain.VeteranService) *CheckVeteranUseCase {
	return &CheckVeteranUseCase{repo: repo, veteranSvc: veteranSvc}
}

func (uc *CheckVeteranUseCase) Execute(ctx context.Context, id string) (bool, error) {
	user, err := uc.repo.FindByID(ctx, id) // リポジトリ経由で取得
	if err != nil {
		return false, err
	}
	return uc.veteranSvc.IsVeteran(user), nil
}
```

> **インターフェース設計の補足:** UseCase は `VeteranChecker` インターフェースを介さず、`domain.VeteranService` という具象型に直接依存しています。これは **シンプルさと疎結合のトレードオフ** です。`VeteranService` は外部依存のない純粋な Domain Service であり、将来不当な副作用を獲得するリスクが低いため、余分なインターフェース層は実益のない儀式（ceremony）になります。デメリットは、`VeteranService` のメソッドシグネチャが変更された場合、UseCase も変更が必要になることです。後に外部依存を獲得する可能性のある Domain Service（例: API を呼び出す可能性があるルックアップサービス）では、最初からインターフェースを定義する方が安全な選択です。

---

## 課題 2: インフラの差し替え (SQL → Active Directory)

急な方針変更で、ユーザー情報の取得先が DB ではなく Active Directory (AD) になりました。

### 2-1. 新しいインフラ実装の追加

`domain.UserRepository` インターフェースを満たす AD 用のリポジトリを作成します。

```go
// internal/adapters/infra/ad_user_repository.go
package infra

import (
	"context"
	"fmt"

	"your-project/internal/domain"
)

type ADUserRepository struct {
	ldapClient *LDAPClient // 架空のライブラリ
}

func (r *ADUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	// LDAP クエリを発行して情報を取得
	entry, err := r.ldapClient.Search(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("ldap search user %s: %w", id, err) // ドライバエラーを変換
	}
	return &domain.User{
		ID:       entry.UID,
		Name:     entry.DisplayName,
		JoinedAt: entry.CreationDate,
	}, nil
}
```

> **エンティティの再構成に関する補足（ベア構造体リテラルの許容）:**  
> クリーンアーキテクチャでは、ドメインエンティティの不変条件（Invariants）を守るために UseCases 層などで `&domain.User{}` のように直接構造体を生成することを禁止しています（通常はバリデーションを行う `NewUser` などのコンストラクタを使用します）。  
> しかし、Infrastructure Adapters でデータベースや外部サービスから既存のエンティティを復元する「再構成（Reconstruction）」においては、すでに妥当とみなされて保存されているデータを復元するため、例外的にベア構造体リテラルでの生成が許容されます。
>
> ※ なお、ドメイン設計でフィールドを非公開（小文字で定義）にしてカプセル化している場合、パッケージ外からベア構造体リテラルは使用できません。この場合は、ドメインパッケージ内に再構成専用の関数（例: `domain.ReconstructUser(...)` などの Reconstructor パターン）を用意し、安全にエンティティを復元します。

### 2-2. 依存性の注入 (DI) による切り替え

メインの処理（エントリポイント）で、注入する具象クラスを入れ替えるだけです。

```go
func main() {
	// 旧: sqlRepo := persistence.NewSQLUserRepository(db)
	// 新:
	adRepo := infra.NewADUserRepository(ldapClient)

	// UseCase は引数がインターフェースなので、adRepo をそのまま受け入れられる
	useCase := usecase.NewCheckVeteranUseCase(adRepo, domain.VeteranService{})

	// この後、useCase.Execute() を呼び出すコードは一切変更不要！
}
```

---

## この実習のポイント

1. **知識の場所**:
    - 勤続年数の計算方法 ＝ **Entity**
    - ベテランの定義（5 年） ＝ **Domain Service**
    - これらは「データベース」や「Web」とは無関係にテスト可能です。
2. **変更の局所化**:
    - データの取得先が DB から AD に変わっても、`infra` レイヤーに新しいコードを追加し、`main` での注入先を変えるだけで済みました。
    - **核心となるビジネスロジック (Domain/UseCase) には 1 行も修正が入っていません。** これがクリーンアーキテクチャの真価です。
3. **ポートと境界**:
    - `domain.UserRepository` は **出力ポート** であり、実装は Infra Adapters (インフラアダプター層) に置きます。
    - `LDAPClient` や DB ドライバなどの詳細は Infra Adapters (インフラアダプター層) に留め、Domain/UseCase に漏らしません。

---

## 理解度チェック

以下の問いに自分で答えてみてください。正解は一つではありませんが、根拠を持って説明できるようになることが目標です。

### 問 1: Domain Service のインターフェース化
`VeteranService` は具象型に直接依存しています。なぜインターフェース化しなかったのでしょうか？逆に、**必ず**インターフェース化すべき場面はどんなときですか？

### 問 2: 変更の影響範囲
もし「ベテランの定義が 5 年から 3 年に変わった」場合、どの層を修正する必要がありますか？逆に、どの層は**一切**触る必要がありませんか？

### 問 3: ベア構造体リテラルのリスク
`ADUserRepository.FindByID` で `&domain.User{...}` のようにベア構造体リテラルを使っています。ドメインの不変条件（例：ID は空文字であってはならない）を守るため、どのような設計を追加できますか？
