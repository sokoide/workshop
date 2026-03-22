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

---

#### 4. Create an App Service and Call the API

This is a Go implementation example for an App Service (Client) acting as a web server that retrieves a token using Managed Identity and displays the result in the browser.

#### 1. Add dependency packages

Use `azidentity` from the Azure SDK for Go.

```bash
go get github.com/Azure/azure-sdk-for-go/sdk/azidentity
go get github.com/Azure/azure-sdk-for-go/sdk/azcore
```

#### 2. Go code example

`azidentity.NewDefaultAzureCredential` allows the code to work in both local environments (logged in via Azure CLI) and Azure environments (Managed Identity) without changes.

```go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

func main() {
	// Get port from environment variable (App Service default is 8080)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1. Authenticate using Managed Identity
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			http.Error(w, "Failed to create credential: "+err.Error(), 500)
			return
		}

		// 2. Attempt to acquire a token
		scope := os.Getenv("API_SCOPE")
		if scope == "" {
			fmt.Fprintln(w, "Warning: API_SCOPE is not set. Using default...")
			scope = "https://graph.microsoft.com/.default" // For testing
		}

		token, err := cred.GetToken(context.Background(), policy.TokenRequestOptions{
			Scopes: []string{scope},
		})

		if err != nil {
			// Display error in browser on failure
			fmt.Fprintf(w, "❌ Token Error: %v\n", err)
			log.Printf("Token Error: %v", err)
			return
		}

		// 3. Display success message in browser
		fmt.Fprintf(w, "✅ Managed Identity Success!\n")
		fmt.Fprintf(w, "Token (first 10 chars): %s...\n", token.Token[:10])
		fmt.Fprintf(w, "Expires On: %v\n\n", token.ExpiresOn)
		
		log.Printf("Successfully retrieved token for scope: %s", scope)

		// 4. Call the API Endpoint
		apiEndpoint := os.Getenv("API_ENDPOINT")
		if apiEndpoint == "" {
			fmt.Fprintln(w, "⚠️ API_ENDPOINT is not set. Skipping API call.")
			return
		}

		client := &http.Client{}
		req, err := http.NewRequest("GET", apiEndpoint, nil)
		if err != nil {
			fmt.Fprintf(w, "❌ Failed to create API request: %v\n", err)
			return
		}

		req.Header.Set("Authorization", "Bearer "+token.Token)
		resp, err := client.Do(req)
		if err != nil {
			fmt.Fprintf(w, "❌ API Call Failed: %v\n", err)
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(w, "✅ API Call Success! (Status: %s)\n", resp.Status)
		fmt.Fprintf(w, "Response Body:\n%s\n", string(body))
	})

	log.Printf("Starting server on port %s...", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
```

#### 3. Important Implementation Notes

- **App Service Behavior**: Once the container is running, navigate to `http://<your-app-name>.azurewebsites.net/` to trigger the Managed Identity token retrieval and subsequent API call. The results will be displayed in your browser.
- **Environment Variables**:
  - `API_SCOPE`: The scope of the API you wish to access (e.g., `api://<api-client-id>/.default`).
  - `API_ENDPOINT`: The URL of the API you want to call (e.g., `https://<api-app-name>.azurewebsites.net/api/profile`).
- **API-side validation**: As mentioned earlier, the API server (Resource Server) must validate that the `aud` claim matches its own Client ID.

---

#### 5. Enable managed identity on the Azure resource

- Open the **Azure portal**
- Go to the target resource, such as App Service
- Open `Identity`
- Enable either `System assigned` or `User assigned`
- Record the managed identity's **principal object ID**

#### 6. Assign the app role to the managed identity

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

The following example uses `github.com/MicahParks/keyfunc/v3` and `github.com/golang-jwt/jwt/v5`. It reads configuration from environment variables and listens on port `8080` (App Service default).

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

func main() {
	// Configuration from environment variables
	tenantID := os.Getenv("TENANT_ID")
	apiClientID := os.Getenv("API_CLIENT_ID")
	requiredScope := os.Getenv("REQUIRED_SCOPE")
	if requiredScope == "" {
		requiredScope = "access_as_user"
	}
	requiredAppRole := os.Getenv("REQUIRED_APP_ROLE")
	if requiredAppRole == "" {
		requiredAppRole = "Svc.Invoke"
	}

	if tenantID == "" || apiClientID == "" {
		log.Fatal("Error: TENANT_ID and API_CLIENT_ID must be set")
	}

	jwksURL := fmt.Sprintf("https://login.microsoftonline.com/%s/discovery/v2.0/keys", tenantID)
	expectedIssuer := fmt.Sprintf("https://login.microsoftonline.com/%s/v2.0", tenantID)

	// Fetch JWKS for token validation
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
		// Validate JWT signature, issuer, and audience
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

		// Check for App Roles (for M2M flow)
		hasValidRole := false
		if rawRoles, ok := claims["roles"].([]any); ok {
			for _, r := range rawRoles {
				if role, ok := r.(string); ok && role == requiredAppRole {
					hasValidRole = true
					break
				}
			}
		}

		// Check for Scopes (for User-delegated flow)
		scp, _ := claims["scp"].(string)
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
			"claims": claims,
		})
	})

	// Use PORT environment variable (default to 8080 for App Service)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting API server on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
```

### Implementation Notes

- **`aud` validation**: Match against the **actual `aud` claim** in the token. With v2 tokens, this is the API app's **client ID (GUID)**.
- **Meaning of `scp` and `access_as_user`**:
  - `access_as_user` is a **Scope** used exclusively in the **"User-delegated flow (Pattern B)"**.
  - It represents a "Delegated Permission" where the client app acts on behalf of the signed-in user. You define this in the "Expose an API" menu in Entra ID.
- **Permissions in M2M Flow**:
  - **The `scp` (Scope) claim is NOT used in the M2M flow (Pattern A).** Instead, **`roles` (App Role)** are used.
  - In communication via Managed Identity, the token contains the assigned App Role in the `roles` claim, while the `scp` claim is typically absent.
- **Validation Logic**: The Go code above is structured to allow access if **either** `hasValidScope` (for users) or `hasValidRole` (for M2M) is true.
- **Environment Variables**: Make sure to set `TENANT_ID` and `API_CLIENT_ID` in your App Service configuration.
- **Deployment**: Use a Dockerfile similar to the one in section 4-1 to build and deploy as an amd64 image.

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
| **`ManagedIdentityCredential: managed identity timed out`** | No App Role assigned to the API | **This is expected if no App Role is assigned.** Entra ID rejects the token request due to insufficient permissions, leading the SDK to timeout. Assign the App Role to the Managed Identity. |

**Expected behavior when successful:**
Once the role is assigned, you will see:

```text
✅ Managed Identity Success!
Token (first 10 chars): eyJ0eXAiOi...
Expires On: 2026-03-23 ...

✅ API Call Success! (Status: 200 OK)
Response Body:
{"claims":{..., "roles":["Svc.Invoke"], ...}, "status":"ok", ...}
```

**Expected behavior if App Role is NOT assigned:**
If Entra ID denies the request due to insufficient permissions, you will see the following error in your browser:

```text
❌ Token Error: DefaultAzureCredential: failed to acquire a token.
Attempted credentials:
	EnvironmentCredential: missing environment variable AZURE_TENANT_ID
	WorkloadIdentityCredential: no client ID specified. Check pod configuration or set ClientID in the options
	ManagedIdentityCredential: managed identity timed out. See https://aka.ms/azsdk/go/identity/troubleshoot#dac for more information
	AzureCLICredential: Azure CLI not found on path
	AzureDeveloperCLICredential: Azure Developer CLI not found on path
```

---
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
