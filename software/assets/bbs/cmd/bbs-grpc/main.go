package main

import (
	"context"
	grpcPresentation "github.com/sokoide/workshop/software/assets/bbs/internal/adapters/presentation/grpc"
	"github.com/sokoide/workshop/software/assets/bbs/internal/adapters/presentation/grpc/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log/slog"
	"net"
	"os"

	"github.com/sokoide/workshop/software/assets/bbs/internal/adapters/infra/persistence/sqlite"
	"github.com/sokoide/workshop/software/assets/bbs/internal/domain/entity"
	"github.com/sokoide/workshop/software/assets/bbs/internal/usecase"
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

	seedBoards(boardRepo)
	listener, err := net.Listen("tcp", "127.0.0.1:9090")
	if err != nil {
		slog.Error("listen failed", "error", err)
		os.Exit(1)
	}
	server := grpc.NewServer()
	pb.RegisterBBSServiceServer(server, grpcPresentation.NewBBSServer(listBoards, listThreads, createThread, listPosts, createPost))
	reflection.Register(server)
	slog.Info("gRPC server starting", "addr", listener.Addr())
	if err := server.Serve(listener); err != nil {
		slog.Error("serve failed", "error", err)
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
