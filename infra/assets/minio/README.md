# MinIO Object Storage Client

A Go CLI that uploads files to MinIO (S3-compatible storage) and issues presigned share links, structured with Clean Architecture. Part of the [MinIO Workshop](../../minio_en.md).

## What this is

`main.go` takes a subcommand (`upload` or `share`) and a filename, then talks to a local MinIO server through the AWS SDK for Go v2 (S3 client with path-style addressing). S3 interaction lives in `internal/infra/minio`, while `internal/usecase` holds the file workflows, so the storage backend could be swapped without touching business logic.

## Requirements

- Go 1.25.5 (`go.mod`)
- Podman or Docker to run MinIO via the bundled compose file
- Defaults: endpoint `http://localhost:9000`, bucket `workshop-images`, credentials `minioadmin` / `minioadmin`

## How to run

> **⚠️ Image note (2026)**: MinIO stopped publishing community images (Oct 2025) and later deleted its Docker Hub/quay.io repositories, so this asset uses the frozen Bitnami mirror `docker.io/bitnamilegacy/minio` — fine for local study, not for production. The last community build has no embedded web Console; everything is verified via the S3 API on `:9000`.

Start MinIO first (S3 API on :9000):

```bash
podman compose up -d   # or: docker compose up -d
```

Then use the CLI:

```bash
go mod tidy
go run main.go upload somefile.jpg
# Output: "Successfully uploaded somefile.jpg to bucket workshop-images"
# (the bucket is created automatically on first run)

go run main.go share somefile.jpg
# Output: a presigned URL valid for 15 minutes
```

The [workshop](../../minio_en.md) walks through the S3 API details with the `aws` CLI.

## Layout

```text
.
├── main.go                 # CLI entry point: upload / share subcommands
├── docker-compose.yml      # MinIO server (S3 API on 9000, minioadmin creds)
└── internal/
    ├── usecase/            # File upload / share-link use cases
    └── infra/minio/        # S3 adapter (AWS SDK v2, path-style)
```
