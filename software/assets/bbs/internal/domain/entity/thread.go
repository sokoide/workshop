package entity

import (
	"errors"
	"time"

	"github.com/sokoide/cleanarch1/internal/domain"
)

type Thread struct {
	ID           int64
	BoardID      int64
	Title        string
	Owner        string // 追加: スレ主
	OwnerOnly    bool   // 追加: スレ主限定モード
	PostCount    int
	CreatedAt    time.Time
	LastPostedAt time.Time
}

func NewThread(boardID int64, title string, owner string) (*Thread, error) {
	if title == "" {
		return nil, domain.ErrEmptyTitle
	}
	if owner == "" {
		owner = DefaultAuthor
	}
	now := time.Now()
	return &Thread{
		BoardID:      boardID,
		Title:        title,
		Owner:        owner,
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

func (t *Thread) CanPost(author string) bool {
	if !t.OwnerOnly {
		return true
	}
	return t.Owner == author
}

func (t *Thread) EnableOwnerOnlyMode(owner string) error {
	if owner == "" {
		return errors.New("owner must not be empty when owner-only mode is enabled")
	}
	t.OwnerOnly = true
	t.Owner = owner
	return nil
}
