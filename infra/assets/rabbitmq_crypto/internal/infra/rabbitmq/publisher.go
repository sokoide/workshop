package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/internal/domain"
)

type publisher struct {
	ch *amqp.Channel
}

// NewPublisher creates a new TradePublisher implementation using RabbitMQ.
func NewPublisher(ch *amqp.Channel) domain.TradePublisher {
	return &publisher{ch: ch}
}

func (p *publisher) Publish(ctx context.Context, trade domain.Trade) error {
	body, err := json.Marshal(trade)
	if err != nil {
		return fmt.Errorf("could not marshal trade: %w", err)
	}

	// Routing Key: market.<symbol>.<target> (e.g., market.btc.usd).
	// Topic exchanges match routing keys case-sensitively, so normalize to
	// lowercase — otherwise bindings like "market.btc.#" never match.
	routingKey := fmt.Sprintf("market.%s.%s",
		strings.ToLower(trade.Symbol), strings.ToLower(trade.TargetCurrency))

	return p.ch.PublishWithContext(ctx,
		ExchangeName, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)
}
