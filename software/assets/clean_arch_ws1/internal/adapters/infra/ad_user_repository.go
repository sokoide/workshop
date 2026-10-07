package infra

import (
	"context"
	"errors"
	"fmt"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
)

// ADUserRepository reads users from Active Directory via an LDAP client.
type ADUserRepository struct {
	ldapClient *LDAPClient // stubbed LDAP library
}

func NewADUserRepository(ldapClient *LDAPClient) *ADUserRepository {
	return &ADUserRepository{ldapClient: ldapClient}
}

// Compile-time check that ADUserRepository satisfies the same domain port as
// the SQL implementation.
var _ domain.UserRepository = (*ADUserRepository)(nil)

func (r *ADUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	// Issue the LDAP query to retrieve the entry
	entry, err := r.ldapClient.Search(ctx, id)
	if err != nil {
		if errors.Is(err, errNoSuchObject) {
			// Translate the driver-specific "not found" into a domain error
			// so the UseCase layer never sees LDAP details.
			return nil, fmt.Errorf("ldap search user %s: %w", id, domain.ErrUserNotFound)
		}
		return nil, fmt.Errorf("ldap search user %s: %w", id, err) // Convert driver error
	}
	// Reconstruct the entity from the directory entry (bare struct literal
	// is allowed here because we restore persisted state, not create new).
	return &domain.User{
		ID:       entry.UID,
		Name:     entry.DisplayName,
		JoinedAt: entry.CreationDate,
	}, nil
}
