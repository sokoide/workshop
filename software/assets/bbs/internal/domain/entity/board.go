package entity

import (
	"time"

	"github.com/sokoide/cleanarch1/internal/domain"
)

type Board struct {
	ID          int64
	Name        string
	Description string
	CreatedAt   time.Time
}

func NewBoard(name, description string) (*Board, error) {
	if name == "" {
		return nil, domain.ErrEmptyBoardName
	}
	if description == "" {
		return nil, domain.ErrEmptyBoardDescription
	}
	return &Board{
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
	}, nil
}

func (b *Board) UpdateDescription(description string) error {
	if description == "" {
		return domain.ErrEmptyBoardDescription
	}
	b.Description = description
	return nil
}
