package infra

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
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

func TestSeedUsers(t *testing.T) {
	users := seedUsers()

	// シーダーの前提: alice はベテラン（10 年）、bob は 1 年。
	// 入会日を「現在」からの相対日にしているので、判定が古くならない。
	aliceAge := time.Since(users["alice"].JoinedAt)
	bobAge := time.Since(users["bob"].JoinedAt)
	if aliceAge < 9*365*24*time.Hour || aliceAge > 11*365*24*time.Hour {
		t.Errorf("alice: expected ~10 years of service, got %v", aliceAge)
	}
	if bobAge < 365*24*time.Hour || bobAge > 2*365*24*time.Hour {
		t.Errorf("bob: expected ~1 year of service, got %v", bobAge)
	}
}
