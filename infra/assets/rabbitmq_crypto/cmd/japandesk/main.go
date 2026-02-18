package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/internal/domain"
	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/internal/infra/rabbitmq"
	"github.com/sokoide/workshop/infra/assets/rabbitmq_crypto/internal/usecase"
)

func main() {
	url := "amqp://guest:guest@localhost:5672/"
	conn, ch, err := rabbitmq.SetupConn(url)
	if err != nil {
		log.Fatalf("Failed to setup RabbitMQ: %v", err)
	}
	defer conn.Close()
	defer ch.Close()

	sub := rabbitmq.NewSubscriber(ch)
	obs := usecase.NewTradeObserver(sub)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("Japan Desk starting (market.*.jpy)...")
	err = obs.Start(ctx, "market.*.jpy", func(trade domain.Trade) error {
		log.Printf("[JPY] %s/JPY: %.2f (Amount: %.4f)", trade.Symbol, trade.Price, trade.Amount)
		return nil
	})
	if err != nil {
		log.Fatalf("Observer error: %v", err)
	}

	<-ctx.Done()
	log.Println("Japan Desk stopped.")
}