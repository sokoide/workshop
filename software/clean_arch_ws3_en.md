# Clean Architecture Workshop (WS3): Swapping Communication Protocols

> **⏱️ Estimated Time**: Approx. 90 minutes
> **🎯 Level**: Intermediate
> **💻 Environment**: Go 1.22+, any OS (grpcurl required; protoc toolchain for Step 2)
>
> > **💡 Glossary**: For technical terms such as [Clean Architecture](../infra/glossary_en.md#architecture), [Interface](../infra/glossary_en.md#go), and [Context](../infra/glossary_en.md#go), please refer to the [Glossary](../infra/glossary_en.md).

In this workshop, you will migrate the BBS (2channel-style bulletin board) REST API to gRPC.
You will replace the **Presentation Adapter** and update the **Composition Root** that wires it, while leaving Domain, UseCase, and Infrastructure Adapters untouched.

## BBS App Overview

The BBS in this workshop is a simple 3-tier bulletin board: Boards → Threads → Posts. For detailed REST API specifications and usage examples, refer to the [BBS README (Japanese)](assets/bbs/README_ja.md).

> **Workshop Focus**: This workshop migrates the existing REST API to **gRPC** by replacing the Presentation Adapter (Handler / Router) and its Composition Root wiring. Domain, UseCase, and Infrastructure Adapters remain untouched.

### About "sage" and Thread Bumping

This BBS implements a 2channel-style "thread bumping" mechanism.

| Term     | Meaning | Behavior                                     |
| :------- | :------ | :------------------------------------------- |
| **age**  | "raise" | Updates `LastPostedAt`, moving thread to top |
| **sage** | "lower" | Does NOT update `LastPostedAt`               |

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

## BBS Project Structure and Dependencies

### Interface (Port) List and Placement Criteria

The "abstractions" (Ports) at the core of Clean Architecture are categorized based on their dependency on domain invariants:

- **Domain Ports**: Placed in the Domain layer if removing the interface would break the domain model's ability to enforce invariants (e.g., a domain policy port such as `PricingPolicy`).
- **UseCase Ports**: Placed in the UseCase layer if the concern is application orchestration (e.g., persistence or transaction boundary control) rather than core domain rules.

#### Domain Ports (Domain Layer)

None exist in the BBS project. Its Domain layer (entities and domain errors) enforces business invariants on its own, so it defines no port abstractions. In other projects, ports that express domain policies (e.g., `PricingPolicy`) would be placed here.

#### UseCase Ports (UseCase Layer)

The BBS defines all of its ports in the UseCase layer. Persistence and transaction boundaries are capabilities needed to complete an application workflow — not part of the core domain language — so the interfaces are owned by the UseCase layer.

| File                              | Interface            | Purpose                                            |
| :-------------------------------- | :------------------- | :------------------------------------------------- |
| `internal/usecase/repository.go`  | `BoardRepository`    | Board persistence (reconstitutes Domain entities)  |
|                                   | `ThreadRepository`   | Thread persistence (reconstitutes Domain entities) |
|                                   | `PostRepository`     | Post persistence (reconstitutes Domain entities)   |
| `internal/usecase/transaction.go` | `TransactionManager` | Transaction boundary control                       |

```go
// internal/usecase/repository.go
// Repository ports are owned by the application workflows that need them.
type BoardRepository interface {
    FindAll(ctx context.Context) ([]*entity.Board, error)
    FindByName(ctx context.Context, name string) (*entity.Board, error)
    Save(ctx context.Context, board *entity.Board) error
}
```

### Complete Dependency Diagram

```text
                   ┌───────────────────────────────────────────┐
                   │  cmd/bbs/main.go (Composition Root)       │
                   │  ───────────────────────────────────────  │
                   │  func main() {                            │
                   │      // 1. Build Infra                    │
                   │      boardRepo := sqlite.NewBoardRepo()   │
                   │      // 2. Build UseCase                  │
                   │      listThreads := usecase.New(...)      │
                   │         └── inject boardRepo              │
                   │      // 3. Build Presentation             │
                   │      handler := NewHandler(listThreads)   │
                   │         └── inject listThreads            │
                   │      // 4. Start server                   │
                   │      http.Serve(router)                   │
                   │  }                                        │
                   └───────────────────────────────────────────┘
                              │
                 ┌────────────┼────────────┐
                 │            │            │
                 ↓            ↓            ↓
        ┌────────────┐  ┌────────────┐  ┌─────────────┐
        │   Infra    │  │  UseCase   │  │ Presentation│
        │  Adapter   │  │   Layer    │  │    Layer    │
        └──────┬─────┘  └──────┬─────┘  └──────┬──────┘
               │               │               │
               │               │               │
┌──────────────┴───────────────┴───────────────┴─────────────────────┐
│                   INTERNAL/DOMAIN + USECASE                        │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  internal/domain/entity/board.go, thread.go, post.go         │  │
│  │  ─────────────────────────────────────────────────────────   │  │
│  │  type Board struct { ID, Name, Description, CreatedAt }      │  │
│  │  type Thread struct { ... }                                  │  │
│  │  type Post struct { ... }                                    │  │
│  │                                                              │  │
│  │  // Domain Logic example                                     │  │
│  │  func (t *Thread) Bump(postedAt time.Time, sage bool) {      │  │
│  │      t.PostCount++                                           │  │
│  │      if !sage { t.LastPostedAt = postedAt }                  │  │
│  │  }                                                           │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  internal/usecase/repository.go     ← UseCase Port           │  │
│  │  type BoardRepository interface { ... }                      │  │
│  └──────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────┘
               ↑ implements
┌──────────────┴───────────────┐
│  internal/adapters/infra/persistence/ │
│  sqlite/board_repo.go        │  ← Infra Adapter (concrete impl)
│  func (r *BoardRepository)   │     (implements UseCase Port)
│  FindByName(...) {           │
│     SELECT ... WHERE name = ?│
│  }                           │
└──────────────────────────────┘
```

### Dependency Direction by Layer

```text
           ┌─────────────────────────────────────┐
           │         Adapters Layer              │
           │  ┌────────────┐ ┌────────────────┐  │
           │  │Presentation│ │ Infrastructure │  │
           │  │ (Inbound)  │ │  (Outbound)    │  │
           │  │HTTP/gRPC/  │ │  DB/External   │  │
           │  │  CLI       │ │    APIs        │  │
           │  └──────┬─────┘ └────────────────┘  │
           └─────────┼───────────────────────────┘
                     │ depends on
                     ↓
           ┌─────────────┐
           │   UseCase   │  ← Application logic
           └──────┬──────┘
                  │ depends on
                  ↓
           ┌────────────────────────────────────┐
           │         Domain Layer               │
           │  Entity + Domain Logic             │  ← Innermost
           └────────────────────────────────────┘
                  ↑ implements
           Infrastructure Adapters (implement Domain/UseCase Ports)
```

### Error Boundary Flow

In Clean Architecture, errors are also transformed as they cross layer boundaries.

```text
┌─────────────────────────────────────────────────────────────────────┐
│ Infra Adapter Layer                                                │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ func (r *ThreadRepo) FindByID(...) {                         │  │
│  │     err := db.QueryRow(...)                                  │  │
│  │     if errors.Is(err, sql.ErrNoRows) {                       │  │
│  │         return nil, domain.ErrThreadNotFound  // ← Convert  │  │
│  │     }                                                        │  │
│  │ }                                                            │  │
│  └──────────────────────────────────────────────────────────────┘  │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ returns domain error
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ UseCase Layer                                                      │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ func (u *CreatePostUseCase) Execute(...) {                  │  │
│  │     thread, err := u.threadRepo.FindByID(...)               │  │
│  │     // err is domain.ErrThreadNotFound or other domain errs  │  │
│  │     return nil, err  // ← Propagate as-is                   │  │
│  │ }                                                            │  │
│  └──────────────────────────────────────────────────────────────┘  │
└───────────────────────────────┬─────────────────────────────────────┘
                                │ returns domain error
                                ↓
┌─────────────────────────────────────────────────────────────────────┐
│ Presentation Layer (HTTP)                                              │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ if errors.Is(err, domain.ErrThreadNotFound) {                │  │
│  │     writeError(w, http.StatusNotFound, err.Error())  // ← Convert │  │
│  │ }                                                            │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
```

**Key points**:

- **Infra Adapter**: Converts driver errors (e.g., `sql.ErrNoRows`) to domain errors (e.g., `domain.ErrThreadNotFound`)
- **UseCase**: Propagates domain errors as-is (unaware of technical details)
- **Presentation**: Converts domain errors to transport errors (HTTP 404, gRPC NotFound)

With each layer handling error conversion, upper layers don't need to know about lower-level technical details.

### Key Points

1. **UseCase doesn't know concrete implementations**: Depends on the `BoardRepository` interface (owned by the UseCase layer), not `sqlite.BoardRepository` (concrete type)
2. **Infra implements UseCase Ports**: `internal/adapters/infra/persistence/sqlite/` implements the port interfaces from `internal/usecase/`
3. **Presentation and Infra Adapters are not directly related**: Both are part of the Adapters layer but split by direction, connected indirectly through UseCase
4. **Only Composition Root (main.go) knows the whole picture**: Decides which Infra implementation and which Presentation to use

```go
// UseCase depends on Interface (not concrete)
type ListThreadsUseCase struct {
    boardRepo BoardRepository  // ← interface owned by UseCase, not sqlite.*
}

// Infra implements Interface
type BoardRepository struct {
    db *sql.DB
}

func (r *BoardRepository) FindByName(...) {  // ← implements Interface
    // SQLite-specific SQL
}
```

---

## Prerequisites

This workshop uses the [BBS project](./assets/bbs/) as the subject code.
You should understand the 3-layer variant (Adapters / UseCases / Domain):

```text
Presentation Adapters ──→ UseCases ──→ Domain
                            ↑              ↑
                        Infrastructure Adapters
        implements ports owned by UseCases or Domain
```

Checklist before starting:

- [ ] Completed [WS1](./clean_arch_ws1_en.md) and [WS2](./clean_arch_ws2_en.md) (or you are comfortable with ports and the Composition Root)
- [ ] Go 1.22+ installed — verify with `go version`
- [ ] The BBS project builds: `cd software/assets/bbs && go build ./...` exits with no output
- [ ] `grpcurl` installed (`grpcurl --version`) — see [grpcurl releases](https://github.com/fullstorydev/grpcurl/releases)
- [ ] For Step 2 only: `protoc`, `protoc-gen-go`, and `protoc-gen-go-grpc` installed — see [gRPC Go quick start](https://grpc.io/docs/languages/go/quickstart/)

## Workshop Scenario

You need to migrate the REST API to gRPC without touching any business logic.

---

## Exercise: HTTP → gRPC Migration

### Identify the Scope (What NOT to Change)

The following layers require **zero modifications**:

| Layer       | Reason                                                                                            |
| ----------- | ------------------------------------------------------------------------------------------------- |
| **Domain**  | Entities (Board, Thread, Post) and domain errors do not depend on any communication protocol      |
| **UseCase** | `Execute(ctx, Input) (Output, error)` signature is unchanged. DTOs are protocol-agnostic          |
| **Infra**   | Repository implementations (SQL queries, error conversion) are unrelated to communication methods |

### Step 1: Review the Current HTTP Handler

Review the current Presentation layer. Handlers follow the "input conversion → UseCase call → output conversion" pattern.

```go
// internal/adapters/presentation/http/handler/thread_handler.go
func (h *ThreadHandler) CreateThread(w http.ResponseWriter, r *http.Request) {
    // HTTP-specific input conversion
    name := r.PathValue("name")
    var req createThreadRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid json")
        return
    }

    // UseCase call (← protocol-independent)
    out, err := h.createThread.Execute(r.Context(), usecase.CreateThreadInput{
        BoardName: name, Title: req.Title, Author: req.Author, Body: req.Body,
    })

    // HTTP-specific output conversion
    if errors.Is(err, domain.ErrBoardNotFound) {
        writeError(w, http.StatusNotFound, err.Error())
        return
    }
    if err != nil {
        // For non-domain errors, hide details and return generic message
        writeError(w, http.StatusInternalServerError, "internal server error")
        return
    }
    writeJSON(w, http.StatusCreated, out)
}
```

**Key observation**: The `Execute` call does not depend on HTTP or gRPC.

### ✅ Checkpoint

- [ ] You located the "input conversion → UseCase call → output conversion" structure in the HTTP handler
- [ ] You confirmed the `Execute` call line contains nothing HTTP-specific (except `r.Context()`)
- [ ] `go build ./...` still passes in `software/assets/bbs` (nothing changed yet)

### Step 2: Create Protobuf Definitions

Create gRPC type definitions. This is a new file — no existing code is modified.

```protobuf
// api/proto/bbs.proto
syntax = "proto3";
package bbs;

import "google/protobuf/timestamp.proto";

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
    Post post = 2;
}

message Thread {
    int64 id = 1;
    int64 board_id = 2;
    string title = 3;
    int32 post_count = 4;
    google.protobuf.Timestamp created_at = 5;
    google.protobuf.Timestamp last_posted_at = 6;
}

message Post {
    int64 id = 1;
    int64 thread_id = 2;
    int32 number = 3;
    string author = 4;
    string body = 5;
    bool sage = 6;
    google.protobuf.Timestamp created_at = 7;
}
```

### ✅ Checkpoint

- [ ] `api/proto/bbs.proto` is a **new file** — no existing file was modified (`git status` shows only the new .proto)
- [ ] Code generation succeeds: `protoc --go_out=. --go-grpc_out=. api/proto/bbs.proto` exits with no output
- [ ] Generated `*.pb.go` files appear under your chosen output directory and `go build ./...` passes

### Step 3: Implement the gRPC Handler

Create a gRPC handler in a new file. Confirm that the **UseCase invocation is identical to the HTTP version**.

First, define the Input Port interfaces in the UseCases layer (if not already defined):

```go
// internal/usecase/port.go
package usecase

import "context"

type ThreadCreator interface {
    Execute(ctx context.Context, in CreateThreadInput) (*CreateThreadOutput, error)
}

type ThreadLister interface {
    Execute(ctx context.Context, in ListThreadsInput) (*ListThreadsOutput, error)
}

// Define similar interfaces for other UseCases as needed
```

Then implement the gRPC handler depending on these interfaces:

```go
// internal/adapters/presentation/grpc/bbs_server.go (new file)
type BBSServer struct {
    pb.UnimplementedBBSServiceServer
    createThread usecase.ThreadCreator  // ← interface (Input Port)
    listThreads  usecase.ThreadLister   // ← interface (Input Port)
    // ...
}

func (s *BBSServer) CreateThread(ctx context.Context, req *pb.CreateThreadRequest) (*pb.CreateThreadResponse, error) {
    // gRPC-specific input conversion (only this part differs)
    out, err := s.createThread.Execute(ctx, usecase.CreateThreadInput{
        BoardName: req.BoardName,
        Title:     req.Title,
        Author:    req.Author,
        Body:      req.Body,
    })

    // gRPC-specific output conversion (HTTP status → gRPC status)
    if errors.Is(err, domain.ErrBoardNotFound) {
        return nil, status.Errorf(codes.NotFound, err.Error())
    }
    if err != nil {
        // For non-domain errors, hide details and return generic message
        return nil, status.Errorf(codes.Internal, "internal server error")
    }

    return &pb.CreateThreadResponse{
        Thread: toProtoThread(out.Thread),
        Post:   toProtoPost(out.Post),
    }, nil
}
```

**Compare**: Line up the HTTP and gRPC `Execute` calls side by side:

```go
// HTTP version
out, err := h.createThread.Execute(r.Context(), usecase.CreateThreadInput{
    BoardName: name, Title: req.Title, Author: req.Author, Body: req.Body,
})

// gRPC version (identical!)
out, err := s.createThread.Execute(ctx, usecase.CreateThreadInput{
    BoardName: req.BoardName, Title: req.Title, Author: req.Author, Body: req.Body,
})
```

The UseCase call is completely identical. Only the conversion target (JSON → protobuf) and error mapping (HTTP status → gRPC status) differ.

### ✅ Checkpoint

- [ ] `bbs_server.go` is a **new file** under `internal/adapters/presentation/grpc/` — Domain, UseCase, and Infra are untouched (`git status` confirms)
- [ ] `BBSServer` depends on `usecase.ThreadCreator` / `usecase.ThreadLister` **interfaces**, not concrete usecase structs
- [ ] Error conversion lives only here: `domain.ErrBoardNotFound → codes.NotFound`, unknown errors → `codes.Internal` with a generic message
- [ ] `go build ./...` passes

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
    // handler := presentation.NewThreadHandler(createThread, listThreads)
    // http.ListenAndServe(":8080", router)

    // New: gRPC server startup (only this changes)
    grpcServer := grpc.NewServer()
    reflection.Register(grpcServer)  // Required for grpcurl service listing
    bbsServer := presentation.NewBBSServer(createThread, listThreads, ...)  // concrete types satisfy interfaces
    pb.RegisterBBSServiceServer(grpcServer, bbsServer)

    lis, err := net.Listen("tcp", ":9090")
    if err != nil {
        slog.Error("failed to listen", "error", err)
        os.Exit(1)
    }
    if err := grpcServer.Serve(lis); err != nil {
        slog.Error("grpc server failed", "error", err)
        os.Exit(1)
    }
}
```

### ✅ Checkpoint

- [ ] Only the server startup block changed in `main.go` — the DI wiring (repositories, usecases) is untouched
- [ ] `go build -o bbs ./cmd/bbs/` still passes after the switch

### Step 5: Verify

```bash
# Build
go build -o bbs ./cmd/bbs/
# Expected output: (none — the bbs binary appears in the working directory)

# Start (keep it running in another terminal)
./bbs
# Expected output: the server starts and listens on :9090

# Test with gRPC client (using grpcurl)
grpcurl -plaintext localhost:9090 bbs.BBSService/ListBoards
# Expected output: JSON listing the seeded boards, e.g.
# {
#   "board": [
#     { "id": "1", "name": "programming", "description": "Programming General" },
#     ...
#   ]
# }

grpcurl -plaintext -d '{"board_name":"program","title":"Go thread","author":"gopher","body":"Go is great!"}' \
    localhost:9090 bbs.BBSService/CreateThread
# Expected output: JSON containing the created "thread" (postCount: "1")
# and the first "post" (number: "1")
```

### ✅ Checkpoint

- [ ] `grpcurl -plaintext localhost:9090 bbs.BBSService/ListBoards` returns the board list as JSON
- [ ] `CreateThread` via gRPC creates the thread and returns both the thread and the first post
- [ ] You confirmed no file outside the Presentation Adapter and `main.go` changed: `git diff --stat` lists only presentation/grpc files, generated .pb.go files, and `cmd/bbs/main.go`
- [ ] `go test ./...` passes in `software/assets/bbs`

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

1. **Localized Impact**: Changes stay in the Presentation Adapter and Composition Root wiring. UseCase's `Execute` call remains untouched.
2. **Protocol Differences Are "Conversion" Differences**: Both HTTP and gRPC follow the same "convert input to DTO → call UseCase" structure.
3. **Easy Swapping**: Production uses gRPC, internal tools use HTTP, tests use CLI — all sharing the same UseCases.

---

## Comprehension Check

Try answering these questions yourself.

### Question 1: Protobuf and the Domain Layer

What problems arise if you directly use Go structs generated from .proto files (e.g., `pb.Post`) as Domain layer entities? According to Clean Architecture, in which layer and how should the conversion be performed?

### Question 2: Differences in Error Handling

In HTTP, returning `404 Not Found` is natural, but in gRPC, you use `status.NotFound`. In which layer should error conversion be performed? If you write "for HTTP" and "for gRPC" branching in the UseCase layer, what problems would occur?

### Question 3: Transaction Boundaries

`CreateThreadUseCase` performs "thread creation" and "first post" in the same transaction. To which layer should this transaction control belong? If you start and end transactions in the Presentation layer (HTTP Handler), which Clean Architecture principle would be violated?

---

## What you learned

- Swapping the transport protocol only touches the **Presentation Adapter** and the **Composition Root** wiring — the `Execute(ctx, Input) (Output, error)` call is byte-for-byte identical between HTTP and gRPC handlers.
- Protocol differences are pure **conversion**: JSON vs protobuf on input, HTTP status vs gRPC status codes on errors.
- Errors cross boundaries translated at each layer: driver errors → domain errors (Infra), domain errors → transport errors (Presentation).
- Protobuf-generated structs stay out of the Domain layer; Presentation adapters map them to/from UseCase DTOs.

**Next**: [WS4: Adding Business Rules](./clean_arch_ws4_en.md) — add an owner-only posting rule and watch the change propagate from Domain outward.

---

## Cleanup

```bash
# Stop the gRPC server (Ctrl+C in its terminal), then:
cd software/assets/bbs
make clean   # removes the bbs binary and bbs.db
```

Or manually:

```bash
rm -f software/assets/bbs/bbs software/assets/bbs/bbs.db
```

---

## References

- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers Language Specification](https://protobuf.dev/programming-guides/proto3/)
- [grpcurl — GitHub](https://github.com/fullstorydev/grpcurl)
- [gRPC Go Quick Start](https://grpc.io/docs/languages/go/quickstart/)
