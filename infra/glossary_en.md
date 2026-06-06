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

An API object that manages external access to services in a cluster, typically providing HTTP/HTTPS routing

```text
┌─────────────┐     veth      ┌─────────────┐
│ Namespace A │<------------->│ Namespace B │
└─────────────┘   (peer)      └─────────────┘
```

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

## OAuth2 & Identity Terms

### OAuth2

A standard protocol for authorization. It allows third-party applications to grant access to resources without knowing the user's password.

### Access Token (JWT)

A token used to prove the right to access specific resources. JSON Web Token (JWT) is a common format.

### Entra ID (Azure AD)

Microsoft's cloud-based identity and access management service.

---

## Message Queue Terms

### Pub/Sub (Publish/Subscribe)

An asynchronous communication pattern where a sender (Publisher) sends messages and receivers (Subscribers) subscribe to them.

### Exchange (RabbitMQ)

A component that routes messages to queues based on rules (Direct, Fanout, Topic, Headers)

A component that handles message routing. RabbitMQ has several types.

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

## Caching Terms

### Sorted Set (ZSET)

A Redis data structure. A set of members sorted by score, ideal for leaderboards.

### Write-Through / Cache-Aside

- **Write-Through**: Data is written to the cache and the backend database simultaneously.
- **Cache-Aside**: The application first checks the cache. If data is missing (cache miss), it reads from the database and updates the cache.

```text
ZADD game_leaderboard 100 player1
ZADD game_leaderboard 250 player2
ZREVRANGE game_leaderboard 0 9  # Retrieve top 10 players
```

### Sets

A set of unique members with no duplicates.

```text
SADD banned_users player2
SISMEMBER banned_users player2  # 1 (true)
```

---

## Protocol Terms

### TCP/IP Stack

A layered implementation of internet communication protocols centered around TCP and IP.

### Encapsulation

The process of wrapping data from a higher layer as the payload of a lower layer.

```text
[ICMP] → [IPv4 [ICMP]] → [Ethernet [IPv4 [ICMP]]]
```

### Raw Socket

A socket that bypasses the OS network stack to send and receive raw packets.

---

## Go Language Terms

### Clean Architecture

A software design pattern that separates concerns into concentric layers (Domain, UseCase, Infra, Framework)

A design pattern where dependencies point inwards.

```text
Framework → UseCase → Domain ← Infra
            (depends on)
```

### Context (context.Context)

The standard Go pattern for propagating cancellation signals, timeouts, and deadlines.

### Interface

A type that defines a set of methods. The principle is "Accept interfaces, return structs."

### CGO

A mechanism for calling C code from Go.
