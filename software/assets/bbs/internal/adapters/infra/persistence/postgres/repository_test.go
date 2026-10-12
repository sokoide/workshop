package postgres_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/sokoide/workshop/software/assets/bbs/internal/adapters/infra/persistence/postgres"
	"github.com/sokoide/workshop/software/assets/bbs/internal/domain"
	"github.com/sokoide/workshop/software/assets/bbs/internal/domain/entity"
	"github.com/sokoide/workshop/software/assets/bbs/internal/usecase"
)

func TestRepositories(t *testing.T) {
	dsn := os.Getenv("BBS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set BBS_TEST_DATABASE_URL to an exercise PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("bbs_test_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := postgres.OpenDB(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	boards := postgres.NewBoardRepository(db)
	threads := postgres.NewThreadRepository(db)
	posts := postgres.NewPostRepository(db)
	tm := postgres.NewTransactionManager(db)
	all, err := boards.FindAll(ctx)
	if err != nil || len(all) != 3 {
		t.Fatalf("seed: %v, %v", all, err)
	}
	if _, err := boards.FindByName(ctx, "missing"); !errors.Is(err, domain.ErrBoardNotFound) {
		t.Fatalf("board error: %v", err)
	}
	if _, err := threads.FindByID(ctx, -1); !errors.Is(err, domain.ErrThreadNotFound) {
		t.Fatalf("thread error: %v", err)
	}
	created, err := usecase.NewCreateThreadUseCase(boards, threads, posts, tm).Execute(ctx, usecase.CreateThreadInput{BoardName: "programming", Title: "test", Author: "owner", Body: "first", OwnerOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	uc := usecase.NewCreatePostUseCase(threads, posts, tm)
	if _, err := uc.Execute(ctx, usecase.CreatePostInput{ThreadID: created.Thread.ID, Author: "other", Body: "denied"}); !errors.Is(err, domain.ErrNotThreadOwner) {
		t.Fatalf("policy: %v", err)
	}
	second, err := uc.Execute(ctx, usecase.CreatePostInput{ThreadID: created.Thread.ID, Author: "owner", Body: "second", Sage: true})
	if err != nil {
		t.Fatal(err)
	}
	if second.Post.Number != 2 || !second.Post.Sage {
		t.Fatalf("second: %+v", second)
	}
	roundtrip, err := threads.FindByID(ctx, created.Thread.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !roundtrip.OwnerOnly || roundtrip.Owner != "owner" || roundtrip.PostCount != 2 {
		t.Fatalf("roundtrip: %+v", roundtrip)
	}
	p, err := entity.NewPost(created.Thread.ID, 2, "owner", "duplicate", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := posts.Save(ctx, p); !errors.Is(err, domain.ErrDuplicatePostNumber) {
		t.Fatalf("duplicate: %v", err)
	}
	sentinel := errors.New("force rollback")
	err = tm.RunInTransaction(ctx, func(txCtx context.Context) error {
		p, err := entity.NewPost(created.Thread.ID, 3, "owner", "rollback", false)
		if err != nil {
			return err
		}
		if err := posts.Save(txCtx, p); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("rollback error: %v", err)
	}
	count, err := posts.CountByThreadID(ctx, created.Thread.ID)
	if err != nil || count != 2 {
		t.Fatalf("rollback count=%d err=%v", count, err)
	}
	const writers = 8
	var wg sync.WaitGroup
	failures := make(chan error, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := uc.Execute(ctx, usecase.CreatePostInput{ThreadID: created.Thread.ID, Author: "owner", Body: "concurrent"})
			failures <- err
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	list, err := posts.FindByThreadID(ctx, created.Thread.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2+writers {
		t.Fatalf("posts=%d", len(list))
	}
	for i, p := range list {
		if p.Number != i+1 {
			t.Fatalf("number=%d at %d", p.Number, i)
		}
	}
}
