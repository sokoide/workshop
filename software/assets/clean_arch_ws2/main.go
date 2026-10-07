package main

import (
	"context"
	"fmt"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/adapters/infra"
	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/domain"
	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/usecase"
	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/usecase/port"
)

func main() {
	ctx := context.Background()

	// --- Exercise 1: swap the notification channel ---------------------
	// ApproveUseCase depends on the port.NotificationGateway port, so wiring
	// a different implementation is a one-line change here.
	fmt.Println("== Exercise 1: notification channel swap ==")
	approveVia(ctx, infra.NewEmailNotifier("noreply@example.com"))
	approveVia(ctx, infra.NewSlackNotifier("https://hooks.slack.com/services/T000/B000/XXXX"))

	// --- Exercise 2: transparent cache (Decorator pattern) -------------
	fmt.Println("\n== Exercise 2: transparent cache ==")
	realRepo := infra.NewSQLUserRepository()
	cachedRepo := infra.NewCachedUserRepository(realRepo, infra.NewMemoryCacheStore())

	// The UseCase sees only domain.UserRepository — it cannot tell whether
	// the repository it received is cached or not.
	uc := usecase.NewCheckVeteranUseCase(cachedRepo, domain.VeteranService{})

	for lookup := 1; lookup <= 3; lookup++ {
		veteran, err := uc.Execute(ctx, "alice")
		if err != nil {
			fmt.Printf("lookup #%d: error: %v\n", lookup, err)
			continue
		}
		fmt.Printf("lookup #%d: alice veteran=%v (SQL queries so far: %d)\n",
			lookup, veteran, realRepo.QueryCount())
	}
}

// approveVia wires the given notification gateway into the use case. It
// works with ANY implementation of port.NotificationGateway.
func approveVia(ctx context.Context, notifier port.NotificationGateway) {
	uc := usecase.NewApproveUseCase(infra.NewSQLUserRepository(), notifier)
	if err := uc.Execute(ctx, "alice"); err != nil {
		fmt.Printf("approve: error: %v\n", err)
	}
}
