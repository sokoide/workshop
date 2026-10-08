# Podman Runtime Notes (rootless) — verified 2026-10-08

Findings from actually running every workshop asset on this machine
(Podman 4.9.3, rootless). Keep this file next to the docker migration:
most fixes below are podman-rootless-specific and can be reverted or
simplified once Docker (rootful) replaces Podman.

## 1. `podman compose` needs the user socket service running

`podman compose` delegates to an external provider (`podman-compose` or
docker-compose) over the Docker-compatible API socket. On this machine the
socket is not started by default (`systemctl --user start podman.socket`
fails without a user session bus), so compose fails with
`Cannot connect to the Docker daemon at unix:///run/user/<uid>/podman/podman.sock`.

Workaround used during verification:

```bash
mkdir -p /run/user/1000/podman
podman system service --time=0 unix:///run/user/1000/podman/podman.sock &
```

Docker note: not needed — docker-compose talks to `/var/run/docker.sock`.

## 2. Rootless cannot bind ports < 1024

Rootless podman maps ports through `rootlessport`, which refuses privileged
ports: traefik's original `80:80` mapping failed with
`rootlessport cannot expose privileged port 80`.

Fix applied: the traefik asset maps host `8088:80` (comment in
`infra/assets/traefik/docker-compose.yml`, updated curls in the docs).
Docker note: with rootful Docker you can switch back to `80:80`.

## 3. Rootless socket path differs from the docs

The rootless API socket lives at `$XDG_RUNTIME_DIR/podman/podman.sock`
(e.g. `/run/user/1000/podman/podman.sock`), not `/run/podman/podman.sock`
(rootful). Compose files that mount it into containers now interpolate the
path so both layouts work:

```yaml
- ${PODMAN_SOCKET:-${XDG_RUNTIME_DIR}/podman/podman.sock}:/var/run/docker.sock:ro
```

(See `infra/assets/traefik/docker-compose.yml`.) Docker note: replace with
`/var/run/docker.sock`.

## 4. `podman run` with a fixed container name is not idempotent

Re-running `make redis-up`-style targets failed with "container name
already in use". Fixed by adding `--replace` to the fixed-name `podman run`
targets in `infra/assets/redis_leaderboard/`, `rabbitmq_crypto/`, and
`secret_management/` Makefiles. Docker note: same behavior on Docker;
`--replace` keeps working.

## 5. kind + rootless podman (kubernetes workshops)

See the kubernetes workshop docs and prior verification notes: `kind load
docker-image` does not work against the rootless socket from
non-interactive shells (use `podman save` + `kind load image-archive`), and
NodePort/ingress access goes through `127.0.0.1` hostPort forwarding.

## 6. Registry-side breakage (not podman's fault, hit during verification)

- **MinIO**: community images are gone (publishing stopped 2025-10; Docker
  Hub/quay.io repos deleted 2026-09; `dl.min.io` returns HTTP 410). The
  minio workshop now uses the frozen Bitnami mirror
  `docker.io/bitnamilegacy/minio` (no embedded web Console on that build).
- **Keycloak**: `quay.io/keycloak/keycloak:26.1` still pulls fine.
- **Kafka**: `docker.io/apache/kafka` works for the go_mq_patterns lab.
