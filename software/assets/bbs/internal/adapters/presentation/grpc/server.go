package grpc

import (
	"context"
	"errors"
	"github.com/sokoide/workshop/software/assets/bbs/internal/adapters/presentation/grpc/pb"
	"github.com/sokoide/workshop/software/assets/bbs/internal/domain"
	"github.com/sokoide/workshop/software/assets/bbs/internal/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BBSServer struct {
	pb.UnimplementedBBSServiceServer
	listBoards   usecase.ListBoardsInputPort
	listThreads  usecase.ListThreadsInputPort
	createThread usecase.CreateThreadInputPort
	listPosts    usecase.ListPostsInputPort
	createPost   usecase.CreatePostInputPort
}

func NewBBSServer(b usecase.ListBoardsInputPort, t usecase.ListThreadsInputPort, c usecase.CreateThreadInputPort, p usecase.ListPostsInputPort, n usecase.CreatePostInputPort) *BBSServer {
	return &BBSServer{listBoards: b, listThreads: t, createThread: c, listPosts: p, createPost: n}
}
func transportError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	case errors.Is(err, domain.ErrBoardNotFound), errors.Is(err, domain.ErrThreadNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, domain.ErrNotThreadOwner):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, domain.ErrEmptyTitle), errors.Is(err, domain.ErrEmptyBody):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domain.ErrDuplicatePostNumber):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
func thread(t usecase.ThreadDTO) *pb.Thread {
	return &pb.Thread{Id: t.ID, BoardId: t.BoardID, Title: t.Title, Owner: t.Owner, OwnerOnly: t.OwnerOnly, PostCount: int32(t.PostCount), CreatedAt: timestamppb.New(t.CreatedAt), LastPostedAt: timestamppb.New(t.LastPostedAt)}
}
func post(p usecase.PostDTO) *pb.Post {
	return &pb.Post{Id: p.ID, ThreadId: p.ThreadID, Number: int32(p.Number), Author: p.Author, Body: p.Body, Sage: p.Sage, CreatedAt: timestamppb.New(p.CreatedAt)}
}
func (s *BBSServer) ListBoards(ctx context.Context, _ *pb.ListBoardsRequest) (*pb.ListBoardsResponse, error) {
	out, err := s.listBoards.Execute(ctx)
	if err != nil {
		return nil, transportError(err)
	}
	r := &pb.ListBoardsResponse{}
	for _, b := range out.Boards {
		r.Boards = append(r.Boards, &pb.Board{Id: b.ID, Name: b.Name, Description: b.Description, CreatedAt: timestamppb.New(b.CreatedAt)})
	}
	return r, nil
}
func (s *BBSServer) ListThreads(ctx context.Context, in *pb.ListThreadsRequest) (*pb.ListThreadsResponse, error) {
	if in.BoardName == "" {
		return nil, status.Error(codes.InvalidArgument, "board name is required")
	}
	out, err := s.listThreads.Execute(ctx, usecase.ListThreadsInput{BoardName: in.BoardName})
	if err != nil {
		return nil, transportError(err)
	}
	r := &pb.ListThreadsResponse{}
	for _, t := range out.Threads {
		r.Threads = append(r.Threads, thread(t))
	}
	return r, nil
}
func (s *BBSServer) CreateThread(ctx context.Context, in *pb.CreateThreadRequest) (*pb.CreateThreadResponse, error) {
	if in.BoardName == "" {
		return nil, status.Error(codes.InvalidArgument, "board name is required")
	}
	out, err := s.createThread.Execute(ctx, usecase.CreateThreadInput{BoardName: in.BoardName, Title: in.Title, Author: in.Author, Body: in.Body, OwnerOnly: in.OwnerOnly})
	if err != nil {
		return nil, transportError(err)
	}
	return &pb.CreateThreadResponse{Thread: thread(out.Thread), Post: post(out.Post)}, nil
}
func (s *BBSServer) ListPosts(ctx context.Context, in *pb.ListPostsRequest) (*pb.ListPostsResponse, error) {
	if in.ThreadId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "positive thread id is required")
	}
	out, err := s.listPosts.Execute(ctx, usecase.ListPostsInput{ThreadID: in.ThreadId})
	if err != nil {
		return nil, transportError(err)
	}
	r := &pb.ListPostsResponse{}
	for _, p := range out.Posts {
		r.Posts = append(r.Posts, post(p))
	}
	return r, nil
}
func (s *BBSServer) CreatePost(ctx context.Context, in *pb.CreatePostRequest) (*pb.CreatePostResponse, error) {
	if in.ThreadId <= 0 {
		return nil, status.Error(codes.InvalidArgument, "positive thread id is required")
	}
	out, err := s.createPost.Execute(ctx, usecase.CreatePostInput{ThreadID: in.ThreadId, Author: in.Author, Body: in.Body, Sage: in.Sage})
	if err != nil {
		return nil, transportError(err)
	}
	return &pb.CreatePostResponse{Post: post(out.Post)}, nil
}
