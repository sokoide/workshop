package domain

import (
	"time"
)

// Trade represents a crypto transaction event.
type Trade struct {
	ID             string    `json:"id"`
	Symbol         string    `json:"symbol"`
	TargetCurrency string    `json:"target_currency"`
	Price          float64   `json:"price"`
	Amount         float64   `json:"amount"`
	Timestamp      time.Time `json:"timestamp"`
}
