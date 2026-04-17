# Infrastructure Workshop Glossary

This document provides concise explanations of technical terms used across various workshops.

## <a name="network"></a>Networking Terms

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

### IP Address

Internet Protocol Address. An address used to identify devices on a network. IPv4 is 32-bit, and IPv6 is 128-bit.

### Subnet Mask

Indicates which part of an IP address is the network portion and which is the host portion.

### Default Gateway

The IP address of the router that forwards packets to destinations outside the local subnet.

### NAT (Network Address Translation)

A technology that translates private IP addresses to global IP addresses and vice versa.

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

A component that routes messages to queues based on rules (Direct, Fanout, Topic, Headers).

---

## Caching Terms

### Sorted Set (ZSET)

A Redis data structure. A set of members sorted by score, ideal for leaderboards.

### Write-Through / Cache-Aside

- **Write-Through**: Data is written to the cache and the backend database simultaneously.
- **Cache-Aside**: The application first checks the cache. If data is missing (cache miss), it reads from the database and updates the cache.

---

## Protocol Terms

### TCP/IP Stack

A layered implementation of internet communication protocols centered around TCP and IP.

### Raw Socket

A socket that bypasses the OS network stack to send and receive raw packets.

---

## Go Language Terms

### Clean Architecture

A software design pattern that separates concerns into concentric layers (Domain, UseCase, Infra, Framework).

### CGO

A mechanism for calling C code from Go.
