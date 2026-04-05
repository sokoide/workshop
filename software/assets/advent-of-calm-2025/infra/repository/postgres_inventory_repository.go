package repository

import (
	"context"
	"fmt"
)

type PostgresInventoryRepository struct{}

func NewPostgresInventoryRepository() *PostgresInventoryRepository {
	return &PostgresInventoryRepository{}
}

func (r *PostgresInventoryRepository) GetStock(ctx context.Context, productID string) (int, error) {
	fmt.Printf("Fetching stock for product %s from Postgres\n", productID)
	// モック実装
	return 100, nil
}

func (r *PostgresInventoryRepository) UpdateStock(ctx context.Context, productID string, quantity int) error {
	fmt.Printf("Updating stock for product %s to %d in Postgres\n", productID, quantity)
	return nil
}
