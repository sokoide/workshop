# Workshop Repository

---

🌐 Available languages:
[English](./README.md) | [日本語](./README_ja.md)

---

This repository provides hands-on technical workshops covering infrastructure and software architecture topics.

This collection is aimed at software engineers. Each workshop is self-contained — follow the
recommended order within each difficulty level below, or start directly with any topic that interests
you. Difficulty labels come from the 🎯 Level marker in each document, and known prerequisite
workshops are noted inline.

**Primary Language:** Go (the required version varies by workshop; check each
`go.mod`)
**Architecture Pattern:** [Clean Architecture (3-Layer Variant)](./software/clean_arch.md)

---

## Choose a Learning Path by Goal

Prepare each exercise from the repository root. Check its prerequisites for tools and privileges. Use at least the version in the asset's go.mod; automatic toolchain download with older Go also requires network access and suitable GOTOOLCHAIN settings.

| Goal                         | Prerequisites and environment                           | First result and path                                                                                                                                                            |
| ---------------------------- | ------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Understand Go design         | Basic Go, Go 1.23+, Python 3, curl; no containers       | Change a greeting app in [WS8](software/clean_arch_ws8_en.md) → [guide](software/clean_arch_en.md) → [WS1](software/clean_arch_ws1_en.md) → [WS2](software/clean_arch_ws2_en.md) |
| Use external services        | Basic Go, Podman, workshop-specific Go version          | Display a leaderboard in [Redis](infra/cache_en.md) → [RabbitMQ](infra/rabbitmq_en.md) → [idempotency](software/idempotency_en.md)                                               |
| Use Kubernetes               | Container basics; kind, kubectl, Helm; Linux/macOS/WSL2 | Publish an app through 3 sessions in [basics](infra/kubernetes_basics_en.md) → [operations](infra/kubernetes_operations_en.md)                                                   |
| Inspect networking internals | Linux, sudo, C compiler, Go; independent advanced path  | Receive frames in [TCP/IP Part 1](infra/tcpip_stack_en.md) → [UDP and integration](infra/tcpip_stack_udp_en.md)                                                                  |

Prepare BBS WS3–WS7 as separate copies with the [start-state tool](tools/prepare_bbs.py). WS3's gRPC code need not carry into WS4. Catalogs can be read directly at the needed pattern.

## Contents

Each section below is grouped by difficulty and listed in the recommended order
within its level. Start with WS8 in the Software Architecture Beginner section.

### Infrastructure

#### Beginner

- [Rate Limiting Workshop: Request Throttling with Redis](./infra/rate_limiting_en.md) — **Beginner**
  - Implement Fixed Window, Sliding Window, and Token Bucket algorithms with Redis.

#### Intermediate

- [Redis Workshop: Real-time Game Leaderboard with Sorted Sets](./infra/cache_en.md) — **Intermediate**
  - Build a high-performance leaderboard using Redis Sorted Sets and manage banned users with Sets.
- [TLS/SSL Certificate Workshop: Building Self-Signed CA and Understanding Certificate Chains](./infra/tls_en.md) — **Intermediate**
  - Step-by-step guide from CA construction to certificate verification using OpenSSL.
- [Reverse Proxy Workshop: Understanding K8s Ingress through Traefik](./infra/traefik_en.md) — **Intermediate**
  - SSL termination, routing, and health checks with Traefik + multiple backend containers. Recommended: the [TLS workshop](./infra/tls_en.md) if you plan to add HTTPS afterwards.
- [Podman Workshop: Creating L3 Isolated Networks with macvlan + Router Containers](./infra/vlan_en.md) — **Intermediate**
  - Create macvlan networks on VLAN sub-interfaces and perform NAT/forwarding with router containers.
- [CoreDNS Workshop: Building Parent-Child DNS Servers to Understand Name Resolution](./infra/coredns_en.md) — **Intermediate**
  - Build authoritative servers and forwarding using two VMs to experience the zone delegation mechanism. Recommended: [TCP/IP Stack Part 2](./infra/tcpip_stack_udp_en.md) to see how DNS queries travel as UDP datagrams.
- [Load Balancer Workshop: Building a Virtual LB with iptables and IPVS](./infra/load_balancer_en.md) — **Intermediate**
  - Learn load balancing algorithms, NAT/DSR modes, and health checks using iptables and IPVS. Optional background deep dive: the [TCP/IP stack workshop](./infra/tcpip_stack_en.md) (NAT/DNAT fundamentals).
- [HTTP Persistent Connection Workshop: Keep-Alive, WebSocket, HTTP/2, gRPC, HTTP/3](./infra/http_persistent_en.md) — **Intermediate**
  - Comparison of connection reuse and 0-RTT mechanisms from HTTP/1.1 to the latest HTTP/3 (QUIC). Optional background deep dive: the [TCP/IP stack workshop](./infra/tcpip_stack_en.md) (TCP handshakes and connection lifecycle).
- [Kubernetes Workshop: From Basics to Practical Container Orchestration](./infra/kubernetes_basics_en.md) — **Intermediate**
  - Comprehensive guide covering Pod, Deployment, Service, ConfigMap, Helm, and more. Continue with the [Kubernetes Operations workshop](./infra/kubernetes_operations_en.md) afterwards.
- [Object Storage Workshop: Learning S3-Compatible API with MinIO](./infra/minio_en.md) — **Intermediate**
  - MinIO construction + bucket operations and signed URL generation using AWS CLI/SDK.
- [Secret Management Workshop: Secure API Key Management with HashiCorp Vault](./infra/secret_management_en.md) — **Intermediate**
  - Centralized secret management, versioning, and Clean Architecture integration using Vault KV v2.
- [Message Queue Workshop: Learning Pub/Sub and Queuing with RabbitMQ](./infra/rabbitmq_en.md) — **Intermediate**
  - Build a real-time crypto monitoring system using RabbitMQ Topic Exchange.
- [Service Mesh Fundamentals: Understanding L7 Traffic Control with Envoy Sidecars](./infra/envoy_en.md) — **Intermediate**
  - Manually deploy Envoy proxies for traffic control (retries, timeouts, routing). Recommended: the [Traefik workshop](./infra/traefik_en.md) first (edge routing before sidecar-based internal traffic control).
- [OAuth2 Workshop: Learning Authorization Flow with Keycloak + Go + Podman](./infra/oauth2_en.md) — **Intermediate**
  - Issue tokens with Keycloak and implement protected APIs by validating JWTs in a Go Resource Server.
- [Web Security Workshop (2 parts): XSS, CSRF, and Clickjacking](./infra/web_security_en.md) — **Intermediate**
  - Experience common web attacks using an intentionally vulnerable application. Complete Part 1 before Part 2 ([Defense](./infra/web_security_defense_en.md)), which applies each countermeasure to a copy of the victim app and re-runs the attacks to verify they are blocked.
- [Supply Chain Security Workshop: Dependency and Container Image Vulnerability Management](./infra/supply_chain_security_en.md) — **Intermediate**
  - Detect vulnerabilities with govulncheck/Trivy, generate SBOMs with syft/grype, and sign images with cosign. Recommended first: the [Web Security workshop](./infra/web_security_en.md) for the attacker's perspective.
- [Terraform Workshop: From Basics to Practical DNS Server Construction](./infra/terraform_en.md) — **Intermediate**
  - Learn the basics with the local provider and manage CoreDNS environments as IaC using Docker/Podman.

#### Advanced

- [Container Runtime Workshop: Understanding the Reality of Containers through namespaces and cgroups](./infra/container_runtime_en.md) — **Advanced**
  - Replicate the magic behind `podman run` by manually creating namespaces/cgroups.
- [TCP/IP Protocol Stack Workshop (2 parts): Building a Network Stack from Scratch](./infra/tcpip_stack_en.md) — **Advanced**
  - Implement Ethernet/IP/ICMP with Go and C (CGO), understanding packets from raw socket. Take Part 1 before Part 2 ([UDP & Stack Integration](./infra/tcpip_stack_udp_en.md)), which adds UDP and the integrated stack.
- [K8s Service (LoadBalancer) Workshop: Virtual Load Balancer built with iptables](./infra/k8s_lb_en.md) — **Advanced**
  - Replicate the operating principles of MetalLB and kube-proxy using VIP addition + DNAT/SNAT. Prerequisite: the [Podman VLAN workshop](./infra/vlan_en.md) (podman networks, router container, and VLAN 10/20 setup are reused).
- [Zero Trust Architecture Workshop: Implementing "Always Verify" with mTLS and Service Mesh](./infra/zero_trust_en.md) — **Advanced**
  - Implement identity-based security (mTLS/AuthZ) using Istio and SPIRE, moving beyond network perimeters. Recommended first: the [Kubernetes Basics](./infra/kubernetes_basics_en.md) and [TLS](./infra/tls_en.md) workshops.
- [iSCSI and Kubernetes PersistentVolume Workshop](./infra/iscsi_pv_en.md) — **Advanced**
  - Build iSCSI targets/initiators and verify mounting/persistence with K8s PV/PVC. Prerequisite: the [Kubernetes Basics workshop](./infra/kubernetes_basics_en.md) (PV/PVC concepts are used from STEP 6).
- [Kubernetes Operations & Security Workshop: Advanced Features for Production](./infra/kubernetes_operations_en.md) — **Advanced**
  - Hands-on with RBAC, NetworkPolicy, HPA, GitOps (ArgoCD), Backup (Velero), and more. Prerequisite: complete the [Kubernetes Basics workshop](./infra/kubernetes_basics_en.md) first.
- [Kerberos / SPNEGO Authentication Workshop: Single Sign-On with NGINX and Podman](./infra/kerberos_en.md) — **Advanced**
  - Experience passwordless authentication flow with KDC construction and NGINX SPNEGO module.
- [OAuth2 Workshop: Learning Authorization Flows with Microsoft Entra ID and Go](./infra/oauth2_entra_en.md) — **Advanced**
  - Learn both managed identity based M2M access and user-delegated API access with Microsoft Entra ID. Recommended first: the [OAuth2 (Keycloak) workshop](./infra/oauth2_en.md), which covers the same flows on a local Keycloak.
- [Sharding Fundamentals: Sparse Table, Segment Tree, Elasticsearch, and Cortex](./infra/sharding_en.md) — **Advanced**
  - Practical sharding design with Elasticsearch and Cortex examples, including a summary of Cortex PRs #7266 and #7270. Recommended: the [Redis caching workshop](./infra/cache_en.md) (storage-layer scaling patterns).

#### Reference

- [Infrastructure Workshop Glossary](./infra/glossary_en.md)
  - Concise explanations of technical terms used across workshops (networking, containers, K8s, security, etc.).

#### Infrastructure: Contents planned for the future

- Observability Workshop: Metric Collection and Visualization with Prometheus + Grafana
  - Metric collection and dashboard construction with Node Exporter + Prometheus + Grafana.

### Software Architecture

#### Beginner

- [Clean Architecture Workshop (WS8): Understanding the Essentials with a Minimal Example](./software/clean_arch_ws8_en.md) — **Beginner**
  - Trace the dependency rule through a small greeting application, then practice replacing business and infrastructure details. **Start here**: no prior Clean Architecture knowledge is required. After WS8, read the [Clean Architecture guide](./software/clean_arch_en.md) and move on to WS1.
- [Clean Architecture (3-Layer Variant: Adapters / UseCases / Domain)](./software/clean_arch_en.md)
  - Visual explanation of dependencies and responsibilities across Domain/UseCase/Adapters. Read between WS8 and WS1.
- [Clean Architecture Workshop (WS1): Learning Design Resilient to Change](./software/clean_arch_ws1_en.md) — **Beginner**
  - Hands-on implementation of Entity/Domain Service and swapping infrastructure from DB to AD. Recommended first: WS8 and the Clean Architecture guide above.
- [SOLID Principles (with Go examples)](./software/solid_en.md) — **Beginner**
  - Learn S/O/L/I/D with practical Go examples and Clean Architecture mapping.

#### Intermediate

- [Clean Architecture Workshop (WS2): Extension and Optimization](./software/clean_arch_ws2_en.md) — **Intermediate**
  - Abstraction of notification channels and introduction of transparent caching using the Decorator pattern. Prerequisite: [WS1](./software/clean_arch_ws1_en.md).
- [Clean Architecture Workshop (WS3): Swapping Communication Protocols](./software/clean_arch_ws3_en.md) — **Intermediate**
  - Migrate a BBS REST API to gRPC by modifying the Presentation Adapter and Composition Root. Prerequisite: [WS1](./software/clean_arch_ws1_en.md) and [WS2](./software/clean_arch_ws2_en.md) (or comfort with ports and the Composition Root).
- [Clean Architecture Workshop (WS4): Adding Business Rules](./software/clean_arch_ws4_en.md) — **Intermediate**
  - Add a new business rule and observe changes propagating from inside to outside. Prerequisite: understanding [WS1](./software/clean_arch_ws1_en.md)/[WS2](./software/clean_arch_ws2_en.md); WS3 is optional preparation.
- [Clean Architecture Workshop (WS5): Swapping the Persistence Layer](./software/clean_arch_ws5_en.md) — **Intermediate**
  - Migrate from SQLite to PostgreSQL by modifying the Infra Adapter and Composition Root. Prerequisite: [WS4](./software/clean_arch_ws4_en.md) — prepare a fresh copy equivalent to completed WS4.
- [Clean Architecture Workshop (WS6): Integrating External Services](./software/clean_arch_ws6_en.md) — **Intermediate**
  - Add Slack notifications using the new Port/Adapter pattern. Prerequisite: [WS5](./software/clean_arch_ws5_en.md) — prepare a fresh copy equivalent to completed WS5.
- [Clean Architecture Workshop (WS7): Adding Authentication](./software/clean_arch_ws7_en.md) — **Intermediate**
  - Add JWT Bearer Token authentication as a cross-cutting concern in the Presentation Adapter. Prerequisite: [WS1](./software/clean_arch_ws1_en.md)–[WS4](./software/clean_arch_ws4_en.md) (or the BBS project builds and tests pass).
- [Idempotency Pattern: Designing Safe Retries](./software/idempotency_en.md) — **Intermediate**
  - Prevent duplicate operations with idempotency keys and examine concurrency, expiration, and failure-handling trade-offs. Pairs well with the [RabbitMQ workshop](./infra/rabbitmq_en.md) (retry-safe consumers).
- [Caching Patterns in Go](./software/go_cache_patterns_en.md) — **Intermediate**
  - Practical cache patterns: TTL, LRU, cache-aside, Redis, and write-through. Reference catalog — jump straight to the pattern you need.
- [Message Queue Patterns in Go](./software/go_mq_patterns_en.md) — **Intermediate**
  - Compare channels, fan-out, worker pools, Kafka, and retry/dead-letter queue patterns with runnable Go examples. Reference catalog — jump straight to the pattern you need.
- [Secure Coding Best Practices (Go)](./software/secure_coding_en.md) — **Intermediate**
  - Prevent SQL injection, path traversal, password hashing mistakes, race conditions, and information leakage in everyday coding.

#### Advanced

- [Introduction to Succinct Data Structures (Go Perspective)](./software/succinct_en.md) — **Advanced**
  - Guide to memory-efficient Trie implementation using LOUDS.

#### Reference

- [Clean Architecture Reference (Simplified)](./software/clean_arch.md)
  - Condensed architecture diagram and key principles — ideal as a quick reference.
- [Design Patterns](https://github.com/sokoide/design-patterns/README.md)
  - Implementation examples and application scenarios of GoF design patterns.

## Maintenance

Install the Markdown tools with `pnpm install` or `npm install` (Python 3.9+ is also required).
Run `make format` to apply Prettier, markdownlint, and textlint using the same file list;
`CLAUDE.md`, dependency caches, and tool metadata are excluded. Tool failures stop the command.
Run `make test-tools` for maintenance script tests and `make check-headings` to compare
heading level order in English/Japanese pairs. The latter exits nonzero for mismatches
or read errors; it does not verify translation accuracy or detect missing counterpart files.

Run `make check-go` to build, vet, and race-test all Go modules. Logs and skipped-test details are saved in `.cache/go-validation/`. Use `make check-go GO_CHECK_FLAGS=--require-integration` to fail on skipped tests or missing Linux-only checks. Raw socket execution requires Linux with CGO; RabbitMQ and Vault integration tests require their services.

`make check-docs` also checks Go requirements against assets. `make check-bbs-states` verifies BBS start/reference states and is included in `make check`.
