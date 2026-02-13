package usecase

import (
	"context"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/internal/domain"
)

// MarketSimulator generates random crypto trade events.
type MarketSimulator struct {
	pub domain.TradePublisher
}

// NewMarketSimulator creates a new MarketSimulator.
func NewMarketSimulator(pub domain.TradePublisher) *MarketSimulator {
	return &MarketSimulator{pub: pub}
}

var (
	symbols = []string{"BTC", "ETH", "SOL", "ADA"}
	targets = []string{"USD", "JPY", "EUR"}
)

// Run starts the simulation loop.
func (s *MarketSimulator) Run(ctx context.Context, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			trade := domain.Trade{
				ID:             uuid.New().String(),
				Symbol:         symbols[rand.Intn(len(symbols))],
				TargetCurrency: targets[rand.Intn(len(targets))],
				Price:          rand.Float64()*50000 + 10,
				Amount:         rand.Float64() * 5,
				Timestamp:      time.Now(),
			}

			if err := s.pub.Publish(ctx, trade); err != nil {
				return err
			}
		}
	}
}
