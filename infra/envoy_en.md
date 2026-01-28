# Service Mesh Fundamentals: Understanding L7 Traffic Control with Envoy Sidecars

In this workshop, you will learn how to implement advanced traffic control (retries, timeouts, circuit breaking) without changing application code, using **Envoy Proxy**, the de facto standard data plane for modern service meshes.

## Goal

By completing this workshop, you will be able to:

- Understand the Sidecar pattern (placing a proxy alongside the application)
- Understand the structure of Envoy configuration files (listeners, clusters, routes)
- Simulate network failures and verify resilience via retries and timeouts
- Use the Envoy Admin interface to check statistics

---

## Challenges and Solutions

In a microservices architecture, unstable network communication is inevitable.

### ❌ Traditional Challenges (Resiliency Logic in Code)

- Implementation of retries and timeouts required in each microservice's code
- Different libraries for different languages lead to inconsistent behavior
- Application redeployment required for configuration changes

### ✅ Service Mesh (Envoy) Approach

- **Out-of-Process**: Decouples communication control logic from the application and delegates it to the sidecar proxy
- **Language Agnostic**: Same control possible whether the app is Go, Python, or Java
- **Observability**: Unified collection of all communication metrics

---

## Architecture

Envoy receives requests from the client and forwards them to the backend service.
In this workshop, we implement a backend app with features to "intentionally delay" or "return errors," and verify how Envoy handles them.

```mermaid
graph LR
    Client[Client] -- HTTP --> Envoy[Envoy Proxy (Sidecar)]
    
    subgraph Service Mesh Pod
        Envoy -- Localhost --> App[Backend Service]
    end

    Envoy -.->|Stats/Admin| Admin[Admin Interface :9901]
```

### Directory Structure

```text
infra/assets/envoy/
├── envoy.yaml              # Envoy configuration file
├── docker-compose.yml      # Configuration definition
└── app/                    # Test backend app (with delay/error generation features)
```

---

## Preparation

### 1. Create Envoy Configuration (`envoy.yaml`)

The following configuration defines a **2-second timeout** and **3 retries** for connections to the backend.

```yaml
static_resources:
  listeners:
  - name: listener_0
    address:
      socket_address: { address: 0.0.0.0, port_value: 10000 }
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: ingress_http
          route_config:
            name: local_route
            virtual_hosts:
            - name: local_service
              domains: ["*"]
              routes:
              - match: { prefix: "/" }
                route: 
                  cluster: service_backend
                  timeout: 2s
                  retry_policy:
                    retry_on: "5xx"
                    num_retries: 3
          http_filters:
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router

  clusters:
  - name: service_backend
    connect_timeout: 0.25s
    type: STRICT_DNS
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: service_backend
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address: { address: backend, port_value: 80 }
```

### 2. Start Environment

```bash
cd infra/assets/envoy
docker-compose up -d
```

---

## Workshop Steps

### STEP 1: Access via Proxy

Verify that you can access the backend via Envoy (port 10000).

```bash
curl -v http://localhost:10000/
```

Check that `server: envoy` is included in the response header. This is proof that it went through Envoy.

### STEP 2: Verify Timeout Behavior

Check Envoy's behavior when the backend service is delayed.
Since `timeout: 2s` is set, request a **3-second sleep** from the backend (a feature of the workshop app).

```bash
curl -v http://localhost:10000/sleep/3
```

**Result**: Verify that Envoy cuts the connection after 2 seconds and returns `504 Gateway Timeout`. This prevents the client from waiting indefinitely due to backend issues.

### STEP 3: Verify Retry Behavior

The configuration sets `num_retries: 3`. Request the backend to return a `500` error.

```bash
curl -v http://localhost:10000/error/500
```

Checking Envoy logs or statistics (Admin IF) reveals that multiple requests were actually sent to the backend. If it were a temporary network error, it could succeed without showing an error to the user.

### STEP 4: Admin Interface

Check Envoy's internal state.

`http://localhost:9901/stats`

Verify that `retry` and `timeout` occurrence counts are recorded as metrics here. This is crucial for monitoring system health.

---

## Relation to Clean Architecture

Envoy can be viewed as the "Framework & Drivers" layer (details) in Clean Architecture.
Business logic (Usecase/Domain) is protected from "infrastructure details" like network instability and retry control, allowing focus on pure logic. This is the ultimate form of delegating infrastructure responsibilities from code to a sidecar.

---

## Cleanup

```bash
docker-compose down
```

---

## Next Steps

- **Circuit Breaking**: Cut-off functionality to protect the entire system during massive error occurrences
- **Istio**: Step up to a service mesh product that centrally manages Envoys via a control plane
