package infra

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
)

func TestADUserRepository_FindByID(t *testing.T) {
	ctx := context.Background()
	client := NewLDAPClient(map[string]LDAPEntry{
		"alice": {UID: "alice", DisplayName: "Alice", CreationDate: time.Date(2019, 4, 1, 0, 0, 0, 0, time.UTC)},
	})
	repo := NewADUserRepository(client)

	t.Run("reconstitutes the entity from the directory entry", func(t *testing.T) {
		user, err := repo.FindByID(ctx, "alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.ID != "alice" || user.Name != "Alice" {
			t.Errorf("unexpected user: %+v", user)
		}
		if !user.JoinedAt.Equal(time.Date(2019, 4, 1, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("unexpected JoinedAt: %v", user.JoinedAt)
		}
	})

	t.Run("translates a missing entry into the domain error", func(t *testing.T) {
		_, err := repo.FindByID(ctx, "missing")
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})
}
