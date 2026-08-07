package usecase

import (
	"context"
)

type ListBoardsInputPort interface {
	Execute(ctx context.Context) (*ListBoardsOutput, error)
}

type ListBoardsUseCase struct {
	boardRepo BoardRepository
}

func NewListBoardsUseCase(boardRepo BoardRepository) *ListBoardsUseCase {
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
