# Infrastructure Workshop Glossary

This document provides concise explanations of technical terms used across various workshops.

## <a name="network"></a>Networking Terms

### L1 / L2 / L3 (Layer 1/2/3)

Refers to the layers of the OSI reference model or the TCP/IP model.

| Layer | Name            | Key Protocols/Devices | Description                                      |
| :---- | :-------------- | :-------------------- | :----------------------------------------------- |
| L1    | Physical Layer  | Cables, NICs          | Transmission and reception of electrical signals |
| L2    | Data Link Layer | Ethernet, MAC address | Communication within the same segment            |
| L3    | Network Layer   | IP, Router            | Routing between different networks               |
| L4    | Transport Layer | TCP, UDP              | End-to-end communication control                 |

### L2 Separation (Layer 2 Separation)

A technology that logically separates communication on the same physical network. Examples include VLAN and VxLAN.

### L3 Routing (Layer 3 Routing)

The function of forwarding packets between different IP subnets, typically handled by a router.

### MAC Address

Media Access Control Address. A unique 48-bit identifier assigned to a network interface (NIC).

```text
Example: 00:1A:2B:3C:4D:5E
```

### IP Address

Internet Protocol Address. An address used to identify devices on a network. IPv4 is 32-bit, and IPv6 is 128-bit.

```text
IPv4 Example: 192.168.1.1
IPv6 Example: 2001:db8::1
```

### Subnet Mask

Indicates which part of an IP address is the network portion and which is the host portion.

```text
Example: 255.255.255.0 (/24)
     -> Range of 192.168.1.0 to 192.168.1.255
```

### Default Gateway

The IP address of the router that forwards packets to destinations outside the local subnet.

### NAT (Network Address Translation)

A technology that translates private IP addresses to global IP addresses and vice versa.

```text
Private: 192.168.1.10 ---[NAT]---> Global: 203.0.113.5
```

---

## <a name="container"></a>Container & Virtualization Terms

### Rootful / Rootless

- **Rootful**: A container runtime operating with root privileges.
- **Rootless**: A container runtime operating with regular user privileges.

### Network Namespace

A Linux kernel feature that isolates the network stack (interfaces, routing tables, firewall rules) for each process.

### veth (Virtual Ethernet)

A pair of virtual Ethernet cables used to connect different Network Namespaces.

```text
┌─────────────┐     veth      ┌─────────────┐
│ Namespace A │<------------->│ Namespace B │
└─────────────┘   (peer)      └─────────────┘
```

---

## <a name="kubernetes"></a>Kubernetes Terms

### Pod

The smallest deployable unit in Kubernetes. One or more containers that share the same network namespace, storage, and IP address.

### Deployment

A resource that manages the desired state of a set of Pods, handling replication, self-healing, and rolling updates.

### Service

An abstraction that defines a logical set of Pods and a policy by which to access them (Stable IP and DNS).

### ConfigMap / Secret

Resources used to inject configuration data and sensitive information (passwords, keys) into Pods separately from the container image.

### PersistentVolume (PV) / PersistentVolumeClaim (PVC)

K8s storage abstraction. A PV is the actual storage resource, while a PVC is a request for storage by a user/Pod.

### Helm

The package manager for Kubernetes. It uses "Charts" to define, install, and upgrade complex Kubernetes applications.

### Ingress

An API object that manages external access to services in a cluster, typically providing HTTP/HTTPS routing.

---

## <a name="security"></a>Security & Zero Trust Terms

### Zero Trust Architecture (ZTA)

A security model based on the principle of "Never Trust, Always Verify," where every access request is authenticated and authorized regardless of origin.

### mTLS (Mutual TLS)

A process in which both the client and server prove their identities using digital certificates before establishing a secure connection.

### SPIFFE / SPIRE

- **SPIFFE**: Secure Production Identity Framework for Everyone. A set of open standards for identifying software services.
- **SPIRE**: SPIFFE Runtime Environment. An implementation of the SPIFFE APIs that issues identities to workloads.

### SVID (SPIFFE Verifiable Identity Document)

A document (usually an X.509 certificate) used by a workload to prove its identity to other workloads.

### RBAC (Role-Based Access Control)

A method of regulating access to computer or network resources based on the roles of individual users within an enterprise.

### NetworkPolicy

A K8s resource that controls the flow of traffic between Pods at the network level (L3/L4).

### TLS / SSL (Transport Layer Security / Secure Sockets Layer)

A protocol that encrypts communication between two endpoints and verifies the server's identity with a certificate. SSL is the deprecated predecessor; TLS is its modern successor.

```text
https:// → HTTP over TLS (default port 443)
```

### CA (Certificate Authority)

A trusted entity that issues digital certificates binding public keys to identities such as domains or organizations.

```text
Chain of trust: Root CA → Intermediate CA → Server certificate
```

### Secret Management

The practice of storing and distributing credentials (API keys, passwords, certificates) centrally instead of embedding them in source code or container images.

### Vault

HashiCorp's open-source tool for secret management. It stores secrets in encrypted form and can issue short-lived, dynamically generated credentials.

### KV Engine (KV Secrets Engine)

A Vault engine that stores secrets as simple key-value pairs. Version 2 (KV v2) adds versioning and soft delete.

### Presigned URL

A time-limited URL with embedded credentials that grants temporary download or upload access to a specific object (e.g., in S3 or MinIO).

---

## <a name="mesh"></a>Service Mesh Terms

### Service Mesh

A dedicated infrastructure layer for managing service-to-service communication, typically providing observability, security, and reliability.

### Sidecar Proxy (Envoy)

A proxy instance deployed alongside an application container within the same Pod. It intercepts and manages all incoming and outgoing traffic.

### Istio

An open-source service mesh that provides a uniform way to secure, connect, and monitor microservices.

### Control Plane / Data Plane

- **Control Plane**: Manages and configures proxies (e.g., Istiod).
- **Data Plane**: Handles actual traffic between services (e.g., Envoy proxies).

---

## <a name="oauth2"></a>OAuth2 & Identity Terms

### OAuth2

A standard protocol for authorization. It allows third-party applications to grant access to resources without knowing the user's password.

### <a name="access-token"></a>Access Token (JWT)

A token used to prove the right to access specific resources. JSON Web Token (JWT) is a common format.

### <a name="resource-server"></a>Resource Server

The server that hosts protected resources and accepts requests authenticated with an access token issued by the authorization server (e.g., a REST API returning user data).

### Entra ID (Azure AD)

Microsoft's cloud-based identity and access management service.

---

## <a name="mq"></a>Message Queue Terms

### Pub/Sub (Publish/Subscribe)

An asynchronous communication pattern where a sender (Publisher) sends messages and receivers (Subscribers) subscribe to them.

### Exchange (RabbitMQ)

A component that routes messages to queues based on rules (Direct, Fanout, Topic, Headers).

| Type    | Description                         |
| :------ | :---------------------------------- |
| Direct  | Routing key must match exactly      |
| Fanout  | Delivers to all queues              |
| Topic   | Wildcard pattern matching           |
| Headers | Matching based on header attributes |

### Routing Key

A key used to specify the destination of a message. Typically in a dot-separated format.

```text
Example: market.btc.usd
```

### Wildcards

Pattern matching characters used in Topic Exchanges.

| Character | Description                | Example                                 |
| :-------- | :------------------------- | :-------------------------------------- |
| `*`       | Matches exactly one word   | `market.*.jpy` matches `market.btc.jpy` |
| `#`       | Matches zero or more words | `market.#` matches `market.btc.usd`     |

---

## <a name="cache"></a>Caching Terms

### Sorted Set (ZSET)

A Redis data structure. A set of members sorted by score, ideal for leaderboards.

```text
ZADD game_leaderboard 100 player1
ZADD game_leaderboard 250 player2
ZREVRANGE game_leaderboard 0 9  # Retrieve top 10 players
```

### Write-Through / Cache-Aside

- **Write-Through**: Data is written to the cache and the backend database simultaneously.
- **Cache-Aside**: The application first checks the cache. If data is missing (cache miss), it reads from the database and updates the cache.

### Sets

A set of unique members with no duplicates.

```text
SADD banned_users player2
SISMEMBER banned_users player2  # 1 (true)
```

---

## <a name="storage"></a>Storage Terms

### iSCSI (Internet Small Computer Systems Interface)

A protocol that carries SCSI block-storage commands over IP networks, letting a host use remote storage as if it were a local disk.

### LUN (Logical Unit Number)

An identifier for a logical unit of storage exported by a storage system. An initiator (client) accesses a LUN as a block device.

### Object Storage

A storage model that manages data as objects (data + metadata + unique ID) instead of files or blocks, suited to unstructured data. Examples: Amazon S3, MinIO.

### Bucket

A container for objects in object storage. Each object is addressed by its bucket name and key (e.g., `my-bucket/photos/1.png`).

---

## <a name="load-balancer"></a>Load Balancer Terms

### Load Balancer

A device or program that distributes traffic across servers to improve availability and scalability. The load balancer itself also needs redundancy to avoid becoming a single point of failure.

### VIP (Virtual IP)

A virtual IP address used as the stable client-facing endpoint of a load balancer.

### DNAT (Destination NAT)

Rewrites the destination address, such as translating a VIP to a backend server address.

```text
192.168.10.100:80 (VIP) → 192.168.20.10:80 (Backend)
```

### SNAT (Source NAT)

Rewrites the source address so that backend responses return through the intended router or load balancer.

### Round Robin

Assigns requests to servers in turn.

### Least Connections

Assigns new connections to the server with the fewest active connections.

### ipvs (IP Virtual Server)

Linux kernel facilities for transport-layer load balancing.

### DSR (Direct Server Return)

Backends send replies directly to clients. The load balancer forwards requests but does not process the return traffic.

---

## <a name="rate-limiting"></a>Rate Limiting Terms

### Rate Limiting

Restricts the request rate to prevent overload and share capacity fairly.

### Fixed Window

Counts requests in fixed time intervals. It is simple, but traffic can spike at window boundaries.

### Sliding Window

Measures requests over a moving interval. A sliding log tracks individual requests; a weighted counter approximates the count using adjacent windows.

### Token Bucket

Consumes tokens from a bucket refilled at a configured rate, allowing bounded bursts while limiting the sustained rate.

```text
[Capacity: 10] [Refill rate: 1/second]
Each request consumes one token.
```

### Throttling

Limits the frequency or rate of operations, such as bandwidth usage or API calls.

---

## <a name="software"></a><a name="idempotency"></a>Idempotency Terms

### Idempotency

Repeating an operation has the same intended effect as performing it once.

```text
x = 5  (idempotent)
x++    (not idempotent)
```

### Idempotency Key

A client-generated identifier reused on retries so the server can detect duplicate operations.

```text
Header: Idempotency-Key: uuid-v4
```

### Retry

Attempts a failed operation again. Repeating non-idempotent operations may duplicate side effects.

### Dead Letter Queue

Stores messages that could not be processed for later inspection or controlled retries.

---

## <a name="protocol"></a>Protocol Terms

### TCP/IP Stack

A layered implementation of internet communication protocols centered around TCP and IP.

### Encapsulation

The process of wrapping data from a higher layer as the payload of a lower layer.

```text
[ICMP] → [IPv4 [ICMP]] → [Ethernet [IPv4 [ICMP]]]
```

### Raw Socket

A socket that bypasses the OS network stack to send and receive raw packets.

### Keep-Alive

An HTTP mechanism that reuses a single TCP connection for multiple requests instead of opening a new connection for each one.

### Multiplexing

Carrying multiple independent request streams over a single connection at the same time, as HTTP/2 does within one TCP connection.

### HoL (Head-of-Line) Blocking

A stall that occurs when data queued earlier blocks data queued later on the same connection. With HTTP/2 over TCP, a single lost packet delays all multiplexed streams.

---

## <a name="architecture"></a>Architecture & IaC Terms

### Clean Architecture

A software design pattern that separates concerns into concentric layers (Domain, UseCase, Infra, Framework) with dependencies pointing inwards.

```text
Framework → UseCase → Domain ← Infra
            (depends on)
```

### IaC (Infrastructure as Code)

The practice of managing infrastructure (servers, networks, cloud resources) through declarative code files instead of manual operations, enabling review, versioning, and reproducibility.

### Terraform

HashiCorp's open-source IaC tool. It reads configuration files, computes a plan to reach the desired state, and applies it to real infrastructure.

### Provider (Terraform)

A Terraform plugin that manages the resources of a specific platform (e.g., AWS, Google Cloud, Kubernetes) via its API.

### Service Discovery

The mechanism by which services locate each other's addresses at runtime, typically via DNS (e.g., Kubernetes Services) or a registry (e.g., Consul).

### Sharding

Splitting a large dataset across multiple independent databases (shards) based on a key, to scale writes and storage horizontally.

### Consistent Hashing

A key-distribution technique in which adding or removing a node remaps only a small fraction of keys, minimizing data movement during resharding.

---

## <a name="go"></a>Go Language Terms

### Context (context.Context)

The standard Go pattern for propagating cancellation signals, timeouts, and deadlines.

### Interface

A type that defines a set of methods. The principle is "Accept interfaces, return structs."

### CGO

A mechanism for calling C code from Go.
