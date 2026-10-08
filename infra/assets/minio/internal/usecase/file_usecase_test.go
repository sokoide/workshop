package usecase

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sokoide/workshop/infra/assets/minio/internal/domain/repository"
)

// mockFileRepository は FileRepository ポートのモック。アップロードされた
// データと署名付き URL 生成時の有効期限を記録する。
type mockFileRepository struct {
	uploadErr    error
	presignErr   error
	uploadedKey  string
	uploadedData []byte
	presignedKey string
	expires      time.Duration
	url          string
}

var _ repository.FileRepository = (*mockFileRepository)(nil)

func (m *mockFileRepository) Upload(ctx context.Context, key string, data []byte) error {
	if m.uploadErr != nil {
		return m.uploadErr
	}
	m.uploadedKey, m.uploadedData = key, data
	return nil
}

func (m *mockFileRepository) GeneratePresignedURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	if m.presignErr != nil {
		return "", m.presignErr
	}
	m.presignedKey, m.expires = key, expires
	return m.url, nil
}

func TestFileUsecase_UploadFile(t *testing.T) {
	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "photo.png")
	content := []byte("fake png bytes")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	t.Run("reads the file and hands its bytes to the repository", func(t *testing.T) {
		repo := &mockFileRepository{}
		uc := NewFileUsecase(repo)

		if err := uc.UploadFile(ctx, path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.uploadedKey != path {
			t.Errorf("expected key %q, got %q", path, repo.uploadedKey)
		}
		if string(repo.uploadedData) != string(content) {
			t.Errorf("expected %d bytes, got %d", len(content), len(repo.uploadedData))
		}
	})

	t.Run("returns the read error for a missing file", func(t *testing.T) {
		repo := &mockFileRepository{}
		uc := NewFileUsecase(repo)

		missing := filepath.Join(t.TempDir(), "missing.png")
		if err := uc.UploadFile(ctx, missing); err == nil {
			t.Error("expected an error for a missing file")
		}
		if repo.uploadedKey != "" {
			t.Errorf("repository should not be called, got key %q", repo.uploadedKey)
		}
	})

	t.Run("propagates repository failures", func(t *testing.T) {
		sentinel := errors.New("bucket unavailable")
		uc := NewFileUsecase(&mockFileRepository{uploadErr: sentinel})

		if err := uc.UploadFile(ctx, path); !errors.Is(err, sentinel) {
			t.Errorf("expected the repository error, got %v", err)
		}
	})
}

func TestFileUsecase_GetShareLink(t *testing.T) {
	ctx := context.Background()

	t.Run("presigns with the business-defined expiry", func(t *testing.T) {
		repo := &mockFileRepository{url: "https://minio.example.com/share"}
		uc := NewFileUsecase(repo)

		url, err := uc.GetShareLink(ctx, "photo.png")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if url != repo.url {
			t.Errorf("expected %q, got %q", repo.url, url)
		}
		if repo.presignedKey != "photo.png" {
			t.Errorf("expected key %q, got %q", "photo.png", repo.presignedKey)
		}
		// 有効期限 15 分はユースケース層のビジネスルール。
		if repo.expires != 15*time.Minute {
			t.Errorf("expected a 15m expiry, got %v", repo.expires)
		}
	})

	t.Run("propagates presign failures", func(t *testing.T) {
		sentinel := errors.New("presign failed")
		uc := NewFileUsecase(&mockFileRepository{presignErr: sentinel})

		if _, err := uc.GetShareLink(ctx, "photo.png"); !errors.Is(err, sentinel) {
			t.Errorf("expected the presign error, got %v", err)
		}
	})
}
