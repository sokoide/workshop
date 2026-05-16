package usecase

import (
	"context"
	"github.com/sokoide/advent-of-calm-2025/cleanarch/internal/domain/entity"
	"github.com/sokoide/advent-of-calm-2025/cleanarch/internal/domain/repository"
)

type CheckInventoryInput struct {
	ProductID string
}

type CheckInventoryOutput struct {
	ProductID string
	Quantity  int
}

type CheckInventoryUsecase struct {
	inventoryRepo repository.InventoryRepository
}

func NewCheckInventoryUsecase(repo repository.InventoryRepository) *CheckInventoryUsecase {
	return &CheckInventoryUsecase{inventoryRepo: repo}
}

func (u *CheckInventoryUsecase) Execute(ctx context.Context, input CheckInventoryInput) (*CheckInventoryOutput, error) {
	if input.ProductID == "" {
		return nil, entity.ErrInvalidProductID
	}

	stock, err := u.inventoryRepo.GetStock(ctx, input.ProductID)
	if err != nil {
		return nil, err
	}

	return &CheckInventoryOutput{
		ProductID: input.ProductID,
		Quantity:  stock,
	}, nil
}
