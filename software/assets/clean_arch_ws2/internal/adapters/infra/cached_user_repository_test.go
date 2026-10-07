package infra

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/domain"
)

// countingRepository counts calls so tests can tell cache hits from misses.
type countingRepository struct {
	user  *domain.User
	calls int
}

func (r *countingRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	r.calls++
	return r.user, nil
}

func newCachedTestRepo() (*countingRepository, *CachedUserRepository) {
	origin := &countingRepository{
		user: &domain.User{ID: "alice", Name: "Alice", JoinedAt: time.Now().AddDate(-10, 0, 0)},
	}
	return origin, NewCachedUserRepository(origin, NewMemoryCacheStore())
}

func TestCachedUserRepository_FindByID(t *testing.T) {
	ctx := context.Background()

	t.Run("second lookup is served from the cache", func(t *testing.T) {
		origin, cached := newCachedTestRepo()

		for i := range 3 {
			user, err := cached.FindByID(ctx, "alice")
			if err != nil {
				t.Fatalf("lookup %d: unexpected error: %v", i+1, err)
			}
			if user.ID != "alice" {
				t.Errorf("lookup %d: unexpected user %+v", i+1, user)
			}
		}
		if origin.calls != 1 {
			t.Errorf("expected the origin to be queried once, got %d", origin.calls)
		}
	})

	t.Run("expired entries fall back to the origin", func(t *testing.T) {
		origin := &countingRepository{
			user: &domain.User{ID: "alice", Name: "Alice", JoinedAt: time.Now().AddDate(-10, 0, 0)},
		}
		store := NewMemoryCacheStore()
		base := time.Now()
		store.now = func() time.Time { return base } // freeze the clock
		cached := NewCachedUserRepository(origin, store)

		if _, err := cached.FindByID(ctx, "alice"); err != nil {
			t.Fatalf("first lookup: unexpected error: %v", err)
		}
		// Advance past the TTL: the entry is gone, so the origin answers again.
		store.now = func() time.Time { return base.Add(2 * userCacheTTL) }
		if _, err := cached.FindByID(ctx, "alice"); err != nil {
			t.Fatalf("second lookup: unexpected error: %v", err)
		}
		if origin.calls != 2 {
			t.Errorf("expected the origin to be queried twice, got %d", origin.calls)
		}
	})

	t.Run("not-found errors are not cached", func(t *testing.T) {
		origin := &failingRepository{}
		cached := NewCachedUserRepository(origin, NewMemoryCacheStore())

		for i := range 2 {
			_, err := cached.FindByID(ctx, "missing")
			if !errors.Is(err, domain.ErrUserNotFound) {
				t.Errorf("lookup %d: expected ErrUserNotFound, got %v", i+1, err)
			}
		}
		if origin.calls != 2 {
			t.Errorf("expected both lookups to reach the origin, got %d", origin.calls)
		}
	})
}

// failingRepository always answers ErrUserNotFound, counting the attempts.
type failingRepository struct {
	calls int
}

func (r *failingRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	r.calls++
	return nil, domain.ErrUserNotFound
}
