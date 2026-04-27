package domain

import "errors"

var (
	ErrBoardNotFound           = errors.New("board not found")
	ErrThreadNotFound          = errors.New("thread not found")
	ErrEmptyBody               = errors.New("post body must not be empty")
	ErrEmptyTitle              = errors.New("thread title must not be empty")
	ErrEmptyBoardName          = errors.New("board name must not be empty")
	ErrEmptyBoardDescription   = errors.New("board description must not be empty")
)
