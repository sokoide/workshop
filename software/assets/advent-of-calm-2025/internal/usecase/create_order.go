package usecase

import (
	"context"
	"time"

	"github.com/sokoide/advent-of-calm-2025/cleanarch/internal/domain/entity"
	"github.com/sokoide/advent-of-calm-2025/cleanarch/internal/domain/repository"
)

type CreateOrderInput struct {
	CustomerID string
	ProductID  string
	Quantity   int
	Amount     float64
}

type CreateOrderUsecase struct {
	orderRepo       repository.OrderRepository
	inventoryClient repository.InventoryClient
	paymentPub      repository.PaymentPublisher
	idGen           repository.IDGenerator
}

func NewCreateOrderUsecase(
	repo repository.OrderRepository,
	invClient repository.InventoryClient,
	pub repository.PaymentPublisher,
	idGen repository.IDGenerator,
) *CreateOrderUsecase {
	return &CreateOrderUsecase{
		orderRepo:       repo,
		inventoryClient: invClient,
		paymentPub:      pub,
		idGen:           idGen,
	}
}

func (u *CreateOrderUsecase) Execute(ctx context.Context, input CreateOrderInput) error {
	// 1. バリデーション
	if input.ProductID == "" {
		return entity.ErrInvalidProductID
	}
	if input.Quantity <= 0 {
		return entity.ErrInvalidQuantity
	}

	// 2. 在庫確保 (Port を直接使用)
	available, err := u.inventoryClient.CheckAndReserve(ctx, input.ProductID, input.Quantity)
	if err != nil {
		return err
	}
	if !available {
		return entity.ErrInsufficientStock
	}

	// 3. エンティティの作成
	order := &entity.Order{
		ID:         u.idGen.GenerateID(),
		CustomerID: input.CustomerID,
		Amount:     input.Amount,
		Status:     entity.OrderStatusPending,
		CreatedAt:  time.Now(),
	}

	// 4. 永続化
	if err := u.orderRepo.Save(ctx, order); err != nil {
		return err
	}

	// 5. 非同期処理の開始（支払い処理をキューへ）
	return u.paymentPub.PublishPaymentTask(ctx, order)
}
