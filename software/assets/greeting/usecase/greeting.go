package usecase

import (
	"workshop/greeting/domain"
)

type GreetingUseCasePort interface {
	Execute(id string) (string, error)
}

type GreetingUseCase struct {
	Repo domain.UserRepository
}

func (u *GreetingUseCase) Execute(id string) (string, error) {
	user, err := u.Repo.FindByID(id)
	if err != nil {
		return "", err
	}
	return "Hello, " + user.Name + "!", nil
}
