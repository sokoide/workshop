# Clean Architecture Workshop (WS7): Adding Authentication

In this workshop, you will add JWT Bearer Token authentication to the BBS (2channel-style bulletin board).
You will experience the **cross-cutting concern pattern** — adding authentication in the Presentation layer — without modifying any inner layers (Domain, UseCase, Infra).

## Prerequisites

This workshop uses the [BBS project](./assets/bbs/) as the subject code.

## Workshop Scenario

Add a requirement that posting requires JWT authentication.
Authentication is a **cross-cutting concern** of the Presentation layer, implemented as middleware.

---

## Exercise: JWT Authentication Middleware

### Identify the Scope (What NOT to Change)

The following layers require **zero modifications**:

| Layer | Reason |
| ------- | -------- |
| **Domain** | Entities and Ports don't know "who" is operating. Authentication is a Presentation responsibility |
| **UseCase** | `Execute(ctx, Input)` signature is unchanged. If authenticated user info is needed, just add a field to the Input DTO |
| **Infra** | DB access is unrelated to authentication |

### Step 1: Implement Auth Middleware

Create JWT validation middleware in a new file.

```go
// internal/presentation/http/middleware/auth.go (new file)
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

// Claims holds user info extracted from the JWT
type Claims struct {
    UserID string `json:"sub"`
    Role   string `json:"role"`
}

// GetClaims extracts authenticated user info from context
func GetClaims(ctx context.Context) *Claims {
    if c, ok := ctx.Value(contextKey{}).(*Claims); ok {
        return c
    }
    return nil
}

// Auth returns middleware that validates JWT Bearer Tokens
func Auth(secret string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Extract token from Authorization header
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

            // Validate JWT
            claims, err := validateToken(token, secret)
            if err != nil {
                writeAuthError(w, http.StatusUnauthorized, "invalid token")
                return
            }

            // Store authenticated user info in context
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

// writeAuthError writes error responses within the auth middleware
func writeAuthError(w http.ResponseWriter, status int, msg string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    if err := json.NewEncoder(w).Encode(map[string]string{"error": msg}); err != nil {
        slog.Error("json encode failed", "error", err)
    }
}
```

### Step 2: Apply Middleware to Router

Require authentication only for write (POST) endpoints. Read (GET) remains publicly accessible.

```go
// internal/presentation/http/router.go (partial change)
func NewRouter(
    boardHandler *handler.BoardHandler,
    threadHandler *handler.ThreadHandler,
    postHandler *handler.PostHandler,
    secret string,  // ← Added: JWT signing secret
) http.Handler {
    mux := http.NewServeMux()

    // No auth (GET — existing behavior preserved)
    mux.HandleFunc("GET /api/boards", boardHandler.ListBoards)
    mux.HandleFunc("GET /api/boards/{name}/threads", threadHandler.ListThreads)
    mux.HandleFunc("GET /api/threads/{threadID}/posts", postHandler.ListPosts)

    // Auth required (POST — apply middleware)
    // auth() returns http.Handler, so use mux.Handle with http.HandlerFunc wrapper
    auth := middleware.Auth(secret)
    mux.Handle("POST /api/boards/{name}/threads", auth(http.HandlerFunc(threadHandler.CreateThread)))
    mux.Handle("POST /api/threads/{threadID}/posts", auth(http.HandlerFunc(postHandler.CreatePost)))

    // Logging middleware applies globally
    return middleware.Logging(mux)
}
```

### Step 3: Pass Secret to Composition Root

```go
// cmd/bbs/main.go (partial change)
func main() {
    // ...existing DI wiring

    // Get JWT secret from environment variable
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        secret = "dev-secret-key" // Development fallback
    }

    // Pass secret to Router
    router := httpPresentation.NewRouter(boardHandler, threadHandler, postHandler, secret)

    slog.Info("server starting", "addr", ":8080")
    if err := http.ListenAndServe(":8080", router); err != nil {
        slog.Error("server failed", "error", err)
        os.Exit(1)
    }
}
```

### Step 4: Verify

```bash
# Build and start
export JWT_SECRET="my-secret-key"
go build -o bbs ./cmd/bbs/
./bbs

# GET without auth → success (existing behavior)
curl http://localhost:8080/api/boards

# POST without auth → 401 Unauthorized
curl -X POST http://localhost:8080/api/boards/program/threads \
  -H 'Content-Type: application/json' \
  -d '{"title":"test","author":"gopher","body":"hello"}'

# Generate a JWT (HMAC-SHA256 signed)
# Install jwt-cli: go install github.com/matyer/jwt-cli@latest
TOKEN=$(jwt encode --secret "my-secret-key" --sub "user123" --role "member")
# Or use Go one-liner:
# TOKEN=$(go run github.com/golang-jwt/jwt/v5/cmd/jwt@latest \
#   --secret "my-secret-key" --claim sub:user123 --claim role:member)

# POST with auth → success
curl -X POST http://localhost:8080/api/boards/program/threads \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"title":"Auth test","author":"gopher","body":"hello with auth"}'
```

---

## Using Authenticated Users in UseCase

In practice, you often need the authenticated user ID in business logic. For example, WS4's "only thread owner can post" rule requires knowing who is posting.

### Handler Side: Extract Claims from Context and Pass to Input DTO

```go
// internal/presentation/http/handler/post_handler.go
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
    // Extract authenticated user from Context
    claims := middleware.GetClaims(r.Context())
    if claims == nil {
        writeError(w, http.StatusUnauthorized, "authentication required")
        return
    }

    // Pass UserID to Input DTO (UseCase is unaware of HTTP or JWT)
    out, err := h.createPost.Execute(r.Context(), usecase.CreatePostInput{
        ThreadID: threadID,
        Author:   claims.UserID,  // JWT sub field → used in business logic
        Body:     req.Body,
        Sage:     req.Sage,
    })
    // ...
}
```

After Step 1, `GetClaims` is already defined in the middleware package.

### UseCase Side: Unaware of Authentication

```go
// usecase/post_usecase.go — unchanged
func (u *CreatePostUseCase) Execute(ctx context.Context, in CreatePostInput) (*CreatePostOutput, error) {
    // Doesn't know where in.Author came from.
    // Whether it came from JWT, test, or CLI is none of its business.
    if !thread.CanPost(in.Author) {
        return nil, domain.ErrNotThreadOwner
    }
    // ...
}
```

This way, **authentication info conversion (JWT Claims → Input DTO) is the Handler's (Presentation layer) responsibility**, and UseCase simply receives it as a string.

---

## Resilient to Auth Method Changes

Switching from JWT → API Key → OAuth requires only swapping the middleware.

```go
// presentation/middleware/apikey.go (alternative auth method)
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
// Swap in Router (one place)
auth := middleware.APIKey(validKeys)  // Changed from JWT to API Key
```

UseCase, Domain, and Infra require **zero changes**.

---

## Easy to Create Unauthenticated Version

For internal APIs or admin tools, just skip the middleware:

```go
// Unauthenticated version
mux.HandleFunc("POST /api/boards/{name}/threads", threadHandler.CreateThread)

// Authenticated version (production)
mux.Handle("POST /api/boards/{name}/threads", auth(http.HandlerFunc(threadHandler.CreateThread)))
```

The same UseCases and Handlers are reused.

---

## Comparison: Without Layer Separation

```go
// NG: Auth mixed with business logic
func CreatePost(w http.ResponseWriter, r *http.Request) {
    token := r.Header.Get("Authorization")  // ← HTTP dependency
    if !validateJWT(token) {                 // ← Auth buried here
        w.WriteHeader(401)
        return
    }
    // ...DB logic
}
```

In this case:

- Every endpoint **duplicates** the auth check
- Auth method changes **propagate to all endpoints**
- UseCase tests need to mock authentication
- "Unauthenticated version" requires code duplication

---

## Key Points

1. **Cross-Cutting Concern Separation**: Authentication is implemented as Presentation layer middleware, never mixing with business logic (UseCase).
2. **Inner Layers Unchanged**: Domain, UseCase, and Infra don't know authentication exists. `Execute(ctx, Input)` signature is unchanged.
3. **Auth Method Swap**: JWT → API Key → OAuth changes require only middleware replacement. UseCase is untouched.
4. **Selective Application**: Granular control like "reads are unauthenticated, writes require auth" is handled entirely in the Router.
5. **Testability**: UseCase tests don't need to mock authentication. Only Handler tests validate auth middleware.
