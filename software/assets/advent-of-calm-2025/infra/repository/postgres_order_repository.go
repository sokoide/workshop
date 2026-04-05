package repository

import (
	"context"
	"fmt"

	"github.com/sokoide/advent-of-calm-2025/cleanarch/domain/entity"
)

type PostgresOrderRepository struct{}

func (r *PostgresOrderRepository) Save(ctx context.Context, order *entity.Order) error {
	fmt.Printf("Saving order %s to Postgres\n", order.ID)
	return nil
}

func (r *PostgresOrderRepository) FindByID(ctx context.Context, id string) (*entity.Order, error) {
	return nil, nil
}
