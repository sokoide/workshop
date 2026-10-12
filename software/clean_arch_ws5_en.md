# Clean Architecture Workshop (WS5): Swapping Persistence

> **⏱️ Estimated Time**: Approx. 60 minutes (including environment setup)
> **🎯 Level**: Intermediate
> **💻 Environment**: Go 1.26.2+, C compiler, Podman, PostgreSQL 16, Python 3, curl

## Goal

Replace SQLite with PostgreSQL and verify board listing, thread creation and owner-only mode without editing Domain, UseCase or HTTP Adapter.

## Prerequisites

- [ ] Understand WS4 posting policy and transaction boundaries
- [ ] Go 1.26.2+, C compiler (SQLite start state), Python 3, curl
- [ ] Podman running; ports 8080 and 55432 free. Docker can run the podman commands below too.

## Start and Finish States

**Start:** HTTP + SQLite equivalent to completed WS4. **Finish:** same HTTP API + PostgreSQL. Do not carry WS3's gRPC version forward. Go dependencies are pinned in the prepared copy.

```bash
# Terminal B — repository root
export WORKSHOP_REPO="$PWD"
python3 tools/prepare_bbs.py ws5 /tmp/bbs-ws5
cd /tmp/bbs-ws5
go test ./...
# Expected output: Prepared ws5 ..., existing tests pass
```

## Steps

### STEP 1: Verify SQLite Baseline

```bash
# Terminal A
cd /tmp/bbs-ws5
go run ./cmd/bbs
# Expected output: server starts on :8080
```

```bash
# Terminal B — /tmp/bbs-ws5
curl -i http://localhost:8080/api/boards
# Expected output: 200 OK; boards contains programming / news / chat
```

Stop terminal A with Ctrl+C.

### STEP 2: Provide the PostgreSQL Adapter

**Predict:** which layers change when the same ports are satisfied? Answer, then compare [SQLite](assets/bbs/internal/adapters/infra/persistence/sqlite/) with the [complete PostgreSQL adapter](assets/bbs/internal/adapters/infra/persistence/postgres/).

| File                                          | Difference to study                                                          |
| --------------------------------------------- | ---------------------------------------------------------------------------- |
| board_repo.go / thread_repo.go / post_repo.go | `?` → `$1`, RETURNING id, boolean storage, error mapping                     |
| model.go                                      | DB model to Entity mapping                                                   |
| transaction.go                                | Implement usecase port; all repositories participate in the same transaction |
| db.go / schema.sql                            | pgx registration, connection check, complete DDL and initial boards          |

FindByID uses FOR UPDATE to avoid concurrent post-number allocation conflicts. Context carrying the transaction is interpreted inside Infra; the use case does not know sql.Tx.

**Runnable commands:** copy the complete files. If implementing them yourself, provide all files above and every repository-port method before checking.

```bash
# Terminal B — /tmp/bbs-ws5
cp -R "$WORKSHOP_REPO/software/assets/bbs/internal/adapters/infra/persistence/postgres" internal/adapters/infra/persistence/
go test ./...
# Expected output: build and tests pass; connection test SKIPs if BBS_TEST_DATABASE_URL is unset
```

### STEP 3: Start DB and Apply DDL and Seed Data

Use a dedicated exercise container. If the name is already taken, choose another name consistently below.

```bash
# Terminal B — /tmp/bbs-ws5
podman run -d --name bbs-ws5-postgres -p 127.0.0.1:55432:5432   -e POSTGRES_USER=bbs -e POSTGRES_PASSWORD=bbs -e POSTGRES_DB=bbs postgres:16
# Expected output: container ID
for attempt in $(seq 1 30); do
  podman exec bbs-ws5-postgres pg_isready -U bbs -d bbs && break
  sleep 1
done
podman exec bbs-ws5-postgres pg_isready -U bbs -d bbs
# Expected output: accepting connections (do not continue unless ready)
podman exec -i bbs-ws5-postgres psql -v ON_ERROR_STOP=1 -U bbs -d bbs < internal/adapters/infra/persistence/postgres/schema.sql
# Expected output: CREATE TABLE and INSERT; no errors
podman exec bbs-ws5-postgres psql -U bbs -d bbs -c 'SELECT name FROM boards ORDER BY name;'
# Expected output: 3 rows: chat / news / programming
```

OpenDB also applies this embedded DDL transactionally; restarts do not duplicate boards. The non-TLS connection string is for this local exercise database.

### STEP 4: Replace the Composition Root

**Complete replacement of an existing file:** read the [PostgreSQL main.go](assets/bbs/cmd/bbs-postgres/main.go), checking repository and transaction-manager wiring. UseCase and HTTP-handler assembly remains the same.

```bash
# Terminal B — /tmp/bbs-ws5
cp "$WORKSHOP_REPO/software/assets/bbs/cmd/bbs-postgres/main.go" cmd/bbs/main.go
go build -o bbs ./cmd/bbs
# Expected output: no output; bbs binary created
BBS_TEST_DATABASE_URL='postgres://bbs:bbs@127.0.0.1:55432/bbs?sslmode=disable' go test -race ./internal/adapters/infra/persistence/postgres -v
# Expected output: connection, persistence, error mapping and rollback tests PASS
```

```bash
# Terminal A — /tmp/bbs-ws5
DATABASE_URL='postgres://bbs:bbs@127.0.0.1:55432/bbs?sslmode=disable' ./bbs
# Expected output: server starts on 127.0.0.1:8080
```

### STEP 5: Verify the Same API and Policy

```bash
# Terminal B — /tmp/bbs-ws5
curl -i http://localhost:8080/api/boards
# Expected output: 200 OK; same boards JSON shape as SQLite
THREAD_ID=$(curl -fsS -X POST http://localhost:8080/api/boards/programming/threads -H 'Content-Type: application/json' -d '{"title":"Postgres","author":"owner","body":"first","owner_only":true}' | python3 -c 'import sys,json; print(json.load(sys.stdin)["thread"]["id"])')
# Expected output: new thread ID
curl -i -X POST "http://localhost:8080/api/threads/$THREAD_ID/posts" -H 'Content-Type: application/json' -d '{"author":"other","body":"denied"}'
# Expected output: 403 Forbidden
curl -i -X POST "http://localhost:8080/api/threads/$THREAD_ID/posts" -H 'Content-Type: application/json' -d '{"author":"owner","body":"allowed"}'
# Expected output: 201 Created, number: 2
diff -ru .baseline/internal/domain internal/domain
diff -ru .baseline/internal/usecase internal/usecase
diff -ru .baseline/internal/adapters/presentation internal/adapters/presentation
# Expected output: no differences from all three commands
```

### ✅ Checkpoint

- [ ] Complete PostgreSQL adapter satisfies all ports; connection tests pass
- [ ] Verified listing, creation, denial and permission via the same HTTP API
- [ ] Domain, UseCase and Presentation unchanged; compared with initial prediction

## What You Learned

- Keep database-specific SQL and drivers inside Infra.
- Align transactions across repositories to preserve posting policy.
- Schema, seed data and connection checking are necessary parts of replacing persistence.

Continue with a dedicated [WS6](clean_arch_ws6_en.md) copy.

## Troubleshooting

| Symptom                 | Fix                                                                      |
| ----------------------- | ------------------------------------------------------------------------ |
| connection refused      | Check container pg_isready and published port 55432                      |
| unknown driver pgx      | Check stdlib registration in db.go and go.mod; copy the complete adapter |
| relation does not exist | Check ON_ERROR_STOP output and schema.sql application                    |
| Expected boards missing | Check seed data in schema.sql                                            |

## Cleanup

Stop terminal A with Ctrl+C first:

```bash
# Terminal B
podman rm -f bbs-ws5-postgres
cd /tmp
rm -r /tmp/bbs-ws5
# Expected output: dedicated DB container, copy and SQLite DB removed
```

## References

- [pgx database/sql adapter](https://pkg.go.dev/github.com/jackc/pgx/v5/stdlib)
- [PostgreSQL CREATE TABLE](https://www.postgresql.org/docs/16/sql-createtable.html)
- [Clean Architecture](clean_arch_en.md)
