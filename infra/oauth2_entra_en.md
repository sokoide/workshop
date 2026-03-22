# OAuth2 Workshop: Learning Authorization Flows with Microsoft Entra ID and Go

In this workshop, you will use **Microsoft Entra ID** as the authorization server and a **Go REST API** as the resource server. The goal is to understand how OAuth 2.0 and OpenID Connect work in realistic Azure environments.

This guide covers two common patterns:

- **A. Machine-to-Machine (M2M)**: an Azure workload calls your API by using a **managed identity**
- **B. User-delegated access**: a client such as React or Postman signs in a **user** and calls your API on that user's behalf

> **Note**:
> Most app registration and API exposure settings are managed in the **Microsoft Entra admin center** (`entra.microsoft.com`).
> Managed identity settings belong to Azure resources, so you usually configure them in the **Azure portal** (`portal.azure.com`).

---

## Which one should you choose?

```text
Is a user involved?
    │
    ├─ YES → B. User-delegated access
    │         (Operation as a user via SPA, Mobile app, or Postman)
    │
    └─ NO  → Is it running on Azure?
              │
              ├─ YES → A. M2M (Managed Identity recommended)
              │         (App Service, Functions, VM batch jobs, etc.)
              │
              └─ NO  → M2M (Client Credentials + Secret/Certificate)
                        (On-premises, connections from other clouds)
```

---

## Values to Keep in Advance

You will need the following values during configuration. It is helpful to note them down.

| Value | Where to find | Where it's used |
| --- | --------- | --------- |
| **Tenant ID** | Entra Admin Center → Overview | Go code (`tenantID`) |
| **API App Client ID** | App registrations → workshop-api | Go code (`apiClientID`), Token requests |
| **Client App Client ID** | App registrations → workshop-client | Postman / SPA configuration |
| **Managed Identity Object ID** | Azure Portal → Resource → Identity | App Role assignment |

---

## First, Separate the Two Patterns

| Item | A. M2M | B. User-delegated access |
| ---- | ------ | ------------------------ |
| Actor | Application / workload | User |
| Typical examples | App Service, Functions, VM, scheduled job | SPA, mobile app, desktop app, Postman |
| OAuth 2.0 flow | Client Credentials flow | Authorization Code flow + PKCE |
| Permission model on the API side | **App Role** | **Scope** |
| Typical claim | `roles` | `scp` |
| Managed identity | Common and recommended | Normally not used |

---

## Important: Difference between v1 and v2 tokens

Entra ID issues two types of access token formats. This workshop uses **v2 tokens**.

| Item | v1 Token | v2 Token |
| ------ | ------------ | ------------ |
| `aud` (audience) | `api://<client-id>` (URI format) | `<client-id>` (GUID format) |
| `iss` (issuer) | `https://sts.windows.net/<tenant-id>/` | `https://login.microsoftonline.com/<tenant-id>/v2.0` |
| Validation complexity | Requires URI matching | Simple GUID matching |

**Why we recommend v2**: It simplifies the validation logic and reduces implementation errors.

> **Configuration**: App registrations → Target App → Manifest → `"requestedAccessTokenVersion": 2`

---

## Debug: Checking Token Contents

To verify your configuration, decode the acquired token and inspect its contents.

### Method 1: jwt.ms (Recommended)

1. Copy the token to your clipboard.
2. Visit [https://jwt.ms](https://jwt.ms).
3. Paste the token to see the decoded Claims.

### Method 2: jwt.io (Local check)

```bash
# Decode token (without signature validation)
echo "<TOKEN>" | cut -d'.' -f2 | base64 -d 2>/dev/null | jq .
```

### Claims to Verify

| Claim | Expected Value | Purpose |
| ------- | -------- | ---------- |
| `aud` | API Client ID | Is it intended for your API? |
| `iss` | `https://login.microsoftonline.com/<tenant-id>/v2.0` | Is it from the correct tenant? |
| `scp` | `access_as_user` | For user-delegated flows |
| `roles` | `["Svc.Invoke"]` | For M2M flows |
| `appid` | Client App ID | Which app requested the token? |

---

## Actors and Roles

| Actor | Example | Role |
| ---- | ---- | ---- |
| Resource Owner | User | Signs in and grants consent |
| Client | React, Postman, Azure workload | Obtains an access token and calls the API |
| Authorization Server | Microsoft Entra ID | Authenticates and issues access tokens |
| Resource Server | Go REST API | Validates tokens and returns protected data |

---

## A. Machine-to-Machine (M2M)

This is service-to-service communication with no interactive user. In Azure, the standard pattern is **Managed Identity + App Role**.

### Architecture

```mermaid
sequenceDiagram
    participant Workload as Azure Workload<br/>(App Service, etc.)
    participant MI as Managed Identity Endpoint<br/>(IMDS / local endpoint)
    participant Entra as Microsoft Entra ID
    participant API as Go Resource Server

    Note over Workload, Entra: 1. The workload gets a token by using its managed identity
    Workload->>MI: Request token via SDK or endpoint
    MI->>Entra: Backend token acquisition
    Entra-->>Workload: Access token (roles: ["Svc.Invoke"])

    Note over Workload, API: 2. The workload calls the API
    Workload->>API: Authorization: Bearer <JWT>
    API->>API: Validate iss / aud / roles / signature
    API-->>Workload: 200 OK
```

### Setup Steps

#### 1. Register the API app

- Open the **Microsoft Entra admin center** → `App registrations` → `New registration`
- Name: `workshop-api`
- After creation, record the **Application (client) ID**
- Go to `Expose an API` and select `Set` to configure the **Application ID URI**
  - Example: `api://<API_APP_CLIENT_ID>`
  - > **Important**: This URI is used as the "resource identifier" when requesting a token. Note that if you enable v2 tokens as described below, the actual `aud` (audience) claim in the token will be the **Client ID (GUID)**, not this URI.

#### 2. Expose an app role for application access

- `App registrations` → `workshop-api` → `App roles` → `Create app role`
- Display name: `Svc.Invoke`
- Allowed member types: `Applications`
- Value: `Svc.Invoke`
- Add a description if needed, then save

#### 3. Configure the API to accept v2 access tokens

- Open `Manifest`
- Set `requestedAccessTokenVersion` to `2` inside the `api` section

```json
"api": {
  "requestedAccessTokenVersion": 2
}
```

> **Note**: Previously this was named `accessTokenAcceptedVersion`, but now `requestedAccessTokenVersion` is used.

#### 4. Enable managed identity on the Azure resource

- Open the **Azure portal**
- Go to the target resource, such as App Service
- Open `Identity`
- Enable either `System assigned` or `User assigned`
- Record the managed identity's **principal object ID**

#### 5. Assign the app role to the managed identity

Assigning a custom API app role to a managed identity is usually easier through **Microsoft Graph / Azure CLI** than through the portal UI.

```bash
# ============================================
# Preparation: Obtain the IDs
# ============================================

# The Application (client) ID of your API app
API_APP_CLIENT_ID=<API_APP_CLIENT_ID>

# The Object ID of your Managed Identity
# Find this in Azure Portal → Resource → Identity → Object ID
# Note: This is the Service Principal Object ID
MI_SP_OBJECT_ID=<MANAGED_IDENTITY_OBJECT_ID>

# ============================================
# Assign the App Role
# ============================================

# Get the service principal object ID of the API app
API_SP_OBJECT_ID=$(az ad sp show --id ${API_APP_CLIENT_ID} --query id -o tsv)

# Get the ID of the app role published by the API app
APP_ROLE_ID=$(az ad sp show --id ${API_APP_CLIENT_ID} \
  --query "appRoles[?value=='Svc.Invoke'].id | [0]" -o tsv)

az rest --method POST \
  --uri "https://graph.microsoft.com/v1.0/servicePrincipals/${MI_SP_OBJECT_ID}/appRoleAssignments" \
  --body "{
    \"principalId\": \"${MI_SP_OBJECT_ID}\",
    \"resourceId\": \"${API_SP_OBJECT_ID}\",
    \"appRoleId\": \"${APP_ROLE_ID}\"
  }"
```

---

## B. User-Delegated Access

This is the pattern for SPAs, mobile apps, desktop apps, or Postman when a signed-in user calls your API. The recommended flow is **Authorization Code + PKCE**.

### Architecture

```mermaid
sequenceDiagram
    participant User as User
    participant Client as Client App<br/>(React / Postman)
    participant Entra as Microsoft Entra ID
    participant API as Go Resource Server

    Note over User, Entra: 1. Sign-in and consent
    User->>Entra: Sign in
    Entra-->>Client: Authorization code

    Note over Client, Entra: 2. Exchange the code with PKCE
    Client->>Entra: code + code_verifier
    Entra-->>Client: Access token (scp: "access_as_user")

    Note over Client, API: 3. Call the API
    Client->>API: Authorization: Bearer <JWT>
    API->>API: Validate iss / aud / scp / signature
    API-->>Client: 200 OK
```

### Setup Steps

#### 1. Expose a scope on the API app

- Open the **Microsoft Entra admin center** → `App registrations` → `workshop-api`
- Go to `Expose an API` → `Add a scope`
- Scope name: `access_as_user`
- Who can consent: `Admins and users`
- Fill in the admin consent and user consent display text, then save

#### 2. Register the client app

- `App registrations` → `New registration` → Name: `workshop-client`
- Open `Authentication` → `Add a platform`
  - For React or another browser SPA: **`Single-page application`**
  - For Postman or native clients: `Mobile and desktop applications`
- Register the redirect URI (e.g., `http://localhost:3000` or `https://oauth.pstmn.io/v1/callback`)

> **Important**: For SPAs, the redirect URI must be registered as a **Single-page application** platform. Otherwise, the authorization code exchange can fail with a CORS error.

#### 3. Add API permissions to the client app

- Open `workshop-client` → `API permissions` → `Add a permission`
- Select `My APIs` → `workshop-api`
- Choose `Delegated permissions`
- Add `access_as_user`

#### 4. Handle consent

| Type | Required for | Performer |
| ---------- | ------------ | ------ |
| **User consent** | User consent allowed tenant + Low impact permissions | The user signing in |
| **Admin consent** | User consent disabled / High impact / Application permissions | Tenant Admin |

Operation:

- **User consent**: A consent screen is shown during the first sign-in.
- **Admin consent**: Go to `API permissions` → `Grant admin consent for [Tenant Name]`.

---

## Go Resource Server Example

The following example uses `github.com/MicahParks/keyfunc/v3` and `github.com/golang-jwt/jwt/v5`.

```go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

const (
	tenantID         = "<YOUR_TENANT_ID>"         // Entra Admin Center → Overview → Tenant ID
	apiClientID      = "<API_APP_CLIENT_ID>"      // App registrations → workshop-api → Application (client) ID
	requiredScope    = "access_as_user"           // Scope to validate for delegated user flow
	requiredAppRole  = "Svc.Invoke"               // App Role to validate for M2M flow
	jwksURL          = "https://login.microsoftonline.com/" + tenantID + "/discovery/v2.0/keys"
	expectedIssuer   = "https://login.microsoftonline.com/" + tenantID + "/v2.0"
)

func main() {
	kf, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		log.Fatalf("failed to create keyfunc: %v", err)
	}

	http.HandleFunc("/api/profile", func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenStr, kf.Keyfunc,
			jwt.WithIssuer(expectedIssuer),
			jwt.WithAudience(apiClientID),
			jwt.WithValidMethods([]string{"RS256"}),
		)
		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid claims", http.StatusUnauthorized)
			return
		}

		scp, _ := claims["scp"].(string)

		hasValidRole := false
		if rawRoles, ok := claims["roles"].([]any); ok {
			for _, r := range rawRoles {
				if role, ok := r.(string); ok && role == requiredAppRole {
					hasValidRole = true
					break
				}
			}
		}

		hasValidScope := false
		for _, s := range strings.Fields(scp) {
			if s == requiredScope {
				hasValidScope = true
				break
			}
		}

		if !hasValidScope && !hasValidRole {
			http.Error(w, "Forbidden: insufficient permissions", http.StatusForbidden)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"user":   claims["name"],
		})
	})

	log.Println("server started on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
```

### Implementation Notes

- **`aud` validation**: Match against the **actual `aud` claim** in the token. With v2 tokens, this is the API app's **client ID (GUID)**.
- **`scp` validation**: The `scp` claim is a **space-delimited** string (e.g., `"access_as_user profile"`). Use `strings.Fields` to split and compare, not `strings.Contains`.
- **M2M differences**: User-delegated tokens carry `scp`, while M2M tokens carry `roles`.

---

## Local Verification Steps

### 1. Start the API Server

```bash
# Passing constants via environment variables (if implemented)
TENANT_ID=<tenant-id> API_CLIENT_ID=<api-client-id> go run main.go

# Or update the constants in main.go and run
go run main.go
```

### 2. Acquire a Token and Test

#### For User-Delegated Flow (Using Postman)

1. Go to the **Authorization** tab in Postman.
2. Type: `OAuth 2.0`
3. Grant type: `Authorization Code (With PKCE)`
4. Callback URL: `https://oauth.pstmn.io/v1/callback`
5. Auth URL: `https://login.microsoftonline.com/<tenant-id>/oauth2/v2.0/authorize`
6. Access Token URL: `https://login.microsoftonline.com/<tenant-id>/oauth2/v2.0/token`
7. Client ID: `<client-app-id>`
8. Scope: `api://<api-client-id>/access_as_user`
9. Code Challenge Method: `S256`
10. Click **Get New Access Token** and follow the sign-in flow.

```bash
# Call the API with the acquired token
curl -H "Authorization: Bearer <TOKEN>" http://localhost:8081/api/profile
```

#### For M2M Flow (Using Azure CLI)

```bash
# To emulate Managed Identity locally for development, use a Service Principal
az login --service-principal -u <client-id> -p <client-secret> --tenant <tenant-id>

# Get an access token
TOKEN=$(az account get-access-token --resource api://<api-client-id> --query accessToken -o tsv)

# Call the API
curl -H "Authorization: Bearer $TOKEN" http://localhost:8081/api/profile
```

---

## Troubleshooting

| Symptom | Cause | Solution |
| ---- | ---- | ---- |
| **401 Unauthorized / `aud` mismatch** | The token was not issued for your API | Decode the token at [jwt.ms](https://jwt.ms) and confirm that `aud` matches the API's client ID. |
| **403 Forbidden** | Signature passed but permissions missing | Check that the required `scp` or `roles` are in the token. Ensure consent was granted or the App Role was assigned. |
| **CORS error in SPA token exchange** | Platform configuration error | Ensure the Redirect URI is under the `Single-page application` type, not `Web`. |
| **`roles` claim is missing** | User flow used instead of M2M / Assignment skipped | If M2M, ensure you used Client Credentials. Verify the `az rest` assignment step. |
| **`iss` mismatch** | Wrong tenant or mixed v1/v2 versions | Recheck `requestedAccessTokenVersion` and the Resource Server's issuer validation settings. |

---

## Configuration Checklist

### M2M (Managed Identity)

- [ ] Register API App and set Application ID URI.
- [ ] Create App Role (`Svc.Invoke`).
- [ ] Set `requestedAccessTokenVersion: 2` in Manifest.
- [ ] Enable Managed Identity on the Azure resource.
- [ ] Assign App Role to Managed Identity (`az rest`).
- [ ] Validate `roles` in the API server.

### User-Delegated Access

- [ ] Expose Scope (`access_as_user`) in the API App.
- [ ] Register Client App (select **Single-page application** for SPAs).
- [ ] Register the Redirect URI.
- [ ] Add API Permissions to the Client App.
- [ ] Grant Admin Consent (if required).
- [ ] Validate `scp` in the API server.

---

## References

- [Microsoft identity platform and OAuth 2.0 authorization code flow](https://learn.microsoft.com/en-us/entra/identity-platform/v2-oauth2-auth-code-flow)
- [How to add a redirect URI to your application](https://learn.microsoft.com/en-us/entra/identity-platform/how-to-add-redirect-uri)
- [Application manifest reference](https://learn.microsoft.com/en-us/entra/identity-platform/reference-app-manifest)
- [Scopes and permissions in the Microsoft identity platform](https://learn.microsoft.com/en-us/entra/identity-platform/scopes-oidc)
- [Assign an application role to a managed identity using PowerShell](https://learn.microsoft.com/en-us/entra/identity/managed-identities-azure-resources/assign-app-role-managed-identity-powershell)
