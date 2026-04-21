package entity

import (
	"time"

	"github.com/sokoide/cleanarch1/internal/domain"
)

type Thread struct {
	ID          int64
	BoardID     int64
	Title       string
	PostCount   int
	CreatedAt   time.Time
	LastPostedAt time.Time
}

func NewThread(boardID int64, title string) (*Thread, error) {
	if title == "" {
		return nil, domain.ErrEmptyTitle
	}
	now := time.Now()
	return &Thread{
		BoardID:      boardID,
		Title:        title,
		PostCount:    0,
		CreatedAt:    now,
		LastPostedAt: now,
	}, nil
}

func (t *Thread) Bump(postedAt time.Time, sage bool) {
	t.PostCount++
	if !sage {
		t.LastPostedAt = postedAt
	}
}
