package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// Pattern 4: Kafka Message Queue
// Production-ready distributed message queue using Apache Kafka

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			BatchTimeout: 10 * time.Millisecond,
			RequiredAcks: kafka.RequireAll,
		},
	}
}

func (p *KafkaProducer) Publish(ctx context.Context, key, value string) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: []byte(value),
		Time:  time.Now(),
	})
}

func (p *KafkaProducer) PublishBatch(ctx context.Context, messages []kafka.Message) error {
	return p.writer.WriteMessages(ctx, messages...)
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}

type KafkaConsumer struct {
	reader *kafka.Reader
}

func NewKafkaConsumer(brokers []string, topic, groupID string) *KafkaConsumer {
	return &KafkaConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			MinBytes:       1,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
			StartOffset:    kafka.FirstOffset,
		}),
	}
}

func (c *KafkaConsumer) Consume(ctx context.Context, handler func(key, value string) error) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // Context cancelled
			}
			return fmt.Errorf("read message: %w", err)
		}

		if err := handler(string(msg.Key), string(msg.Value)); err != nil {
			log.Printf("Handler error for key %s: %v", string(msg.Key), err)
		}
	}
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}

// KafkaQueue wraps producer and consumer for easy use
type KafkaQueue struct {
	producer *KafkaProducer
	consumer *KafkaConsumer
	topic    string
}

func NewKafkaQueue(brokers []string, topic, consumerGroup string) *KafkaQueue {
	return &KafkaQueue{
		producer: NewKafkaProducer(brokers, topic),
		consumer: NewKafkaConsumer(brokers, topic, consumerGroup),
		topic:    topic,
	}
}

func (q *KafkaQueue) Publish(ctx context.Context, key, value string) error {
	return q.producer.Publish(ctx, key, value)
}

func (q *KafkaQueue) Subscribe(ctx context.Context, handler func(key, value string) error) error {
	return q.consumer.Consume(ctx, handler)
}

func (q *KafkaQueue) Close() error {
	if err := q.producer.Close(); err != nil {
		return err
	}
	return q.consumer.Close()
}

func main() {
	// Kafka connection settings
	brokers := []string{"localhost:9092"}
	topic := "orders"
	groupID := "order-processor"

	queue := NewKafkaQueue(brokers, topic, groupID)
	defer queue.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Start consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := queue.Subscribe(ctx, func(key, value string) error {
			fmt.Printf("Received: key=%s, value=%s\n", key, value)
			return nil
		})
		if err != nil {
			log.Printf("Consumer error: %v", err)
		}
	}()

	// Give consumer time to start
	time.Sleep(time.Second)

	// Publish messages
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("order-%d", i)
		value := fmt.Sprintf(`{"id": %d, "status": "pending"}`, i)

		if err := queue.Publish(ctx, key, value); err != nil {
			log.Printf("Publish error: %v", err)
		} else {
			fmt.Printf("Published: key=%s\n", key)
		}
	}

	// Wait for messages to be processed
	time.Sleep(2 * time.Second)
	cancel()
	wg.Wait()
}
