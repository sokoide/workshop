# Sharding: From First Principles to Go Implementation

This guide is a clean, implementation-oriented view of sharding for distributed systems.

It explains:

1. What sharding is and why it exists
2. How to choose and validate a sharding strategy
3. How to handle hotspots, fan-out, and re-sharding
4. Why control-plane stability is as important as data-plane throughput
5. How Sparse Table and Segment Tree support shard-aware decisions
6. How to implement core patterns in Go
7. How Elasticsearch and Cortex map to these concepts

---

## 1) What Is Sharding?

Sharding is horizontal partitioning of data and traffic across multiple nodes.

Instead of:

```text
all data -> one machine
```

you do:

```text
data partitioned -> many machines
```

Each partition is a **shard**.

Sharding becomes necessary when one node can no longer absorb:

- storage growth
- write throughput
- query concurrency
- failure-domain blast radius

Sharding solves scale limits, but introduces distributed-systems costs:

- cross-shard query fan-out
- key skew and hotspots
- online rebalancing complexity
- routing correctness
- control-plane convergence under churn

---

## 2) Core Concepts You Must Get Right

### 2.1 Shard key

The shard key answers: "which shard owns this record?"

A bad key causes:

- uneven load
- hot partitions
- expensive multi-shard queries

A good key keeps scaling predictable.

### 2.2 Routing function

Once a key is selected, routing is:

```text
route(key) -> shard
```

Common routing families:

- modulo hash
- consistent hash ring
- range map
- tenant/domain mapping

### 2.3 Replication

Sharding distributes ownership. Replication provides availability.

Together they define:

- write quorum and durability
- read consistency model
- behavior during partial failures

---

## 3) Strategy Selection (Hash, Range, Tenant)

### 3.1 Hash-based sharding

Route by hash (modulo or ring).

Best for:

- high write rates
- near-uniform key distribution

Tradeoff:

- point writes distribute well
- analytical queries often fan out

### 3.2 Range-based sharding

Route by key interval (time range, lexical range, numeric range).

Best for:

- time-series windows
- range scans and pagination

Tradeoff:

- range reads are efficient
- latest range often becomes hot

Common mitigation:

- time partition + hash suffix

### 3.3 Tenant-aware sharding

Route by tenant/org/customer boundary.

Best for:

- multi-tenant SaaS
- noisy-neighbor isolation
- explicit per-tenant SLOs

Tradeoff:

- large tenants dominate capacity unless split further

Common mitigation:

- shuffle sharding
- intra-tenant hashing

---

## 4) Consistent Hashing and Rehash Cost

Naive modulo routing:

```text
hash(key) % N
```

Problem: changing `N` remaps most keys.

Consistent hashing limits movement during scale events.

Typical implementations:

- token ring + virtual nodes
- rendezvous (HRW) hashing
- jump consistent hash

Operational guidance:

- use enough virtual nodes for smooth distribution
- make replica placement topology-aware
- rate-limit scale events to avoid cache churn and queue spikes

---

## 5) Sparse Table and Segment Tree: Why They Matter to Sharding

These are not shard-routing strategies. They are **control-plane data structures** used to make routing and balancing decisions efficiently.

### 5.1 Where they fit in a sharded system

- Data plane:
  - write/read traffic using hash/range/tenant routing
- Control plane:
  - shard health snapshots
  - load-aware shard selection
  - rebalancing decisions

Sparse Table and Segment Tree belong to the control plane.

### 5.2 Sparse Table (static range query)

Use when shard metadata changes infrequently between rebuild windows.

Properties:

- build: `O(n log n)`
- query (idempotent ops like min/max): `O(1)`
- updates: expensive (rebuild-oriented)

Sharding-related use cases:

- precomputed minimum-latency shard per interval from a snapshot
- read-heavy planner paths where metadata is batch-refreshed
- "choose best shard set for this time window" with immutable window stats

### 5.3 Segment Tree (dynamic range query)

Use when shard metrics change continuously.

Properties:

- build: `O(n)` or `O(n log n)` depending on implementation
- point update: `O(log n)`
- range query: `O(log n)`

Sharding-related use cases:

- live shard load tracking (QPS, p95, queue depth)
- choosing least-loaded shard in a subset
- online balancing with frequent updates from telemetry

### 5.4 Practical rule

- metadata mostly static + heavy query rate -> Sparse Table
- metadata highly dynamic + frequent updates -> Segment Tree

The connection to sharding is direct: they make control decisions fast enough to keep routing stable under load.

---

## 6) Fan-out, Hotspots, and Re-sharding

### 6.1 Fan-out is the hidden tax

When one query touches many shards:

- tail latency increases
- memory and merge cost increase
- probability of partial failure increases

Mitigation:

- align shard key with dominant filters
- constrain time windows
- route with tenant/partition hints
- use two-phase query (discover candidate shards -> targeted fetch)

### 6.2 Hotspot patterns

- write hotspot: key salting, adaptive split, write buffering
- read hotspot: coalescing, replicas, cache tiers
- tenant hotspot: shuffle sharding, quotas, fairness schedulers

### 6.3 Safe re-sharding sequence

1. dual-write + read-old
2. backfill new layout
3. shadow-read compare
4. switch read traffic
5. decommission old layout after verification window

Never treat shard-key migration as an emergency shortcut.

---

## 7) Control Plane vs Data Plane

Data plane handles user traffic.
Control plane handles ownership, ring updates, health propagation, and balancing loops.

Most large incidents are caused by control-plane instability, not by hash math itself.

### Cortex ring loop optimization notes

- PR #7266: merged on **February 16, 2026**
  - [cortexproject/cortex#7266](https://github.com/cortexproject/cortex/pull/7266)
  - replaced repeated `time.After(...)` calls in DynamoDB watch loops with reusable timers
  - benchmark in PR discussion: around `248 B/op, 3 allocs/op` -> `0 B/op, 0 allocs/op`
- PR #7270: merged on **February 20, 2026**
  - [cortexproject/cortex#7270](https://github.com/cortexproject/cortex/pull/7270)
  - extended timer reuse to lifecycler and backoff loops, with shared timer helpers

Why this matters to sharding:

Control loops run continuously. Per-iteration allocations add GC pressure and jitter, which can delay ownership convergence and destabilize shard transitions during churn.

---

## 8) Mapping to Real Systems

### 8.1 Elasticsearch

- index split into primary shards + replicas
- default routing hashes `_id` (or custom routing key)
- query path often fans out then merges

Design implications:

- choose routing key for locality where possible
- constrain index/time scope to reduce fan-out
- plan shard sizing and lifecycle early

### 8.2 Cortex

- distributor hashes series labels into a consistent-hash ring
- ring selects replication set (multiple ingesters)
- ring convergence quality directly affects ingestion/query stability

Design implications:

- ring backend latency affects ownership propagation speed
- token movement should be controlled during rollout
- control-loop efficiency directly impacts ring stability

---

## 9) Go Reference Implementation

A runnable Go example is included at:

- `infra/assets/sharding/main.go`

It demonstrates:

1. modulo hash router
2. consistent hash ring with virtual nodes and replication set lookup
3. tenant-aware routing with per-tenant ring
4. Sparse Table for static range-min snapshot queries
5. Segment Tree for dynamic load-aware range-min queries

Run it:

```bash
go run infra/assets/sharding/main.go
```

---

## 10) Final Design Rules

1. Choose shard key by dominant query shape, not schema aesthetics.
2. Design re-sharding before production.
3. Track per-shard metrics (QPS, p95/p99, queue depth, fan-out).
4. Keep control loops allocation-light and predictable.
5. Replication and topology awareness are mandatory for production.
6. Use Sparse Table / Segment Tree intentionally based on update dynamics.

---

## References

- Elasticsearch routing and shards:
  - [Routing field](https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping-routing-field.html)
  - [Clusters, nodes, and shards](https://www.elastic.co/docs/deploy-manage/distributed-architecture/clusters-nodes-shards)
- Cortex hash ring docs:
  - [Hash rings](https://cortexmetrics.io/docs/configuration/arguments/#hash-rings)
- Cortex PRs:
  - [PR #7266](https://github.com/cortexproject/cortex/pull/7266)
  - [PR #7270](https://github.com/cortexproject/cortex/pull/7270)
