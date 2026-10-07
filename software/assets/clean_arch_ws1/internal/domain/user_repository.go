package domain

import "context"

// UserRepository is the output port for user persistence. The UseCase layer
// depends on this interface — never on SQL drivers, LDAP clients, or any
// other infrastructure detail.
type UserRepository interface {
	FindByID(ctx context.Context, id string) (*User, error)
}
