package main

import (
	"context"
	"encoding/json"
	"fmt"
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

	log.Println("Logger starting (market.#)...")
	err = obs.Start(ctx, "market.#", func(trade domain.Trade) error {
		b, _ := json.Marshal(trade)
		fmt.Println(string(b))
		return nil
	})
	if err != nil {
		log.Fatalf("Observer error: %v", err)
	}

	<-ctx.Done()
	log.Println("Logger stopped.")
}
