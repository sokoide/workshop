package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
)

// mockUserRepository is a test double for the domain.UserRepository port.
type mockUserRepository struct {
	user *domain.User
	err  error
}

func (m *mockUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return m.user, m.err
}

func TestCheckVeteranUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("returns true for a veteran user", func(t *testing.T) {
		repo := &mockUserRepository{
			user: &domain.User{ID: "alice", Name: "Alice", JoinedAt: time.Now().AddDate(-10, 0, 0)},
		}
		uc := NewCheckVeteranUseCase(repo, domain.VeteranService{})

		veteran, err := uc.Execute(ctx, "alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !veteran {
			t.Error("expected veteran=true for 10 years of service")
		}
	})

	t.Run("returns false for a junior user", func(t *testing.T) {
		repo := &mockUserRepository{
			user: &domain.User{ID: "bob", Name: "Bob", JoinedAt: time.Now().AddDate(-1, 0, 0)},
		}
		uc := NewCheckVeteranUseCase(repo, domain.VeteranService{})

		veteran, err := uc.Execute(ctx, "bob")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if veteran {
			t.Error("expected veteran=false for 1 year of service")
		}
	})

	t.Run("repository error propagates", func(t *testing.T) {
		repo := &mockUserRepository{err: domain.ErrUserNotFound}
		uc := NewCheckVeteranUseCase(repo, domain.VeteranService{})

		_, err := uc.Execute(ctx, "missing")
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})
}
