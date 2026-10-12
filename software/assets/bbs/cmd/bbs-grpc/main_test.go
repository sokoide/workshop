package main

import (
	"context"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/sokoide/workshop/software/assets/bbs/internal/adapters/infra/persistence/sqlite"
	presentation "github.com/sokoide/workshop/software/assets/bbs/internal/adapters/presentation/grpc"
	"github.com/sokoide/workshop/software/assets/bbs/internal/adapters/presentation/grpc/pb"
	"github.com/sokoide/workshop/software/assets/bbs/internal/domain/entity"
	"github.com/sokoide/workshop/software/assets/bbs/internal/usecase"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestGRPCContract(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := sqlite.OpenDB(filepath.Join(t.TempDir(), "bbs.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	boards := sqlite.NewBoardRepository(db)
	threads := sqlite.NewThreadRepository(db)
	posts := sqlite.NewPostRepository(db)
	tm := sqlite.NewTransactionManager(db)
	board, err := entity.NewBoard("programming", "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := boards.Save(ctx, board); err != nil {
		t.Fatal(err)
	}
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	defer server.Stop()
	pb.RegisterBBSServiceServer(server, presentation.NewBBSServer(usecase.NewListBoardsUseCase(boards), usecase.NewListThreadsUseCase(boards, threads), usecase.NewCreateThreadUseCase(boards, threads, posts, tm), usecase.NewListPostsUseCase(posts), usecase.NewCreatePostUseCase(threads, posts, tm)))
	go func() { _ = server.Serve(listener) }()
	connection, err := grpc.NewClient("passthrough:///bufnet", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	client := pb.NewBBSServiceClient(connection)
	all, err := client.ListBoards(ctx, &pb.ListBoardsRequest{})
	if err != nil || len(all.Boards) != 1 {
		t.Fatalf("boards=%v err=%v", all, err)
	}
	_, err = client.CreateThread(ctx, &pb.CreateThreadRequest{BoardName: "missing", Title: "x", Body: "x"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("missing: %v", err)
	}
	_, err = client.CreatePost(ctx, &pb.CreatePostRequest{ThreadId: -1})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid: %v", err)
	}
	created, err := client.CreateThread(ctx, &pb.CreateThreadRequest{BoardName: "programming", Title: "grpc", Author: "owner", Body: "first", OwnerOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if !created.Thread.OwnerOnly || created.Post.Number != 1 {
		t.Fatalf("mapping: %v", created)
	}
	_, err = client.CreatePost(ctx, &pb.CreatePostRequest{ThreadId: created.Thread.Id, Author: "other", Body: "denied"})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("denied: %v", err)
	}
	allowed, err := client.CreatePost(ctx, &pb.CreatePostRequest{ThreadId: created.Thread.Id, Author: "owner", Body: "second"})
	if err != nil || allowed.Post.Number != 2 {
		t.Fatalf("allowed: %v, %v", allowed, err)
	}
	list, err := client.ListPosts(ctx, &pb.ListPostsRequest{ThreadId: created.Thread.Id})
	if err != nil || len(list.Posts) != 2 {
		t.Fatalf("posts=%v err=%v", list, err)
	}
	ts, err := client.ListThreads(ctx, &pb.ListThreadsRequest{BoardName: "programming"})
	if err != nil || len(ts.Threads) != 1 || ts.Threads[0].PostCount != 2 {
		t.Fatalf("threads=%v err=%v", ts, err)
	}
}
