# Web Security Workshop: XSS, CSRF, and Clickjacking

## Purpose

In this workshop, you will learn the mechanisms of and countermeasures against common Web application vulnerabilities: XSS (Cross-Site Scripting), CSRF (Cross-Site Request Forgery), and Clickjacking, using an intentionally vulnerable application.

## Goals

- Understand the mechanisms of XSS (Stored/Reflected) and verify JavaScript execution.
- Understand the mechanism of CSRF and verify how unauthorized requests are sent on behalf of an authenticated user.
- Understand the mechanism of Clickjacking and verify visual deception techniques using iframes.
- Understand defensive measures (escaping, tokens, header settings) for each vulnerability.

---

## Actors and Attack Flows

### XSS (Cross-Site Scripting)

An attacker embeds a malicious script into a web page and executes it in other users' browsers.

```mermaid
sequenceDiagram
    participant A as Attacker
    participant S as Vulnerable Site
    participant V as Victim

    Note over A, S: 1. Post script (<script>alert(1)</script>)
    A->>S: POST request
    Note over S: Save to DB (Stored)
    
    Note over V, S: 2. View page
    V->>S: GET /xss/stored
    S-->>V: HTML (contains script)
    Note over V: Script is executed
```text

### CSRF (Cross-Site Request Forgery)

Tricks an authenticated user into sending an unintended request (like changing a password or email address) to a vulnerable site.

```mermaid
sequenceDiagram
    participant V as Victim (Logged in)
    participant E as Malicious Site
    participant S as Vulnerable Site

    Note over V, S: 1. Logged into Vulnerable Site (Cookie held)
    V->>E: 2. Visit malicious site
    E-->>V: HTML containing attack form
    Note over V: Form is (automatically) submitted
    V->>S: 3. Unauthorized request (Cookie automatically attached)
    Note over S: Processed as user's operation
```text

### Clickjacking

Uses a transparent iframe to make the user click a button different from the one they intended.

```mermaid
graph TD
    subgraph Browser ["Browser View"]
        AttackerPage["Attacker Page (Background)"]
        VictimIframe["Vulnerable Site iframe (Transparent)"]
    end
    
    AttackerPage -->|Overlaid| VictimIframe
    UserClick["User Click"] --> VictimIframe
```text

---

## Anti-patterns and Solutions

| Vulnerability | ❌ Dangerous Implementation | ✅ Secure Countermeasure |
| :--- | :--- | :--- |
| **XSS** | Output user input as-is in HTML | Escape output appropriately, set CSP |
| **CSRF** | Rely solely on Cookies for authentication | Use CSRF tokens, set SameSite attribute |
| **Clickjacking** | Allow iframe embedding from external sites | Set `X-Frame-Options` or CSP `frame-ancestors` |

---

## Architecture

This workshop provides both "Victim Site" and "Attacker Site" endpoints within a single Go application.

### Directory Structure

```text
infra/assets/web_security/
├── main.go         # Vulnerable application
├── go.mod
├── Dockerfile
└── docker-compose.yml
```text

---

## Preparation

### 1. Start the Application

```bash
cd infra/assets/web_security
podman compose up -d
```text

### 2. Verification

Access `http://localhost:8080` in your browser.

---

## Workshop Steps

### STEP 1: Stored XSS

Save a script in a board-like function and verify it executes every time the page is opened.

1. Navigate to the `Stored XSS Demo` page.
2. Enter `<script>alert('XSS!');</script>` into the input field and submit.
3. Verify that the page reloads and the alert is displayed.

### STEP 2: Reflected XSS

Verify that a script contained in URL parameters is reflected directly on the page.

1. Navigate to the `Reflected XSS Demo` page.
2. Access the URL with a script appended to the end:
   - `http://localhost:8080/xss/reflected?q=<script>alert(document.cookie);</script>`
3. Verify that the cookie information is displayed (accessible because HttpOnly attribute is missing).

### STEP 3: CSRF (Cross-Site Request Forgery)

1. Click `Login` to set the session Cookie.
2. Return to the `Victim Site` top page and confirm the current email is `victim@example.com`.
3. Click `CSRF Attack Page` under `Attacker's Links`.
4. Click the "Claim Prize!" button on the attacker's site.
5. Return to the top page of the victim site and verify that the email address has been changed to `hacker@evil.com` without your authorization.

### STEP 4: Clickjacking

1. Click `Clickjacking Attack Page` under `Attacker's Links`.
2. Try to click the "GET COOKIES" button on the attacker site, and verify that you actually clicked the "Confirm Transfer" button on the hidden victim site behind it.
   - *Note: iframe is semi-transparent for demonstration purposes.*

---

## Implementation of Countermeasures

### XSS Prevention

Go's `html/template` package escapes variables by default.

```go
// ❌ Dangerous: Print without escaping
fmt.Fprintf(w, "<li>%s</li>", comment)

// ✅ Secure: Use template engine (automatic escaping)
// See official Go documentation for details
```text

### CSRF Prevention

- **CSRF Tokens**: Issue a unique token per request and verify it on the server.
- **SameSite Cookie**: Set `SameSite=Lax` or `Strict` for Cookies.

### Clickjacking Prevention

Add the following to response headers:

- `X-Frame-Options: DENY` or `SAMEORIGIN`
- `Content-Security-Policy: frame-ancestors 'self'`

---

## Cleanup

```bash
podman compose down
```text

---

## References

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [IPA: How to build secure websites](https://www.ipa.go.jp/security/vuln/websecurity.html)
