package infra

import (
	"errors"
	"workshop/greeting/domain"
)

type MemoryUserRepo struct {
	users map[string]*domain.User
}

func NewMemoryUserRepo() *MemoryUserRepo {
	return &MemoryUserRepo{
		users: map[string]*domain.User{
			"1": {ID: "1", Name: "Alice"},
			"2": {ID: "2", Name: "Bob"},
			"3": {ID: "3", Name: "Charles"},
			"4": {ID: "4", Name: "David"},
			"5": {ID: "5", Name: "Eve"},
		},
	}
}

func (r *MemoryUserRepo) FindByID(id string) (*domain.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return user, nil
}
