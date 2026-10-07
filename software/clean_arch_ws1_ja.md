# クリーンアーキテクチャ実習 (WS1): 変更に強い設計を学ぶ

> **⏱️ 所要時間**: 約 60 分
> **🎯 レベル**: Beginner
> **💻 環境**: Go 1.23+、OS 不問（外部依存ゼロ）
>
> > **💡 用語集**: [クリーンアーキテクチャ](../infra/glossary_ja.md#architecture)や[インターフェース](../infra/glossary_ja.md#go)などの専門用語は [用語集](../infra/glossary_ja.md) を参照してください。

この実習では、既存のサンプルを拡張しながら、ビジネスルールの追加とインフラの差し替えを行い、クリーンアーキテクチャの利点を体験します。

## 実習のシナリオ

1. **ビジネスルールの追加**: 「勤続 5 年以上の社員をベテランと認定する」というルールを、Entity と Domain Service を使って実装します。
2. **インフラの差し替え**: データの保存先を SQL データベースから Active Directory (LDAP) に変更します。この際、UseCase や Domain に一切手を加えないことを確認します。

## 前提知識

- [ ] Go 1.23 以上がインストール済み（`go version` で確認）
- [ ] [クリーンアーキテクチャガイド](./clean_arch.md)（3 層バリアント: Adapters / UseCases / Domain）を読んでいる
- [ ] 作業用ワークスペースの準備（次節を参照）

### ワークスペースの準備

この実習の完全な参照実装（実行可能）は [`software/assets/clean_arch_ws1/`](./assets/clean_arch_ws1/) にあります。行き詰まったときや完成形を確認したいときは、いつでも `go run .` で実行して比べることができます。

以下のどちらかの方法で進めてください:

```bash
# 方法 A: 参照実装をコピーして改造する（クイックスタート）
cp -r software/assets/clean_arch_ws1 ~/ws1 && cd ~/ws1

# 方法 B: 空のモジュールから始めて自分で書く
mkdir ws1 && cd ws1
go mod init example.com/ws1
mkdir -p internal/domain internal/usecase internal/adapters/infra
```

> 以下のコード断片は参照実装のモジュールパス `github.com/sokoide/workshop/software/assets/clean_arch_ws1` を使います。方法 B を選んだ場合は、`go mod init` に渡したモジュールパスに読み替えてください。

---

## 課題 1: Entity と Domain Service の実装

「ベテラン判定」というビジネス知識を Domain 層に閉じ込めます。

### 1-1. Entity の作成

ユーザーの属性と、勤続年数を計算する知識を持たせます。

```go
// internal/domain/user.go
package domain

import "time"

// User is the user entity.
type User struct {
	ID       string
	Name     string
	JoinedAt time.Time // Joining date
}

// GetTenureYears returns the number of years of service as of now.
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

// VeteranService decides who counts as a veteran.
type VeteranService struct{}

// IsVeteran determines if a user is a veteran (5 or more years of service).
func (s VeteranService) IsVeteran(user *User) bool {
	return user.GetTenureYears() >= 5
}
```

### 1-3. リポジトリポートの定義

UseCase はリポジトリ経由でユーザーを取得するため、まず Domain 層にポート（インターフェース）を定義します。実装は Infra Adapters 層に置きます — これが課題 2 のインフラ差し替えを可能にする境目（シーム）です。（リポジトリポートは、永続化をアプリケーションオーケストレーションと捉える場合には UseCase 層に置く流儀もあります — BBS シリーズの実習はそちらです。基準は [Port 所有権のガイダンス](./clean_arch_ja.md#port-所有権のガイダンス)。本実習では `UserRepository` がこのドメインの語彙の一部であるため、契約を Domain に置いています。）

```go
// internal/domain/user_repository.go
package domain

import "context"

// UserRepository is the output port for user persistence. The UseCase layer
// depends on this interface — never on SQL drivers, LDAP clients, or any
// other infrastructure detail.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
}
```

```go
// internal/domain/errors.go
package domain

import "errors"

// ErrUserNotFound is returned when the repository cannot find the user.
// It belongs to the Domain layer so every caller can test for it with
// errors.Is, no matter which infrastructure produced it.
var ErrUserNotFound = errors.New("user not found")
```

### 1-4. UseCase の実装

Domain オブジェクトを組み合わせてユースケースを実現します。

```go
// internal/usecase/check_veteran.go
package usecase

import (
	"context"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
)

// CheckVeteranUseCase answers "is this user a veteran?".
type CheckVeteranUseCase struct {
	repo       domain.UserRepository
	veteranSvc domain.VeteranService
}

func NewCheckVeteranUseCase(repo domain.UserRepository, veteranSvc domain.VeteranService) *CheckVeteranUseCase {
	return &CheckVeteranUseCase{repo: repo, veteranSvc: veteranSvc}
}

func (uc *CheckVeteranUseCase) Execute(ctx context.Context, id string) (bool, error) {
	user, err := uc.repo.FindByID(ctx, id) // Retrieved via repository
	if err != nil {
		return false, err
	}
	return uc.veteranSvc.IsVeteran(user), nil
}
```

> **インターフェース設計の補足:** UseCase は `VeteranChecker` インターフェースを介さず、`domain.VeteranService` という具象型に直接依存しています。これは **シンプルさと疎結合のトレードオフ** です。`VeteranService` は外部依存のない純粋な Domain Service であり、将来不当な副作用を獲得するリスクが低いため、余分なインターフェース層は実益のない儀式（ceremony）になります。デメリットは、`VeteranService` のメソッドシグネチャが変更された場合、UseCase も変更が必要になることです。後に外部依存を獲得する可能性のある Domain Service（例: API を呼び出す可能性があるルックアップサービス）では、最初からインターフェースを定義する方が安全な選択です。

### ✅ チェックポイント

- [ ] `internal/domain/user.go` と `internal/domain/veteran_service.go` がコンパイルできる — `go build ./...` が何も出力せずに終了する
- [ ] `CheckVeteranUseCase` が `infra` パッケージを **一切 import せずに** コンパイルできる
- [ ] DB や HTTP のセットアップなしに `VeteranService.IsVeteran` の単体テストを書ける

確認しましょう — Domain 層は DB や HTTP のセットアップなしに単体テストできます:

```bash
go test ./internal/domain/
# Expected output: ok  github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain  0.007s
```

---

## 課題 2: インフラの差し替え (SQL → Active Directory)

急な方針変更で、ユーザー情報の取得先が DB ではなく Active Directory (AD) になりました。

### 2-1. 出発点: SQL リポジトリ（スタンドイン）

まずは _現在の_ 実装として、SQL データベース背後のリポジトリを用意します。参照実装では本物の DB の代わりに in-memory のスタンドインを使います — 外部からは SQL リポジトリとまったく同じ振る舞い（クエリ、Entity の再構成、ミス時にはドメインエラーを返す）をするため、実習は **外部依存ゼロ** で動きます。

```go
// internal/adapters/infra/sql_user_repository.go
package infra

import (
	"context"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
)

// userRow mimics one row of the users table.
type userRow struct {
	ID       string
	Name     string
	JoinedAt time.Time
}

// memoryDB is a stand-in for a real SQL database. It keeps rows in a map so
// the workshop runs with zero external dependencies.
type memoryDB struct {
	users map[string]userRow
}

// seedUsers keeps the sample dataset in one place. Joining dates are relative
// to "now" so the veteran verdicts never go stale: alice has 10 years of
// service (veteran), bob has 1 year (not yet a veteran).
func seedUsers() map[string]userRow {
	now := time.Now()
	return map[string]userRow{
		"alice": {ID: "alice", Name: "Alice", JoinedAt: now.AddDate(-10, 0, 0)},
		"bob":   {ID: "bob", Name: "Bob", JoinedAt: now.AddDate(-1, 0, 0)},
	}
}

// SQLUserRepository behaves like a repository backed by a SQL database: it
// "queries" the in-memory users table and reconstitutes domain entities.
type SQLUserRepository struct {
	db *memoryDB
}

func NewSQLUserRepository() *SQLUserRepository {
	return &SQLUserRepository{db: &memoryDB{users: seedUsers()}}
}

// Compile-time check that SQLUserRepository satisfies the domain port.
var _ domain.UserRepository = (*SQLUserRepository)(nil)

// FindByID pretends to run:
//
//	SELECT id, name, joined_at FROM users WHERE id = ?
func (r *SQLUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	row, ok := r.db.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return &domain.User{
		ID:       row.ID,
		Name:     row.Name,
		JoinedAt: row.JoinedAt,
	}, nil
}
```

### 2-2. LDAP クライアントのスタブ

新しいインフラは LDAP クライアントライブラリ経由で Active Directory と通信します。参照実装には本物の代わりに **スタブ** を同梱します: 本物と同じ振る舞い（`Search` が UID をディレクトリエントリに解決するか、ドライバ風のエラーを返す）をしつつ、エントリはメモリに保持します。後で本物のライブラリに差し替えるときも、このファイルだけ触れば済みます。

```go
// internal/adapters/infra/ldap_client.go
package infra

import (
	"context"
	"errors"
	"time"
)

// LDAPEntry mimics an entry returned by an LDAP search.
type LDAPEntry struct {
	UID          string
	DisplayName  string
	CreationDate time.Time
}

// errNoSuchObject mimics the error a real LDAP library returns when the
// search base holds no matching entry.
var errNoSuchObject = errors.New("ldap: no such object")

// LDAPClient is a stub of a real LDAP client library (such as go-ldap/ldap).
// It keeps its directory entries in memory so the workshop has zero external
// dependencies, while still behaving like the real thing: Search resolves a
// UID to an entry or returns a driver-style error.
type LDAPClient struct {
	entries map[string]LDAPEntry
}

func NewLDAPClient(entries map[string]LDAPEntry) *LDAPClient {
	return &LDAPClient{entries: entries}
}

// Search resolves a UID to a directory entry.
func (c *LDAPClient) Search(ctx context.Context, uid string) (LDAPEntry, error) {
	entry, ok := c.entries[uid]
	if !ok {
		return LDAPEntry{}, errNoSuchObject
	}
	return entry, nil
}
```

### 2-3. 新しいインフラ実装の追加

同じ `domain.UserRepository` インターフェースを満たす AD 用のリポジトリを作成します。

```go
// internal/adapters/infra/ad_user_repository.go
package infra

import (
	"context"
	"errors"
	"fmt"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
)

// ADUserRepository reads users from Active Directory via an LDAP client.
type ADUserRepository struct {
	ldapClient *LDAPClient // stubbed LDAP library
}

func NewADUserRepository(ldapClient *LDAPClient) *ADUserRepository {
	return &ADUserRepository{ldapClient: ldapClient}
}

// Compile-time check that ADUserRepository satisfies the same domain port as
// the SQL implementation.
var _ domain.UserRepository = (*ADUserRepository)(nil)

func (r *ADUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	// Issue the LDAP query to retrieve the entry
	entry, err := r.ldapClient.Search(ctx, id)
	if err != nil {
		if errors.Is(err, errNoSuchObject) {
			// Translate the driver-specific "not found" into a domain error
			// so the UseCase layer never sees LDAP details.
			return nil, fmt.Errorf("ldap search user %s: %w", id, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("ldap search user %s: %w", id, err) // Convert driver error
	}
	// Reconstruct the entity from the directory entry (bare struct literal
	// is allowed here because we restore persisted state, not create new).
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

### 2-4. 依存性の注入 (DI) による切り替え

`main.go` は Composition Root（合成ルート）— 具象実装を知っている唯一の場所です。両方のデータソースに対して同じベテラン判定を実行し、差し替えの様子を確認できます:

```go
// main.go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/adapters/infra"
	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/usecase"
)

func main() {
	ctx := context.Background()

	// Phase 1: users are served by the SQL database.
	checkWith(ctx, "SQL database", infra.NewSQLUserRepository())

	// Phase 2: policy change — the user directory moved to Active Directory.
	// Only this wiring changes; internal/domain and internal/usecase are untouched.
	now := time.Now()
	directory := infra.NewLDAPClient(map[string]infra.LDAPEntry{
		"alice": {UID: "alice", DisplayName: "Alice", CreationDate: now.AddDate(-10, 0, 0)},
		"bob":   {UID: "bob", DisplayName: "Bob", CreationDate: now.AddDate(-1, 0, 0)},
	})
	checkWith(ctx, "Active Directory (LDAP)", infra.NewADUserRepository(directory))
}

// checkWith wires the given repository into the use case and prints the
// veteran verdicts. It works with ANY implementation of domain.UserRepository.
func checkWith(ctx context.Context, source string, repo domain.UserRepository) {
	fmt.Printf("== data source: %s ==\n", source)
	uc := usecase.NewCheckVeteranUseCase(repo, domain.VeteranService{})
	for _, id := range []string{"alice", "bob"} {
		veteran, err := uc.Execute(ctx, id)
		if err != nil {
			fmt.Printf("  %s: error: %v\n", id, err)
			continue
		}
		fmt.Printf("  %s: veteran=%v\n", id, veteran)
	}
}
```

UseCase は引数がインターフェースなので、`adRepo` をそのまま受け入れられます — エントリポイントで注入する具象クラスを入れ替えるだけです。実行してみましょう:

```bash
go run .
# Expected output:
# == data source: SQL database ==
#   alice: veteran=true
#   bob: veteran=false
# == data source: Active Directory (LDAP) ==
#   alice: veteran=true
#   bob: veteran=false
```

両方のデータソースで判定結果は同一です — ビジネスロジックは差し替えに気づいていません。

### ✅ チェックポイント

- [ ] `ADUserRepository` が `domain.UserRepository` を満たしている — `var _ domain.UserRepository = (*infra.ADUserRepository)(nil)` がコンパイルできる
- [ ] データ取得先の差し替えで変更したのは `main.go`（Composition Root）だけ
- [ ] `go run .` が SQL / LDAP 両データソースで同一のベテラン判定結果を出力する
- [ ] AD リポジトリを配線した状態でも `go build ./...` が通り、`internal/domain` / `internal/usecase` は無変更のままである（`git diff` で差分がないことを確認）

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
   - `LDAPClient` スタブや DB ドライバなどの詳細は Infra Adapters (インフラアダプター層) に留め、Domain/UseCase に漏らしません。

---

## 理解度チェック

以下の問いに自分で答えてみてください。正解は一つではありませんが、根拠を持って説明できるようになることが目標です。

### 問 1: Domain Service のインターフェース化

`VeteranService` は具象型に直接依存しています。なぜインターフェース化しなかったのでしょうか？逆に、**必ず**インターフェース化すべき場面はどんなときですか？

### 問 2: 変更の影響範囲

もし「ベテランの定義が 5 年から 3 年に変わった」場合、どの層を修正する必要がありますか？逆に、どの層は**一切**触る必要がありませんか？

### 問 3: ベア構造体リテラルのリスク

`ADUserRepository.FindByID` で `&domain.User{...}` のようにベア構造体リテラルを使っています。ドメインの不変条件（例：ID は空文字であってはならない）を守るため、どのような設計を追加できますか？

---

## 学んだこと

- 「誰がベテランか」というビジネス知識は Domain 層に住む: 計算は **Entity** のメソッド、基準（5 年）は **Domain Service** として表現され、DB・HTTP・モックなしでテストできる。
- UseCase は `domain.UserRepository` という **出力ポート** にのみ依存するため、SQL から Active Directory への差し替えは `main.go` だけで完結し、Domain と UseCase は **ゼロ行変更** だった。
- Infra Adapters がベア構造体リテラルを使ってよいのは、永続化済みデータの **再構成** の場合のみ。新規生成は不変条件を守るためコンストラクタ経由にする。
- インターフェースを定義すべきかどうかは、相手が外部依存を持つ（持ち得る）かどうかで判断する。純粋な Domain Service すべてに不要なインターフェースは作らない。
- 外部システム（SQL データベースや LDAP ディレクトリ）は in-memory スタンドインやスタブで置き換えられ、実習を外部依存ゼロで実行できる。

**次**: [WS2: 拡張と最適化](./clean_arch_ws2_ja.md) — 通知の抽象化と Decorator パターンによる透過的キャッシュの導入を学びます。

---

## 参考文献

- [The Clean Architecture — Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Effective Go: Interfaces](https://go.dev/doc/effective_go#interfaces_and_types)
- [Go Documentation](https://go.dev/doc/)
