package usecase

import (
	"context"

	"github.com/sokoide/cleanarch1/internal/domain"
	"github.com/sokoide/cleanarch1/internal/domain/entity"
	"github.com/sokoide/cleanarch1/internal/domain/port"
)

type ListPostsUseCase struct {
	postRepo port.PostRepository
}

func NewListPostsUseCase(postRepo port.PostRepository) *ListPostsUseCase {
	return &ListPostsUseCase{postRepo: postRepo}
}

func (u *ListPostsUseCase) Execute(ctx context.Context, in ListPostsInput) (*ListPostsOutput, error) {
	posts, err := u.postRepo.FindByThreadID(ctx, in.ThreadID)
	if err != nil {
		return nil, err
	}

	dtos := make([]PostDTO, len(posts))
	for i, p := range posts {
		dtos[i] = toPostDTO(p)
	}
	return &ListPostsOutput{Posts: dtos}, nil
}

type CreatePostUseCase struct {
	threadRepo port.ThreadRepository
	postRepo   port.PostRepository
	tm         port.TransactionManager
}

func NewCreatePostUseCase(threadRepo port.ThreadRepository, postRepo port.PostRepository, tm port.TransactionManager) *CreatePostUseCase {
	return &CreatePostUseCase{threadRepo: threadRepo, postRepo: postRepo, tm: tm}
}

func (u *CreatePostUseCase) Execute(ctx context.Context, in CreatePostInput) (*CreatePostOutput, error) {
	thread, err := u.threadRepo.FindByID(ctx, in.ThreadID)
	if err != nil {
		return nil, err
	}
	if thread == nil {
		return nil, domain.ErrThreadNotFound
	}

	count, err := u.postRepo.CountByThreadID(ctx, thread.ID)
	if err != nil {
		return nil, err
	}

	post, err := entity.NewPost(thread.ID, count+1, in.Author, in.Body, in.Sage)
	if err != nil {
		return nil, err
	}

	var out *CreatePostOutput
	if err := u.tm.RunInTransaction(ctx, func(txCtx context.Context) error {
		if err := u.postRepo.Save(txCtx, post); err != nil {
			return err
		}

		thread.Bump(post.CreatedAt, in.Sage)
		if err := u.threadRepo.Save(txCtx, thread); err != nil {
			return err
		}

		out = &CreatePostOutput{Post: toPostDTO(post)}
		return nil
	}); err != nil {
		return nil, err
	}

	return out, nil
}
