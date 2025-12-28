# Workshop Repository

---

🌐 Available languages:
[English](./README.md) | [日本語](./README_ja.md)

---

This repository contains hands-on materials for various technical topics.

## Contents

### Infrastructure

- [CoreDNS Workshop: Building Parent-Child DNS Servers to Understand Name Resolution](./infra/coredns_en.md)
  - Build authoritative servers and forwarding using two VMs to experience the zone delegation mechanism.
- [Podman Workshop: Creating L3 Isolated Networks with macvlan + Router Containers](./infra/vlan_en.md)
  - Create macvlan networks on VLAN sub-interfaces and perform NAT/forwarding with router containers.
- [K8s Service (LoadBalancer) Workshop: Virtual Load Balancer built with iptables](./infra/k8s_lb_en.md)
  - Replicate the operating principles of MetalLB and kube-proxy using VIP addition + DNAT/SNAT.
- [iSCSI and Kubernetes PersistentVolume Workshop](./infra/iscsi_pv_en.md)
  - Build iSCSI targets/initiators and verify mounting/persistence with K8s PV/PVC.
- [Terraform Workshop: From Basics to Practical DNS Server Construction](./infra/terraform_en.md)
  - Learn the basics with the local provider and manage CoreDNS environments as IaC using Docker/Podman.
- [TLS/SSL Certificate Workshop: Building Self-Signed CA and Understanding Certificate Chains](./infra/tls_en.md)
  - Step-by-step guide from CA construction to certificate verification using OpenSSL.
- Reverse Proxy Workshop: Understanding K8s Ingress through Traefik (Planned)
  - SSL termination, routing, and health checks with Traefik + multiple backend containers.
- Service Mesh Fundamentals: Understanding L7 Traffic Control with Envoy Sidecars (Planned)
  - Manually deploy Envoy proxies for traffic control (retries, timeouts, routing).
- Container Runtime Workshop: Understanding the Reality of Containers through namespaces and cgroups (Planned)
  - Replicate the magic behind `docker run` by manually creating namespaces/cgroups.
- Message Queue Workshop: Learning Pub/Sub and Queuing with RabbitMQ (Planned)
  - Asynchronous messaging with RabbitMQ + Producer/Consumer containers.
- Object Storage Workshop: Learning S3-Compatible API with MinIO (Planned)
  - MinIO construction + bucket operations and signed URL generation using AWS CLI/SDK.
- Observability Workshop: Metric Collection and Visualization with Prometheus + Grafana (Planned)
  - Metric collection and dashboard construction with Node Exporter + Prometheus + Grafana.
- Secret Management Workshop: Dynamic Secret Management with HashiCorp Vault (Planned)
  - Vault construction + dynamic secret retrieval from applications.

### Software Architecture

- [Clean Architecture (3-Layer Structure)](./software/clean_arch_en.md)
  - Visual explanation of dependencies and responsibilities across Domain/UseCase/Infra.
- [Clean Architecture Workshop (WS1): Learning Design Resilient to Change](./software/clean_arch_ws1_en.md)
  - Hands-on implementation of Entity/Domain Service and swapping infrastructure from DB to AD.
- [Clean Architecture Workshop (WS2): Extension and Optimization](./software/clean_arch_ws2_en.md)
  - Abstraction of notification channels and introduction of transparent caching using the Decorator pattern.
- [Clean Architecture Workshop (WS3): E-commerce Platform](./software/clean_arch_ws3_en.md)
  - Designing an order/inventory management system with microservice isolation in mind.
- [Design Patterns](https://github.com/sokoide/design-patterns/README.md)
  - Implementation examples and application scenarios of GoF design patterns.
