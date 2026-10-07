package usecase

import (
	"context"
	"fmt"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/domain"
	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/usecase/port"
)

// ApproveUseCase approves a membership and notifies the user.
type ApproveUseCase struct {
	repo     domain.UserRepository
	notifier port.NotificationGateway
}

func NewApproveUseCase(repo domain.UserRepository, notifier port.NotificationGateway) *ApproveUseCase {
	return &ApproveUseCase{repo: repo, notifier: notifier}
}

func (uc *ApproveUseCase) Execute(ctx context.Context, userID string) error {
	user, err := uc.repo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("find user %s: %w", userID, err)
	}
	// The UseCase does not care whether this goes out as Email or Slack.
	message := fmt.Sprintf("Membership approved: welcome, %s!", user.Name)
	return uc.notifier.Send(ctx, message)
}
