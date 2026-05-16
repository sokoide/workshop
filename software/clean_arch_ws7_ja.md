# クリーンアーキテクチャ実習 (WS7): 認証の追加

この実習では、BBS（2 ちゃんねる風掲示板）に JWT Bearer Token 認証を追加します。
**Presentation 層に横断的関心を追加するパターン**を体験し、内側の層（Domain・UseCase・Infra）を一切変更しないことを確認します。

## 前提知識

本実習は [BBS プロジェクト](./assets/bbs/) のコードを題材にします。

## 実習のシナリオ

「投稿時に JWT で認証を要求する」という要件に対応します。
認証は Presentation 層の **横断的関心事（Cross-Cutting Concern）** であり、ミドルウェアとして実装します。

---

## 課題: JWT 認証ミドルウェアの追加

### 変更範囲の確認（やってはいけないこと）

以下の層は **1行も変更しません**。

| 層 | 理由 |
| ---- | ------ |
| **Domain** | Entity と Port は「誰が」操作しているかを知らない。認証は Presentation の責務 |
| **UseCase** | `Execute(ctx, Input)` のシグネチャ不変。認証済みユーザーが必要なら Input DTO にフィールドを追加するだけで対応可能 |
| **Infra** | DB アクセスは認証とは無関係 |

### Step 1: 認証ミドルウェアの実装

新しいファイルに JWT 検証ミドルウェアを作ります。

```go
// internal/adapters/internal/adapters/presentation/http/middleware/auth.go（新規ファイル）
package middleware

import (
    "context"
    "encoding/json"
    "errors"
    "log/slog"
    "net/http"
    "strings"

    "github.com/golang-jwt/jwt/v5"
)

type contextKey struct{}

// Claims は JWT から抽出するユーザー情報です
type Claims struct {
    UserID string `json:"sub"`
    Role   string `json:"role"`
}

// GetClaims は context から認証済みユーザー情報を取り出します
func GetClaims(ctx context.Context) *Claims {
    if c, ok := ctx.Value(contextKey{}).(*Claims); ok {
        return c
    }
    return nil
}

// Auth は JWT Bearer Token を検証するミドルウェアを返します
func Auth(secret string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Authorization ヘッダーからトークンを取得
            header := r.Header.Get("Authorization")
            if header == "" {
                writeAuthError(w, http.StatusUnauthorized, "missing authorization header")
                return
            }

            token := strings.TrimPrefix(header, "Bearer ")
            if token == header {
                writeAuthError(w, http.StatusUnauthorized, "invalid authorization format")
                return
            }

            // JWT を検証
            claims, err := validateToken(token, secret)
            if err != nil {
                writeAuthError(w, http.StatusUnauthorized, "invalid token")
                return
            }

            // 認証済みユーザー情報を context に格納
            ctx := context.WithValue(r.Context(), contextKey{}, claims)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func validateToken(tokenString string, secret string) (*Claims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("unexpected signing method")
        }
        return []byte(secret), nil
    })
    if err != nil {
        return nil, err
    }

    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || !token.Valid {
        return nil, errors.New("invalid token claims")
    }

    sub, ok := claims["sub"].(string)
    if !ok {
        return nil, errors.New("invalid token: missing or non-string sub claim")
    }
    role, ok := claims["role"].(string)
    if !ok {
        return nil, errors.New("invalid token: missing or non-string role claim")
    }

    return &Claims{UserID: sub, Role: role}, nil
}

// writeAuthError は認証ミドルウェア内でエラーレスポンスを書き込みます
func writeAuthError(w http.ResponseWriter, status int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
        slog.Error("json encode failed", "error", err)
    }
}
```

### Step 2: Router にミドルウェアを適用する

書き込み（POST）エンドポイントにだけ認証を要求します。読み取り（GET）は認証なしでアクセス可能です。

```go
// internal/adapters/internal/adapters/presentation/http/router.go（一部変更）
func NewRouter(
    boardHandler *handler.BoardHandler,
    threadHandler *handler.ThreadHandler,
    postHandler *handler.PostHandler,
    secret string,  // ← 追加: JWT署名用シークレット
) http.Handler {
    mux := http.NewServeMux()

    // 認証なし（GET — 既存動作を維持）
    mux.HandleFunc("GET /api/boards", boardHandler.ListBoards)
    mux.HandleFunc("GET /api/boards/{name}/threads", threadHandler.ListThreads)
    mux.HandleFunc("GET /api/threads/{threadID}/posts", postHandler.ListPosts)

    // 認証あり（POST — ミドルウェアを適用）
    // auth() は http.Handler を返すため mux.Handle を使用
    auth := middleware.Auth(secret)
    mux.Handle("POST /api/boards/{name}/threads", auth(http.HandlerFunc(threadHandler.CreateThread)))
    mux.Handle("POST /api/threads/{threadID}/posts", auth(http.HandlerFunc(postHandler.CreatePost)))

    // ロギングミドルウェアは全体に適用
    return middleware.Logging(mux)
}
```

### Step 3: Composition Root に secret を渡す

```go
// cmd/bbs/main.go（一部変更）
func main() {
    // ...既存の DI 組み立て

    // JWT シークレットを環境変数から取得
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        secret = "dev-secret-key" // 開発用フォールバック
    }

    // Router に secret を渡す
    router := httpPresentation.NewRouter(boardHandler, threadHandler, postHandler, secret)

    slog.Info("server starting", "addr", ":8080")
    if err := http.ListenAndServe(":8080", router); err != nil {
        slog.Error("server failed", "error", err)
        os.Exit(1)
    }
}
```

### Step 4: 動作確認

```bash
# ビルド・起動
export JWT_SECRET="my-secret-key"
go build -o bbs ./cmd/bbs/
./bbs

# 認証なしで GET → 成功（既存動作）
curl http://localhost:8080/api/boards

# 認証なしで POST → 401 Unauthorized
curl -X POST http://localhost:8080/api/boards/program/threads \
  -H 'Content-Type: application/json' \
  -d '{"title":"テスト","author":"gopher","body":"hello"}'

# JWT を生成（HMAC-SHA256 署名付き）
# jwt-cli をインストール: go install github.com/matyer/jwt-cli@latest
TOKEN=$(jwt encode --secret "my-secret-key" --sub "user123" --role "member")
# または Go のワンライナーで生成:
# TOKEN=$(go run github.com/golang-jwt/jwt/v5/cmd/jwt@latest \
#   --secret "my-secret-key" --claim sub:user123 --claim role:member)

# 認証ありで POST → 成功
curl -X POST http://localhost:8080/api/boards/program/threads \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"認証テスト","author":"gopher","body":"hello with auth"}'
```

---

## 認証済みユーザーを UseCase で使う

実務では「認証されたユーザーID をビジネスロジックで使いたい」ケースが多々あります。例えば WS4 の「スレ主しか書き込めない」というルールでは、投稿者が誰かを UseCase に伝える必要があります。

### Handler 側: Context から Claims を取り出して Input DTO に渡す

```go
// internal/adapters/internal/adapters/presentation/http/handler/post_handler.go
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
    // Context から認証済みユーザーを取り出す
    claims := middleware.GetClaims(r.Context())
    if claims == nil {
        writeError(w, http.StatusUnauthorized, "authentication required")
        return
    }

    // Input DTO に UserID を渡す（UseCase は HTTP や JWT を知らない）
    out, err := h.createPost.Execute(r.Context(), usecase.CreatePostInput{
        ThreadID: threadID,
        Author:   claims.UserID,  // JWT の sub フィールド → ビジネスロジックで使用
        Body:     req.Body,
        Sage:     req.Sage,
    })
    // ...
}
```

### UseCase 側: 認証の存在を知らない

```go
// internal/usecase/post_usecase.go — 変更なし
func (u *CreatePostUseCase) Execute(ctx context.Context, in CreatePostInput) (*CreatePostOutput, error) {
    // in.Author に誰が入っているかは知らない。
    // JWT から来たのか、テストから来たのか、CLI から来たのかは関知しない。
    if !thread.CanPost(in.Author) {
        return nil, domain.ErrNotThreadOwner
    }
    // ...
}
```

このように、**認証情報の変換（JWT Claims → Input DTO）は Handler（Presentation 層）の責務**であり、UseCase は単なる文字列として受け取るだけです。

---

## 認証方式の変更にも強い

JWT → API Key → OAuth に認証方式を変更する場合、ミドルウェアを差し替えるだけです。

```go
// internal/adapters/presentation/middleware/apikey.go（別の認証方式）
func APIKey(validKeys map[string]bool) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            key := r.Header.Get("X-API-Key")
            if !validKeys[key] {
                writeAuthError(w, http.StatusUnauthorized, "invalid API key")
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

```go
// Router で差し替え（1箇所だけ）
auth := middleware.APIKey(validKeys)  // JWT から API Key に変更
```

UseCase、Domain、Infra は **一切変更不要** です。

---

## 認証なし版も簡単に用意できる

社内 API や管理ツール用に認証なし版を作る場合、ミドルウェアを適用しない Router を用意するだけです。

```go
// 認証なし版
mux.HandleFunc("POST /api/boards/{name}/threads", threadHandler.CreateThread)

// 認証あり版（本番用）
mux.Handle("POST /api/boards/{name}/threads", auth(http.HandlerFunc(threadHandler.CreateThread)))
```

同じ UseCase・同じ Handler を使い回せます。

---

## レイヤー分離がない場合との比較

```go
// NG: 認証がビジネスロジックに混ざっている
func CreatePost(w http.ResponseWriter, r *http.Request) {
    token := r.Header.Get("Authorization")  // ← HTTP 依存
    if !validateJWT(token) {                 // ← 認証がここに埋まっている
        w.WriteHeader(401)
        return
    }
    // ...DB 処理
}
```

この場合:

- 全てのエンドポイントで認証チェックを **重複して書く** 必要がある
- 認証方式の変更が **全エンドポイントに波及** する
- UseCase のテストで認証をモックする必要がある
- 「認証なし版」を作るためにコードを複製する必要がある

---

## この実習のポイント

1. **横断的関心事の分離**: 認証は Presentation 層のミドルウェアとして実装し、ビジネスロジック（UseCase）に混ざらない。
2. **内側の層は不変**: Domain・UseCase・Infra は認証の存在を知らない。`Execute(ctx, Input)` のシグネチャは変わらない。
3. **認証方式の差し替え**: JWT → API Key → OAuth への変更は、ミドルウェアの差し替えだけで完了。UseCase は変更不要。
4. **選択的適用**: 読み取りは認証なし、書き込みは認証あり、という粒度の制御が Router で完結する。
5. **テスト容易性**: UseCase のテストで認証をモックする必要がない。Handler のテストだけが認証ミドルウェアを検証する。
