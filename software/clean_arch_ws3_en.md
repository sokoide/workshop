# Clean Architecture Workshop (WS3): Swapping Communication Protocols

> **⏱️ Estimated Time**: Approx. 90 minutes
> **🎯 Level**: Intermediate
> **💻 Environment**: Go 1.26.2+, HTTP → gRPC, Linux / macOS / WSL2

## Goal

- Connect HTTP and gRPC input/output to the same use cases.
- Verify that changes stay in the Presentation Adapter and Composition Root.

## Prerequisites

- [ ] Go 1.26.2+, C compiler (SQLite CGO), Python 3, curl, grpcurl
- [ ] WS1/WS2 completed, or understanding of ports and dependency injection
- [ ] Ports 8080 and 9090 available
- protoc and generation plugins are needed only for regeneration; generated Go code is included.

## Start and Finish States

**Start:** HTTP + SQLite BBS, including owner-only mode. **Finish:** the same features via gRPC on `127.0.0.1:9090`. WS3 is independent: do not carry this copy into WS4.

Prepare from the repository root. Existing copies are never overwritten; choose another name for a rerun.

```bash
# Terminal B — repository root
export WORKSHOP_REPO="$PWD"
python3 tools/prepare_bbs.py ws3 /tmp/bbs-ws3
cd /tmp/bbs-ws3
go test ./...
# Expected output: Prepared ws3 ..., existing tests pass
```

## Steps

### STEP 1: Check HTTP Baseline

```bash
# Terminal A
cd /tmp/bbs-ws3
go run ./cmd/bbs
# Expected output: server starting addr=:8080
```

```bash
# Terminal B — /tmp/bbs-ws3
curl -i http://localhost:8080/api/boards
# Expected output: 200 OK; boards contains programming / news / chat
```

### ✅ Checkpoint: Baseline

- [ ] Retrieved boards over HTTP
- [ ] Predicted which files must change for gRPC

### STEP 2: Add Proto and gRPC Adapter

Read the [complete Proto](assets/bbs/api/proto/bbs.proto) and [complete adapter](assets/bbs/internal/adapters/presentation/grpc/server.go). Inputs map to usecase DTOs; outputs map to Proto. Domain errors map to gRPC status; unknown error details are hidden.

**Runnable commands:** copy the reference implementation. If implementing it yourself, provide all five RPCs and DTO mappings before running the same checks.

```bash
# Terminal B — /tmp/bbs-ws3
mkdir -p api/proto
cp "$WORKSHOP_REPO/software/assets/bbs/api/proto/bbs.proto" api/proto/
cp -R "$WORKSHOP_REPO/software/assets/bbs/internal/adapters/presentation/grpc" internal/adapters/presentation/
mkdir -p /tmp/bbs-ws3-check
go build -o /tmp/bbs-ws3-check/ ./...
# Expected output: no compilation errors (directory output because there are multiple main packages)
```

The included `pb/` matches this Proto. To regenerate it, follow pinned-tool instructions in the [BBS README](assets/bbs/README.md).

### STEP 3: Replace the Composition Root

Stop terminal A with Ctrl+C. This is a complete replacement of the existing `main.go`. Compare the original wiring with the [gRPC Composition Root](assets/bbs/cmd/bbs-grpc/main.go), then copy it:

```bash
# Terminal B — /tmp/bbs-ws3
cp "$WORKSHOP_REPO/software/assets/bbs/cmd/bbs-grpc/main.go" cmd/bbs/main.go
go test ./...
# Expected output: all pass
```

```bash
# Terminal A — /tmp/bbs-ws3
go run ./cmd/bbs
# Expected output: gRPC server starting addr=127.0.0.1:9090
```

### STEP 4: Verify the Same Features over gRPC

```bash
# Terminal B — /tmp/bbs-ws3
grpcurl -plaintext localhost:9090 bbs.BBSService/ListBoards
# Expected output: boards contains programming / news / chat
grpcurl -plaintext -d '{"board_name":"programming","title":"gRPC","author":"gopher","body":"hello"}' localhost:9090 bbs.BBSService/CreateThread
# Expected output: JSON containing thread and post (number: 1)
grpcurl -plaintext -d '{"board_name":"missing","title":"x","body":"x"}' localhost:9090 bbs.BBSService/CreateThread
# Expected output: Code: NotFound (nonzero exit is expected)
diff -ru .baseline/internal/domain internal/domain
diff -ru .baseline/internal/usecase internal/usecase
diff -ru .baseline/internal/adapters/infra internal/adapters/infra
# Expected output: no differences from all three commands
```

### ✅ Checkpoint: Complete

- [ ] Board listing and thread creation succeed; missing board returns NotFound
- [ ] Domain, UseCase and Infra are unchanged, matching the initial prediction
- [ ] Can explain how transportError corresponds to HTTP status mapping

## What You Learned

- Proto is an outer transport DTO; usecase DTOs define the inner application boundary.
- Transport replacement changes the Presentation Adapter and Composition Root.
- Matching RPC input/output and error mapping allows reuse of the same use cases.

Start [WS4](clean_arch_ws4_en.md) from a new HTTP copy.

## Troubleshooting

| Symptom                | Check and fix                                                           |
| ---------------------- | ----------------------------------------------------------------------- |
| Cannot connect to 9090 | Inspect terminal A logs and check that the HTTP server was stopped      |
| Proto types missing    | Copy both the adapter and generated pb directory                        |
| WORKSHOP_REPO empty    | Use the prepared terminal B or reset it to the repository absolute path |

## Cleanup

Stop terminal A with Ctrl+C, then remove the copy in terminal B:

```bash
cd /tmp
rm -r /tmp/bbs-ws3
rm -rf /tmp/bbs-ws3-check
# Expected output: copy and build outputs removed; original untouched
```

## References

- [gRPC Go Quick Start](https://grpc.io/docs/languages/go/quickstart/)
- [Protocol Buffers Go Guide](https://protobuf.dev/reference/go/go-generated/)
- [Clean Architecture](clean_arch_en.md)
