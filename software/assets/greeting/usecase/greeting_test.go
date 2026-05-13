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
	// 1. Mockの設定
	mockRepo := &MockUserRepository{
		FindByIDFunc: func(id string) (*domain.User, error) {
			return &domain.User{ID: id, Name: "TestUser"}, nil
		},
	}

	// 2. UseCaseの作成
	uc := &GreetingUseCase{Repo: mockRepo}

	// 3. 実行
	result, err := uc.Execute("123")

	// 4. 検証
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	expected := "Hello, TestUser!"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
