package usecase

import (
	"context"
	"os"
	"time"

	"github.com/sokoide/workshop/infra/assets/minio/internal/domain/repository"
)

type FileUsecase struct {
	repo repository.FileRepository
}

func NewFileUsecase(repo repository.FileRepository) *FileUsecase {
	return &FileUsecase{repo: repo}
}

func (u *FileUsecase) UploadFile(ctx context.Context, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return u.repo.Upload(ctx, filename, data)
}

func (u *FileUsecase) GetShareLink(ctx context.Context, filename string) (string, error) {
	// ビジネスロジックとしての有効期限設定
	return u.repo.GeneratePresignedURL(ctx, filename, 15*time.Minute)
}
