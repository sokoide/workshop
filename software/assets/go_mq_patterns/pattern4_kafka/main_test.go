package main

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	metadataAPI "github.com/segmentio/kafka-go/protocol/metadata"
	produceAPI "github.com/segmentio/kafka-go/protocol/produce"
)

// stubTransport は kafka.Writer の Transport（RoundTripper インターフェース）を
// 差し替え、ブローカーなしで publish ロジックを検証するためのスタブ。
// メタデータ要求には 2 パーティションのトピックを、produce 要求には
// 成功応答を返す。
type stubTransport struct {
	mu        sync.Mutex
	fail      bool
	topic     string
	partition int32
	produces  int
	metadata  int
}

func (t *stubTransport) RoundTrip(ctx context.Context, addr net.Addr, req kafka.Request) (kafka.Response, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch r := req.(type) {
	case *metadataAPI.Request:
		t.metadata++
		if t.fail {
			return nil, errors.New("stub: metadata lookup failed")
		}
		return &metadataAPI.Response{
			Brokers: []metadataAPI.ResponseBroker{{NodeID: 1, Host: "localhost", Port: 9092}},
			Topics: []metadataAPI.ResponseTopic{{
				Name:       t.topic,
				Partitions: []metadataAPI.ResponsePartition{{PartitionIndex: 0, LeaderID: 1}, {PartitionIndex: 1, LeaderID: 1}},
			}},
		}, nil
	case *produceAPI.Request:
		t.produces++
		if t.fail {
			return nil, errors.New("stub: produce failed")
		}
		resp := &produceAPI.Response{}
		for _, topic := range r.Topics {
			rt := produceAPI.ResponseTopic{Topic: topic.Topic}
			for _, part := range topic.Partitions {
				rt.Partitions = append(rt.Partitions, produceAPI.ResponsePartition{Partition: part.Partition})
			}
			resp.Topics = append(resp.Topics, rt)
		}
		return resp, nil
	default:
		return nil, errors.New("stub: unexpected request type")
	}
}

func (t *stubTransport) stats() (produces, metadata int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.produces, t.metadata
}

func newStubbedProducer(t *testing.T, topic string) (*KafkaProducer, *stubTransport) {
	t.Helper()

	p := NewKafkaProducer([]string{"localhost:9092"}, topic)
	tr := &stubTransport{topic: topic}
	p.writer.Transport = tr // 同パッケージなので transport を差し替えられる
	t.Cleanup(func() {
		if err := p.Close(); err != nil {
			t.Errorf("failed to close producer: %v", err)
		}
	})
	return p, tr
}

func TestNewKafkaProducer_Configuration(t *testing.T) {
	p := NewKafkaProducer([]string{"localhost:9092"}, "orders")
	t.Cleanup(func() { _ = p.Close() })

	w := p.writer
	if w.Topic != "orders" {
		t.Errorf("unexpected topic: %q", w.Topic)
	}
	if _, ok := w.Balancer.(*kafka.LeastBytes); !ok {
		t.Errorf("expected LeastBytes balancer, got %T", w.Balancer)
	}
	if w.RequiredAcks != kafka.RequireAll {
		t.Errorf("expected RequireAll acks, got %v", w.RequiredAcks)
	}
	if w.BatchTimeout != 10*time.Millisecond {
		t.Errorf("unexpected batch timeout: %v", w.BatchTimeout)
	}
}

func TestKafkaProducer_Publish(t *testing.T) {
	p, tr := newStubbedProducer(t, "orders")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.Publish(ctx, "order-1", `{"id": 1}`); err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}

	// スタブ経由で、メタデータ解決後に 1 回 produce が走ったことを確認する。
	produces, metadata := tr.stats()
	if produces != 1 {
		t.Errorf("expected 1 produce request, got %d", produces)
	}
	if metadata == 0 {
		t.Error("expected at least one metadata request before producing")
	}
}

func TestKafkaProducer_PublishBatch(t *testing.T) {
	p, tr := newStubbedProducer(t, "orders")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	messages := []kafka.Message{
		{Key: []byte("order-1"), Value: []byte(`{"id": 1}`)},
		{Key: []byte("order-2"), Value: []byte(`{"id": 2}`)},
	}
	if err := p.PublishBatch(ctx, messages); err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}

	if produces, _ := tr.stats(); produces == 0 {
		t.Error("expected the batch to reach the transport")
	}
}

func TestKafkaProducer_PublishReturnsTransportError(t *testing.T) {
	p, tr := newStubbedProducer(t, "orders")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tr.mu.Lock()
	tr.fail = true
	tr.mu.Unlock()

	if err := p.Publish(ctx, "order-1", "value"); err == nil {
		t.Fatal("expected an error when the transport fails")
	}
}

func TestKafkaConsumer_ConsumeReturnsNilOnCancelledContext(t *testing.T) {
	c := NewKafkaConsumer([]string{"localhost:9092"}, "orders", "group-1")
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// コンテキストが既にキャンセルされていれば、エラーではなく nil を返す。
	done := make(chan error, 1)
	go func() {
		done <- c.Consume(ctx, func(key, value string) error { return nil })
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected nil on cancelled context, got %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Consume did not return after context cancellation")
	}
}
