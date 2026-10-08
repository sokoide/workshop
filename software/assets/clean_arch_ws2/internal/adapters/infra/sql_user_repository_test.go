package infra

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/domain"
)

func TestSQLUserRepository_FindByID(t *testing.T) {
	ctx := context.Background()
	repo := NewSQLUserRepository()

	t.Run("reconstitutes a seeded user as a domain entity", func(t *testing.T) {
		user, err := repo.FindByID(ctx, "alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if user.ID != "alice" || user.Name != "Alice" {
			t.Errorf("unexpected user: %+v", user)
		}

		// シーダーは alice を 10 年前に入会させている（ベテラン判定用）。
		if got := time.Since(user.JoinedAt); got < 9*365*24*time.Hour {
			t.Errorf("alice should have joined about 10 years ago, got %v", got)
		}
	})

	t.Run("translates a missing row into the domain error", func(t *testing.T) {
		_, err := repo.FindByID(ctx, "missing")
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})
}

func TestSQLUserRepository_QueryCount(t *testing.T) {
	ctx := context.Background()
	repo := NewSQLUserRepository()

	if got := repo.QueryCount(); got != 0 {
		t.Fatalf("expected 0 queries at start, got %d", got)
	}

	// ヒットしたクエリも Not-Found のクエリも 1 回としてカウントされる。
	// main.go のキャッシュデモはこの計数に依存している。
	if _, err := repo.FindByID(ctx, "alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := repo.FindByID(ctx, "missing"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
	if _, err := repo.FindByID(ctx, "alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := repo.QueryCount(); got != 3 {
		t.Errorf("expected 3 queries, got %d", got)
	}
}
