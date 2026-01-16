package main

import (
	"context"
	"log"

	"github.com/sokoide/advent-of-calm-2025/cleanarch/domain/service"
	"github.com/sokoide/advent-of-calm-2025/cleanarch/infra/client"
	"github.com/sokoide/advent-of-calm-2025/cleanarch/infra/messaging"
	"github.com/sokoide/advent-of-calm-2025/cleanarch/infra/repository"
	"github.com/sokoide/advent-of-calm-2025/cleanarch/infra/util"
	"github.com/sokoide/advent-of-calm-2025/cleanarch/usecase"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Application failed: %v", err)
	}
}

func run() error {
	// 1. Setup Infrastructure
	orderRepo := &repository.PostgresOrderRepository{}
	inventoryClient := &client.RestInventoryClient{}
	paymentPub := &messaging.RabbitMQPaymentPublisher{}
	idGen := &util.UUIDGenerator{}

	// 2. Setup Domain Service
	orderDomainSvc := service.NewOrderDomainService(inventoryClient)
	inventoryRepo := &repository.PostgresInventoryRepository{}
	inventoryDomainSvc := service.NewInventoryDomainService(inventoryRepo)

	// 3. Setup Usecase
	createOrderUsecase := usecase.NewCreateOrderUsecase(orderRepo, orderDomainSvc, paymentPub, idGen)
	checkInventoryUsecase := usecase.NewCheckInventoryUsecase(inventoryDomainSvc)
	updateInventoryUsecase := usecase.NewUpdateInventoryUsecase(inventoryDomainSvc)

	// 4. Run Usecase (Customer Flow)
	ctx := context.Background()
	input := usecase.CreateOrderInput{
		CustomerID: "customer-123",
		ProductID:  "product-456",
		Quantity:   1,
		Amount:     99.99,
	}

	err := createOrderUsecase.Execute(ctx, input)
	if err != nil {
		return err
	}

	// 5. Run Usecase (Admin Flow)
	// Admin checks inventory
	checkInput := usecase.CheckInventoryInput{ProductID: "product-456"}
	output, err := checkInventoryUsecase.Execute(ctx, checkInput)
	if err != nil {
		return err
	}
	log.Println("Current stock:", output.Quantity)

	// Admin updates inventory
	updateInput := usecase.UpdateInventoryInput{ProductID: "product-456", Quantity: 150}
	err = updateInventoryUsecase.Execute(ctx, updateInput)
	if err != nil {
		return err
	}
	
	return nil
}
