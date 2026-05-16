package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/sokoide/cleanarch1/internal/domain/entity"
	httpPresentation "github.com/sokoide/cleanarch1/internal/adapters/presentation/http"
	"github.com/sokoide/cleanarch1/internal/adapters/presentation/http/handler"
	"github.com/sokoide/cleanarch1/internal/adapters/infra/persistence/sqlite"
	"github.com/sokoide/cleanarch1/internal/usecase"
)

func main() {
	dsn := os.Getenv("BBS_DB")
	if dsn == "" {
		dsn = "bbs.db"
	}

	db, err := sqlite.OpenDB(dsn)
	if err != nil {
		slog.Error("failed to open db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Infra Adapter
	boardRepo := sqlite.NewBoardRepository(db)
	threadRepo := sqlite.NewThreadRepository(db)
	postRepo := sqlite.NewPostRepository(db)
	tm := sqlite.NewTransactionManager(db)

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

	addr := ":8080"
	slog.Info("server starting", "addr", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func seedBoards(boardRepo *sqlite.BoardRepository) {
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
