package entity

import (
	"strings"
	"time"

	"github.com/sokoide/cleanarch1/internal/domain"
)

type Post struct {
	ID        int64
	ThreadID  int64
	Number    int
	Author    string
	Body      string
	Sage      bool
	CreatedAt time.Time
}

const DefaultAuthor = "名無しさん"

func NewPost(threadID int64, number int, author, body string, sage bool) (*Post, error) {
	if strings.TrimSpace(body) == "" {
		return nil, domain.ErrEmptyBody
	}
	if author == "" {
		author = DefaultAuthor
	}
	return &Post{
		ThreadID:  threadID,
		Number:    number,
		Author:    author,
		Body:      body,
		Sage:      sage,
		CreatedAt: time.Now(),
	}, nil
}
