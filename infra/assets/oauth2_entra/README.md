# OAuth2 with Microsoft Entra ID

The OAuth2 pair from the [OAuth2 Workshop](../../oauth2_entra_en.md) adapted to Microsoft Entra ID (Azure AD): an API that validates Entra-issued JWTs (scopes and app roles) and a client that authenticates via `azidentity`.

> **⚠️ Azure costs**: this asset talks to a real Microsoft Entra tenant. Running it requires an Azure subscription with Entra ID, and app registrations in the Azure portal. Depending on your tenant, the activities involved may incur Azure costs — the [workshop](../../oauth2_entra_en.md) lists exactly what must be created (app registrations, redirect URIs, app roles, admin consent).

## What this is

Two independent Go modules, each with a Dockerfile and a Makefile for building/pushing container images. `api/` validates access tokens against the tenant's JWKS endpoint and enforces a required scope and app role. `client/` uses `DefaultAzureCredential` to obtain tokens (supporting managed identity, environment variables, or Azure CLI sign-in) and calls the API.

## Requirements

- Go 1.26.1+ (each module has its own `go.mod`)
- An Azure subscription and Entra tenant; `TENANT_ID` and `API_CLIENT_ID` must be set for the API
- Podman (for the image-based workflow) with `REGISTRY_USERNAME` set for the Makefile targets (`build`, `run`, `push`, `clean`, `login`)
- Azure CLI (`az login`) or another `DefaultAzureCredential` source for the client

## How to run

Locally after completing the Azure setup in the [workshop](../../oauth2_entra_en.md):

```bash
# Terminal 1: API
cd api
TENANT_ID=<tenant-id> API_CLIENT_ID=<api-app-id> go run main.go

# Terminal 2: client (obtains a token, then calls the API)
cd client
API_SCOPE=<scope> API_ENDPOINT=http://localhost:8081 go run main.go
```

Container workflow per module:

```bash
make build   # podman build --platform linux/amd64
make run     # runs the image with the required env vars
```

## Layout

```text
.
├── api/       # Resource server module: Entra JWT validation (scope + app role)
│   ├── main.go
│   ├── Dockerfile
│   └── Makefile
└── client/    # Client module: azidentity DefaultAzureCredential
    ├── main.go
    ├── Dockerfile
    └── Makefile
```
