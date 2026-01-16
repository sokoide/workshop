package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/pkg/domain"
	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/pkg/infra/rabbitmq"
	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/pkg/usecase"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func run() error {
	url := "amqp://guest:guest@localhost:5672/"
	conn, ch, err := rabbitmq.SetupConn(url)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer ch.Close()

	sub := rabbitmq.NewSubscriber(ch)
	obs := usecase.NewTradeObserver(sub)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("Whale Alert starting (market.btc.#)...")
	err = obs.Start(ctx, "market.btc.#", func(trade domain.Trade) error {
		if trade.Amount >= 3.0 {
			log.Printf("🚨 WHALE ALERT: %s bought %.2f %s at %.2f %s", trade.ID[:8], trade.Amount, trade.Symbol, trade.Price, trade.TargetCurrency)
		}
		return nil
	})
	if err != nil {
		return err
	}

	<-ctx.Done()
	log.Println("Whale Alert stopped.")
	return nil
}
