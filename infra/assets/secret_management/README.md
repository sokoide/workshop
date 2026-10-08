# HashiCorp Vault Secret Management

A set of small CLIs for storing, retrieving, and listing secrets in HashiCorp Vault's KV v2 engine, plus an API client that demonstrates consuming a secret. Part of the [Secret Management Workshop](../../secret_management_en.md).

## What this is

Four commands under `cmd/` share a Vault adapter (`internal/infra/vault`) and secret use cases (`internal/usecase`). The adapter reads `VAULT_ADDR` (default `http://localhost:8200`) and `VAULT_TOKEN` (default `dev-workshop-token`, matching the dev server started by the Makefile). `api-client` runs a short demo that fetches a stored key and "uses" it against a fake endpoint.

## Requirements

- Go 1.25.5 (`go.mod`)
- Podman for Vault
- Makefile targets: `vault-up` (dev server on :8200 with root token `dev-workshop-token`), `init` (enable KV v2), `vault-logs`, `vault-status`, `vault-down`

## How to run

```bash
make vault-up
make init
go mod tidy

# Store and retrieve
go run cmd/put-secret/main.go api/external-key "sk-live-abc123xyz789"
# Output: "=== Secret Stored Successfully ===" with key and value
go run cmd/get-secret/main.go api/external-key
# Output: "=== Secret Retrieved ===" showing the value (Version: 1)

# List stored keys
go run cmd/list-secrets/main.go

# Demo: an API client consuming the secret
go run cmd/api-client/main.go
# Output: "[INFO] Demo completed successfully"
```

The [workshop](../../secret_management_en.md) covers Vault concepts (dynamic secrets, leasing, and why KV v2) and the full walkthrough.

## Layout

```text
.
├── cmd/
│   ├── put-secret/     # Store a key/value pair
│   ├── get-secret/     # Retrieve a value (with version)
│   ├── list-secrets/   # List stored keys
│   └── api-client/     # Demo client that consumes a secret
├── internal/
│   ├── domain/         # Secret domain types
│   ├── usecase/        # Secret use cases
│   └── infra/vault/    # Vault API adapter (env-based config)
└── Makefile            # vault-up / init / vault-down / logs / status
```
