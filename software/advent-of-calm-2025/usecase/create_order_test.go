package usecase_test

import (
	"context"
	"testing"

	"github.com/sokoide/advent-of-calm-2025/cleanarch/domain/entity"
	"github.com/sokoide/advent-of-calm-2025/cleanarch/usecase"
)

// MockOrderRepository implements repository.OrderRepository
type MockOrderRepository struct {
	SaveFunc func(ctx context.Context, order *entity.Order) error
}

func (m *MockOrderRepository) Save(ctx context.Context, order *entity.Order) error {
	return m.SaveFunc(ctx, order)
}
func (m *MockOrderRepository) FindByID(ctx context.Context, id string) (*entity.Order, error) {
	return nil, nil
}

// MockInventoryClient implements repository.InventoryClient
type MockInventoryClient struct {
	CheckAndReserveFunc func(ctx context.Context, productID string, quantity int) (bool, error)
}

func (m *MockInventoryClient) CheckAndReserve(ctx context.Context, productID string, quantity int) (bool, error) {
	return m.CheckAndReserveFunc(ctx, productID, quantity)
}

// MockPaymentPublisher implements repository.PaymentPublisher
type MockPaymentPublisher struct {
	PublishFunc func(ctx context.Context, order *entity.Order) error
}

func (m *MockPaymentPublisher) PublishPaymentTask(ctx context.Context, order *entity.Order) error {
	return m.PublishFunc(ctx, order)
}

// MockIDGenerator implements repository.IDGenerator
type MockIDGenerator struct{}

func (m *MockIDGenerator) GenerateID() string { return "test-order-id" }

func TestCreateOrderUsecase_Execute(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		repo := &MockOrderRepository{
			SaveFunc: func(ctx context.Context, order *entity.Order) error { return nil },
		}
		inv := &MockInventoryClient{
			CheckAndReserveFunc: func(ctx context.Context, productID string, quantity int) (bool, error) { return true, nil },
		}
		pub := &MockPaymentPublisher{
			PublishFunc: func(ctx context.Context, order *entity.Order) error { return nil },
		}
		idGen := &MockIDGenerator{}

		u := usecase.NewCreateOrderUsecase(repo, inv, pub, idGen)

		input := usecase.CreateOrderInput{
			CustomerID: "c1",
			ProductID:  "p1",
			Quantity:   1,
			Amount:     100.0,
		}

		err := u.Execute(ctx, input)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("InsufficientStock", func(t *testing.T) {
		inv := &MockInventoryClient{
			CheckAndReserveFunc: func(ctx context.Context, productID string, quantity int) (bool, error) { return false, nil },
		}
		u := usecase.NewCreateOrderUsecase(nil, inv, nil, nil)

		input := usecase.CreateOrderInput{
			ProductID: "p1",
			Quantity:  1,
		}

		err := u.Execute(ctx, input)
		if err != entity.ErrInsufficientStock {
			t.Errorf("expected ErrInsufficientStock, got %v", err)
		}
	})
}
