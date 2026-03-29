# Infrastructure Workshop Glossary

This document provides concise explanations of technical terms used across various workshops.

## Networking Terms

### L1 / L2 / L3 (Layer 1/2/3)

Refers to the layers of the OSI reference model or the TCP/IP model.

| Layer | Name | Key Protocols/Devices | Description |
| :--- | :--- | :--- | :--- |
| L1 | Physical Layer | Cables, NICs | Transmission and reception of electrical signals |
| L2 | Data Link Layer | Ethernet, MAC address | Communication within the same segment |
| L3 | Network Layer | IP, Router | Routing between different networks |
| L4 | Transport Layer | TCP, UDP | End-to-end communication control |

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

## Container & Virtualization Terms

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

## Security Terms

### TLS/SSL

Transport Layer Security / Secure Sockets Layer. Protocols used to encrypt communication. TLS is the current standard.

### CA (Certificate Authority)

A trusted third-party entity that issues digital certificates.

### Certificate Chain

A hierarchical structure: Root CA → Intermediate CA → Server Certificate.

### SAN (Subject Alternative Name)

An extension that specifies valid domain names for a certificate. Modern browsers prioritize SAN over CN.

### CSR (Certificate Signing Request)

A file containing a server's public key and information, used to request a signature from a CA.

### OAuth2

A standard protocol for authorization. It allows third-party applications to grant access to resources without knowing the user's password.

### Access Token

A token used to prove the right to access specific resources to a resource server. Typically in JWT (JSON Web Token) format.

### Resource Server

A server that holds protected user resources. It validates access tokens and provides resources for legitimate requests.

---

## Message Queue Terms

### Pub/Sub (Publish/Subscribe)

An asynchronous communication pattern where a sender (Publisher) sends messages and receivers (Subscribers) subscribe to them.

### Exchange

A component that handles message routing. RabbitMQ has several types.

| Type | Description |
| :--- | :--- |
| Direct | Routing key must match exactly |
| Fanout | Delivers to all queues |
| Topic | Wildcard pattern matching |
| Headers | Matching based on header attributes |

### Routing Key

A key used to specify the destination of a message. Typically in a dot-separated format.

```text
Example: market.btc.usd
```

### Wildcards

Pattern matching characters used in Topic Exchanges.

| Character | Description | Example |
| :--- | :--- | :--- |
| `*` | Matches exactly one word | `market.*.jpy` matches `market.btc.jpy` |
| `#` | Matches zero or more words | `market.#` matches `market.btc.usd` |

---

## Caching Terms

### Sorted Set (ZSET)

A Redis data structure. A set of members sorted by score.

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

### Promiscuous Mode

A mode where all packets are received, even those not addressed to the device. Used by packet analysis tools.

### Checksum

A verification value used to detect data corruption. RFC 1071 uses the "1's complement of the 1's complement sum."

### MTU (Maximum Transmission Unit)

The maximum data size that can be sent in one transmission. Typically 1500 bytes for Ethernet.

---

## Podman Terms

### Podman

A container engine compatible with Docker. It is daemonless and capable of rootless execution.

### Pod

A group of one or more containers that share the same network namespace.

---

## Go Language Terms

### Clean Architecture

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

A mechanism for calling C code from Go. Enabled by `import "C"`.

### unsafe.Pointer

A pointer type that bypasses Go's type system constraints, used for interoperation with C.
