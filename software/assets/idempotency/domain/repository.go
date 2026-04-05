package domain

import (
	"context"
)

type Account struct {
	ID      string
	Balance int
}

type AccountRepository interface {
	Get(ctx context.Context, id string) (*Account, error)
	UpdateBalance(ctx context.Context, id string, amount int) error
}

type IdempotencyStore interface {
	GetResult(ctx context.Context, key string) ([]byte, error)
	SaveResult(ctx context.Context, key string, result []byte) error
	Lock(ctx context.Context, key string) (bool, error)
	Unlock(ctx context.Context, key string) error
}
