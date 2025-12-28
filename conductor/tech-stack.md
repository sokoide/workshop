# Technology Stack

本プロジェクトで使用されている、および今後導入予定の技術要素を定義します。

## Core
- **Programming Language:** Go (v1.25.5)
- **Architecture:** 3-Layer Clean Architecture
    - **Domain:** Pure Go (No external dependencies)
    - **Usecase:** Application logic, orchestration
    - **Infrastructure:** Adapters for external systems

## Communication & Integration
- **API Protocols:**
    - **REST:** External client communication and legacy service integration.
    - **gRPC:** High-performance service-to-service communication.
- **Messaging:** RabbitMQ (Task-based asynchronous communication, simulated in early stages).

## Data Management
- **Primary Database:** PostgreSQL (Simulated implementation for workshop simplicity, focused on repository patterns).

## Tools & Automation
- **Build System:** Make (Tasks defined in `Makefile`)
- **Containerization:** Podman/Docker (Recommended for infrastructure components)
- **Documentation:** Mermaid.js (Integrated in Markdown)
