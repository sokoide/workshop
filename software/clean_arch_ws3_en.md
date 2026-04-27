# Clean Architecture Workshop (WS3): Swapping Communication Protocols

In this workshop, you will migrate the BBS (2channel-style bulletin board) REST API to gRPC.
You will modify **only the Framework layer**, confirming that Domain, UseCase, and Infra remain completely untouched.

## BBS App Overview

The BBS in this workshop is a simple 3-tier bulletin board: Boards → Threads → Posts.

| Resource | Description |
| -------- | ----------- |
| **Board** | A bulletin board. Identified by `name` (e.g., `programming`) |
| **Thread** | A thread. Belongs to a specific Board, identified by `threadID` (numeric) |
| **Post** | A post (reply). Belongs to a specific Thread, with sequential numbering. The `sage` flag prevents the thread from floating to the top |

### REST API List

The current HTTP endpoints are these 5:

| Method | Path | Description |
| -------- | ---- | ----------- |
| GET | `/api/boards` | List boards |
| GET | `/api/boards/{name}/threads` | List threads |
| POST | `/api/boards/{name}/threads` | Create thread |
| GET | `/api/threads/{threadID}/posts` | List posts |
| POST | `/api/threads/{threadID}/posts` | Create reply |

### curl Usage Examples

```bash
# List boards
curl localhost:8080/api/boards

# List threads
curl localhost:8080/api/boards/programming/threads

# Create thread
curl -X POST localhost:8080/api/boards/programming/threads \
  -H 'Content-Type: application/json' \
  -d '{"title":"Test","author":"anonymous","body":"First post"}'

# List posts
curl localhost:8080/api/threads/1/posts

# Create reply (with sage)
curl -X POST localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"Anonymous","body":"sage","sage":true}'
```

### Step-by-Step Example (Create Thread → Reply)

```bash
# 1. Check boards
curl localhost:8080/api/boards
# → {"boards":[{"id":1,"name":"programming","name":"Programming General","created_at":"2025-01-01T00:00:00Z"}]}

# 2. Create a thread on the programming board
curl -s -X POST localhost:8080/api/boards/programming/threads \
  -H 'Content-Type: application/json' \
  -d '{"title":"Go Thread","author":"gopher","body":"Go is great!"}' | jq .
# → { "thread": { "id": 1, "title": "Go Thread", ... }, "post": { "id": 1, "number": 1, ... } }

# 3. Post a reply (using the thread.id from step 2)
curl -s -X POST localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"Anon","body":"Indeed"}' | jq .
# → { "id": 2, "number": 2, "author": "Anon", "body": "Indeed", ... }

# 4. Post a reply with sage
curl -s -X POST localhost:8080/api/threads/1/posts \
  -H 'Content-Type: application/json' \
  -d '{"author":"sage","body":"sage","sage":true}' | jq .
# → { "id": 3, "number": 3, "sage": true, ... }

# 5. Verify the posts list
curl -s localhost:8080/api/threads/1/posts | jq .
# → [ { "number": 1, "author": "gopher", "body": "Go is great!" },
#     { "number": 2, "author": "Anon", "body": "Indeed" },
#     { "number": 3, "author": "sage", "body": "sage", "sage": true } ]
```

### About "sage" and Thread Bumping

This BBS implements a 2channel-style "thread bumping" mechanism.

| Term | Meaning | Behavior |
| :--- | :--- | :--- |
| **age** | "raise" | Updates `LastPostedAt`, moving thread to top |
| **sage** | "lower" | Does NOT update `LastPostedAt` |

**Default behavior:**

- `sage: false` or omitted → **bump** (thread floats to top)
- `sage: true` → **no bump** (thread stays in place)

**Thread list ordering:**

```sql
ORDER BY last_posted_at DESC  -- Most recently active threads appear first
```

**Concrete example:**

```bash
# 1. Three threads created: Ruby, Python, Go
#    List: Ruby(10:00) → Python(10:05) → Go(10:10)

# 2. Post a bump reply to Ruby thread (sage: false)
curl -X POST localhost:8080/api/threads/3/posts -d '{"author":"A","body":"hello"}'
#    Ruby's LastPostedAt updates to now → floats to top
#    List: Ruby(10:15) ← bumped! → Go(10:10) → Python(10:05)

# 3. Post a no-bump reply to Python thread (sage: true)
curl -X POST localhost:8080/api/threads/2/posts -d '{"author":"B","body":"sage","sage":true}'
#    Python's LastPostedAt remains old → position unchanged
#    List: Ruby(10:15) → Go(10:10) → Python(10:05) ← stays put
```

**When to use:**

- **bump (default)**: When you want to indicate "this conversation is active"
- **no bump (sage)**: For minor updates, corrections, or when you don't want to disturb older threads

**Implementation logic:**

```go
// Thread.Bump() - Updates the thread's last post timestamp
func (t *Thread) Bump(postedAt time.Time, sage bool) {
    t.PostCount++
    if !sage {  // Only update if NOT sage
        t.LastPostedAt = postedAt
    }
}
```

Note: The first post in a new thread is always a bump, so new threads appear at the top of the list.

---

## Prerequisites

This workshop uses the [BBS project](./assets/bbs/) as the subject code.
You should understand the 4-layer structure:

```text
Framework (HTTP/gRPC)  →  UseCase (Application logic)  →  Domain (Business rules)
                                                         →  Port Interface (abstraction)
                              ↑                                   ↑
                         Infra Adapter (DB) ─────────────────────┘  (DIP: concrete depends on abstract)
```

## Workshop Scenario

You need to migrate the REST API to gRPC without touching any business logic.

---

## Exercise: HTTP → gRPC Migration

### Identify the Scope (What NOT to Change)

The following layers require **zero modifications**:

| Layer | Reason |
| ------- | -------- |
| **Domain** | Entities (Board, Thread, Post), Port Interfaces do not depend on any communication protocol |
| **UseCase** | `Execute(ctx, Input) (Output, error)` signature is unchanged. DTOs are protocol-agnostic |
| **Infra** | Repository implementations (SQL queries) are unrelated to communication methods |

### Step 1: Review the Current HTTP Handler

Review the current Framework layer. Handlers follow the "input conversion → UseCase call → output conversion" pattern.

```go
// framework/handler/thread_handler.go
func (h *ThreadHandler) CreateThread(w http.ResponseWriter, r *http.Request) {
    // HTTP-specific input conversion
    name := r.PathValue("name")
    var req createThreadRequest
    json.NewDecoder(r.Body).Decode(&req)

    // UseCase call (← protocol-independent)
    out, err := h.createThread.Execute(r.Context(), usecase.CreateThreadInput{
        BoardSlug: name, Title: req.Title, Author: req.Author, Body: req.Body,
    })

    // HTTP-specific output conversion
    if errors.Is(err, domain.ErrBoardNotFound) {
        writeError(w, http.StatusNotFound, err.Error())
        return
    }
    writeJSON(w, http.StatusCreated, out)
}
```

**Key observation**: The `Execute` call does not depend on HTTP or gRPC.

### Step 2: Create Protobuf Definitions

Create gRPC type definitions. This is a new file — no existing code is modified.

```protobuf
// api/proto/bbs.proto
syntax = "proto3";
package bbs;

service BBSService {
    rpc ListBoards(ListBoardsRequest) returns (ListBoardsResponse);
    rpc ListThreads(ListThreadsRequest) returns (ListThreadsResponse);
    rpc CreateThread(CreateThreadRequest) returns (CreateThreadResponse);
    rpc ListPosts(ListPostsRequest) returns (ListPostsResponse);
    rpc CreatePost(CreatePostRequest) returns (CreatePostResponse);
}

message CreateThreadRequest {
    string board_name = 1;
    string title = 2;
    string author = 3;
    string body = 4;
}

message CreateThreadResponse {
    Thread thread = 1;
}

message Thread {
    int64 id = 1;
    string title = 2;
    string author = 3;
    int32 response_count = 4;
    string created_at = 5;
    string last_posted_at = 6;
}
```

### Step 3: Implement the gRPC Handler

Create a gRPC handler in a new file. Confirm that the **UseCase invocation is identical to the HTTP version**.

```go
// framework/grpc/bbs_server.go (new file)
type BBSServer struct {
    pb.UnimplementedBBSServiceServer
    createThread usecase.CreateThreadUseCase
    listThreads  usecase.ListThreadsUseCase
    // ...
}

func (s *BBSServer) CreateThread(ctx context.Context, req *pb.CreateThreadRequest) (*pb.CreateThreadResponse, error) {
    // gRPC-specific input conversion (only this part differs)
    out, err := s.createThread.Execute(ctx, usecase.CreateThreadInput{
        BoardSlug: req.BoardSlug,
        Title:     req.Title,
        Author:    req.Author,
        Body:      req.Body,
    })

    // gRPC-specific output conversion (HTTP status → gRPC status)
    if errors.Is(err, domain.ErrBoardNotFound) {
        return nil, status.Errorf(codes.NotFound, err.Error())
    }
    if err != nil {
        return nil, status.Errorf(codes.Internal, err.Error())
    }

    return &pb.CreateThreadResponse{Thread: toProtoThread(out)}, nil
}
```

**Compare**: Line up the HTTP and gRPC `Execute` calls side by side:

```go
// HTTP version
out, err := h.createThread.Execute(r.Context(), usecase.CreateThreadInput{
    BoardSlug: name, Title: req.Title, Author: req.Author, Body: req.Body,
})

// gRPC version (identical!)
out, err := s.createThread.Execute(ctx, usecase.CreateThreadInput{
    BoardSlug: req.BoardSlug, Title: req.Title, Author: req.Author, Body: req.Body,
})
```

The UseCase call is completely identical. Only the conversion target (JSON → protobuf) and error mapping (HTTP status → gRPC status) differ.

### Step 4: Switch the Entry Point

Change the startup method from HTTP to gRPC in the Composition Root.

```go
// cmd/bbs/main.go
func main() {
    // DI wiring (UseCase injection) remains unchanged
    boardRepo := sqlite.NewBoardRepository(db)
    threadRepo := sqlite.NewThreadRepository(db)
    postRepo := sqlite.NewPostRepository(db)
    tm := sqlite.NewTransactionManager(db)

    createThread := usecase.NewCreateThreadUseCase(boardRepo, threadRepo, postRepo, tm)
    listThreads := usecase.NewListThreadsUseCase(boardRepo, threadRepo)
    // ...

    // Old: HTTP server startup
    // handler := framework.NewThreadHandler(createThread, listThreads)
    // http.ListenAndServe(":8080", router)

    // New: gRPC server startup (only this changes)
    grpcServer := grpc.NewServer()
    bbsServer := framework.NewBBSServer(createThread, listThreads, ...)
    pb.RegisterBBSServiceServer(grpcServer, bbsServer)

    lis, _ := net.Listen("tcp", ":9090")
    grpcServer.Serve(lis)
}
```

### Step 5: Verify

```bash
# Build
go build -o bbs ./cmd/bbs/

# Start
./bbs

# Test with gRPC client (using grpcurl)
grpcurl -plaintext localhost:9090 bbs.BBSService/ListBoards

grpcurl -plaintext -d '{"board_name":"program","title":"Go thread","author":"gopher","body":"Go is great!"}' \
    localhost:9090 bbs.BBSService/CreateThread
```

---

## Comparison: Without Layer Separation

```go
// Everything mixed into one function
func CreateThread(w http.ResponseWriter, r *http.Request) {
    name := r.PathValue("name")           // HTTP dependency
    db, _ := sql.Open("sqlite3", "bbs.db") // DB dependency
    tx, _ := db.Begin()                    // Technical detail
    res, _ := tx.Exec("INSERT INTO ...")   // SQL
    w.WriteHeader(201)                     // HTTP dependency
    json.NewEncoder(w).Encode(res)         // JSON dependency
}
```

In this case, switching to gRPC requires **rewriting the entire function**.

---

## Key Points

1. **Localized Impact**: Only the Framework layer changes. UseCase's `Execute` call remains untouched.
2. **Protocol Differences Are "Conversion" Differences**: Both HTTP and gRPC follow the same "convert input to DTO → call UseCase" structure.
3. **Easy Swapping**: Production uses gRPC, internal tools use HTTP, tests use CLI — all sharing the same UseCases.
