package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/internal/domain"
)

type mockPublisher struct {
	trades []domain.Trade
}

func (m *mockPublisher) Publish(ctx context.Context, trade domain.Trade) error {
	m.trades = append(m.trades, trade)
	return nil
}

func TestMarketSimulator_Run(t *testing.T) {
	mock := &mockPublisher{}
	sim := NewMarketSimulator(mock)

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	// Run with short interval
	_ = sim.Run(ctx, 100*time.Millisecond)

	// Should have published 2-3 trades
	if len(mock.trades) < 2 {
		t.Errorf("expected at least 2 trades, got %d", len(mock.trades))
	}

	for _, tr := range mock.trades {
		if tr.ID == "" || tr.Symbol == "" || tr.TargetCurrency == "" {
			t.Errorf("invalid trade generated: %+v", tr)
		}
	}
}
