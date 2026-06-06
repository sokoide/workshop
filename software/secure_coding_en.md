# Secure Coding Best Practices (Go)

Learn about vulnerabilities commonly introduced during everyday coding and the habits to prevent them. Unlike HTTP-level attacks covered in `infra/web_security_en.md` (XSS, CSRF, etc.), this guide focuses on risks hidden inside **application implementation**.

---

## Table of Contents

1. [SQL Injection Prevention](#bp1-sql-injection-prevention)
2. [Input Validation](#bp2-input-validation)
3. [Path Traversal Prevention](#bp3-path-traversal-prevention)
4. [Secure Password Hashing](#bp4-secure-password-hashing)
5. [Secure Random Number Generation](#bp5-secure-random-number-generation)
6. [Error Handling Without Information Leakage](#bp6-error-handling-without-information-leakage)
7. [Logging Without Exposing Secrets](#bp7-logging-without-exposing-secrets)
8. [Race Condition Prevention](#bp8-race-condition-prevention)

---

## BP1: SQL Injection Prevention

Embedding user input directly into SQL strings allows attackers to execute arbitrary SQL.

### Bad Example

```go
func GetUser(db *sql.DB, username string) (*User, error) {
    // ❌ String concatenation of user input into SQL
    query := fmt.Sprintf("SELECT id, name, email FROM users WHERE name = '%s'", username)
    row := db.QueryRow(query)
    // ...
}
```

**Attack examples:**

```text
Input: ' OR '1'='1' --
Expanded: SELECT id, name, email FROM users WHERE name = '' OR '1'='1' --'
Result: Returns all users' data
```

```text
Input: '; DROP TABLE users; --
Expanded: SELECT id, name, email FROM users WHERE name = ''; DROP TABLE users; --'
Result: The users table is dropped
```

### Good Example

```go
func GetUser(db *sql.DB, username string) (*User, error) {
    // ✅ Use placeholders (parameters are automatically escaped)
    row := db.QueryRow("SELECT id, name, email FROM users WHERE name = $1", username)

    var u User
    err := row.Scan(&u.ID, &u.Name, &u.Email)
    if err != nil {
        return nil, err
    }
    return &u, nil
}
```

**Placeholder syntax varies by RDBMS:**

| Database   | Syntax        | Example           |
| :--------- | :------------ | :---------------- |
| PostgreSQL | `$1, $2, ...` | `WHERE name = $1` |
| MySQL      | `?`           | `WHERE name = ?`  |
| SQLite     | `?` or `$1`   | `WHERE name = ?`  |

**Why placeholders are safe:**

```text
With placeholders:
  SQL: SELECT ... WHERE name = $1
  Parameter: "' OR '1'='1' --"
  ↓
  The database treats the parameter as "data"
  → Not interpreted as SQL syntax
  → Searches for a user literally named "' OR '1'='1' --"
  → No results → empty result set
```

---

## BP2: Input Validation

Never trust user input. Always validate at the boundary.

### Bad Example

```go
func CreateUser(r *http.Request) error {
    name := r.FormValue("name")
    email := r.FormValue("email")
    age := r.FormValue("age")

    // ❌ No validation at all
    _, err := db.Exec("INSERT INTO users (name, email, age) VALUES (?, ?, ?)", name, email, age)
    return err
}
```

**Potential problems:**

```text
name: "" (empty)           → Data inconsistency
email: "not-an-email"      → Invalid format
age: "-1"                  → Invalid value
name: 1000-char string     → Buffer overflow / DoS
name: "<script>alert(1)</script>" → XSS (when rendered)
```

### Good Example

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

    // Name: required, length limit
    name = strings.TrimSpace(name)
    if name == "" {
        errs = append(errs, "name is required")
    }
    if utf8.RuneCountInString(name) > 100 {
        errs = append(errs, "name must be 100 characters or less")
    }

    // Email: format check
    if _, err := mail.ParseAddress(email); err != nil {
        errs = append(errs, "invalid email format")
    }

    // Age: integer, range check
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

**Validation principles:**

```text
Allowlist approach: ✅ Recommended
  Define "only these formats are valid" and reject everything else
  Example: regex for email, enum values list

Blocklist approach: ❌ Not recommended
  Define "these characters are dangerous" and allow everything else
  Example: block "<script>" → "<SCRIPT>" or "<img onerror>" still passes

Reason: Cannot keep up with unknown attack patterns
```

---

## BP3: Path Traversal Prevention

Using user input in file paths can lead to unauthorized file access.

### Bad Example

```go
func GetAvatar(w http.ResponseWriter, r *http.Request) {
    username := r.URL.Query().Get("user")

    // ❌ User input directly concatenated into a file path
    path := filepath.Join("/var/avatars", username)
    http.ServeFile(w, r, path)
}
```

**Attack example:**

```text
Request: GET /avatar?user=../../../etc/passwd
Expanded: /var/avatars/../../../etc/passwd
Resolved: /etc/passwd → System file is read!
```

### Good Example

```go
func GetAvatar(w http.ResponseWriter, r *http.Request) {
    username := r.URL.Query().Get("user")

    // ✅ 1. Validate input (alphanumeric only)
    if !isValidUsername(username) {
        http.Error(w, "invalid username", http.StatusBadRequest)
        return
    }

    // ✅ 2. Resolve path and verify it stays within the base directory
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

**Why `filepath.Join` alone is not enough:**

```text
filepath.Join("/var/avatars", "../../../etc/passwd")
  → /var/avatars/../../../etc/passwd
  → filepath.Abs resolves it
  → /etc/passwd  ← Outside the base directory!

Use strings.HasPrefix to verify the resolved absolute path starts with the base directory
```

---

## BP4: Secure Password Hashing

Storing passwords in plaintext or using fast hashes (MD5, SHA) makes them easy to recover when leaked.

### Bad Example

```go
import "crypto/sha256"

func HashPassword(password string) string {
    // ❌ SHA-256 is a fast hash → vulnerable to brute-force attacks
    // ❌ No salt → same password produces the same hash
    h := sha256.Sum256([]byte(password))
    return fmt.Sprintf("%x", h)
}
```

**Attack example (rainbow table):**

```text
SHA-256("password123") → ef92b778... (always the same)
  ↓
Attacker obtains the hash
  ↓
Looks it up in a rainbow table (precomputed hash dictionary)
  ↓
Instantly recovers "password123"
```

### Good Example

```go
import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
    // ✅ bcrypt is computationally expensive (resistant to brute-force)
    // ✅ Salt is automatically applied (same password → different hashes)
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

**Why bcrypt is secure:**

```text
MD5 / SHA-256:
  Single hash computation: < 1μs
  Attempts per second: billions
  → An 8-character password can be cracked in hours

bcrypt (cost=10):
  Single hash computation: ~100ms
  Attempts per second: ~10
  → Same 8-character password is infeasible to crack
```

**Hash algorithm comparison:**

| Algorithm   | Use Case                     | Password Storage             |
| :---------- | :--------------------------- | :--------------------------- |
| MD5 / SHA-1 | Checksums, non-security uses | ❌ Never                     |
| SHA-256     | Data integrity, signatures   | ❌ Too fast                  |
| bcrypt      | Password hashing             | ✅ Recommended               |
| argon2id    | Password hashing (newer)     | ✅ Recommended (memory-hard) |
| scrypt      | Password hashing             | ✅ Recommended               |

---

## BP5: Secure Random Number Generation

`math/rand` generates predictable pseudorandom numbers and must not be used for security purposes. Use `crypto/rand` for tokens, session IDs, and password reset links.

### Bad Example

```go
import "math/rand"

func GenerateToken() string {
    // ❌ math/rand is predictable (reproducible if seed is known)
    b := make([]byte, 16)
    rand.Read(b)  // same seed → same sequence
    return hex.EncodeToString(b)
}

func GeneratePasswordResetToken() string {
    // ❌ Time-based → guessable
    return fmt.Sprintf("%d", time.Now().UnixNano())
}
```

**Attack example:**

```text
Token generated with math/rand:
  If the seed is known, all tokens are predictable
  Seed can be guessed from process start time or PID

Time-based token:
  Try timestamps ± a few seconds from the reset request
  → A few hundred attempts to guess the token
```

### Good Example

```go
import (
    "crypto/rand"
    "encoding/hex"
)

func GenerateToken() string {
    // ✅ crypto/rand uses the OS's cryptographically secure random source
    b := make([]byte, 32) // 256 bits
    _, err := rand.Read(b)
    if err != nil {
        // OS random source exhausted → should stop the server
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

**When to use which:**

| Use Case                 | Package                     | Reason                                         |
| :----------------------- | :-------------------------- | :--------------------------------------------- |
| Test data, simulations   | `math/rand`                 | Predictability is fine                         |
| Session IDs, CSRF tokens | `crypto/rand`               | Must be unguessable                            |
| Password reset tokens    | `crypto/rand`               | Must be unguessable                            |
| UUIDs                    | `crypto/rand`-based UUID v4 | Must be unguessable                            |
| Lotteries, games         | `math/rand`                 | Fairness needed but not cryptographic security |

---

## BP6: Error Handling Without Information Leakage

Returning internal error details (stack traces, SQL statements, file paths) to users gives attackers valuable information for reconnaissance.

### Bad Example

```go
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")

    var user User
    err := db.QueryRow("SELECT * FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name)
    if err != nil {
        // ❌ Internal error returned directly to the user
        http.Error(w, err.Error(), http.StatusInternalServerError)
        // → "sql: no rows in result set" or
        //   "connection refused: tcp 10.0.1.5:5432" leaked
        return
    }

    json.NewEncoder(w).Encode(user)
}
```

**Examples of leaked information:**

```text
"pq: relation \"admin_users\" does not exist"
  → Table name revealed to attacker

"dial tcp 10.0.1.5:5432: connect: connection refused"
  → Internal network IP and database port revealed

"open /etc/app/secrets.yaml: permission denied"
  → File path and config file existence revealed
```

### Good Example

```go
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Query().Get("id")

    var user User
    err := db.QueryRow("SELECT * FROM users WHERE id = $1", id).Scan(&user.ID, &user.Name)
    if err != nil {
        // ✅ Return a generic message to the user
        // ✅ Log the details server-side only
        log.Printf("getUser failed: id=%s err=%v", id, err)
        http.Error(w, "internal server error", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(user)
}
```

**Principles:**

```text
Error responses:
  User-facing:  Generic messages (e.g., "Authentication failed" for login)
  Server logs:  Detailed information (error type, stack trace, context)

Development vs Production:
  Development: Debug information is acceptable
  Production:  Never expose internal information

Status codes:
  400: User input error (validation failure)
  401: Unauthenticated
  403: Insufficient permissions
  404: Resource not found (be careful — "not found" itself can be information)
  500: Internal server error (no details in response)
```

---

## BP7: Logging Without Exposing Secrets

Writing passwords, tokens, or credit card numbers to logs exposes them to everyone with log access.

### Bad Example

```go
func LoginHandler(w http.ResponseWriter, r *http.Request) {
    username := r.FormValue("username")
    password := r.FormValue("password")

    // ❌ Password written to logs
    log.Printf("login attempt: user=%s password=%s", username, password)

    // ❌ Entire request logged (headers may contain cookies or API keys)
    log.Printf("request: %+v", r)
}
```

```go
func PaymentHandler(w http.ResponseWriter, r *http.Request) {
    cardNumber := r.FormValue("card_number")

    // ❌ Credit card number written to logs
    log.Printf("payment: card=%s amount=%d", cardNumber, amount)
}
```

**Risks of log leakage:**

```text
Who can read logs:
  Developers → Operators → SRE → Auditors
  All of them get access to sensitive data

Where logs are stored:
  Files / Cloud Storage / Log aggregation services (Datadog, Splunk, etc.)
  → Security depends on the storage backend

Historical logs:
  Logs are retained for long periods → data can leak later
```

### Good Example

```go
func LoginHandler(w http.ResponseWriter, r *http.Request) {
    username := r.FormValue("username")
    password := r.FormValue("password")

    // ✅ Password is never logged
    log.Printf("login attempt: user=%s", username)

    ok := checkPassword(username, password)
    if !ok {
        // ✅ Only the fact of failure is logged (not the password)
        log.Printf("login failed: user=%s", username)
        http.Error(w, "invalid credentials", http.StatusUnauthorized)
        return
    }
    // ...
}
```

**Masking sensitive information:**

```go
func maskCardNumber(card string) string {
    if len(card) < 4 {
        return "****"
    }
    return strings.Repeat("*", len(card)-4) + card[len(card)-4:]
}

// Example:
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

// Example:
// maskEmail("alice@example.com") → "al***@example.com"
```

**Data that must never appear in logs:**

| Data                               | Example               | Handling                      |
| :--------------------------------- | :-------------------- | :---------------------------- |
| Passwords                          | `password=s3cret`     | Never log                     |
| Session tokens                     | `session_id=abc123`   | Log ID only (not token value) |
| API keys                           | `api_key=sk-xxxx`     | Mask                          |
| Credit card numbers                | `card=4111...1111`    | Show last 4 digits only       |
| PII (personally identifiable info) | Address, phone number | Mask or omit                  |
| Authorization headers              | `Bearer eyJ...`       | Never log                     |

---

## BP8: Race Condition Prevention

When multiple goroutines access shared data concurrently, data corruption or security issues can occur.

### Bad Example (TOCTOU: Time-of-Check to Time-of-Use)

```go
var balance = 10000

func WithdrawHandler(w http.ResponseWriter, r *http.Request) {
    amount, _ := strconv.Atoi(r.FormValue("amount"))

    // ❌ Another request can slip in between check and use
    if balance >= amount {           // Time-of-Check: verify balance
        // ← Another request can pass the same check here!
        time.Sleep(10 * time.Millisecond) // Simulate processing delay
        balance -= amount             // Time-of-Use: update balance
        log.Printf("withdrawn %d, balance=%d", amount, balance)
    }
}
```

**Attack example (double withdrawal):**

```text
Initial balance: 10,000
  Request A: Withdraw 8,000 → Check OK (balance >= 8,000)
  Request B: Withdraw 8,000 → Check OK (balance still 10,000)
  Request A: balance -= 8,000 → balance = 2,000
  Request B: balance -= 8,000 → balance = -6,000 ← Balance goes negative!
```

### Good Example

```go
import "sync"

var (
    balance int = 10000
    mu      sync.Mutex
)

func WithdrawHandler(w http.ResponseWriter, r *http.Request) {
    amount, _ := strconv.Atoi(r.FormValue("amount"))

    // ✅ Mutex protects the critical section
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

**Mutex usage tips:**

```go
// ✅ Always use defer to ensure Unlock
mu.Lock()
defer mu.Unlock()
// Operations here execute exclusively

// ❌ Forgetting defer causes deadlock
mu.Lock()
if err != nil {
    return  // Never unlocked!
}
balance -= amount
mu.Unlock()
```

**Safer approach (database level):**

```go
// ✅ Use database transactions and row-level locking
func WithdrawDB(db *sql.DB, userID int, amount int) error {
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // SELECT FOR UPDATE locks the row
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

**Race condition patterns:**

| Pattern       | Description                                   | Countermeasure                   |
| :------------ | :-------------------------------------------- | :------------------------------- |
| TOCTOU        | State changes between check and use           | Mutex / DB transaction           |
| Double submit | Form submitted twice                          | Idempotency key                  |
| Counter race  | Multiple goroutines updating the same counter | `sync/atomic` / Mutex            |
| File race     | Concurrent writes to the same file            | File locking / exclusive control |

---

## Defense Checklist

| Category             | Item                                                     | Check |
| :------------------- | :------------------------------------------------------- | :---- |
| **SQL**              | Using placeholders (parameterized queries)               | □     |
| **Input validation** | Using allowlist-based validation                         | □     |
|                      | Setting string length limits                             | □     |
| **File access**      | Path traversal prevention (verify within base directory) | □     |
| **Passwords**        | Hashing with bcrypt / argon2id                           | □     |
|                      | Not storing passwords in plaintext                       | □     |
| **Random numbers**   | Using `crypto/rand` for security purposes                | □     |
| **Error handling**   | Not returning internal error details in production       | □     |
|                      | Logging details server-side only                         | □     |
| **Logging**          | Not writing passwords / tokens / API keys to logs        | □     |
|                      | Masking sensitive information                            | □     |
| **Concurrency**      | Protecting shared data access with Mutex / atomic        | □     |
|                      | Using DB transactions and row locks appropriately        | □     |

---

## Post-Workshop Quiz

1. **Why is it dangerous to build SQL with `fmt.Sprintf`?**
   - <details><summary>Answer</summary>If user input contains SQL special characters (such as single quotes), unintended SQL can be executed. Placeholders treat parameters as data, preventing them from affecting SQL syntax.</details>

2. **Why can't `filepath.Join` alone prevent path traversal?**
   - <details><summary>Answer</summary>`filepath.Join` normalizes paths but does not prevent the resolved path from pointing outside the base directory when `..` is included. You must verify the resolved absolute path starts with the base directory using `strings.HasPrefix`.</details>

3. **Why is SHA-256 unsuitable for password hashing?**
   - <details><summary>Answer</summary>SHA-256 is fast to compute, making it vulnerable to brute-force attacks. It also requires manual salt management. bcrypt and argon2id have adjustable computational cost and apply salts automatically.</details>

4. **Explain when to use `math/rand` vs `crypto/rand`.**
   - <details><summary>Answer</summary>Use `math/rand` for test data and simulations where predictability is fine. Use `crypto/rand` for session tokens, CSRF tokens, and any value that must be unguessable.</details>

5. **What is a TOCTOU vulnerability, and how do you prevent it?**
   - <details><summary>Answer</summary>TOCTOU (Time-of-Check to Time-of-Use) occurs when state changes between a condition check and the actual operation, invalidating the check. Prevent it with mutual exclusion (Mutex) or database transactions with row-level locking (SELECT FOR UPDATE).</details>

---

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP Cheat Sheet Series](https://cheatsheetseries.owasp.org/)
- [OWASP Input Validation Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html)
- [Go Security Checklist](https://github.com/guardrailsio/awesome-golang-security)
- [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency)
