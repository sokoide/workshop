package domain

import "context"

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
}
