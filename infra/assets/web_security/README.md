# Web Security Victim and Attacker Sites

> **⚠️ INTENTIONALLY VULNERABLE — educational use only.**
> These two Go servers implement real XSS, CSRF, clickjacking, and CORS vulnerabilities on purpose so you can exploit them in a browser and then fix them. **Never deploy this code, never expose it to a network, and never reuse these patterns in production.** Run it only on localhost for study.

A deliberately insecure "victim" site and a hostile "attacker" site for hands-on web vulnerability exercises. Part of the [Web Security Workshop](../../web_security_en.md); the hardened comparison version lives in the [Defense Workshop](../../web_security_defense_en.md), which copies this asset and fixes each flaw.

## What this is

`victim/` is a small site on `:8080` with a session cookie, a stored/reflected comment area, an email-change form, and a profile API — all without any of the usual defenses. `attacker/` on `:8081` serves pages that attack the victim site: a CSRF form that silently changes the victim's email, a clickjacking page that frames the victim site, and CORS probes. Each `main.go` is a single self-contained file using only the standard library.

## Requirements

- Go 1.26.2+ (separate `go.mod` in `victim/` and `attacker/`)
- A browser on the same machine (the attacks are browser-based)
- No containers or external tools needed

## How to run

In two terminals:

```bash
cd victim
go run main.go
# Output: "victim site starting" addr=:8080

cd attacker
go run main.go
# Output: "attacker site starting" addr=:8081
```

Open <http://localhost:8080> and follow the exercises: log in to set the session cookie, then use the attacker pages at <http://localhost:8081> to steal it (XSS), change the email without consent (CSRF), trap clicks (clickjacking), or read the profile cross-origin (CORS). The [workshop](../../web_security_en.md) gives step-by-step instructions per vulnerability.

## Layout

```text
.
├── victim/    # Insecure target site (:8080) — XSS sinks, no CSRF tokens,
│              # no security headers, permissive data export
│    └── main.go
└── attacker/  # Attack pages (:8081) — CSRF form, clickjacking iframe, CORS probe
    └── main.go
```
