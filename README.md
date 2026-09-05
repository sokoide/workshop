# Workshop Repository

---

🌐 Available languages:
[English](./README.md) | [日本語](./README_ja.md)

---

This repository provides hands-on technical workshops covering infrastructure and software architecture topics.

**Primary Language:** Go (the required version varies by workshop; check each
`go.mod`)
**Architecture Pattern:** [Clean Architecture (3-Layer Variant)](./software/clean_arch.md)

---

## Maintenance

Install the Markdown tools with `pnpm install` or `npm install` (Python 3.9+ is also required).
Run `make format` to apply Prettier, markdownlint, and textlint using the same file list;
`CLAUDE.md`, dependency caches, and tool metadata are excluded. Tool failures stop the command.
Run `make test-tools` for maintenance script tests and `make check-headings` to compare
heading level order in English/Japanese pairs. The latter exits nonzero for mismatches
or read errors; it does not verify translation accuracy or detect missing counterpart files.

Run `make check-go` to build, vet, and race-test all Go modules. Logs and skipped-test details are saved in `.cache/go-validation/`. Use `make check-go GO_CHECK_FLAGS=--require-integration` to fail on skipped tests or missing Linux-only checks. Raw socket execution requires Linux with CGO; RabbitMQ and Vault integration tests require their services.

## Contents

### Infrastructure

- [CoreDNS Workshop: Building Parent-Child DNS Servers to Understand Name Resolution](./infra/coredns_en.md)
  - Build authoritative servers and forwarding using two VMs to experience the zone delegation mechanism.
- [Podman Workshop: Creating L3 Isolated Networks with macvlan + Router Containers](./infra/vlan_en.md)
  - Create macvlan networks on VLAN sub-interfaces and perform NAT/forwarding with router containers.
- [K8s Service (LoadBalancer) Workshop: Virtual Load Balancer built with iptables](./infra/k8s_lb_en.md)
  - Replicate the operating principles of MetalLB and kube-proxy using VIP addition + DNAT/SNAT.
- [Load Balancer Workshop: Building a Virtual LB with iptables and IPVS](./infra/load_balancer_en.md)
  - Learn load balancing algorithms, NAT/DSR modes, and health checks using iptables and IPVS.
- [iSCSI and Kubernetes PersistentVolume Workshop](./infra/iscsi_pv_en.md)
  - Build iSCSI targets/initiators and verify mounting/persistence with K8s PV/PVC.
- [Terraform Workshop: From Basics to Practical DNS Server Construction](./infra/terraform_en.md)
  - Learn the basics with the local provider and manage CoreDNS environments as IaC using Docker/Podman.
- [TLS/SSL Certificate Workshop: Building Self-Signed CA and Understanding Certificate Chains](./infra/tls_en.md)
  - Step-by-step guide from CA construction to certificate verification using OpenSSL.
- [Container Runtime Workshop: Understanding the Reality of Containers through namespaces and cgroups](./infra/container_runtime_en.md)
  - Replicate the magic behind `podman run` by manually creating namespaces/cgroups.
- [Redis Workshop: Real-time Game Leaderboard with Sorted Sets](./infra/cache_en.md)
  - Build a high-performance leaderboard using Redis Sorted Sets and manage banned users with Sets.
- [Rate Limiting Workshop: Request Throttling with Redis](./infra/rate_limiting_en.md)
  - Implement Fixed Window, Sliding Window, and Token Bucket algorithms with Redis.
- [Message Queue Workshop: Learning Pub/Sub and Queuing with RabbitMQ](./infra/rabbitmq_en.md)
  - Build a real-time crypto monitoring system using RabbitMQ Topic Exchange.
- [Secret Management Workshop: Secure API Key Management with HashiCorp Vault](./infra/secret_management_en.md)
  - Centralized secret management, versioning, and Clean Architecture integration using Vault KV v2.
- [Kerberos / SPNEGO Authentication Workshop: Single Sign-On with NGINX and Podman](./infra/kerberos_en.md)
  - Experience passwordless authentication flow with KDC construction and NGINX SPNEGO module.
- [OAuth2 Workshop: Learning Authorization Flow with Keycloak + Go + Podman](./infra/oauth2_en.md)
  - Issue tokens with Keycloak and implement protected APIs by validating JWTs in a Go Resource Server.
- [OAuth2 Workshop: Learning Authorization Flows with Microsoft Entra ID and Go](./infra/oauth2_entra_en.md)
  - Learn both managed identity based M2M access and user-delegated API access with Microsoft Entra ID.
- [Reverse Proxy Workshop: Understanding K8s Ingress through Traefik](./infra/traefik_en.md)
  - SSL termination, routing, and health checks with Traefik + multiple backend containers.
- [Service Mesh Fundamentals: Understanding L7 Traffic Control with Envoy Sidecars](./infra/envoy_en.md)
  - Manually deploy Envoy proxies for traffic control (retries, timeouts, routing).
- [Object Storage Workshop: Learning S3-Compatible API with MinIO](./infra/minio_en.md)
  - MinIO construction + bucket operations and signed URL generation using AWS CLI/SDK.
- [HTTP Persistent Connection Workshop: Keep-Alive, WebSocket, HTTP/2, gRPC, HTTP/3](./infra/http_persistent_en.md)
  - Comparison of connection reuse and 0-RTT mechanisms from HTTP/1.1 to the latest HTTP/3 (QUIC).
- [Sharding Fundamentals: Sparse Table, Segment Tree, Elasticsearch, and Cortex](./infra/sharding_en.md)
  - Practical sharding design with Elasticsearch and Cortex examples, including a summary of Cortex PRs #7266 and #7270.
- [TCP/IP Protocol Stack Workshop: Building a Network Stack from Scratch](./infra/tcpip_stack_en.md)
  - Implement Ethernet/IP/ICMP/UDP with Go and C (CGO), understanding packets from raw socket.
- [Kubernetes Workshop: From Basics to Practical Container Orchestration](./infra/kubernetes_basics_en.md)
  - Comprehensive guide covering Pod, Deployment, Service, ConfigMap, Helm, and more.
- [Kubernetes Operations & Security Workshop: Advanced Features for Production](./infra/kubernetes_operations_en.md)
  - Hands-on with RBAC, NetworkPolicy, HPA, GitOps (ArgoCD), Backup (Velero), and more.
- [Zero Trust Architecture Workshop: Implementing "Always Verify" with mTLS and Service Mesh](./infra/zero_trust_en.md)
  - Implement identity-based security (mTLS/AuthZ) using Istio and SPIRE, moving beyond network perimeters.
- [Web Security Workshop: XSS, CSRF, and Clickjacking](./infra/web_security_en.md)
  - Experience common web attacks and learn defense mechanisms (escaping, tokens, etc.) using an intentionally vulnerable application.
- [Supply Chain Security Workshop: Dependency and Container Image Vulnerability Management](./infra/supply_chain_security_en.md)
  - Detect vulnerabilities with govulncheck/Trivy, generate SBOMs with syft/grype, and sign images with cosign.
- [Infrastructure Workshop Glossary](./infra/glossary_en.md)
  - Concise explanations of technical terms used across workshops (networking, containers, K8s, security, etc.).

#### Infrastructure: Contents planned for the future

- Observability Workshop: Metric Collection and Visualization with Prometheus + Grafana
  - Metric collection and dashboard construction with Node Exporter + Prometheus + Grafana.

### Software Architecture

- [Clean Architecture (3-Layer Variant: Adapters / UseCases / Domain)](./software/clean_arch_en.md)
  - Visual explanation of dependencies and responsibilities across Domain/UseCase/Adapters.
- [Clean Architecture Workshop (WS1): Learning Design Resilient to Change](./software/clean_arch_ws1_en.md)
  - Hands-on implementation of Entity/Domain Service and swapping infrastructure from DB to AD.
- [Clean Architecture Workshop (WS2): Extension and Optimization](./software/clean_arch_ws2_en.md)
  - Abstraction of notification channels and introduction of transparent caching using the Decorator pattern.
- [Clean Architecture Workshop (WS3): Swapping Communication Protocols](./software/clean_arch_ws3_en.md)
  - Migrate a BBS REST API to gRPC by modifying only the Framework layer.
- [Clean Architecture Workshop (WS4): Adding Business Rules](./software/clean_arch_ws4_en.md)
  - Add a new business rule and observe changes propagating from inside to outside.
- [Clean Architecture Workshop (WS5): Swapping the Persistence Layer](./software/clean_arch_ws5_en.md)
  - Migrate from SQLite to PostgreSQL by modifying only the Infra layer.
- [Clean Architecture Workshop (WS6): Integrating External Services](./software/clean_arch_ws6_en.md)
  - Add Slack notifications using the new Port/Adapter pattern.
- [Clean Architecture Workshop (WS7): Adding Authentication](./software/clean_arch_ws7_en.md)
  - Add JWT Bearer Token authentication as a cross-cutting concern in the Framework layer.
- [Clean Architecture Workshop (WS8): Understanding the Essentials with a Minimal Example](./software/clean_arch_ws8_en.md)
  - Trace the dependency rule through a small greeting application, then practice replacing business and infrastructure details.
- [Idempotency Pattern: Designing Safe Retries](./software/idempotency_en.md)
  - Prevent duplicate operations with idempotency keys and examine concurrency, expiration, and failure-handling trade-offs.
- [Message Queue Patterns in Go](./software/go_mq_patterns_en.md)
  - Compare channels, fan-out, worker pools, Kafka, and retry/dead-letter queue patterns with runnable Go examples.
- [SOLID Principles (with Go examples)](./software/solid_en.md)
  - Learn S/O/L/I/D with practical Go examples and Clean Architecture mapping.
- [Secure Coding Best Practices (Go)](./software/secure_coding_en.md)
  - Prevent SQL injection, path traversal, password hashing mistakes, race conditions, and information leakage in everyday coding.
- [Caching Patterns in Go](./software/go_cache_patterns_en.md)
  - Practical cache patterns: TTL, LRU, cache-aside, Redis, and write-through.
- [Introduction to Succinct Data Structures (Go Perspective)](./software/succinct_en.md)
  - Guide to memory-efficient Trie implementation using LOUDS.
- [Clean Architecture Reference (Simplified)](./software/clean_arch.md)
  - Condensed architecture diagram and key principles — ideal as a quick reference.
- [Design Patterns](https://github.com/sokoide/design-patterns/README.md)
  - Implementation examples and application scenarios of GoF design patterns.
