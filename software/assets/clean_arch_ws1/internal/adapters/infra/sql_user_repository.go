package infra

import (
	"context"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
)

// userRow mimics one row of the users table.
type userRow struct {
	ID       string
	Name     string
	JoinedAt time.Time
}

// memoryDB is a stand-in for a real SQL database. It keeps rows in a map so
// the workshop runs with zero external dependencies.
type memoryDB struct {
	users map[string]userRow
}

// seedUsers keeps the sample dataset in one place. Joining dates are relative
// to "now" so the veteran verdicts never go stale: alice has 10 years of
// service (veteran), bob has 1 year (not yet a veteran).
func seedUsers() map[string]userRow {
	now := time.Now()
	return map[string]userRow{
		"alice": {ID: "alice", Name: "Alice", JoinedAt: now.AddDate(-10, 0, 0)},
		"bob":   {ID: "bob", Name: "Bob", JoinedAt: now.AddDate(-1, 0, 0)},
	}
}

// SQLUserRepository behaves like a repository backed by a SQL database: it
// "queries" the in-memory users table and reconstitutes domain entities.
type SQLUserRepository struct {
	db *memoryDB
}

func NewSQLUserRepository() *SQLUserRepository {
	return &SQLUserRepository{db: &memoryDB{users: seedUsers()}}
}

// Compile-time check that SQLUserRepository satisfies the domain port.
var _ domain.UserRepository = (*SQLUserRepository)(nil)

// FindByID pretends to run:
//
//	SELECT id, name, joined_at FROM users WHERE id = ?
func (r *SQLUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	row, ok := r.db.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	return &domain.User{
		ID:       row.ID,
		Name:     row.Name,
		JoinedAt: row.JoinedAt,
	}, nil
}
