# Traefik Reverse Proxy with Docker Labels

A docker-compose stack where Traefik v3.1 discovers two `whoami` containers via Docker labels and load-balances them on a single hostname. Part of the [Traefik Workshop](../../traefik_en.md).

## What this is

This asset is configuration-only — there is no Go code or `go.mod`. `docker-compose.yml` starts Traefik (with the Docker provider and insecure dashboard) plus two identical `traefik/whoami` services, both labeled with the same router rule `Host(\`whoami.localhost\`)`, so Traefik load-balances requests between them. The label-based discovery (no file editing, no restarts) is the point of the demo.

## Requirements

- Podman (or Docker) with `podman compose` — the stack mounts `/run/podman/podman.sock` into the container as `/var/run/docker.sock`, so a Docker-compatible socket is required
- Makefile targets: `up`, `down`, `logs`, `clean`
- Ports 80 (entrypoint) and 8080 (dashboard)

## How to run

```bash
make up   # podman compose up -d

curl -H "Host: whoami.localhost" http://localhost/
# Output: whoami response; repeat the call and the "Hostname" line
# alternates between the two whoami containers
```

Browse <http://localhost:8080> for the Traefik dashboard: the `whoami.localhost` router, the service, and both containers are visible there. Teardown: `make down` (or `make clean` to also remove volumes).

Details and exercises are in the [workshop](../../traefik_en.md).

## Layout

```text
.
├── docker-compose.yml   # Traefik + two whoami services with router labels
└── Makefile             # up / down / logs / clean
```
