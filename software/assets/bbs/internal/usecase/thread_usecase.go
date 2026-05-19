package usecase

import (
	"context"

	"github.com/sokoide/cleanarch1/internal/domain"
	"github.com/sokoide/cleanarch1/internal/domain/entity"
	"github.com/sokoide/cleanarch1/internal/domain/port"
)

type ListThreadsInputPort interface {
	Execute(ctx context.Context, in ListThreadsInput) (*ListThreadsOutput, error)
}

type ListThreadsUseCase struct {
	boardRepo  port.BoardRepository
	threadRepo port.ThreadRepository
}

func NewListThreadsUseCase(boardRepo port.BoardRepository, threadRepo port.ThreadRepository) *ListThreadsUseCase {
	return &ListThreadsUseCase{boardRepo: boardRepo, threadRepo: threadRepo}
}

func (u *ListThreadsUseCase) Execute(ctx context.Context, in ListThreadsInput) (*ListThreadsOutput, error) {
	board, err := u.boardRepo.FindByName(ctx, in.BoardName)
	if err != nil {
		return nil, err
	}
	if board == nil {
		return nil, domain.ErrBoardNotFound
	}

	threads, err := u.threadRepo.FindByBoardID(ctx, board.ID)
	if err != nil {
		return nil, err
	}

	dtos := make([]ThreadDTO, len(threads))
	for i, t := range threads {
		dtos[i] = toThreadDTO(t)
	}
	return &ListThreadsOutput{Threads: dtos}, nil
}

type CreateThreadInputPort interface {
	Execute(ctx context.Context, in CreateThreadInput) (*CreateThreadOutput, error)
}

type CreateThreadUseCase struct {
	boardRepo  port.BoardRepository
	threadRepo port.ThreadRepository
	postRepo   port.PostRepository
	tm         TransactionManager
}

func NewCreateThreadUseCase(boardRepo port.BoardRepository, threadRepo port.ThreadRepository, postRepo port.PostRepository, tm TransactionManager) *CreateThreadUseCase {
	return &CreateThreadUseCase{boardRepo: boardRepo, threadRepo: threadRepo, postRepo: postRepo, tm: tm}
}

func (u *CreateThreadUseCase) Execute(ctx context.Context, in CreateThreadInput) (*CreateThreadOutput, error) {
	board, err := u.boardRepo.FindByName(ctx, in.BoardName)
	if err != nil {
		return nil, err
	}
	if board == nil {
		return nil, domain.ErrBoardNotFound
	}

	thread, err := entity.NewThread(board.ID, in.Title, in.Author)
	if err != nil {
		return nil, err
	}
	if in.OwnerOnly {
		if err := thread.EnableOwnerOnlyMode(in.Author); err != nil {
			return nil, err
		}
	}

	post, err := entity.NewPost(thread.ID, 1, in.Author, in.Body, false)
	if err != nil {
		return nil, err
	}

	var out *CreateThreadOutput
	if err := u.tm.RunInTransaction(ctx, func(txCtx context.Context) error {
		if err := u.threadRepo.Save(txCtx, thread); err != nil {
			return err
		}

		post.ThreadID = thread.ID
		thread.Bump(post.CreatedAt, false)
		if err := u.postRepo.Save(txCtx, post); err != nil {
			return err
		}

		if err := u.threadRepo.Save(txCtx, thread); err != nil {
			return err
		}

		out = &CreateThreadOutput{
			Thread: toThreadDTO(thread),
			Post:   toPostDTO(post),
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return out, nil
}

func toThreadDTO(t *entity.Thread) ThreadDTO {
	return ThreadDTO{
		ID:           t.ID,
		BoardID:      t.BoardID,
		Title:        t.Title,
		Owner:        t.Owner,
		OwnerOnly:    t.OwnerOnly,
		PostCount:    t.PostCount,
		CreatedAt:    t.CreatedAt,
		LastPostedAt: t.LastPostedAt,
	}
}

func toPostDTO(p *entity.Post) PostDTO {
	return PostDTO{
		ID:        p.ID,
		ThreadID:  p.ThreadID,
		Number:    p.Number,
		Author:    p.Author,
		Body:      p.Body,
		Sage:      p.Sage,
		CreatedAt: p.CreatedAt,
	}
}
