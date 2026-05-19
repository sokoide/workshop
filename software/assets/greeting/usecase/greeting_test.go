package usecase

import (
	"context"
	"testing"
	"workshop/greeting/domain"
)

// MockUserRepository はテスト用のリポジトリ実装です
type MockUserRepository struct {
	FindByIDFunc func(ctx context.Context, id string) (*domain.User, error)
}

func (m *MockUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return m.FindByIDFunc(ctx, id)
}

func TestGreetingUseCase_Execute(t *testing.T) {
	ctx := context.Background()
	t.Run("returns greeting for existing user", func(t *testing.T) {
		mockRepo := &MockUserRepository{
			FindByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return &domain.User{ID: id, Name: "TestUser"}, nil
			},
		}

		uc := &GreetingUseCase{Repo: mockRepo}
		result, err := uc.Execute(ctx, "123")

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		expected := "Hello, TestUser!"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("returns error for missing user", func(t *testing.T) {
		mockRepo := &MockUserRepository{
			FindByIDFunc: func(ctx context.Context, id string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
		}

		uc := &GreetingUseCase{Repo: mockRepo}
		_, err := uc.Execute(ctx, "999")

		if err != domain.ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})
}
