package usecase

import (
	"context"

	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/internal/domain"
)

// TradeObserver monitors trade events from a subscriber.
type TradeObserver struct {
	sub domain.TradeSubscriber
}

// NewTradeObserver creates a new TradeObserver.
func NewTradeObserver(sub domain.TradeSubscriber) *TradeObserver {
	return &TradeObserver{sub: sub}
}

// Start begins observing trades with the given routing key and handler.
func (o *TradeObserver) Start(ctx context.Context, routingKey string, handler func(domain.Trade) error) error {
	return o.sub.Subscribe(ctx, routingKey, handler)
}
