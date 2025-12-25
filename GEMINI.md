# Workshop Repository Context

This directory contains hands-on workshop materials for various technical topics, primarily aimed at software engineers and system administrators. The materials are available in both English and Japanese.

## Directory Overview

The repository is a collection of Markdown-based guides for practical exercises in networking, infrastructure as code, storage, and container orchestration. Each workshop is designed to be self-contained and provides step-by-step instructions.

## Key Files

### Entry Points

- `README_ja.md`: The main index of workshops in Japanese.
- `README_en.md`: The main index of workshops in English.

### Workshop Guides (Stored in `@infra/` directory)

- `@infra/coredns_ja.md`: CoreDNS workshop focusing on parent-child DNS server construction and name resolution.
- `@infra/vlan_ja.md`: Podman workshop for building L3 isolated networks using `macvlan` and router containers.
- `@infra/k8s_lb_ja.md`: Kubernetes Service (LoadBalancer) workshop, demonstrating how to build a virtual LB using `iptables`.
- `@infra/iscsi_pv_ja.md`: iSCSI and Kubernetes PersistentVolume workshop, covering iSCSI target/initiator setup and K8s integration.
- `@infra/terraform_ja.md`: Terraform workshop ranging from basics to practical DNS server construction using Podman/Docker.

## Usage

These materials are intended for hands-on learning. To use them:

1. Choose a topic and open the corresponding Markdown file in the `@infra/` directory (either `_ja.md` or `_en.md`).
2. Follow the prerequisites and setup instructions provided in the guide.
3. Execute the commands and configuration steps as described.
4. Verify the results using the provided validation steps (e.g., `dig`, `kubectl`, `terraform plan`).

Most workshops require a Linux environment (e.g., Ubuntu VMs) or container runtimes (Docker/Podman).
