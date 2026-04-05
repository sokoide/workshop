package messaging

import (
	"context"
	"fmt"

	"github.com/sokoide/advent-of-calm-2025/cleanarch/domain/entity"
)

type RabbitMQPaymentPublisher struct{}

func (p *RabbitMQPaymentPublisher) PublishPaymentTask(ctx context.Context, order *entity.Order) error {
	fmt.Printf("Publishing payment task for order %s to RabbitMQ\n", order.ID)
	return nil
}
