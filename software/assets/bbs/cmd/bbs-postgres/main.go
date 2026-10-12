package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/sokoide/workshop/software/assets/bbs/internal/adapters/infra/persistence/postgres"
	httpPresentation "github.com/sokoide/workshop/software/assets/bbs/internal/adapters/presentation/http"
	"github.com/sokoide/workshop/software/assets/bbs/internal/adapters/presentation/http/handler"
	"github.com/sokoide/workshop/software/assets/bbs/internal/domain/entity"
	"github.com/sokoide/workshop/software/assets/bbs/internal/usecase"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := postgres.OpenDB(ctx, dsn)
	if err != nil {
		slog.Error("failed to open db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Infra Adapter
	boardRepo := postgres.NewBoardRepository(db)
	threadRepo := postgres.NewThreadRepository(db)
	postRepo := postgres.NewPostRepository(db)
	tm := postgres.NewTransactionManager(db)

	// UseCase
	listBoards := usecase.NewListBoardsUseCase(boardRepo)
	listThreads := usecase.NewListThreadsUseCase(boardRepo, threadRepo)
	createThread := usecase.NewCreateThreadUseCase(boardRepo, threadRepo, postRepo, tm)
	listPosts := usecase.NewListPostsUseCase(postRepo)
	createPost := usecase.NewCreatePostUseCase(threadRepo, postRepo, tm)

	// Presentation - Handler
	boardHandler := handler.NewBoardHandler(listBoards)
	threadHandler := handler.NewThreadHandler(listThreads, createThread)
	postHandler := handler.NewPostHandler(listPosts, createPost)

	// Seed default boards if empty
	seedBoards(boardRepo)

	// Router
	router := httpPresentation.NewRouter(boardHandler, threadHandler, postHandler)

	addr := "127.0.0.1:8080"
	slog.Info("server starting", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func seedBoards(boardRepo usecase.BoardRepository) {
	boards, err := boardRepo.FindAll(context.Background())
	if err != nil {
		slog.Warn("seed check failed", "error", err)
		return
	}
	if len(boards) > 0 {
		return
	}

	defaults := []struct{ name, description string }{
		{"programming", "Programming General"},
		{"news", "News & Current Events"},
		{"chat", "Casual Chat"},
	}
	for _, d := range defaults {
		b, err := entity.NewBoard(d.name, d.description)
		if err != nil {
			slog.Warn("seed board skipped", "name", d.name, "error", err)
			continue
		}
		if err := boardRepo.Save(context.Background(), b); err != nil {
			slog.Warn("seed board save failed", "name", d.name, "error", err)
		}
	}
	slog.Info("seeded default boards")
}
