package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	pub := rabbitmq.NewPublisher(ch)
	sim := usecase.NewMarketSimulator(pub)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("Ticker starting... (Ctrl+C to stop)")
	if err := sim.Run(ctx, 500*time.Millisecond); err != nil && err != context.Canceled {
		log.Fatalf("Ticker error: %v", err)
	}
	log.Println("Ticker stopped.")
}
