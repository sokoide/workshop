# Web Security Reference

Read individual background topics as needed. Runnable exercises remain in [the defense workshop](web_security_defense_en.md).

## Reference: Background Not Exercised in This Lab

> The topics below are **reference material**: they could not be turned into a hands-on exercise against this victim application (its structure does not exercise them), but they complete the countermeasure picture.

### CORS Fundamentals

CORS (Cross-Origin Resource Sharing) is the mechanism by which the browser controls access to resources from "different origins" (domain, port, protocol).

**Origin examples:**

```text
https://example.com:443/api/data
       ^^^^^^^^^^^^^^ ^^^
       Origin         Path

Same origin:
  ✓ https://example.com/api/data
  ✓ https://example.com:443/api/data

Different origin:
  ✗ https://api.example.com/api/data  (Different domain)
  ✗ http://example.com/api/data       (Different protocol)
  ✗ https://example.com:8080/api/data (Different port)
```

**Why CORS is necessary:** browsers restrict cross-origin requests with the **Same-Origin Policy (SOP)**. Without it, any site's JavaScript could `fetch()` your logged-in services and read the responses. CORS is the controlled way to relax SOP:

```text
1. Browser adds Origin header to the request
   Origin: https://example.com

2. Server adds Access-Control-Allow-Origin to the response
   Access-Control-Allow-Origin: https://example.com  (allowed)
   Access-Control-Allow-Origin: *                        (allow all)

3. Browser verifies the match
   - Match → passes response to JavaScript
   - Mismatch → throws an error
```

**Key CORS headers:**

| Header                             | Description                  | Pitfall                  |
| :--------------------------------- | :--------------------------- | :----------------------- |
| `Access-Control-Allow-Origin`      | Allowed origin               | Use `*` cautiously       |
| `Access-Control-Allow-Credentials` | Allow sending credentials    | Cannot be used with `*`  |
| `Access-Control-Allow-Methods`     | Allowed HTTP methods         | Only necessary ones      |
| `Access-Control-Allow-Headers`     | Allowed request headers      | Only necessary ones      |
| `Access-Control-Max-Age`           | Cache duration for preflight | Set an appropriate value |

**Preflight requests (OPTIONS method):**

```text
For complex requests (PUT, DELETE, custom headers, etc.):

1. Browser sends OPTIONS request first
   OPTIONS /api/data HTTP/1.1
   Origin: https://example.com
   Access-Control-Request-Method: PUT

2. Server returns permission
   Access-Control-Allow-Origin: https://example.com
   Access-Control-Allow-Methods: GET, PUT, DELETE

3. Browser sends the actual PUT request
   PUT /api/data HTTP/1.1
   Origin: https://example.com
```

**CORS and CSRF together:**

```text
Without CORS enforcement:
  fetch() from attacker site → request sent
                            → Cookie automatically attached
                            → response readable
                            → CSRF-style attacks are easy

With CORS enforcement (STEP 6):
  fetch() from attacker site → request still sent
                            → Cookie still attached (outside CORS)
                            → response blocked
                            → attacker cannot read the response

*Note: even when the response cannot be read, side effects (POST/DELETE etc.)
   still happen → CSRF tokens and SameSite cookies remain necessary.
```

### CSRF and CORS - Different Protection Goals

|                       | **CSRF**                                           | **CORS**                                   |
| :-------------------- | :------------------------------------------------- | :----------------------------------------- |
| **Primary Target**    | **Request Submission**                             | **Response Reading**                       |
| **Protection Target** | **Server-side** (preventing unintended operations) | **Client-side** (preventing data theft)    |
| **Attack Nature**     | Impersonation to **perform actions**               | Data **theft** from other sites            |
| **Browser's Role**    | Auto-attaches Cookies (enables attacks)            | Blocks response reading (provides defense) |

**CSRF protection goal:** verify "who sent the request"

```text
Attacker: Wants to send a "change password" request while impersonating the user
         ↓
Server: Needs to verify if it is truly the user's own operation
         ↓
CSRF Token: A secret value that only a user who viewed the form possesses
         ↓
Result: Attacker site doesn't have the token → request rejected ✓
```

**CORS protection goal:** control "who can read the response"

```text
Attacker's JavaScript: fetch('https://victim.com/api/profile')
         ↓
Browser: Sends request with Cookies attached
         ↓
Server: Authentication succeeds → response includes sensitive data
         ↓
[This is where CORS makes the decision]
         ↓
CORS denied: browser blocks the response ✓
           → JavaScript cannot read the data

CORS misconfigured: browser passes the response to JavaScript ✗
           → attacker steals sensitive data
```

**Complementary relationship:**

```text
[Without CSRF protection]
Attacker's form → unauthorized request sent → server executes it
                 → data gets modified ✗

[Without CORS protection]
Attacker's fetch → request sent → server executes it
                 → response unreadable, but side effects still occur ✗

[Both necessary]
CSRF: guarantees request authenticity (action defense)
CORS: restricts response reading (information defense)
```

### Why Preflight Requests Alone Cannot Prevent CSRF

**Conclusion:** preflight requests (OPTIONS method) cannot fully prevent CSRF.

**When preflight occurs** — only for "complex requests":

```javascript
// ✅ Preflight occurs
fetch(url, {
  method: "PUT", // PUT/DELETE/PATCH
  headers: { "X-Custom": "value" }, // Custom headers
});

// ❌ No preflight (simple request)
fetch(url, {
  method: "POST",
  headers: { "Content-Type": "application/x-www-form-urlencoded" },
});
```

**Simple POST request conditions:**

- Method: GET, HEAD, POST only
- Headers: `Accept`, `Accept-Language`, `Content-Language`, `Content-Type` only
- `Content-Type`: `application/x-www-form-urlencoded`, `multipart/form-data`, `text/plain` only

**CSRF attacks bypass preflight** by using form submissions:

```html
<!-- Attacker Site HTML -->
<form action="https://victim.com/update-email" method="POST">
  <input type="hidden" name="email" value="hacker@evil.com" />
</form>
<script>
  document.forms[0].submit(); // Auto-submit
</script>
```

This request:

```text
POST /update-email HTTP/1.1
Host: victim.com
Content-Type: application/x-www-form-urlencoded
Cookie: session_id=...  ← automatically attached

email=hacker@evil.com
```

is sent **without preflight** → bypasses the CORS check entirely.

**Attack method comparison:**

| Attack Method              | Preflight? | CORS Can Block? | CSRF Token Can Block? |
| :------------------------- | :--------: | :-------------: | :-------------------: |
| **Form POST**              |   ❌ No    |      ❌ No      |        ✅ Yes         |
| **fetch + PUT/DELETE**     |   ✅ Yes   |     ✅ Yes      |        ✅ Yes         |
| **fetch + Custom Headers** |   ✅ Yes   |     ✅ Yes      |        ✅ Yes         |
| **img tag GET**            |   ❌ No    |      ❌ No      |      ⚠️ Limited       |

**Summary:**

1. **Cannot prevent form submission attacks** — the most common CSRF vector; requests are sent without preflight.
2. **Cannot prevent GET request attacks** — `<img src="https://victim.com/delete-account">`; GET never triggers preflight.
3. **Risks remain without SameSite cookies** — even preflighted requests send Cookies; reading can be blocked, side effects cannot.

Therefore, CORS serves only as information-leak protection; CSRF defense still requires CSRF tokens or SameSite cookies — exactly the layering you built in STEP 3 and STEP 4.

### CSP Deep Dive

#### What is CSP?

CSP is an HTTP response header that tells the browser which sources of resources (scripts, images, styles, etc.) are allowed for the current page. It is a powerful defense layer that significantly mitigates XSS risks.

**Without CSP:**

```text
Browser: Executes all <script> tags on the page.
         → Malicious injections like <script>alert(document.cookie)</script> run.
```

**With CSP:**

```text
Server:  Content-Security-Policy: script-src 'self'
Browser: "Only execute scripts that come from the same origin ('self')."
         → Inline <script> tags are blocked.
         → Injected scripts are not executed ✓
```

#### How CSP Works

```text
1. Server declares policy via HTTP response header
   Content-Security-Policy: script-src 'self'; style-src 'self'

2. Browser evaluates policy during page load
   - Checks the source of each resource
   - Blocks resources that violate the policy

3. On violation
   - Outputs error message to console
   - Sends violation report (if configured)
```

#### Common Directives

| Directive         | Controls                              | Example                                    |
| :---------------- | :------------------------------------ | :----------------------------------------- |
| `script-src`      | Where JavaScript can be loaded from   | `'self'`, `'unsafe-inline'`, `'nonce-abc'` |
| `style-src`       | Where CSS can be loaded from          | `'self'`, `'unsafe-inline'`                |
| `img-src`         | Where images can be loaded from       | `'self'`, `https://images.example.com`     |
| `font-src`        | Where fonts can be loaded from        | `'self'`, `https://fonts.googleapis.com`   |
| `connect-src`     | Destinations for fetch/XHR/WebSockets | `'self'`, `https://api.example.com`        |
| `frame-src`       | Where iframes can be loaded from      | `'self'`, `https://trusted.com`            |
| `frame-ancestors` | Which pages can embed this page       | `'none'`, `'self'` (Clickjacking defense)  |
| `default-src`     | Fallback for other directives         | `'self'`                                   |
| `object-src`      | Plugin content                        | `'none'` (Recommended)                     |
| `base-uri`        | Restricts `<base>` tags               | `'self'`                                   |
| `form-action`     | Where forms can be submitted          | `'self'`                                   |

#### Source Values

| Value             | Meaning                                          | Risk                               |
| :---------------- | :----------------------------------------------- | :--------------------------------- |
| `'self'`          | Same origin only                                 | Safe (Recommended)                 |
| `'none'`          | Allow nothing                                    | Safest                             |
| `'unsafe-inline'` | Allow inline scripts/styles                      | Dangerous (Negates XSS protection) |
| `'unsafe-eval'`   | Allow `eval()` etc.                              | Dangerous                          |
| `'nonce-<value>'` | Only allow scripts with matching nonce attribute | Safe (Recommended)                 |
| `'sha256-<hash>'` | Only allow scripts matching the hash             | Safe                               |
| `https://...`     | Allow specific domain                            | Moderate                           |
| `*`               | Allow everything                                 | Dangerous                          |

#### Step-by-Step CSP Introduction

**Phase 1: Report-Only mode (no impact)**

```go
// Content-Security-Policy-Report-Only blocks nothing, only reports violations
w.Header().Set("Content-Security-Policy-Report-Only",
    "default-src 'self'; "+
    "report-uri /csp-report")

// Use this in production first to collect violation logs before
// switching to an enforcing CSP
```

**Phase 2: Lenient policy (migration phase)**

```go
// Temporarily allow inline scripts (while refactoring existing code)
w.Header().Set("Content-Security-Policy",
    "default-src 'self'; "+
    "script-src 'self' 'unsafe-inline'; "+ // Remove gradually
    "style-src 'self' 'unsafe-inline'; "+
    "img-src 'self' data:; "+
    "frame-ancestors 'none'")
```

**Phase 3: Strict policy (nonce-based, recommended)**

```go
func generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

func secureHandler(w http.ResponseWriter, r *http.Request) {
	nonce := generateNonce()

	w.Header().Set("Content-Security-Policy",
		"default-src 'none'; "+
			"script-src 'nonce-"+nonce+"'; "+ // Only allow scripts with this nonce
			"style-src 'self'; "+
			"img-src 'self'; "+
			"connect-src 'self'; "+
			"frame-ancestors 'none'; "+
			"base-uri 'self'; "+
			"form-action 'self'")

	// Add the nonce attribute in the HTML
	fmt.Fprintf(w, `
        <script nonce="%s">
            // Only this script will execute
            // Injected <script> tags have no nonce → blocked
            console.log('legitimate script');
        </script>
    `, nonce)
}
```

**How nonce-based defense works:**

```text
Server generates: <script nonce="abc123">...</script>
CSP header:       script-src 'nonce-abc123'

Attacker injects: <script>alert('XSS')</script>
                   ↑ No nonce attribute → browser blocks ✓

<script nonce="abc123">alert('XSS')</script>
                   ↑ Attacker cannot know the nonce value → blocked ✓
```

#### What CSP Can and Cannot Prevent

| Attack                     | Prevented by CSP?                  | Reason                                  |
| :------------------------- | :--------------------------------- | :-------------------------------------- |
| Inline `<script>` XSS      | Yes, with `script-src 'self'`      | Inline scripts are blocked              |
| External malicious scripts | Yes, with `script-src 'self'`      | Non-allowed domains are blocked         |
| `eval()`-based XSS         | Yes, without `unsafe-eval`         | `eval()` is blocked                     |
| Clickjacking               | Yes, with `frame-ancestors 'none'` | iframe embedding is blocked             |
| CSRF                       | No                                 | CSRF uses form submissions, not scripts |
| `<img onerror=...>` XSS    | Yes, without `unsafe-inline`       | Inline event handlers are blocked       |

#### Implementation: Go CSP Middleware

```go
func cspMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := generateNonce()
		r.Header.Set("X-CSP-Nonce", nonce) // Make available to handlers

		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; "+
				"script-src 'nonce-"+nonce+"'; "+
				"style-src 'self'; "+
				"img-src 'self'; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'; "+
				"report-uri /csp-report")

		next.ServeHTTP(w, r)
	})
}
```

#### Checking CSP Violations

```text
Chrome DevTools:
  Console tab → "Refused to execute inline script because it violates
                  the following Content Security Policy directive: 'script-src'"

  Network tab → Check POST requests to "csp-report"

Example report body:
{
  "csp-report": {
    "document-uri": "http://localhost:8080/xss/stored",
    "violated-directive": "script-src 'self'",
    "blocked-uri": "inline"
  }
}
```

### Advanced Attack Techniques

#### DOM-based XSS

Unlike traditional XSS, the server response is not vulnerable; the vulnerability arises when client-side JavaScript performs DOM operations unsafely.

```javascript
// Vulnerable Code Example
const hash = location.hash.substring(1);
document.getElementById("output").innerHTML = hash; // Direct insertion

// Attack URL: http://victim.com/#<img src=x onerror=alert('XSS')>
```

**Countermeasures:**

- Use `textContent` or `innerText` instead of `innerHTML`.
- Use sanitization libraries like DOMPurify.

#### Self-XSS

An attacker tricks a user into executing a malicious script themselves, often by claiming it's a "required command" to gain extra features.

**Countermeasures:**

- User education (never paste untrusted code into the console).
- Require re-authentication for critical operations.
