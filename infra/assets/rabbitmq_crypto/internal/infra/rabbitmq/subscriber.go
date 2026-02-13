package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/internal/domain"
	amqp "github.com/rabbitmq/amqp091-go"
)

type subscriber struct {
	ch *amqp.Channel
}

// NewSubscriber creates a new TradeSubscriber implementation using RabbitMQ.
func NewSubscriber(ch *amqp.Channel) domain.TradeSubscriber {
	return &subscriber{ch: ch}
}

func (s *subscriber) Subscribe(ctx context.Context, routingKey string, handler func(domain.Trade) error) error {
	// 1. Declare a temporary queue (exclusive to this consumer)
	q, err := s.ch.QueueDeclare(
		"",    // random name
		false, // non-durable
		true,  // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return fmt.Errorf("could not declare queue: %w", err)
	}

	// 2. Bind the queue to the exchange with the routing key
	err = s.ch.QueueBind(
		q.Name,       // queue name
		routingKey,   // routing key
		ExchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not bind queue: %w", err)
	}

	// 3. Start consuming
	msgs, err := s.ch.Consume(
		q.Name, // queue
		"",     // consumer tag
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return fmt.Errorf("could not start consume: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case d, ok := <-msgs:
				if !ok {
					return
				}
				var trade domain.Trade
				if err := json.Unmarshal(d.Body, &trade); err != nil {
					log.Printf("Error unmarshaling trade: %v", err)
					continue
				}
				if err := handler(trade); err != nil {
					log.Printf("Error handling trade: %v", err)
				}
			}
		}
	}()

	return nil
}
