package usecase

import (
	"context"

	"workshop/greeting/domain"
)

type GreetingUseCasePort interface {
	Execute(ctx context.Context, id string) (string, error)
}

type GreetingUseCase struct {
	Repo domain.UserRepository
}

func (u *GreetingUseCase) Execute(ctx context.Context, id string) (string, error) {
	user, err := u.Repo.FindByID(ctx, id)
	if err != nil {
		return "", err
	}
	return "Hello, " + user.Name + "!", nil
}
