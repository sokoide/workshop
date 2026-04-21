# Clean Architecture Workshop (WS3): Swapping Communication Protocols

In this workshop, you will migrate the BBS (2channel-style bulletin board) REST API to gRPC.
You will modify **only the Framework layer**, confirming that Domain, UseCase, and Infra remain completely untouched.

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
    slug := r.PathValue("slug")
    var req createThreadRequest
    json.NewDecoder(r.Body).Decode(&req)

    // UseCase call (← protocol-independent)
    out, err := h.createThread.Execute(r.Context(), usecase.CreateThreadInput{
        BoardSlug: slug, Title: req.Title, Author: req.Author, Body: req.Body,
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
    string board_slug = 1;
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
    BoardSlug: slug, Title: req.Title, Author: req.Author, Body: req.Body,
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

grpcurl -plaintext -d '{"board_slug":"program","title":"Go thread","author":"gopher","body":"Go is great!"}' \
    localhost:9090 bbs.BBSService/CreateThread
```

---

## Comparison: Without Layer Separation

```go
// Everything mixed into one function
func CreateThread(w http.ResponseWriter, r *http.Request) {
    slug := r.PathValue("slug")           // HTTP dependency
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
