package repository

import (
	"context"

	"github.com/sokoide/advent-of-calm-2025/cleanarch/domain/entity"
)

type OrderRepository interface {
	Save(ctx context.Context, order *entity.Order) error
	FindByID(ctx context.Context, id string) (*entity.Order, error)
}

type InventoryRepository interface {
	GetStock(ctx context.Context, productID string) (int, error)
	UpdateStock(ctx context.Context, productID string, quantity int) error
}

type PaymentPublisher interface {
	PublishPaymentTask(ctx context.Context, order *entity.Order) error
}

type InventoryClient interface {
	CheckAndReserve(ctx context.Context, productID string, quantity int) (bool, error)
}

type IDGenerator interface {
	GenerateID() string
}
