package port

import (
	"context"

	"github.com/sokoide/cleanarch1/internal/domain/entity"
)

type BoardRepository interface {
	FindAll(ctx context.Context) ([]*entity.Board, error)
	FindByName(ctx context.Context, name string) (*entity.Board, error)
	Save(ctx context.Context, board *entity.Board) error
}

type ThreadRepository interface {
	FindByBoardID(ctx context.Context, boardID int64) ([]*entity.Thread, error)
	FindByID(ctx context.Context, id int64) (*entity.Thread, error)
	Save(ctx context.Context, thread *entity.Thread) error
}

type PostRepository interface {
	FindByThreadID(ctx context.Context, threadID int64) ([]*entity.Post, error)
	CountByThreadID(ctx context.Context, threadID int64) (int, error)
	Save(ctx context.Context, post *entity.Post) error
}
