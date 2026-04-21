package port

import "context"

// TransactionManager controls transaction boundaries without exposing technical details.
// UseCase defines the boundary; Infra Adapter decides the mechanism (sql.Tx, etc).
type TransactionManager interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
