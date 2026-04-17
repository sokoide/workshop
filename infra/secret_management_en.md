# Secret Management Workshop: Secure API Key Management with HashiCorp Vault

In this workshop, you will build a secure secret management system using **HashiCorp Vault** and **Go**, learning how to manage sensitive data without hardcoding it.

> **💡 Glossary**: Please refer to [Secret Management](glossary.md#security), [Vault](glossary.md#security), or [KV Engine](glossary.md#security) in the [Glossary](glossary.md) for technical terms used in this workshop.

## Goal

Build and experience a secure secret management flow, acquiring the following skills:

- Understand why secret management is critical for security.
- Identify and avoid common anti-patterns (hardcoded credentials, .env files).
- Master HashiCorp Vault fundamentals (KV v2, tokens).
- Integrate Vault into applications using Clean Architecture.
- Implement secret versioning and rotation.

---

## Challenges in Secret Management (Anti-Patterns)

Before learning the right way, let's look at common mistakes and their risks.

### ❌ Anti-Pattern 1: Hardcoded Credentials

Writing credentials directly into the Git history.

- **Problem**: They remain in the history forever and are exposed to everyone with repository access.

### ❌ Anti-Pattern 2: Environment Variable Files (.env)

- **Problem**: Easy to accidentally commit to Git, and lacks an audit log of who accessed them.

### ✅ External Secret Manager Solutions

Using Vault provides centralized management, audit logs, encryption at rest, and automatic rotation.

---

## Architecture

We implement **Clean Architecture** with proper separation of concerns.

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
```text

### Directory Structure

```text
infra/assets/secret_management/
├── cmd/                        # Entry points for operations
├── domain/                     # Pure Go definitions
├── usecase/                    # Orchestration of secret management
├── infra/                      # Vault adapter implementation
├── Makefile                    # Container management commands
└── go.mod
```text

---

## Preparation

### 1. Start Vault

```bash
cd infra/assets/secret_management
make vault-up
```text

### 2. Enable KV v2 Secrets Engine

```bash
make init
```text

### ✅ Verification Checkpoints

- [ ] Confirmed `workshop-vault` is running via `podman ps`.
- [ ] Confirmed the KV v2 engine is mounted at `secret/` after `make init`.
- [ ] Confirmed the `VAULT_TOKEN` environment variable is set.

---

## Workshop Steps

### STEP 1: Identify Anti-Patterns

Look for hardcoded passwords in other workshop assets. Realize how terrifying it is that once committed, they stay in the history even if deleted later.

### STEP 2: Basic Secret Operations (Put/Get)

```bash
# Store
go run cmd/put-secret/main.go api/external-key "sk-live-abc123xyz789"
# Retrieve
go run cmd/get-secret/main.go api/external-key
```text

### STEP 3: Practical Example - API Client

Verify the pattern of retrieving a key from Vault at startup and using it for a (simulated) API call.

```bash
go run cmd/api-client/main.go
```text

### STEP 4: Versioning and Rotation

In Vault KV v2, writing a new value to the same key automatically increments the version.

```bash
# Update
go run cmd/put-secret/main.go api/key "new-value"
# Check history (using container command)
podman exec workshop-vault vault kv metadata get secret/api/key
```text

Examine the `internal/usecase` layer code and confirm it has zero dependencies on Vault (it only depends on the interface defined in `internal/domain`).

### ✅ Verification Checkpoints

- [ ] Confirmed `go run cmd/put-secret` completed without error.
- [ ] Confirmed `go run cmd/get-secret` retrieves the stored value.
- [ ] Confirmed the secret version is incremented via the `metadata get` command.

---

## Clean Architecture Highlights

The business logic doesn't care **"whether the secret is in Vault or AWS."** It retrieves secrets through abstract operations defined in `internal/domain/repository.go`, while infrastructure details are implemented in `internal/infra`. This makes the system extremely resilient to infrastructure changes.

---

## Cleanup

```bash
make vault-down
```text

---

## Next Steps

- **Dynamic Secrets**: Learn how to issue temporary database access on-demand.
- **Transit Secrets Engine**: Learn how to encrypt data without storing the decryption key in the application.

---

## 🔧 Troubleshooting

### Authentication Error (403 Forbidden)

**Symptoms**: `Error making API request: 403 Forbidden`

**Causes and Solutions**:

- **Invalid Token**: Re-verify your `VAULT_TOKEN`. Check if it has expired.
- **Policy Restrictions**: Ensure your token has permissions for the specified path (e.g., `secret/api/*`).

### Connection Refused

**Symptoms**: `dial tcp 127.0.0.1:8200: connect: connection refused`

**Causes and Solutions**:

- **Vault Not Running**: Ensure `make vault-up` finished without errors.
- **Incorrect Address**: Check that `VAULT_ADDR` is set to `http://127.0.0.1:8200`.

---

## 💻 Environment Notes

### For macOS Users

- Use `127.0.0.1` explicitly if `localhost` fails to connect.

### For Windows Users

- Ensure port-forwarding is active to access the Vault UI from your Windows browser.
- Manage `VAULT_ADDR` in your WSL configuration (e.g., `.bashrc`) rather than Windows environment variables.
