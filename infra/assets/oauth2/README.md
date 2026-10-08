# OAuth2 Client and Resource Server

A minimal OAuth2 Authorization Code + PKCE pair: a Go web client that walks the browser flow and a Go resource server that validates JWT access tokens. Part of the [OAuth2 Workshop](../../oauth2_en.md).

## What this is

Two independent Go modules. `client/` is a web app on `:3000` that starts the authorization flow against Keycloak, handles the `/callback`, stores the token in a session cookie, and calls the API with it. `api/` is a resource server on `:8081` that fetches the issuer's JWKS and verifies the JWT signature, issuer, and audience before serving `/api/profile`.

## Requirements

- Go 1.22+ (each module has its own `go.mod`)
- A Keycloak authorization server reachable at `http://localhost:8080` with realm `workshop`, client `workshop-client`, and a test user — the [workshop](../../oauth2_en.md) covers the full Keycloak setup (realm import, client, PKCE settings)
- Podman or Docker to run Keycloak
- A browser for the interactive login

## How to run

After Keycloak is configured as described in the workshop:

```bash
# Terminal 1: resource server
cd api
go run main.go
# Output: "resource server started on :8081"

# Terminal 2: client app
cd client
go run main.go
# Output: "oauth2 client app started on :3000"
```

Open <http://localhost:3000>, click through the login, and the client displays the token and the response from `http://localhost:8081/api/profile`.

Both modules read environment variables with the defaults shown above (`OIDC_ISSUER`, `OIDC_AUDIENCE`, `OAUTH_CLIENT_ID`, `RESOURCE_API_URL`, etc.) — see the top of each `main.go`.

## Layout

```text
.
├── api/          # Resource server module: JWKS validation + /api/profile (Go 1.22)
│   └── main.go
├── client/       # Client web app module: auth-code + PKCE flow, :3000 (Go 1.22)
│   └── main.go
└── compose.yaml  # Keycloak 26.1 (start-dev, admin/admin, :8080) for the workshop
```
