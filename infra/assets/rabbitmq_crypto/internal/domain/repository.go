package domain

import "context"

// TradePublisher defines the interface for publishing trade events.
type TradePublisher interface {
	Publish(ctx context.Context, trade Trade) error
}

// TradeSubscriber defines the interface for subscribing to trade events.
type TradeSubscriber interface {
	Subscribe(ctx context.Context, routingKey string, handler func(Trade) error) error
}
