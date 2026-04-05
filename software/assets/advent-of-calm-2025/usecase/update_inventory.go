package usecase

import (
	"context"
	"github.com/sokoide/advent-of-calm-2025/cleanarch/domain/entity"
	"github.com/sokoide/advent-of-calm-2025/cleanarch/domain/repository"
)

type UpdateInventoryInput struct {
	ProductID string
	Quantity  int
}

type UpdateInventoryUsecase struct {
	inventoryRepo repository.InventoryRepository
}

func NewUpdateInventoryUsecase(repo repository.InventoryRepository) *UpdateInventoryUsecase {
	return &UpdateInventoryUsecase{inventoryRepo: repo}
}

func (u *UpdateInventoryUsecase) Execute(ctx context.Context, input UpdateInventoryInput) error {
	if input.ProductID == "" {
		return entity.ErrInvalidProductID
	}
	if input.Quantity < 0 {
		return entity.ErrInvalidQuantity
	}

	return u.inventoryRepo.UpdateStock(ctx, input.ProductID, input.Quantity)
}
