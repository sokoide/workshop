package rabbitmq

import (
	"context"
	"testing"
	"time"

	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/internal/domain"
)

func TestSubscriber_Subscribe(t *testing.T) {
	conn, ch, err := SetupConn("amqp://guest:guest@localhost:5672/")
	if err != nil {
		t.Skip("RabbitMQ not available, skipping integration test")
		return
	}
	defer conn.Close()
	defer ch.Close()

	sub := NewSubscriber(ch)
	pub := NewPublisher(ch)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	received := make(chan domain.Trade, 1)
	err = sub.Subscribe(ctx, "market.btc.usd", func(trade domain.Trade) error {
		received <- trade
		return nil
	})
	if err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	// Give it a moment to setup
	time.Sleep(100 * time.Millisecond)

	trade := domain.Trade{
		ID:             "test-id",
		Symbol:         "BTC",
		TargetCurrency: "USD",
		Price:          50000.0,
	}
	err = pub.Publish(ctx, trade)
	if err != nil {
		t.Fatalf("failed to publish: %v", err)
	}

	select {
	case r := <-received:
		if r.ID != trade.ID {
			t.Errorf("expected ID %s, got %s", trade.ID, r.ID)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for trade")
	}
}
