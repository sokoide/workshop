# Secret Management Workshop: Secure API Key Management with HashiCorp Vault

In this workshop, you will build a secure secret management system using **HashiCorp Vault** and **Go**. You will learn best practices for managing API keys, database credentials, and other sensitive data without hardcoding them in your source code.

## Goal

By completing this workshop, you will be able to:

- Understand why secret management is critical for security
- Identify and avoid common anti-patterns (hardcoded credentials, .env files in git)
- Learn HashiCorp Vault fundamentals (KV v2 secrets engine, tokens, leases)
- Integrate Vault into applications using Clean Architecture
- Implement secret versioning and rotation

---

## Anti-Patterns to Avoid

Before learning the right way, let's look at common mistakes:

### ❌ Anti-Pattern 1: Hardcoded Credentials

This is the **worst** practice. Credentials are permanently visible in git history:

```go
// From infra/assets/rabbitmq_crypto/cmd/ticker/main.go
url := "amqp://guest:guest@localhost:5672/"  // NEVER DO THIS
```

**Problems:**

- Git history contains the secret forever (even if you "delete" it later)
- Anyone with repository access can see credentials
- Secrets are exposed in code reviews and pull requests
- No way to rotate credentials without changing code

### ❌ Anti-Pattern 2: Environment Variable Files (.env)

While better than hardcoding, `.env` files are still risky:

```bash
# .env file (often accidentally committed to git)
DATABASE_URL=postgresql://user:password123@localhost/db
API_KEY=sk-live-abc123xyz789
```

**Problems:**

- Easy to accidentally commit `.env` to git
- No audit trail of who accessed secrets
- No automatic rotation
- Secrets are plaintext on disk

### ✅ The Right Way: External Secret Manager

Vault provides:

- Centralized secret storage
- Audit logging
- Automatic secret rotation
- Fine-grained access control
- Encryption at rest

---

## Architecture

This workshop implements **Clean Architecture** with proper separation of concerns:

```mermaid
graph TB
    subgraph Framework ["Framework Layer (cmd/)"]
        GS[get-secret]
        PS[put-secret]
        LS[list-secrets]
        AC[api-client]
    end

    subgraph Usecase ["Usecase Layer (usecase/)"]
        SM[SecretManager]
        API[APIClient]
    end

    subgraph Domain ["Domain Layer (domain/)"]
        Entity[Secret Entity]
        Repo[SecretRepository Interface]
    end

    subgraph Infra ["Infra Layer (infra/vault/)"]
        VA[Vault Adapter]
        SDK[Vault Go SDK]
    end

    subgraph External ["External"]
        V[Vault Server KV v2]
    end

    GS --> SM
    PS --> SM
    LS --> SM
    AC --> API
    SM --> Repo
    API --> SM
    VA -.->|implements| Repo
    VA --> SDK
    SDK --> V

    style Repo fill:#e1f5fe
    style VA fill:#fff3e0
    style V fill:#f3e5f5
```

### Directory Structure

```text
infra/assets/secret_management/
├── cmd/                        # Framework Layer (Entry Points)
│   ├── get-secret/main.go      # Retrieve a secret
│   ├── put-secret/main.go      # Store a secret
│   ├── list-secrets/main.go    # List all secrets
│   └── api-client/main.go      # API client with Vault auth
├── domain/                     # Domain Layer (Pure Go)
│   ├── secret.go               # Secret entity
│   └── repository.go           # SecretRepository interface
├── usecase/                    # Usecase Layer (Business Logic)
│   ├── secret_manager.go       # Secret orchestration
│   └── api_client.go           # API client pattern
├── infra/                      # Infra Layer (Vault Adapter)
│   └── vault/
│       ├── client.go           # Vault client setup
│       ├── secret.go           # KV v2 implementation
│       └── secret_test.go      # Integration tests
├── Makefile                    # Container management
└── go.mod
```

### Key Principles

- **Domain Layer**: Pure Go with no external dependencies. Defines `SecretRepository` interface.
- **Usecase Layer**: Business logic that depends only on domain interfaces.
- **Infra Layer**: Vault adapter implements `SecretRepository` using HashiCorp Vault SDK.
- **Framework Layer**: CLI commands that wire dependencies together.

---

## Preparation

### 1. Start Vault

Start a Vault Dev container. Dev mode is **insecure** but perfect for learning:

```bash
cd infra/assets/secret_management
make vault-up
```

This starts Vault with:

- Address: `http://localhost:8200`
- Root token: `dev-workshop-token`
- Unsealed automatically (dev mode only)

### 2. Enable KV v2 Secrets Engine

The KV v2 secrets engine must be enabled before storing secrets.
Even if you don't have the `vault` command locally, you can use `make init` to configure it inside the container:

```bash
# Enable KV v2 (executes command inside container)
make init
```

Alternatively, you can manually execute the command inside the container:

```bash
podman exec workshop-vault vault secrets enable -path=secret kv-v2
```

**Note:** KV v2 provides versioning for all secrets, allowing you to track changes and roll back if needed.

### 3. Install Dependencies

```bash
go mod tidy
```

---

## Workshop Steps

### STEP 1: Explore the Anti-Pattern (5 minutes)

Before using Vault, see how secrets are commonly mishandled:

```bash
cd ../rabbitmq_crypto
grep -r "guest:guest" cmd/
# Result: url := "amqp://guest:guest@localhost:5672/"
```

**Key Takeaway:** Once a secret is committed to git, it's there forever (even in history).

### STEP 2: Basic Secret Operations (15 minutes)

Navigate back to the secret management project:

```bash
cd ../secret_management
```

#### Step 1: Store Your First Secret

Use the `put-secret` command:

```bash
go run cmd/put-secret/main.go api/external-key "sk-live-abc123xyz789"
```

Expected output:

```text
[DEBUG] Connecting to Vault at http://localhost:8200
[INFO] Storing secret: api/external-key
[INFO] Successfully stored secret: api/external-key

=== Secret Stored Successfully ===
Key:   api/external-key
Value: sk-live-abc123xyz789

You can retrieve it with:
  go run cmd/get-secret/main.go api/external-key
```

#### Step 2: Retrieve the Secret

```bash
go run cmd/get-secret/main.go api/external-key
```

Expected output:

```text
[DEBUG] Connecting to Vault at http://localhost:8200
[INFO] Retrieving secret: api/external-key
[INFO] Successfully retrieved secret: api/external-key (version 1)

=== Secret Retrieved ===
Key:     api/external-key
Value:   sk-live-abc123xyz789
Version: 1
Created: 2025-01-28T10:30:00Z
```

#### Step 3: List All Secrets

```bash
go run cmd/list-secrets/main.go
```

### STEP 3: Practical Example - API Client (10 minutes)

Real-world applications need to retrieve secrets at runtime and use them for API calls. This example demonstrates the pattern:

```bash
# First, store an API key
go run cmd/put-secret/main.go api/external-key "sk-live-abc123xyz789"

# Then run the API client
go run cmd/api-client/main.go
```

The `api-client` demonstrates:

1. Retrieving the API key from Vault at startup
2. Using the key for an API call (simulated)
3. No hardcoded credentials in the code

**Code Pattern (from `usecase/api_client.go`):**

```go
func (ac *apiClient) CallAPI(ctx context.Context, secretKey, apiEndpoint string) error {
    // Step 1: Retrieve API key from Vault
    secret, err := ac.secretManager.RetrieveSecret(ctx, secretKey)
    if err != nil {
        return fmt.Errorf("failed to retrieve API key: %w", err)
    }

    apiKey := secret.Value

    // Step 2: Make API call with the key
    statusCode, _, err := ac.httpClient.Get(apiEndpoint, apiKey)
    // ...
}
```

### STEP 4: Secret Versioning and Rotation (10 minutes)

Vault KV v2 automatically versions secrets. Let's see this in action:

#### Step 1: Update a Secret

```bash
# Store version 1
go run cmd/put-secret/main.go api/payment-gateway "sk-pay-v1-abc"

# Update to version 2
go run cmd/put-secret/main.go api/payment-gateway "sk-pay-v2-xyz"

# Update to version 3
go run cmd/put-secret/main.go api/payment-gateway "sk-pay-v3-new"
```

#### Step 2: Retrieve Current Version

```bash
go run cmd/get-secret/main.go api/payment-gateway
# Shows version 3 (latest)
```

#### Step 3: View History with Vault CLI (using container)

```bash
# List versions
podman exec workshop-vault vault kv metadata get secret/api/payment-gateway

# Retrieve specific version (version 1)
podman exec workshop-vault vault kv get -version=1 secret/api/payment-gateway
```

**Use Cases for Versioning:**

- **Rollback**: If a new secret is incorrect, revert to a previous version
- **Audit**: Track who changed what and when
- **Rotation**: Deploy new secrets while keeping old ones accessible temporarily

### STEP 5: Understanding the Architecture (5 minutes)

Review the code structure:

```bash
# Domain Layer (pure Go, no external dependencies)
cat domain/secret.go
cat domain/repository.go

# Usecase Layer (business logic)
cat usecase/secret_manager.go

# Infra Layer (Vault adapter)
cat infra/vault/secret.go
```

**Key Observation:** The `usecase` layer has no idea it's talking to Vault. It only knows about the `SecretRepository` interface defined in the `domain` layer. This means you could swap Vault for AWS Secrets Manager or Azure Key Vault without changing any business logic.

### STEP 6: Verification via Tests

Run unit and integration tests to verify the system behavior:

```bash
# Unit tests (use mocks, no Vault required)
go test ./usecase/... -v

# Integration tests (require Vault container)
go test ./infra/vault/... -v
```

---

## Clean Architecture Highlights

### Domain Layer (`domain/repository.go`)

Pure Go interface with zero external dependencies:

```go
type SecretRepository interface {
    GetSecret(ctx context.Context, key string) (*Secret, error)
    PutSecret(ctx context.Context, key, value string) error
    ListSecrets(ctx context.Context) ([]SecretMetadata, error)
}
```

### Infra Layer (`infra/vault/secret.go`)

Vault-specific implementation:

```go
type VaultSecretRepository struct {
    client *api.Client
    path   string // KV v2 mount path
}

func (r *VaultSecretRepository) GetSecret(ctx context.Context, key string) (*domain.Secret, error) {
    secret, err := r.client.KVv2(r.path).Get(ctx, key)
    // Map Vault response to domain.Secret
}
```

### Usecase Layer (`usecase/secret_manager.go`)

Business logic that doesn't know about Vault:

```go
type secretManager struct {
    repo domain.SecretRepository
}

func (sm *secretManager) RetrieveSecret(ctx context.Context, key string) (*domain.Secret, error) {
    return sm.repo.GetSecret(ctx, key)
}
```

---

## Environment Variables

The application uses these environment variables with sensible defaults:

| Variable      | Default                   | Description                |
|---------------|---------------------------|----------------------------|
| `VAULT_ADDR`  | `http://localhost:8200`   | Vault server address       |
| `VAULT_TOKEN` | `dev-workshop-token`      | Vault authentication token |

You can override these at runtime:

```bash
VAULT_ADDR=http://vault.example.com:8200 \
VAULT_TOKEN=your-token-here \
go run cmd/get-secret/main.go api/key
```

---

## Cleanup

When you are finished with the workshop:

```bash
make vault-down
```

This stops and removes the Vault container. Note: Data in dev mode is ephemeral and will be lost when the container stops.

---

## Next Steps

After completing this workshop, consider:

1. **Production Deployment**: Learn about Vault production hardening (TLS, Raft storage, auto-unseal)
2. **Dynamic Secrets**: Explore database credentials that auto-rotate
3. **Transit Secrets Engine**: Encrypt/decrypt data without storing it
4. **Identity**: Use Vault's authentication methods (GitHub, AWS, Kubernetes)
5. **Audit Logging**: Enable detailed audit trails for compliance

---

## References

- [HashiCorp Vault Documentation](https://developer.hashicorp.com/vault/docs)
- [Vault Go API Client](https://github.com/hashicorp/vault/api)
- [Clean Architecture by Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
