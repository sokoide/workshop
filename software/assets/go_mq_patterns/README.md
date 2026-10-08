# Go Message Queue Patterns — 5 Executable Examples

Five standalone, runnable programs covering messaging patterns in Go — from plain channels to Kafka and retry-with-DLQ. Part of the [Message Queue Patterns catalog](../../go_mq_patterns_en.md).

## What this is

Each `patternN_*` directory is an independent `main` package demonstrating one pattern. Patterns 1, 2, 3, and 5 run entirely in-process using Go channels and finish by themselves. Pattern 4 demonstrates a production-style Kafka producer/consumer with `kafka-go` and needs a real broker.

## Requirements

- Go 1.21 (`go.mod`)
- Apache Kafka on `localhost:9092` for `pattern4_kafka` only (see the [workshop](../../go_mq_patterns_en.md) for a broker setup); all other patterns have no external dependencies

## How to run

```bash
go mod tidy
cd pattern1_simple_channel && go run .   # then pattern2_*, pattern3_*, (4 with Kafka), pattern5_*
```

Verified outputs (abridged):

```text
pattern1_simple_channel: Received: ID=msg-0..4, Payload=Hello 0..4 / Consumer stopped
pattern2_fanout:         every consumer receives the same broadcast messages
pattern3_worker_pool:    Worker N processing job job-N / Job job-N completed
pattern4_kafka:          produced/consumed via kafka-go (needs a broker)
pattern5_retry_dlq:      Message msg-2 failed (attempt 1/3 ... 2/3) -> moving to DLQ
```

## Layout

```text
.
├── pattern1_simple_channel/  # Producer/consumer on a plain channel
├── pattern2_fanout/          # Fan-out broadcast to all consumers
├── pattern3_worker_pool/     # Bounded worker pool consuming jobs
├── pattern4_kafka/           # Kafka producer/consumer with kafka-go
└── pattern5_retry_dlq/       # Retry with exponential backoff + dead letter queue
```
