# HTTP Persistent Connection Workshop: Keep-Alive, WebSocket, HTTP/2, gRPC, HTTP/3

> **Note on Terminology**: The term "Persistent Connection" used in this workshop refers to the definition in HTTP specifications (such as RFC 7230). It differs from "Data Persistence" in databases; it refers to maintaining a single TCP connection or a logical connection provided by QUIC to reuse it for multiple requests.

In this workshop, you will learn about **connection reuse and streaming**, which are the foundations of modern web applications. You will understand how connections are optimized from HTTP/1.1 to the latest HTTP/3 through hands-on exercises.

## Goals

- Understand the effect of connection reuse with HTTP/1.1 **Keep-Alive**.
- Experience bi-directional, full-duplex communication with **WebSocket**.
- Understand the efficiency of resource acquisition through **HTTP/2** multiplexing.
- Understand the difference between Unary and Streaming in **gRPC** (based on HTTP/2).

---

## Why are "Persistent Connections" Necessary?

Re-establishing TCP connections (3-way handshake) and TLS connections (handshake) for every communication causes significant overhead, especially in high-latency environments.

| Protocol | Connection Handling | Features |
| :--- | :--- | :--- |
| **HTTP/1.0** | Short-lived | Disconnect after each request. High overhead. |
| **HTTP/1.1** | Persistent (Keep-Alive) | Reuse connections. In practice, requests are processed serially, leading to **HoL Blocking** due to response waiting. |
| **WebSocket** | Bi-directional | Upgraded from HTTP. Allows bi-directional sending, but **TCP-level HoL Blocking** remains. |
| **SSE** | Server-Sent Events | Unidirectional stream from server to client over HTTP. Lightweight for notifications. **TCP-level HoL Blocking** remains. |
| **HTTP/2** | Multiplexed | Multiple "streams" within one connection. Allows parallel processing within server limits, but **TCP-level HoL Blocking** remains. |
| **gRPC** | Streaming | Based on HTTP/2, utilizes streams and flow control for efficient bi-directional communication. |
| **HTTP/3** | QUIC/UDP | Reduces TCP and TLS overhead. Rebuilds reliability over UDP, **resolving TCP-level HoL Blocking**. |

> **HoL Blocking (Head-of-Line Blocking)**:
> A phenomenon where the first request or packet in a queue blocks all subsequent requests or packets from being processed, even if they have arrived normally.
>
> - **HTTP/1.1 Level**: A "wait-in-line" at the application layer where the next request cannot be sent on the same connection until the previous response is received.
> - **TCP Level (WebSocket/SSE/HTTP/2)**:
>   TCP guarantees "ordered delivery." When data for "Image A" and "Image B" are mixed in one connection, if **even one packet of Image A is lost**, the OS holds back subsequent normal packets of Image B in the buffer (waiting for Image A's retransmission).
>   Consequently, Image B's display is blocked due to Image A's trouble. This is the essence of "pipe clogging."
>
> **Relationship between HTTP/3 and QUIC**:
> Often confused, but accurately, **the "HTTP/3 application layer" sits on top of the "QUIC transport layer protocol (replacing TCP)."**
>
> - **QUIC**: A new "communication foundation" based on UDP, featuring encryption via TLS 1.3, mobility resilience via Connection IDs, and packet loss resilience.
> - **HTTP/3**: A "convention" for sending HTTP requests by directly utilizing QUIC's multiplexing capabilities.
> - **2026 Current Supplement**: **gRPC over HTTP/3** has become a common option for implementation and operation, especially in unstable mobile networks where QUIC's robustness supports gRPC's reliability.

---

## Guide to Choosing Update Notifications (Event Delivery) (2026 Edition)

There are multiple ways to immediately inform a client that "a state has changed on the server." Here is a quick guide based on use cases:

1. **Bi-directional communication needed in browser** (Chat, Games) → **WebSocket**
2. **Unidirectional notification to browser is sufficient** (News feeds, Stock updates) → **SSE (Server-Sent Events)**
    - Implementation is very simple (`text/event-stream`), and automatic reconnection is supported by standard browsers.
3. **High network fluctuation / Modern low-latency requirements** → **WebTransport (HTTP/3)**
4. **Backend-to-Backend communication** → **gRPC Streaming**
    - Type-safe (IDL/proto) with support for server, client, and bi-directional streaming.

---

## Architecture

We will start one Go server and observe the differences in communication across different paths and protocols.

### HTTP/1.0 (Short-lived Connections)

**Estimated Sessions: 2** (for 2 requests)
Establishes a TCP connection for each request and disconnects immediately after the response. This is the most inefficient method.

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server

    Note over C,S: Request 1
    C->>S: TCP Handshake
    C->>S: GET /foo
    S-->>C: Response
    Note over C,S: TCP Close

    Note over C,S: Request 2
    C->>S: TCP Handshake
    C->>S: GET /bar
    S-->>C: Response
    Note over C,S: TCP Close
```

### HTTP/1.1 (Keep-Alive / Connection Pooling)

HTTP/1.1 introduced "connection reuse."

- **Sequential Sending: Estimated Sessions 1**
  Reuses the same connection to process requests one by one.
- **Parallel Sending: Estimated Sessions 2+ (Max ~6)**
  Most browser implementations (Chrome/Firefox, etc.) limit the number of simultaneous connections to **around 6 per domain** to speed up processing through parallelism.

> **About Pipelining**:
> While HTTP Pipelining exists in the specification to send the next request without waiting for a response, it is effectively disabled in major browsers due to the difficulty of guaranteeing response order and compatibility issues with middleboxes. In practice, it results in serial processing of one request per connection.

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server

    Note over C,S: Sequential (1 Session)
    C->>S: TCP Handshake
    C->>S: GET /foo
    S-->>C: Response
    C->>S: GET /bar (Re-use)
    S-->>C: Response

    Note over C,S: Parallel (2 Sessions)
    C->>S: Conn 1: GET /image1.jpg
    C->>S: Conn 2: GET /image2.jpg
    S-->>C: Response (from Conn 1)
    S-->>C: Response (from Conn 2)
```

### WebSocket (Upgrade)

**Estimated Sessions: 1**
Starts with an HTTP connection and then exclusively uses the same TCP session for bi-directional communication.

```mermaid
sequenceDiagram
    participant C as Client (Browser/Terminal)
    participant S as Server (Go)

    C->>S: GET /ws (Upgrade: websocket)
    S-->>C: 101 Switching Protocols
    rect rgba(145, 145, 145, 0.1)
        Note over C,S: Bi-directional session (Same TCP)
        C->>S: Data from Client
        S->>C: Data from Server
    end
```

### HTTP/2 (Multiplexing)

**Estimated Sessions: 1**
Multiple streams can be multiplexed within a single connection, allowing parallel processing within the server's limit (`SETTINGS_MAX_CONCURRENT_STREAMS`: typically 100–256).

- **Downloading 100 Images**:
  Requests can be sent without the browser's 6-connection limit. However, since the underlying layer is TCP, **a single packet loss will stall the progress of all streams on that connection** (TCP-level HoL Blocking).

> **About Server Push**:
> While Server Push using even-numbered IDs exists in the specification, it is currently rarely used.
>
> - **Original Purpose**: An optimization to "pre-push" CSS/JS that will definitely be needed after the HTML, without waiting for a client request.
> - **Caution**: It is not a general-purpose push notification mechanism like WebSocket; it is a technology for static resource delivery optimization within the same origin and cache context.
> - **2026 Trend**: **103 Early Hints** is recommended for resource preloading, while **WebSocket** or **gRPC Streaming** should be chosen for real-time notifications.

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server

    Note over C,S: Single TCP Connection
    C->>S: GET /img1.jpg (Stream 1)
    C->>S: GET /img2.jpg (Stream 3)
    C->>S: ... (Stream 5~199)
    C->>S: GET /img100.jpg (Stream 199)
    S-->>C: Data for Stream 3
    S-->>C: Data for Stream 1
    S-->>C: ...
```

### HTTP/3 (QUIC / 0-RTT)

**Estimated Sessions: 1** (Logical connection via UDP/QUIC)
Based on UDP, but QUIC ensures reliability. While the initial connection requires 1-RTT, **session resumption** allows 0-RTT, where requests can be sent before the handshake completes.

- **Downloading 100 Images**:
  In addition to HTTP/2's benefits, QUIC performs order guarantee on a "per-stream" basis. Packet loss only causes the specific image data to wait for retransmission, while **other images (streams) continue to be transferred without interruption**. This completely resolves TCP-level HoL Blocking.
  *Note: Application-layer or single-stream HoL Blocking due to ordering dependencies may still occur.*

```mermaid
sequenceDiagram
    participant C as Client
    participant S as Server

    Note over C,S: Single QUIC Session (UDP)
    C->>S: QUIC Handshake + GET /img1.jpg (0-RTT if resumed)
    C->>S: GET /img2.jpg ... GET /img100.jpg
    S-->>C: Response (Stream 0, 4, 8...)
```

---

## Preparation

1. **Navigate to the Repository**

    ```bash
    cd infra/assets/http_persistent_conn
    ```

2. **Install Tools**

   Install the system-level tools using `apt`, and use **Homebrew (Linuxbrew)** for development tools to ensure the latest versions and easy installation of `websocat`.

   ```bash
   # 1. Install system essentials via apt
   sudo apt update
   sudo apt install -y podman podman-compose git make openssl curl

   # 2. Install development tools via Homebrew
   # If you haven't installed Homebrew: /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
   brew install websocat go protobuf grpcurl
   ```

   > [!IMPORTANT]
   > Ensure both Homebrew and Go binary directories are in your `PATH`.
>
   > ```bash
   > # Example for ~/.bashrc or ~/.zshrc
   > eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
   > export PATH=$PATH:$(go env GOPATH)/bin
   > ```

   Next, set up the Go-specific Protobuf plugins using `make setup`:

   ```bash
   make setup
   ```

   ```bash
   # (Tool paths appear; no output if already installed)
   which protoc-gen-go grpcurl
   ```

1. **Generate Self-Signed Certificate**

    Since HTTP/2/3 require TLS/QUIC, create a local certificate.

    ```bash
    make cert
    ```

2. **Generate Protobuf Code**

    Generate Go stubs from `proto/greeter.proto`.

    ```bash
    make gen
    ```

3. **Start the Server**

    ```bash
    make run
    ```

    Port List:
    - `:8080` → HTTP/1.1 + WebSocket + SSE
    - `:8443` → HTTP/2 (TLS)
    - `:8444` → HTTP/3 (QUIC)
    - `:50051` → gRPC (HTTP/2)

    Keep the server running in this terminal and use another terminal for the following steps.

---

## Workshop Steps

### STEP 1: Verifying HTTP/1.1 Keep-Alive

In HTTP/1.1, connections are maintained by default. Verify this using `curl -v`.

```bash
# 1. Default (with Keep-Alive): Send two consecutive requests
curl -v http://localhost:8080/ http://localhost:8080/

# 2. Disable Keep-Alive with "Connection: close" (simulating HTTP/1.0-like behavior)
curl -v -H "Connection: close" http://localhost:8080/ http://localhost:8080/
```

**Observation Points**:

- **With Keep-Alive (Default)**:
    1. **curl Logs**: Check for `Re-using existing connection! (#0) with host localhost` in the second request's log. This confirms the TCP connection is reused.
    2. **Socket Status (ss command)**: In another terminal, verify that there is **only one** connection to `:8080`.

        ```bash
        # Ubuntu 24.04: -a allows you to see connection remnants like TIME-WAIT
        watch -n 0.1 "ss -ntp | grep :8080"
        ```

    3. **Criteria**: If only **one line** (e.g., TIME-WAIT) remains after both requests finish, it proves a single connection was reused.

        > The Linux watcher above uses `ss -ntp`, so it suppresses TIME_WAIT entries by default. Run it again with `ss -ntap` or add `-a` if you want to observe TIME_WAIT. On macOS, you can also iterate:
        >
        > ```bash
        > while true; do
        >     clear
        >     netstat -anp tcp | grep 8080 | grep -v TIME_WAIT
        >     sleep 0.1
        > done
        > ```
        >
        > Removing `| grep -v TIME_WAIT` shows everything, including TIME_WAIT lines.

- **Without Keep-Alive (Connection: close)**:
    1. **curl Logs**: Note that `Closing connection 0` appears after the first response, and `Re-using existing connection!` **does not appear** for the second request.
    2. **Socket Status**: While monitoring with `ss`, you will see **two distinct connections** (with different client-side port numbers) being created and moving to completion (e.g., TIME-WAIT).

    *Note: On macOS, use `watch -n 0.1 "lsof -iTCP:8080 -sTCP:ESTABLISHED"` or similar.*

> **Note on Timeout**:
> Servers typically have a **Keep-Alive Timeout**. If no request arrives within a certain period, the server sends a TCP disconnect (FIN). If the connection disappears during the exercise, simply send another request to perform a new handshake.

### STEP 2: Bi-directional Communication with WebSocket (Full-duplex beyond HTTP)

WebSocket starts with an **HTTP/1.1 `Upgrade` header**, but once established, it switches to **"Full-duplex communication"** where both parties can send data at any time, ignoring the HTTP request-response framework.

| Feature | HTTP/1.1 (Keep-Alive) | WebSocket |
| :--- | :--- | :--- |
| **Communication Direction** | Client-initiated Request/Response | Full-duplex (Either side can send anytime) |
| **Data Unit** | HTTP Message (Header + Body) | Lightweight Frame (Binary/Text) |
| **Overhead** | Headers required for every request | Minimal frame headers after connection |

> **When using Proxies (Nginx / Traefik)**:
>
> - **Nginx**: Does not pass `Upgrade` headers to the backend by default. Requires explicit configuration like `proxy_set_header Upgrade $http_upgrade;`.
> - **Traefik**: Modern design automatically detects WebSocket and acts as a TCP tunnel.
> - **Common Caution**: Both proxies have Idle Timeout settings that can drop inactive connections. Application-level **Heartbeats (Ping/Pong)** are essential for maintaining WebSockets.

```bash
# Send 5 messages and observe the echo responses
( for i in {1..5}; do
    printf 'message %d\n' "$i"
    sleep 1
done
) | websocat -v ws://localhost:8080/ws
```

**Observation Points**:

1. **Handshake**: Check for `HTTP/1.1 101 Switching Protocols` in the `websocat -v` log.
2. **Bi-directional Check**: This sample is an "echo server." Verify that whatever you type is immediately returned.
3. **Socket Monitoring**: Use `ss`. The socket remains in **ESTABLISHED** state while you interact, remaining as a single socket.

### STEP 2b: WebSocket Connection via Traefik (Optional)

Experience how modern reverse proxies handle WebSockets as "tunnels" in a containerized environment.

```bash
# Start server and Traefik (file-provider configuration)
podman-compose up --build -d

# Connect via Traefik (Host 18080) using the same 5-message loop from STEP 2
(
    for i in {1..5}; do
        printf 'message %d\n' "$i"
        sleep 1
    done
) | websocat -v ws://localhost:18080/ws
```

**Observation Points**:

1. **Explicit Routing by Files**: The static `traefik.yml` + `traefik-dynamic.yml` pair ensures Traefik listens for `Host(`localhost`)` and forwards to `http://app:8080` without sharing `/run/podman/podman.sock`.
2. **Host Port 18080 Access**: Compose binds host `18080:80`, so confirm Traefik → app communication using `websocat -v ws://localhost:18080/ws` or `curl http://localhost:18080/`.
3. **Socket Monitoring (Platform Specific)**:
    - **Linux**:

        ```bash
        watch -n 0.1 "ss -ntp | grep :18080"
        ```

    - **macOS**:

        ```bash
        while true; do
            clear
            lsof -nP -iTCP:18080 | grep 18080
            sleep 0.1
        done
        ```

        > The Linux watcher above uses `ss -ntp` to keep TIME_WAIT hidden. Re-run it with `ss -ntap` when you want to see TIME_WAIT entries.

4. **Transparency**: From the client's perspective, it behaves exactly like a direct connection.
5. **Note**: Don't forget to cleanup with `podman-compose down`.

### STEP 3: HTTP/2 Multiplexing

HTTP/2 creates multiple virtual "streams" within a single TCP connection to process requests in **complete parallel**.

```bash
# Force HTTP/2 to download multiple files
curl --http2 -k -v https://localhost:8443/a https://localhost:8443/b
```

**Observation Points**:

1. **Multiplexing**: In the `curl` log, check for different odd stream IDs (e.g., `[HTTP/2] [1] GET /a`, `[HTTP/2] [3] GET /b`) running simultaneously.
2. **Socket Monitoring**: Verify that the OS-level TCP socket remains **always single** even when multiple requests are flying.
3. **Socket Monitoring (Platform Specific)**:
    - **Linux**:

        ```bash
        watch -n 0.1 "ss -ntp | grep :8443"
        ```

    - **macOS**:

        ```bash
        while true; do
            clear
            lsof -nP -iTCP:8443 | grep 8443
            sleep 0.1
        done
        ```

        > The Linux watcher uses `ss -ntp`, which hides TIME_WAIT entries. Re-run it with `ss -ntap` if you need to observe queued sockets.

### STEP 4: HTTP/3 (QUIC) 0-RTT and Transition to UDP

HTTP/3 completely abandons TCP in favor of the UDP-based **QUIC** protocol.

> **Note**: The stock `curl` on Ubuntu 24.04 may not include HTTP/3 support. Run `curl --version` and look for `HTTP3` under `Features`. If `HTTP3` is missing, install the Homebrew (Linuxbrew) build of `curl` you already installed for the workshop tools and prepend it to `PATH`:

```bash
brew install curl
alias curl=$(find $(brew --prefix) -name curl |grep bin/curl)
curl --version | grep HTTP3
```

> **Why abandon TCP? (Resolving TCP-level HoL Blocking)**:
> TCP treats all data as a "single pipe" and performs order guarantee for the entire stream. QUIC performs order guarantee on a **"per-stream"** basis. If a packet for one stream is lost, only that stream waits for retransmission, while other streams continue without being blocked. This is the technical essence of resolving HoL Blocking.

```bash
# Access via HTTP/3
curl --http3 -k -v https://localhost:8444/
```

**Observation Points**:

1. **Protocol Difference**: Check for `ALPN: h3` in the `curl` log.
2. **Socket Monitoring (UDP)**:
    - **Status Display**: Since UDP is a "connectionless" protocol.
    - Linux: `sudo tcpdump -i lo -n port 8444`
    - macos: `sudo tcpdump -i lo0 -n port 8444`

### STEP 5: Diverse Streaming Experiences with gRPC (HTTP/2)

gRPC utilizes HTTP/2's long-lived connections and stream multiplexing.

#### Continuous Unary Calls vs. HTTP/1.1

When using a single `ClientConn`, gRPC Unary offers significant advantages over HTTP/1.1 Keep-Alive:

| Feature | HTTP/1.1 Keep-Alive | gRPC Unary (HTTP/2) |
| :--- | :--- | :--- |
| **Parallelism** | Serial (Must wait for response) | **Multiplexing** |
| **HoL Blocking** | Likely at connection level | **HTTP-layer HoL Mitigated** (TCP-level remains) |
| **Resource Efficiency** | Parallelism needs multiple TCP conns | Many streams over **1 TCP connection** |

#### gRPC Advantages over WebSocket

While both maintain connections, gRPC is more refined:

1. **Sematics Preservation**: WebSocket loses HTTP concepts (paths, types) after connection, but gRPC maintains a clear "Request-Response" model.
2. **Header Compression (HPACK)**: Compresses redundant headers (auth tokens, etc.), making it extremely lightweight.
3. **Standard Flow Control**: Built-in HTTP/2 window control prevents the receiver from being overwhelmed.

```bash
# 1. Continuous Unary Test
grpcurl -plaintext -d '{"name": "req1"}' localhost:50051 pb.Greeter/SayHello
grpcurl -plaintext -d '{"name": "req2"}' localhost:50051 pb.Greeter/SayHello

# 2. Server Streaming Test
grpcurl -plaintext -d '{"name": "stream"}' localhost:50051 pb.Greeter/SayHelloStream

# 3. Bidirectional Streaming Test
# (Press Ctrl+D to end input)
grpcurl -plaintext -d '{"name": "Alice"}' -d '{"name": "Bob"}' localhost:50051 pb.Greeter/Chat
```

**Observation Points**:

- Note that `grpcurl` starts a new process for each command, usually resulting in a new connection per call. The true power of gRPC (1-connection multiplexing) is maximized when reusing a long-lived `ClientConn` within an application.
- **Socket Monitoring**:
    1. **Linux**:

        ```bash
        watch -n 0.1 "ss -ntp | grep :50051"
        ```

        > The Linux watcher hides TIME_WAIT entries by using `ss -ntp`. Run it again with `ss -ntap` if you want to examine them.

    2. **macOS**:

        ```bash
        while true; do
            clear
            netstat -anp tcp | grep 50051 | grep -v TIME_WAIT
            sleep 0.1
        done
        ```

        > Remove `| grep -v TIME_WAIT` if you want to include TIME_WAIT entries in the output.

### STEP 6: Lightweight Notifications with Server-Sent Events (SSE)

Observe SSE, the easiest way to implement "server-to-client notifications" for browsers.

```bash
# Request SSE from HTTP/1.1 port
curl -v http://localhost:8080/sse
```

**Observation Points**:

1. **Content-Type**: Check for `text/event-stream`. Data arrives sequentially without closing the connection.
2. **Lightweight**: Observe how it's implemented as a "long HTTP response" rather than complex framing like WebSocket.
3. **Socket Monitoring (Platform Specific)**:
    - **Linux**:

        ```bash
        watch -n 0.1 "ss -ntp | grep :8080"
        ```

        > The Linux watcher keeps TIME_WAIT hidden via `ss -ntp`. Re-run with `ss -ntap` if you want to include those entries.

    - **macOS**:

        ```bash
        while true; do
            clear
            lsof -nP -iTCP:8080 | grep 8080
            sleep 0.1
        done
        ```

---

## Relationship with Clean Architecture

In Clean Architecture, communication protocols belong to the outermost **"Frameworks & Drivers"** layer. We isolate business logic from these details through **Dependency Inversion (DIP)**.

```mermaid
graph LR
    subgraph Domain
        DomainLogic[Domain Logic]
        DomainPort["Notification Port (Interface)"]
    end

    subgraph Use Case
        UC[Use Case Interactor]
    end

    subgraph Infra Adapters
        InboundCtrl[Inbound Controller]
        InfraImpl[Infra Adapter]
    end

    subgraph Framework and Drivers
        HTTP_Hdl[HTTP Handler]
        GRPC_SDK[gRPC SDK / Library]
    end

    HTTP_Hdl --> InboundCtrl
    InboundCtrl --> UC
    UC --> DomainPort
    InfraImpl -- implements --> DomainPort
    DomainPort --> DomainLogic
    InfraImpl --> GRPC_SDK
```

1. **Inbound (Receiving)**: Frameworks (HTTP Handlers, gRPC/gateway) receive requests and hand control to Controllers, which invoke UseCases.
2. **Domain Interfaces**: The Domain layer owns ports/interfaces (e.g., Notification Port). UseCases depend on these abstractions rather than concrete infra.
3. **Outbound (Sending)**: Infra Adapters implement the Domain port/interface and carry out the concrete communication (gRPC SDK, HTTP handler, etc.).

**Design Pattern: BFF (Backend For Frontend)**
In practice, directly hitting gRPC streaming from a browser can be difficult. A common, Clean Architecture-aligned approach is to use a **BFF that communicates with the backend via gRPC and relays information to the frontend via WebSocket or SSE**.

---

## Conclusion: How should you choose?

In modern system design, the following segregation is common:

- **Backend to Backend (Microservices)**: **gRPC** is the primary choice. Benefits from type safety, high-speed binary, and multiplexing. (REST may be chosen based on organizational standards).
- **Browser to Backend (Real-time)**: **WebSocket** is mainstream for chats, charts, and notifications. **SSE** is a strong alternative for unidirectional lightweight notifications.
- **Browser to Backend (Normal API)**: **REST (JSON/HTTP)** or **gRPC-Web**.

> [!NOTE]
> **2026 Perspective**:
> **WebTransport (HTTP/3 based)** is gaining attention. While it may be more suitable than WebSocket in some cases, it hasn't completely replaced WebSocket as of 2026. Choosing based on requirements and browser support is practical.

---

## Appendix: HTTP/3 Adoption Status (As of 2026)

- **Browser Support**: Standard support in major browsers (Chrome, Edge, Firefox, Safari).
- **Server / Infrastructure**: Accounts for a very high percentage of traffic through major CDNs. Nginx and cloud load balancers have supported it as stable for years.
- **Mobile**: QUIC's Connection ID-based continuity is reported to be highly effective in mobile networks (e.g., 5G) while moving.

---

## Cleanup

```bash
# Terminate processes
kill $(jobs -p)
# Stop containers
podman-compose down
```

---

## Next Steps

- **Challenge WebTransport**: Evaluate it as an alternative to WebSocket, comparing its multi-stream and low-latency (datagram) capabilities.
- **Load Balancer Configuration**: Investigate how L4 LBs and L7 LBs handle persistent connections differently (e.g., connection imbalance issues).
