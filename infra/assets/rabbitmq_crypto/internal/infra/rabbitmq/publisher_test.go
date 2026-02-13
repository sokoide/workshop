package rabbitmq

import (
	"context"
	"testing"
	"time"

	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/internal/domain"
)

func TestPublisher_Publish(t *testing.T) {
	// Skip if RabbitMQ is not running
	conn, ch, err := SetupConn("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Skip("RabbitMQ not available, skipping integration test")
		return
	}
	defer conn.Close()
	defer ch.Close()

	pub := NewPublisher(ch)
	trade := domain.Trade{
		ID:             "test-id",
		Symbol:         "BTC",
		TargetCurrency: "USD",
		Price:          50000.0,
		Amount:         1.0,
		Timestamp:      time.Now(),
	}

	err = pub.Publish(context.Background(), trade)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}
}
