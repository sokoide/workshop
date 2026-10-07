package domain

import "errors"

// ErrUserNotFound is returned when the repository cannot find the user.
// It belongs to the Domain layer so every caller can test for it with
// errors.Is, no matter which infrastructure produced it.
var ErrUserNotFound = errors.New("user not found")
