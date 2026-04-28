package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sokoide/cleanarch1/internal/domain"
	"github.com/sokoide/cleanarch1/internal/domain/entity"
	"github.com/sokoide/cleanarch1/internal/domain/port"
)

// --- Mocks ---

type txCtxKey struct{}

type mockThreadRepo struct {
	thread   *entity.Thread
	err      error
	saveErr  error
	saved    *entity.Thread
	gotTxCtx bool
}

func (m *mockThreadRepo) FindByBoardID(ctx context.Context, boardID int64) ([]*entity.Thread, error) {
	return nil, nil
}

func (m *mockThreadRepo) FindByID(ctx context.Context, id int64) (*entity.Thread, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.gotTxCtx = ctx.Value(txCtxKey{}) != nil
	return m.thread, nil
}

func (m *mockThreadRepo) Save(ctx context.Context, thread *entity.Thread) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = thread
	return nil
}

type mockPostRepo struct {
	count    int
	countErr error
	saveErr  error
	saved    *entity.Post
	gotTxCtx bool
}

func (m *mockPostRepo) FindByThreadID(ctx context.Context, threadID int64) ([]*entity.Post, error) {
	return nil, nil
}

func (m *mockPostRepo) CountByThreadID(ctx context.Context, threadID int64) (int, error) {
	m.gotTxCtx = ctx.Value(txCtxKey{}) != nil
	return m.count, m.countErr
}

func (m *mockPostRepo) Save(ctx context.Context, post *entity.Post) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = post
	return nil
}

type mockTM struct {
	committed  bool
	rolledBack bool
}

func (m *mockTM) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	txCtx := context.WithValue(ctx, txCtxKey{}, true)
	err := fn(txCtx)
	if err != nil {
		m.rolledBack = true
		return err
	}
	m.committed = true
	return nil
}

// --- Tests ---

func TestCreatePost_Success(t *testing.T) {
	thread := &entity.Thread{
		ID:           1,
		BoardID:      1,
		Title:        "test",
		PostCount:    0,
		CreatedAt:    time.Now(),
		LastPostedAt: time.Now(),
	}

	tRepo := &mockThreadRepo{thread: thread}
	pRepo := &mockPostRepo{count: 0}
	tm := &mockTM{}

	uc := NewCreatePostUseCase(tRepo, pRepo, tm)
	out, err := uc.Execute(context.Background(), CreatePostInput{
		ThreadID: 1,
		Author:   "tester",
		Body:     "hello",
		Sage:     false,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil {
		t.Fatal("expected output")
	}
	if pRepo.saved == nil {
		t.Fatal("post should be saved")
	}
	if tRepo.saved == nil {
		t.Fatal("thread should be saved (bumped)")
	}
	if !tm.committed {
		t.Error("transaction should be committed")
	}
	if pRepo.saved.Number != 1 {
		t.Errorf("post number = %d, want 1", pRepo.saved.Number)
	}
}

func TestCreatePost_UsesTxCtx(t *testing.T) {
	thread := &entity.Thread{
		ID:           1,
		BoardID:      1,
		Title:        "test",
		PostCount:    0,
		CreatedAt:    time.Now(),
		LastPostedAt: time.Now(),
	}

	tRepo := &mockThreadRepo{thread: thread}
	pRepo := &mockPostRepo{count: 0}
	tm := &mockTM{}

	uc := NewCreatePostUseCase(tRepo, pRepo, tm)
	_, err := uc.Execute(context.Background(), CreatePostInput{
		ThreadID: 1,
		Author:   "tester",
		Body:     "hello",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !tRepo.gotTxCtx {
		t.Error("FindByID should receive tx context from RunInTransaction")
	}
	if !pRepo.gotTxCtx {
		t.Error("CountByThreadID should receive tx context from RunInTransaction")
	}
}

func TestCreatePost_ThreadNotFound(t *testing.T) {
	tRepo := &mockThreadRepo{thread: nil}
	pRepo := &mockPostRepo{count: 0}
	tm := &mockTM{}

	uc := NewCreatePostUseCase(tRepo, pRepo, tm)
	_, err := uc.Execute(context.Background(), CreatePostInput{
		ThreadID: 999,
		Author:   "tester",
		Body:     "hello",
	})

	if !errors.Is(err, domain.ErrThreadNotFound) {
		t.Errorf("error = %v, want ErrThreadNotFound", err)
	}
	if !tm.rolledBack {
		t.Error("transaction should be rolled back")
	}
}

func TestCreatePost_RollbackOnSaveError(t *testing.T) {
	thread := &entity.Thread{
		ID:           1,
		BoardID:      1,
		Title:        "test",
		PostCount:    0,
		CreatedAt:    time.Now(),
		LastPostedAt: time.Now(),
	}

	tRepo := &mockThreadRepo{thread: thread}
	pRepo := &mockPostRepo{
		count:   0,
		saveErr: errors.New("db error"),
	}
	tm := &mockTM{}

	uc := NewCreatePostUseCase(tRepo, pRepo, tm)
	_, err := uc.Execute(context.Background(), CreatePostInput{
		ThreadID: 1,
		Author:   "tester",
		Body:     "hello",
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if !tm.rolledBack {
		t.Error("transaction should be rolled back on save error")
	}
	if tm.committed {
		t.Error("transaction should NOT be committed on save error")
	}
}

func TestCreatePost_RollbackOnThreadSaveError(t *testing.T) {
	thread := &entity.Thread{
		ID:           1,
		BoardID:      1,
		Title:        "test",
		PostCount:    0,
		CreatedAt:    time.Now(),
		LastPostedAt: time.Now(),
	}

	tRepo := &mockThreadRepo{thread: thread, saveErr: errors.New("bump failed")}
	pRepo := &mockPostRepo{count: 0}
	tm := &mockTM{}

	uc := NewCreatePostUseCase(tRepo, pRepo, tm)
	_, err := uc.Execute(context.Background(), CreatePostInput{
		ThreadID: 1,
		Author:   "tester",
		Body:     "hello",
	})

	if err == nil {
		t.Fatal("expected error")
	}
	if !tm.rolledBack {
		t.Error("transaction should be rolled back when thread bump fails")
	}
}

func TestCreatePost_DuplicatePostNumber(t *testing.T) {
	thread := &entity.Thread{
		ID:           1,
		BoardID:      1,
		Title:        "test",
		PostCount:    0,
		CreatedAt:    time.Now(),
		LastPostedAt: time.Now(),
	}

	tRepo := &mockThreadRepo{thread: thread}
	pRepo := &mockPostRepo{
		count:   0,
		saveErr: domain.ErrDuplicatePostNumber,
	}
	tm := &mockTM{}

	uc := NewCreatePostUseCase(tRepo, pRepo, tm)
	_, err := uc.Execute(context.Background(), CreatePostInput{
		ThreadID: 1,
		Author:   "tester",
		Body:     "hello",
	})

	if !errors.Is(err, domain.ErrDuplicatePostNumber) {
		t.Errorf("error = %v, want ErrDuplicatePostNumber", err)
	}
}

func TestCreatePost_PostNumberSequence(t *testing.T) {
	thread := &entity.Thread{
		ID:           1,
		BoardID:      1,
		Title:        "test",
		PostCount:    0,
		CreatedAt:    time.Now(),
		LastPostedAt: time.Now(),
	}

	for existingCount := 0; existingCount < 5; existingCount++ {
		tRepo := &mockThreadRepo{thread: thread}
		pRepo := &mockPostRepo{count: existingCount}
		tm := &mockTM{}

		uc := NewCreatePostUseCase(tRepo, pRepo, tm)
		out, err := uc.Execute(context.Background(), CreatePostInput{
			ThreadID: 1,
			Author:   "tester",
			Body:     "hello",
		})

		if err != nil {
			t.Fatalf("count=%d: unexpected error: %v", existingCount, err)
		}
		want := existingCount + 1
		if out.Post.Number != want {
			t.Errorf("count=%d: post number = %d, want %d", existingCount, out.Post.Number, want)
		}
	}
}

// Verify port.TransactionManager compliance
var _ port.TransactionManager = (*mockTM)(nil)
