# HTTP Persistent Connections Playground

A single Go server that demonstrates every persistent-connection style — HTTP/1.1 keep-alive, WebSocket, SSE, HTTP/2, HTTP/3 (QUIC), and gRPC — each with an artificial 2-second delay so concurrency differences are visible. Part of the [HTTP Persistent Connections Workshop](../../http_persistent_en.md).

## What this is

One `main.go` starts four listeners: HTTP/1.1 + WebSocket + SSE on `:8080`, HTTP/2 (TLS) on `:8443`, HTTP/3 (QUIC) on `:8444`, and gRPC on `:50051`. Because every handler sleeps 2 seconds, running two requests in parallel shows whether the protocol multiplexes them over one connection (HTTP/2, HTTP/3, gRPC, WebSocket) or opens a new connection each time (plain HTTP/1.1 without keep-alive).

## Requirements

- Go 1.23+ (`go.mod`)
- `make` targets: `setup` (protoc plugins + grpcurl), `cert` (openssl self-signed cert), `gen` (protoc codegen), `run`, `test-http1`, `test-ws`, `test-grpc`
- A self-signed `server.crt` / `server.key` pair is committed; regenerate with `make cert` if expired
- Optional clients: `websocat` (WebSocket), `grpcurl` (gRPC), `curl` (everything else)

## How to run

```bash
make run   # go mod tidy && go run main.go
```

The server logs each listener as it starts. From other terminals (or follow the [workshop](../../http_persistent_en.md)):

```bash
make test-http1   # curl -v http://localhost:8080/ twice — observe connection reuse
make test-ws      # echo over WebSocket at ws://localhost:8080/ws (2s echo delay)
make test-grpc    # unary and streaming RPCs against localhost:50051

# SSE: receives 5 events, one every 2 seconds
curl -N http://localhost:8080/sse

# HTTP/2 (TLS) and HTTP/3 (QUIC)
curl -k --http2 https://localhost:8443/
curl -k --http3 https://localhost:8444/
```

Regenerate gRPC stubs after editing `proto/greeter.proto` with `make gen` (requires `protoc`).

## Layout

```text
.
├── main.go               # All four servers: HTTP/1.1+WS+SSE, HTTP/2, HTTP/3, gRPC
├── proto/greeter.proto   # gRPC service definition
├── pb/                   # Generated protobuf/gRPC Go code
├── server.crt|key        # Self-signed TLS certificate for :8443 / :8444
├── traefik.yml           # Traefik static config (file provider, :80 entrypoint)
├── traefik-dynamic.yml   # Traefik dynamic routing to the app on :8080
├── Dockerfile            # Containerized server
└── Makefile              # setup / cert / gen / run / test-* targets
```
