package usecase

import (
	"testing"
	"workshop/greeting/domain"
)

// MockUserRepository はテスト用のリポジトリ実装です
type MockUserRepository struct {
	FindByIDFunc func(id string) (*domain.User, error)
}

func (m *MockUserRepository) FindByID(id string) (*domain.User, error) {
	return m.FindByIDFunc(id)
}

func TestGreetingUseCase_Execute(t *testing.T) {
	t.Run("returns greeting for existing user", func(t *testing.T) {
		mockRepo := &MockUserRepository{
			FindByIDFunc: func(id string) (*domain.User, error) {
				return &domain.User{ID: id, Name: "TestUser"}, nil
			},
		}

		uc := &GreetingUseCase{Repo: mockRepo}
		result, err := uc.Execute("123")

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
			FindByIDFunc: func(id string) (*domain.User, error) {
				return nil, domain.ErrUserNotFound
			},
		}

		uc := &GreetingUseCase{Repo: mockRepo}
		_, err := uc.Execute("999")

		if err != domain.ErrUserNotFound {
			t.Errorf("expected ErrUserNotFound, got %v", err)
		}
	})
}
