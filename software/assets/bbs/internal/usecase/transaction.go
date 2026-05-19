package usecase

import "context"

// TransactionManager controls transaction boundaries without exposing technical details.
// UseCase defines the boundary; Infra Adapter decides the mechanism (sql.Tx, etc).
// Owned by UseCase layer, not Domain, because transaction management is an
// application policy concern, not a domain concept.
type TransactionManager interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
