# セキュアコーディング Best Practices (Go)

日々のコーディングで作り込みやすい脆弱性と、それを防ぐための習慣を学びます。XSS や CSRF といった Web レベルの攻撃（`infra/web_security_ja.md` 参照）とは別に、**アプリケーション内部の実装** に潜むリスクを取り上げます。

---

## 目次

1. [SQL インジェクション対策](#bp1-sql-インジェクション対策)
2. [入力検証 (Input Validation)](#bp2-入力検証-input-validation)
3. [パストラバーサル対策](#bp3-パストラバーサル対策-path-traversal)
4. [パスワードの安全なハッシュ化](#bp4-パスワードの安全なハッシュ化)
5. [安全な乱数生成](#bp5-安全な乱数生成)
6. [エラー処理で情報を漏らさない](#bp6-エラー処理で情報を漏らさない)
7. [ログに機密情報を書かない](#bp7-ログに機密情報を書かない)
8. [競合状態 (Race Condition) の防止](#bp8-競合状態-race-condition-の防止)

---

## BP1: SQL インジェクション対策

ユーザー入力を SQL 文字列に直接埋め込むと、攻撃者が任意の SQL を実行できます。

### 悪い例

```go
func GetUser(db *sql.DB, username string) (*User, error) {
    // ❌ ユーザー入力を文字列結合で SQL に埋め込み
    query := fmt.Sprintf("SELECT id, name, email FROM users WHERE name = '%s'", username)
    row := db.QueryRow(query)
    // ...
}
```

**攻撃例:**

```text
入力: ' OR '1'='1' --
展開後: SELECT id, name, email FROM users WHERE name = '' OR '1'='1' --'
結果: 全ユーザーのデータが返る
```

```text
入力: '; DROP TABLE users; --
展開後: SELECT id, name, email FROM users WHERE name = ''; DROP TABLE users; --'
結果: users テーブルが削除される
```

### 改善例

```go
func GetUser(db *sql.DB, username string) (*User, error) {
    // ✅ プレースホルダーを使用（パラメータは自動的にエスケープされる）
    row := db.QueryRow("SELECT id, name, email FROM users WHERE name = $1", username)

    var u User
    err := row.Scan(&u.ID, &u.Name, &u.Email)
    if err != nil {
        return nil, err
    }
    return &u, nil
}
```

**プレースホルダーのプレースホルダー記号（RDBMS により異なる）:**

| データベース | 記号            | 例                |
|:-------------|:----------------|:------------------|
| PostgreSQL   | `$1, $2, ...`   | `WHERE name = $1` |
| MySQL        | `?`             | `WHERE name = ?`  |
| SQLite       | `?` または `$1` | `WHERE name = ?`  |

**なぜプレースホルダーが安全なのか？**

```text
プレースホルダー使用時:
  SQL: SELECT ... WHERE name = $1
  パラメータ: "' OR '1'='1' --"
  ↓
  データベースがパラメータを「データ」として扱う
  → SQL の構文として解釈されない
  → name が "' OR '1'='1' --" という文字列のユーザーを検索するだけ
  → 該当なし → 空の結果
```

---

## BP2: 入力検証 (Input Validation)

ユーザー入力を信用せず、境界（boundary）で必ず検証します。

### 悪い例

```go
func CreateUser(r *http.Request) error {
    name := r.FormValue("name")
    email := r.FormValue("email")
    age := r.FormValue("age")

    // ❌ 一切の検証なし
    _, err := db.Exec("INSERT INTO users (name, email, age) VALUES (?, ?, ?)", name, email, age)
    return err
}
```

**起こりうる問題:**

```text
name: "" (空文字)        → データ不整合
email: "not-an-email"    → 無効な形式
age: "-1"                → 不正な値
name: 1000文字の文字列    → バッファオーバーフロー / DoS
name: "<script>alert(1)</script>" → XSS (出力時の問題に直結)
```

### 改善例

```go
import (
    "fmt"
    "net/mail"
    "strconv"
    "strings"
    "unicode/utf8"
)

type CreateUserInput struct {
    Name  string
    Email string
    Age   int
}

func validateCreateUser(name, email, ageStr string) (*CreateUserInput, error) {
    var errs []string

    // 名前: 必須、長さ制限
    name = strings.TrimSpace(name)
    if name == "" {
        errs = append(errs, "name is required")
    }
    if utf8.RuneCountInString(name) > 100 {
        errs = append(errs, "name must be 100 characters or less")
    }

    // メールアドレス: 形式チェック
    if _, err := mail.ParseAddress(email); err != nil {
        errs = append(errs, "invalid email format")
    }

    // 年齢: 整数、範囲チェック
    age, err := strconv.Atoi(ageStr)
    if err != nil {
        errs = append(errs, "age must be a number")
    } else if age < 0 || age > 150 {
        errs = append(errs, "age must be between 0 and 150")
    }

    if len(errs) > 0 {
        return nil, fmt.Errorf("validation failed: %s", strings.Join(errs, ", "))
    }

    return &CreateUserInput{Name: name, Email: email, Age: age}, nil
}
```

**検証の原則:**

```text
Allowlist (許可リスト) 方式: ✅ 推奨
  「正しい形式はこれだけ」を定義し、それ以外を拒否
  例: メールアドレスの正規表現、enum 値のリスト

Blocklist (拒否リスト) 方式: ❌ 非推奨
  「危険な文字はこれ」を定義し、それ以外を許可
  例: "<script>" をブロック → "<SCRIPT>" や "<img onerror>" は通る

理由: 未知の攻撃パターンに対応できないため
```

---

## BP3: パストラバーサル対策 (Path Traversal)

ユーザー入力をファイルパスに使うと、意図しないファイルにアクセスされる可能性があります。

### 悪い例

```go
func GetAvatar(w http.ResponseWriter, r *http.Request) {
    username := r.URL.Query().Get("user")

    // ❌ ユーザー入力をそのままパスに結合
    path := filepath.Join("/var/avatars", username)
    http.ServeFile(w, r, path)
}
```

**攻撃例:**

```text
リクエスト: GET /avatar?user=../../../etc/passwd
展開後: /var/avatars/../../../etc/passwd
解決後: /etc/passwd → システムファイルが読み取られる！
```

### 改善例

```go
func GetAvatar(w http.ResponseWriter, r *http.Request) {
    username := r.URL.Query().Get("user")

    // ✅ 1. 入力を検証 (英数字のみ許可)
    if !isValidUsername(username) {
        http.Error(w, "invalid username", http.StatusBadRequest)
        return
    }

    // ✅ 2. パスを解決し、ベースディレクトリ内か確認
    basePath := "/var/avatars"
    fullPath := filepath.Join(basePath, username)
    absPath, err := filepath.Abs(fullPath)
    if err != nil {
        http.Error(w, "invalid path", http.StatusBadRequest)
        return
    }

    absBase, _ := filepath.Abs(basePath)
    if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) {
        http.Error(w, "access denied", http.StatusForbidden)
        return
    }

    http.ServeFile(w, r, absPath)
}

func isValidUsername(s string) bool {
    for _, r := range s {
        if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
            return false
        }
    }
    return len(s) > 0 && len(s) <= 64
}
```

**なぜ filepath.Join だけでは不十分なのか？**

```text
filepath.Join("/var/avatars", "../../../etc/passwd")
  → /var/avatars/../../../etc/passwd
  → filepath.Abs で解決
  → /etc/passwd  ← ベースディレクトリの外！

strings.HasPrefix で「解決後の絶対パスがベースディレクトリで始まるか」を
確認することで、トラバーサルを防ぐ
```

---

## BP4: パスワードの安全なハッシュ化

パスワードを平文や単方向ハッシュ (MD5, SHA) で保存すると、漏洩時に簡単に元のパスワードが復元されます。

### 悪い例

```go
import "crypto/sha256"

func HashPassword(password string) string {
    // ❌ SHA-256 は高速なハッシュ → ブルートフォース攻撃に弱い
    // ❌ ソルトなし → 同じパスワードが同じハッシュになる
    h := sha256.Sum256([]byte(password))
    return fmt.Sprintf("%x", h)
}
```

**攻撃例 (レインボーテーブル):**

```text
SHA-256("password123") → ef92b778... (常に同じ)
  ↓
攻撃者がハッシュを入手
  ↓
レインボーテーブル (事前計算済みハッシュ辞書) で検索
  ↓
"password123" を即座に特定
```

### 改善例

```go
import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
    // ✅ bcrypt は計算コストが高い (ブルートフォースに強い)
    // ✅ ソルトが自動的に付与される (同じパスワードでも異なるハッシュ)
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }
    return string(bytes), nil
}

func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

**なぜ bcrypt が安全なのか？**

```text
MD5 / SHA-256:
  1回のハッシュ計算: < 1μs
  1秒間の試行: 数十億回
  → 8桁パスワードを数時間で破れる可能性

bcrypt (cost=10):
  1回のハッシュ計算: ~100ms
  1秒間の試行: ~10回
  → 同じ8桁パスワードでも現実的な時間では破れない
```

**ハッシュアルゴリズムの比較:**

| アルゴリズム | 用途                             | パスワード保存         |
|:-------------|:---------------------------------|:-----------------------|
| MD5 / SHA-1  | チェックサム、非セキュリティ用途 | ❌ 絶対NG              |
| SHA-256      | データ整合性、署名               | ❌ 速すぎる            |
| bcrypt       | パスワードハッシュ               | ✅ 推奨                |
| argon2id     | パスワードハッシュ (より新しい)  | ✅ 推奨 (メモリハード) |
| scrypt       | パスワードハッシュ               | ✅ 推奨                |

---

## BP5: 安全な乱数生成

`math/rand` は予測可能な擬似乱数を生成するため、セキュリティ用途には使えません。トークン、セッション ID、パスワードリセットリンク等には `crypto/rand` を使用します。

### 悪い例

```go
import "math/rand"

func GenerateToken() string {
    // ❌ math/rand は予測可能 (シードが既知なら再現可能)
    b := make([]byte, 16)
    rand.Read(b)  // seed が同じなら同じ乱数列
    return hex.EncodeToString(b)
}

func GeneratePasswordResetToken() string {
    // ❌ 現在時刻ベース → 推測可能
    return fmt.Sprintf("%d", time.Now().UnixNano())
}
```

**攻撃例:**

```text
math/rand で生成されたトークン:
  シードが分かれば全てのトークンを予測可能
  シードの推測: プロセス起動時刻や PID から

時刻ベースのトークン:
  パスワードリセット要求時刻 ± 数秒を試行
  → 数百回の試行でトークンを特定
```

### 改善例

```go
import (
    "crypto/rand"
    "encoding/hex"
)

func GenerateToken() string {
    // ✅ crypto/rand は OS の暗号学的に安全な乱数源を使用
    b := make([]byte, 32) // 256ビット
    _, err := rand.Read(b)
    if err != nil {
        // OS の乱数源が枯渇 → サーバーを止めるべき
        panic("crypto/rand failed: " + err.Error())
    }
    return hex.EncodeToString(b)
}

func GenerateSessionID() string {
    b := make([]byte, 32)
    _, _ = rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}
```

**使い分け:**

| 用途                           | パッケージ                     | 理由                                             |
|:-------------------------------|:-------------------------------|:-------------------------------------------------|
| テストデータ、シミュレーション | `math/rand`                    | 予測可能でも問題ない                             |
| セッション ID、CSRF トークン   | `crypto/rand`                  | 推測されては困る                                 |
| パスワードリセットトークン     | `crypto/rand`                  | 推測されては困る                                 |
| UUID                           | `crypto/rand` ベースの UUID v4 | 推測されては困る                                 |
| 抽選、ゲーム                   | `math/rand`                    | 公平性は必要だが暗号学的安全性は不要な場合が多い |

---

## BP6: エラー処理で情報を漏らさない

内部のエラー詳細（スタックトレース、SQL 文、ファイルパス）をユーザーに返すと、攻撃者の情報収集に利用されます。

### 悪い例

```go
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")

    var user User
    err := db.QueryRow("SELECT * FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name)
    if err != nil {
        // ❌ 内部エラーをそのままユーザーに返す
        http.Error(w, err.Error(), http.StatusInternalServerError)
        // → "sql: no rows in result set" や
        //   "connection refused: tcp 10.0.1.5:5432" が漏洩
        return
    }

    json.NewEncoder(w).Encode(user)
}
```

**漏洩する情報の例:**

```text
"pq: relation \"admin_users\" does not exist"
  → テーブル名が攻撃者に知られる

"dial tcp 10.0.1.5:5432: connect: connection refused"
  → 内部ネットワークの IP アドレス、DB のポート番号が知られる

"open /etc/app/secrets.yaml: permission denied"
  → ファイルパス、設定ファイルの存在が知られる
```

### 改善例

```go
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")

    var user User
    err := db.QueryRow("SELECT * FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name)
    if err != nil {
        // ✅ ユーザーには汎用メッセージを返す
        // ✅ 詳細はサーバーログにのみ出力
        log.Printf("getUser failed: id=%s err=%v", id, err)
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(user)
}
```

**原則:**

```text
エラーレスポンス:
  ユーザー向け: 汎用的なメッセージ (ログインページなら「認証に失敗しました」)
  ログ:         詳細な情報 (エラーの種類、スタックトレース、コンテキスト)

開発環境 vs 本番環境:
  開発: デバッグ情報を表示してよい (スタックトレース等)
  本番: 絶対に内部情報を漏らさない

ステータスコードの使い分け:
  400: ユーザーの入力ミス (入力検証エラー)
  401: 未認証
  403: 権限不足
  404: リソースが存在しない (「存在しない」こと自体が情報にならないよう注意)
  500: サーバー内部エラー (詳細は返さない)
```

---

## BP7: ログに機密情報を書かない

パスワード、トークン、クレジットカード番号などの機密情報をログに出力すると、ログへのアクセス権がある人全員に漏洩します。

### 悪い例

```go
func LoginHandler(w http.ResponseWriter, r *http.Request) {
    username := r.FormValue("username")
    password := r.FormValue("password")

    // ❌ パスワードをログに出力
    log.Printf("login attempt: user=%s password=%s", username, password)

    // ❌ リクエスト全体をログに出力 (ヘッダーに Cookie や API キーが含まれる)
    log.Printf("request: %+v", r)
}
```

```go
func PaymentHandler(w http.ResponseWriter, r *http.Request) {
    cardNumber := r.FormValue("card_number")

    // ❌ クレジットカード番号をログに出力
    log.Printf("payment: card=%s amount=%d", cardNumber, amount)
}
```

**ログ漏洩のリスク:**

```text
ログの閲覧権限:
  開発者 → 運用者 → SRE → 監査人
  全員が機密データにアクセスできてしまう

ログの保存先:
  ファイル / Cloud Storage / ログ集約サービス (Datadog, Splunk 等)
  → 保存先のセキュリティレベルに依存

過去ログ:
  ログは長期間保存される → 後から情報が漏洩するリスク
```

### 改善例

```go
func LoginHandler(w http.ResponseWriter, r *http.Request) {
    username := r.FormValue("username")
    password := r.FormValue("password")

    // ✅ パスワードはログに出さない
    log.Printf("login attempt: user=%s", username)

    ok := checkPassword(username, password)
    if !ok {
        // ✅ 認証失敗の事実だけを記録 (パスワードは記録しない)
        log.Printf("login failed: user=%s", username)
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
        return
    }
    // ...
}
```

**機密情報のマスキング:**

```go
func maskCardNumber(card string) string {
    if len(card) < 4 {
        return "****"
    }
    return strings.Repeat("*", len(card)-4) + card[len(card)-4:]
}

// 使用例:
// maskCardNumber("4111111111111111") → "************1111"

func maskEmail(email string) string {
    parts := strings.SplitN(email, "@", 2)
    if len(parts) != 2 {
        return "***@***"
    }
    name := parts[0]
    if len(name) <= 2 {
        return "**@" + parts[1]
    }
    return name[:2] + "***@" + parts[1]
}

// 使用例:
// maskEmail("alice@example.com") → "al***@example.com"
```

**ログに出力してはいけない情報:**

| 情報                   | 例                  | 対策                           |
|:-----------------------|:--------------------|:-------------------------------|
| パスワード             | `password=s3cret`   | ログに出さない                 |
| セッショントークン     | `session_id=abc123` | ID のみ出力 (トークン値は不可) |
| API キー               | `api_key=sk-xxxx`   | マスキング                     |
| クレジットカード       | `card=4111...1111`  | 下 4 桁のみ表示                |
| 個人情報 (PII)         | 住所、電話番号      | マスキングまたは非出力         |
| Authorization ヘッダー | `Bearer eyJ...`     | ログに出さない                 |

---

## BP8: 競合状態 (Race Condition) の防止

複数の goroutine が共有データに同時にアクセスすると、データ不整合やセキュリティ上の問題が発生します。

### 悪い例 (TOCTOU: Time-of-Check to Time-of-Use)

```go
var balance = 10000

func WithdrawHandler(w http.ResponseWriter, r *http.Request) {
    amount, _ := strconv.Atoi(r.FormValue("amount"))

    // ❌ チェックと使用の間に別のリクエストが割り込める
    if balance >= amount {           // Time-of-Check: 残高確認
        // ← ここで別のリクエストが同じ条件を通過可能！
        time.Sleep(10 * time.Millisecond) // 処理の遅延をシミュレート
        balance -= amount             // Time-of-Use: 残高更新
        log.Printf("withdrawn %d, balance=%d", amount, balance)
    }
}
```

**攻撃例 (二重引き出し):**

```text
初期残高: 10,000
  リクエスト A: 引き出し 8,000 → チェック OK (残高 >= 8,000)
  リクエスト B: 引き出し 8,000 → チェック OK (残高はまだ 10,000)
  リクエスト A: 残高 -= 8,000 → 残高 = 2,000
  リクエスト B: 残高 -= 8,000 → 残高 = -6,000 ← 残高がマイナスに！
```

### 改善例

```go
import "sync"

var (
    balance int = 10000
    mu      sync.Mutex
)

func WithdrawHandler(w http.ResponseWriter, r *http.Request) {
    amount, _ := strconv.Atoi(r.FormValue("amount"))

    // ✅ Mutex でクリティカルセクションを保護
    mu.Lock()
    defer mu.Unlock()

    if balance >= amount {
        balance -= amount
        log.Printf("withdrawn %d, balance=%d", amount, balance)
        fmt.Fprintf(w, "success: balance=%d", balance)
        return
    }

    http.Error(w, "insufficient balance", http.StatusBadRequest)
}
```

**Mutex の使い方のポイント:**

```go
// ✅ defer で確実に Unlock
mu.Lock()
defer mu.Unlock()
// この間の操作は一意に実行される

// ❌ defer を忘れるとデッドロック
mu.Lock()
if err != nil {
    return  // Unlock されない！
}
balance -= amount
mu.Unlock()
```

**より安全なアプローチ (データベースレベル):**

```go
// ✅ データベースのトランザクションと行ロックを使用
func WithdrawDB(db *sql.DB, userID int, amount int) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // SELECT FOR UPDATE で行をロック
    var balance int
    err = tx.QueryRow("SELECT balance FROM accounts WHERE user_id = $1 FOR UPDATE", userID).Scan(&balance)
    if err != nil {
        return err
    }

    if balance < amount {
        return fmt.Errorf("insufficient balance")
    }

    _, err = tx.Exec("UPDATE accounts SET balance = balance - $1 WHERE user_id = $2", amount, userID)
    if err != nil {
        return err
    }

    return tx.Commit()
}
```

**競合状態のパターン:**

| パターン     | 説明                                | 対策                        |
|:-------------|:------------------------------------|:----------------------------|
| TOCTOU       | チェックと使用の間に状態が変化      | Mutex / DB トランザクション |
| 二重送信     | フォームが二重送信される            | 冪等キー (idempotency key)  |
| カウンタ競合 | 複数 goroutine が同じカウンタを更新 | `sync/atomic` / Mutex       |
| ファイル競合 | 同じファイルへの並行書き込み        | ファイルロック /排他制御    |

---

## 防御策チェックリスト

| カテゴリ     | 項目                                                   | チェック |
|:-------------|:-------------------------------------------------------|:---------|
| **SQL**      | プレースホルダー（パラメータ化クエリ）を使用しているか | □        |
| **入力検証** | Allowlist 方式で入力を検証しているか                   | □        |
||文字列長の上限を設定しているか|□|
|**ファイルアクセス**|パストラバーサル対策 (ベースディレクトリ内か確認) を実装しているか|□|
|**パスワード**|bcrypt / argon2id でハッシュ化しているか|□|
||平文でパスワードを保存していないか|□|
|**乱数**|セキュリティ用途に `crypto/rand` を使用しているか|□|
|**エラー処理**|本番環境で内部エラー詳細を返していないか|□|
||エラー詳細はサーバーログにのみ出力しているか|□|
|**ログ**|パスワード / トークン / API キーをログに出していないか|□|
||機密情報をマスキングしているか|□|
|**並行処理**|共有データへのアクセスを Mutex / atomic で保護しているか|□|
||DB のトランザクションと行ロックを適切に使用しているか|□|

---

## 演習後の理解度確認

### 確認クイズ

1. **`fmt.Sprintf` で SQL を組み立てるのが危険な理由は何ですか？**
   - <details><summary>解答</summary>ユーザー入力に SQL の特殊文字（シングルクォート等）が含まれている場合、意図しない SQL が実行されるためです。プレースホルダーを使用すると、パラメータはデータとして扱われ SQL 構文に影響しません。</details>

2. **`filepath.Join` を使ってもパストラバーサルを防げないのはなぜですか？**
   - <details><summary>解答</summary>`filepath.Join` はパスを正規化しますが、`..` を含む入力を解決した結果がベースディレクトリの外を指すことを防ぐ機能はないためです。解決後の絶対パスがベースディレクトリ配下か `strings.HasPrefix` で確認する必要があります。</details>

3. **SHA-256 でパスワードをハッシュ化するのが不適切な理由は何ですか？**
   - <details><summary>解答</summary>SHA-256 は高速に計算できるため、ブルートフォース攻撃（全パターンを試す攻撃）に対して脆弱です。またソルトを自分で管理する必要があります。bcrypt や argon2id は計算コストが高く調整可能で、ソルトも自動付与されます。</details>

4. **`math/rand` と `crypto/rand` の使い分けを説明してください。**
   - <details><summary>解答</summary>`math/rand` は予測可能な擬似乱数を生成するため、テストデータやシミュレーション等のセキュリティに関係ない用途に使います。セッショントークンや CSRF トークン等、推測されては困る値には `crypto/rand` を使用します。</details>

5. **TOCTOU 脆弱性とは何ですか？ どのように防ぎますか？**
   - <details><summary>解答</summary>Time-of-Check to Time-of-Use の略で、条件チェック（Check）と実際の操作（Use）の間に別の処理が割り込み、チェック時の条件が満たされなくなっている脆弱性です。Mutex による排他制御や、データベースのトランザクション + 行ロック (SELECT FOR UPDATE) で防ぎます。</details>

---

## 参考文献

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP Cheat Sheet Series](https://cheatsheetseries.owasp.org/)
- [OWASP Input Validation Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html)
- [Go Security Checklist](https://github.com/guardrailsio/awesome-golang-security)
- [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency)
- [IPA: 安全なSQLの呼び出し方](https://www.ipa.go.jp/security/vuln/websecurity.html)
