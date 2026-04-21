package entity

import (
	"time"

	"github.com/sokoide/cleanarch1/internal/domain"
)

type Board struct {
	ID        int64
	Slug      string
	Name      string
	CreatedAt time.Time
}

func NewBoard(slug, name string) (*Board, error) {
	if slug == "" {
		return nil, domain.ErrEmptyBoardSlug
	}
	if name == "" {
		return nil, domain.ErrEmptyBoardName
	}
	return &Board{
		Slug:      slug,
		Name:      name,
		CreatedAt: time.Now(),
	}, nil
}

func (b *Board) Rename(name string) error {
	if name == "" {
		return domain.ErrEmptyBoardName
	}
	b.Name = name
	return nil
}
