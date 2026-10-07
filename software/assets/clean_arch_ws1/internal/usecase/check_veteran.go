package usecase

import (
	"context"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
)

// CheckVeteranUseCase answers "is this user a veteran?".
type CheckVeteranUseCase struct {
	repo       domain.UserRepository
	veteranSvc domain.VeteranService
}

func NewCheckVeteranUseCase(repo domain.UserRepository, veteranSvc domain.VeteranService) *CheckVeteranUseCase {
	return &CheckVeteranUseCase{repo: repo, veteranSvc: veteranSvc}
}

func (uc *CheckVeteranUseCase) Execute(ctx context.Context, id string) (bool, error) {
	user, err := uc.repo.FindByID(ctx, id) // Retrieved via repository
	if err != nil {
		return false, err
	}
	return uc.veteranSvc.IsVeteran(user), nil
}
