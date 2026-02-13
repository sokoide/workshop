package usecase

import (
	"context"
	"testing"

	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/internal/domain"
)

type mockSubscriber struct {
	routingKey string
	handler    func(domain.Trade) error
}

func (m *mockSubscriber) Subscribe(ctx context.Context, routingKey string, handler func(domain.Trade) error) error {
	m.routingKey = routingKey
	m.handler = handler
	return nil
}

func TestTradeObserver_Start(t *testing.T) {
	mock := &mockSubscriber{}
	obs := NewTradeObserver(mock)

	ctx := context.Background()
	key := "market.btc.#"
	
	err := obs.Start(ctx, key, func(t domain.Trade) error { return nil })
	if err != nil {
		t.Fatalf("failed to start observer: %v", err)
	}

	if mock.routingKey != key {
		t.Errorf("expected routing key %s, got %s", key, mock.routingKey)
	}

	if mock.handler == nil {
		t.Error("expected handler to be set")
	}
}
