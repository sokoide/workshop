package entity

import "errors"

var (
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrOrderNotFound     = errors.New("order not found")
	ErrInvalidProductID  = errors.New("invalid product id")
	ErrInvalidQuantity   = errors.New("invalid quantity")
)
