package domain

import (
	"testing"
	"time"
)

func TestTrade(t *testing.T) {
	trade := Trade{
		ID:             "test-id",
		Symbol:         "BTC",
		TargetCurrency: "USD",
		Price:          50000.0,
		Amount:         1.5,
		Timestamp:      time.Now(),
	}

	if trade.Symbol != "BTC" {
		t.Errorf("expected BTC, got %s", trade.Symbol)
	}
}
