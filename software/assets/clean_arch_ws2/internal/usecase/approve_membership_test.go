package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/domain"
	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/usecase/port"
)

// mockNotificationGateway records Send calls instead of notifying anyone.
type mockNotificationGateway struct {
	sent []string
	err  error
}

func (m *mockNotificationGateway) Send(ctx context.Context, message string) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, message)
	return nil
}

// Compile-time check that the mock satisfies the UseCase-owned port.
var _ port.NotificationGateway = (*mockNotificationGateway)(nil)

func TestApproveUseCase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("sends a notification addressed to the user", func(t *testing.T) {
		repo := &mockUserRepository{
			user: &domain.User{ID: "alice", Name: "Alice", JoinedAt: time.Now().AddDate(-10, 0, 0)},
		}
		notifier := &mockNotificationGateway{}
		uc := NewApproveUseCase(repo, notifier)

		if err := uc.Execute(ctx, "alice"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(notifier.sent) != 1 {
			t.Fatalf("expected exactly 1 notification, got %d", len(notifier.sent))
		}
		if !strings.Contains(notifier.sent[0], "Alice") {
			t.Errorf("notification should mention the user name, got %q", notifier.sent[0])
		}
	})

	t.Run("skips the notification when the user is missing", func(t *testing.T) {
		repo := &mockUserRepository{err: domain.ErrUserNotFound}
		notifier := &mockNotificationGateway{}
		uc := NewApproveUseCase(repo, notifier)

		err := uc.Execute(ctx, "missing")
		if !errors.Is(err, domain.ErrUserNotFound) {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
		if len(notifier.sent) != 0 {
			t.Errorf("expected no notification on error, got %v", notifier.sent)
		}
	})
}
