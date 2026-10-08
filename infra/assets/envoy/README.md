# Envoy Reverse Proxy Demo

Configuration-driven demo of Envoy as a reverse proxy (timeouts and retries) in front of a small Go backend. Part of the [Envoy Workshop](../../envoy_en.md).

## What this is

A `docker-compose.yml` stack that runs Envoy v1.31 and a Go backend image. The `envoy.yaml` route applies a 2s timeout and 3 retries on 5xx, so you can observe Envoy absorbing slow or failing backend responses. There is no root-level Go module; the backend in `app/` is its own module built by Docker.

## Requirements

- Podman (`podman compose`) or Docker (`docker compose`)
- curl
- The Envoy admin interface uses port 9901, the proxy listens on 10000

## How to run

```bash
podman compose up -d   # or: docker compose up -d
```

Then verify routing, timeout, and retry behavior:

```bash
# Plain routing through Envoy to the backend
curl http://localhost:10000/
# Output: backend ok

# Timeout: backend sleeps 3s, but the route timeout is 2s
curl -i http://localhost:10000/sleep/3
# Output: HTTP 504 upstream timeout

# Retry: backend returns 500, Envoy retries 3 times then gives up
curl -i http://localhost:10000/error/500
# Output: HTTP 500 with "error 500"
```

Envoy's admin dashboard is available at <http://localhost:9901>. Teardown: `podman compose down`.

See the [workshop](../../envoy_en.md) for the full walkthrough.

## Layout

```text
.
├── envoy.yaml           # Envoy listener/route/cluster config (timeout + retry policy)
├── docker-compose.yml   # Envoy + backend services
└── app/                 # Go backend module ("/", "/sleep/N", "/error/N" handlers)
    ├── main.go
    ├── Dockerfile
    └── go.mod
```
