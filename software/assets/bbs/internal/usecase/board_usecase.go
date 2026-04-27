package usecase

import (
	"context"

	"github.com/sokoide/cleanarch1/internal/domain/port"
)

type ListBoardsUseCase struct {
	boardRepo port.BoardRepository
}

func NewListBoardsUseCase(boardRepo port.BoardRepository) *ListBoardsUseCase {
	return &ListBoardsUseCase{boardRepo: boardRepo}
}

func (u *ListBoardsUseCase) Execute(ctx context.Context) (*ListBoardsOutput, error) {
	boards, err := u.boardRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]BoardDTO, len(boards))
	for i, b := range boards {
		dtos[i] = BoardDTO{
			ID:          b.ID,
			Name:        b.Name,
			Description: b.Description,
			CreatedAt:   b.CreatedAt,
		}
	}
	return &ListBoardsOutput{Boards: dtos}, nil
}
