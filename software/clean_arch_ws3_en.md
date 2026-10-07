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
For the API specification and the `sage` request parameter, refer to the [BBS README (Japanese)](assets/bbs/README_ja.md#レスの投稿).

> **Summary**: `age` ("raise") bumps the thread to the top of the list by updating `LastPostedAt`; `sage` ("lower") does not update it, so the thread stays in place. The default (`sage: false` or omitted) is a bump.

---

## BBS Project Structure and Dependencies

The BBS follows the 3-layer variant (Adapters / UseCases / Domain) described in [clean_arch.md](./clean_arch_en.md): dependencies point inward — Presentation Adapters call UseCases, and Infrastructure Adapters implement ports owned by UseCases or Domain. Presentation and Infra Adapters never depend on each other directly; only the Composition Root (`cmd/bbs/main.go`) knows the whole picture and decides which Infra implementation and which Presentation to wire together. For the layer diagram, dependency matrix, and layer definitions, see [Layer Structure and Dependencies](./clean_arch_en.md#layer-structure-and-dependencies); for the BBS-specific composition diagram, see the [BBS README](assets/bbs/README_ja.md#アーキテクチャ構成図).

### Ports in the BBS Project

The Domain layer defines no ports — its entities and domain errors enforce business invariants on their own. All ports are owned by the UseCase layer because persistence and transaction boundaries are capabilities needed to complete an application workflow, not part of the core domain language. For the placement criteria, see [Port Ownership Guidance](./clean_arch_en.md#port-ownership-guidance).

| File                              | Interface            | Purpose                                            |
| :-------------------------------- | :------------------- | :------------------------------------------------- |
| `internal/usecase/repository.go`  | `BoardRepository`    | Board persistence (reconstitutes Domain entities)  |
|                                   | `ThreadRepository`   | Thread persistence (reconstitutes Domain entities) |
|                                   | `PostRepository`     | Post persistence (reconstitutes Domain entities)   |
| `internal/usecase/transaction.go` | `TransactionManager` | Transaction boundary control                       |

### Error Boundaries

Errors are transformed as they cross layer boundaries: the Infra Adapter converts driver errors (`sql.ErrNoRows`) to domain errors (`domain.ErrThreadNotFound`), the UseCase propagates domain errors as-is, and the Presentation Adapter converts them to transport errors (HTTP 404 / gRPC `NotFound`). For the full rules, see [Error Boundary Rules](./clean_arch_en.md#error-boundary-rules).

---

## Prerequisites

This workshop uses the [BBS project](./assets/bbs/) as the subject code.
You should understand the 3-layer variant (Adapters / UseCases / Domain) — see [Layer Structure and Dependencies](./clean_arch_en.md#layer-structure-and-dependencies).

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
