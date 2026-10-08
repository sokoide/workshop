package messaging

import (
	"context"
	"fmt"

	"github.com/sokoide/workshop/software/assets/advent-of-calm-2025/internal/domain/entity"
)

type RabbitMQPaymentPublisher struct{}

func (p *RabbitMQPaymentPublisher) PublishPaymentTask(ctx context.Context, order *entity.Order) error {
	fmt.Printf("Publishing payment task for order %s to RabbitMQ\n", order.ID)
	return nil
}
