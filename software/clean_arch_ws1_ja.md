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
// domain/user.go
package domain

import "time"

type User struct {
 ID       string
 Name     string
 JoinedAt time.Time // 入社日
}

// GetTenureYears は勤続年数を返します
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
// domain/veteran_service.go
package domain

type VeteranService struct{}

// IsVeteran は、ユーザーがベテラン（勤続5年以上）かどうかを判定します
func (s *VeteranService) IsVeteran(user *User) bool {
 return user.GetTenureYears() >= 5
}
```

### 1-3. UseCase の実装

Domain オブジェクトを組み合わせてユースケースを実現します。

```go
// usecase/check_veteran.go
package usecase

import (
 "context"
 "your-project/domain"
)

type CheckVeteranUseCase struct {
 repo       domain.UserRepository
 veteranSvc domain.VeteranService
}

func (uc *CheckVeteranUseCase) Execute(ctx context.Context, id string) (bool, error) {
 user, err := uc.repo.FindByID(ctx, id) // リポジトリ経由で取得
 if err != nil {
  return false, err
 }
 return uc.veteranSvc.IsVeteran(user), nil
}
```

---

## 課題 2: インフラの差し替え (SQL → Active Directory)

急な方針変更で、ユーザー情報の取得先が DB ではなく Active Directory (AD) になりました。

### 2-1. 新しいインフラ実装の追加

`domain.UserRepository` インターフェースを満たす AD 用のリポジトリを作成します。

```go
// infra/ad_user_repository.go
package infra

type AdUserRepository struct {
 ldapClient *LdapClient // 架空のライブラリ
}

func (r *AdUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
 // LDAP クエリを発行して情報を取得
 entry, _ := r.ldapClient.Search(id)
 return &domain.User{
  ID:       entry.UID,
  Name:     entry.DisplayName,
  JoinedAt: entry.CreationDate,
 }, nil
}
```

### 2-2. 依存性の注入 (DI) による切り替え

メインの処理（エントリポイント）で、注入する具象クラスを入れ替えるだけです。

```go
func main() {
 // 旧: sqlRepo := infra.NewSqlUserRepository(db)
 // 新:
 adRepo := infra.NewAdUserRepository(ldapClient)

 // UseCase は引数がインターフェースなので、adRepo をそのまま受け入れられる
 useCase := usecase.NewCheckVeteranUseCase(adRepo)

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
