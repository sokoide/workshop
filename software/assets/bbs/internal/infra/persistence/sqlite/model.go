package sqlite

import (
	"time"

	"github.com/sokoide/cleanarch1/internal/domain/entity"
)

// DB models — separate from domain entities to avoid leaking persistence concerns.

type BoardModel struct {
	ID        int64
	Slug      string
	Name      string
	CreatedAt time.Time
}

func (m BoardModel) ToEntity() *entity.Board {
	return &entity.Board{
		ID:        m.ID,
		Slug:      m.Slug,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
	}
}

type ThreadModel struct {
	ID           int64
	BoardID      int64
	Title        string
	PostCount    int
	CreatedAt    time.Time
	LastPostedAt time.Time
}

func (m ThreadModel) ToEntity() *entity.Thread {
	return &entity.Thread{
		ID:           m.ID,
		BoardID:      m.BoardID,
		Title:        m.Title,
		PostCount:    m.PostCount,
		CreatedAt:    m.CreatedAt,
		LastPostedAt: m.LastPostedAt,
	}
}

type PostModel struct {
	ID        int64
	ThreadID  int64
	Number    int
	Author    string
	Body      string
	Sage      bool
	CreatedAt time.Time
}

func (m PostModel) ToEntity() *entity.Post {
	return &entity.Post{
		ID:        m.ID,
		ThreadID:  m.ThreadID,
		Number:    m.Number,
		Author:    m.Author,
		Body:      m.Body,
		Sage:      m.Sage,
		CreatedAt: m.CreatedAt,
	}
}
