package usecase

import (
	"context"

	"github.com/sokoide/cleanarch1/internal/domain"
	"github.com/sokoide/cleanarch1/internal/domain/entity"
)

type ListPostsInputPort interface {
	Execute(ctx context.Context, in ListPostsInput) (*ListPostsOutput, error)
}

type ListPostsUseCase struct {
	postRepo PostRepository
}

func NewListPostsUseCase(postRepo PostRepository) *ListPostsUseCase {
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

type CreatePostInputPort interface {
	Execute(ctx context.Context, in CreatePostInput) (*CreatePostOutput, error)
}

type CreatePostUseCase struct {
	threadRepo ThreadRepository
	postRepo   PostRepository
	tm         TransactionManager
}

func NewCreatePostUseCase(threadRepo ThreadRepository, postRepo PostRepository, tm TransactionManager) *CreatePostUseCase {
	return &CreatePostUseCase{threadRepo: threadRepo, postRepo: postRepo, tm: tm}
}

func (u *CreatePostUseCase) Execute(ctx context.Context, in CreatePostInput) (*CreatePostOutput, error) {
	var out *CreatePostOutput
	if err := u.tm.RunInTransaction(ctx, func(txCtx context.Context) error {
		thread, err := u.threadRepo.FindByID(txCtx, in.ThreadID)
		if err != nil {
			return err
		}
		if thread == nil {
			return domain.ErrThreadNotFound
		}

		if !thread.CanPost(in.Author) {
			return domain.ErrNotThreadOwner
		}

		count, err := u.postRepo.CountByThreadID(txCtx, thread.ID)
		if err != nil {
			return err
		}

		post, err := entity.NewPost(thread.ID, count+1, in.Author, in.Body, in.Sage)
		if err != nil {
			return err
		}

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
