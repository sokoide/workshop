package usecase

import "time"

// Board
type ListBoardsOutput struct {
	Boards []BoardDTO `json:"boards"`
}

type BoardDTO struct {
	ID        int64     `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// Thread
type ListThreadsInput struct {
	BoardSlug string
}

type ListThreadsOutput struct {
	Threads []ThreadDTO `json:"threads"`
}

type CreateThreadInput struct {
	BoardSlug string
	Title     string
	Author    string
	Body      string
}

type CreateThreadOutput struct {
	Thread ThreadDTO `json:"thread"`
	Post   PostDTO   `json:"post"`
}

type ThreadDTO struct {
	ID           int64     `json:"id"`
	BoardID      int64     `json:"board_id"`
	Title        string    `json:"title"`
	PostCount    int       `json:"post_count"`
	CreatedAt    time.Time `json:"created_at"`
	LastPostedAt time.Time `json:"last_posted_at"`
}

// Post
type ListPostsInput struct {
	ThreadID int64
}

type ListPostsOutput struct {
	Posts []PostDTO `json:"posts"`
}

type CreatePostInput struct {
	ThreadID int64
	Author   string
	Body     string
	Sage     bool
}

type CreatePostOutput struct {
	Post PostDTO `json:"post"`
}

type PostDTO struct {
	ID        int64     `json:"id"`
	ThreadID  int64     `json:"thread_id"`
	Number    int       `json:"number"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	Sage      bool      `json:"sage"`
	CreatedAt time.Time `json:"created_at"`
}
