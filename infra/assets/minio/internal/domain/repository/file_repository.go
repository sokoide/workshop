package repository

import (
	"context"
	"time"
)

type FileRepository interface {
	Upload(ctx context.Context, key string, data []byte) error
	GeneratePresignedURL(ctx context.Context, key string, expires time.Duration) (string, error)
}
