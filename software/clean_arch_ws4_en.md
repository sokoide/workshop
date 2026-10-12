# Clean Architecture Workshop (WS4): Adding Business Rules

> **⏱️ Estimated Time**: Approx. 60 minutes
> **🎯 Level**: Intermediate
> **💻 Environment**: Go 1.26.2+, curl, Linux / macOS / WSL2

## Goal

- Implement the owner-only posting rule in Domain and call it from the UseCase.
- Verify allowed/denied outcomes and that denied posts are not saved.

## Prerequisites

- [ ] Go 1.26.2+, C compiler, Python 3, curl
- [ ] Understand WS1/WS2 layers and ports. WS3 is recommended; do not carry its gRPC code forward.

## Start and Finish States

**Start:** HTTP + SQLite. Entity Owner/OwnerOnly fields, DTOs, DB columns and HTTP 403 mapping are supplied as scaffolding. The CanPost decision and usecase rejection are TODOs; other authors are still allowed.

**Finish:** owner-only mode allows only the owner. The repository asset is the completed reference; use a prepared start copy. Focus on the two policy changes instead of recreating the supplied plumbing.

```bash
# Terminal B — repository root
export WORKSHOP_REPO="$PWD"
python3 tools/prepare_bbs.py ws4 /tmp/bbs-ws4
cd /tmp/bbs-ws4
go test ./...
# Expected output: start-state tests pass; rejection tests are added next
```

## Steps

### STEP 1: Add Rejection Tests First

**Complete test files:** copy the reference tests to expose the missing specification.

```bash
# Terminal B — /tmp/bbs-ws4
cp "$WORKSHOP_REPO/software/assets/bbs/internal/domain/entity/thread_test.go" internal/domain/entity/
cp "$WORKSHOP_REPO/software/assets/bbs/internal/usecase/post_usecase_test.go" internal/usecase/
go test ./internal/domain/entity ./internal/usecase
# Expected output: non-owner rejection tests FAIL (nonzero exit is expected here)
```

### STEP 2: Implement the Domain Decision

**Replace an existing method:** replace the TODO-containing CanPost in `internal/domain/entity/thread.go`:

```go
func (t *Thread) CanPost(author string) bool {
    if !t.OwnerOnly {
        return true
    }
    return t.Owner == author
}
```

### STEP 3: Reject before Saving in the UseCase

**Edit to an existing file:** replace the TODO in `internal/usecase/post_usecase.go`:

```go
if !thread.CanPost(in.Author) {
    return domain.ErrNotThreadOwner
}
```

Place the decision inside the transaction, before counting or saving posts. The existing HTTP handler maps this error to 403.

```bash
go test ./...
# Expected output: all tests, including Domain and UseCase, pass
```

### ✅ Checkpoint: Policy

- [ ] Anyone can post when owner-only mode is off
- [ ] Owner-only mode permits the matching author name and saves no denied post

### STEP 4: Verify over HTTP

```bash
# Terminal A
cd /tmp/bbs-ws4
go run ./cmd/bbs
# Expected output: server starts on :8080
```

```bash
# Terminal B — /tmp/bbs-ws4
THREAD_ID=$(curl -fsS -X POST http://localhost:8080/api/boards/programming/threads -H 'Content-Type: application/json' -d '{"title":"Owner only","author":"owner","body":"first","owner_only":true}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["thread"]["id"])')
# Expected output: THREAD_ID contains the newly created thread ID
curl -i -X POST "http://localhost:8080/api/threads/$THREAD_ID/posts" -H 'Content-Type: application/json' -d '{"author":"other","body":"denied"}'
# Expected output: 403 Forbidden
curl -i -X POST "http://localhost:8080/api/threads/$THREAD_ID/posts" -H 'Content-Type: application/json' -d '{"author":"owner","body":"allowed"}'
# Expected output: 201 Created, number: 2 (denied post was not counted)
```

### ✅ Checkpoint: Behavior and Understanding

- [ ] Other author gets 403, owner gets 201, post number is 2
- [ ] Can explain why the HTTP handler does not own the decision

<details>
<summary>Hint and sample answer</summary>

A Domain rule can be used from gRPC or CLI as well. The use case calls it before saving; Presentation maps the error to its transport. Author-name comparison is an educational identity model, not authenticated identity assurance. Authentication is covered in WS7.

</details>

## What You Learned

- Domain owns the decision, UseCase owns when to call it, Presentation maps the error.
- Failing tests expose the specification before the implementation passes.
- With DTO/DB plumbing supplied, the rule change can be learned independently.

Start [WS5](clean_arch_ws5_en.md) from a new completed HTTP copy.

## Troubleshooting

| Symptom                          | Fix                                                     |
| -------------------------------- | ------------------------------------------------------- |
| Rejection tests pass immediately | Prepare a fresh ws4 start state at a new path           |
| 404 response                     | Use the newly returned THREAD_ID rather than a fixed ID |
| Non-owner gets 201               | Check both CanPost and the usecase call before saving   |

## Cleanup

Stop terminal A with Ctrl+C and remove the copy:

```bash
# Terminal B
cd /tmp
rm -r /tmp/bbs-ws4
# Expected output: copy and SQLite DB removed; original untouched
```

## References

- [Clean Architecture](clean_arch_en.md)
- [Go Testing](https://pkg.go.dev/testing)
