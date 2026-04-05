package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/sokoide/workshop/software/assets/idempotency/domain"
	"github.com/sokoide/workshop/software/assets/idempotency/infra"
	"github.com/sokoide/workshop/software/assets/idempotency/usecase"
)

type RedisAccountRepo struct {
	client *redis.Client
}

func (r *RedisAccountRepo) Get(ctx context.Context, id string) (*domain.Account, error) {
	val, err := r.client.Get(ctx, "balance:"+id).Result()
	if err == redis.Nil {
		// Initialize with 1000
		r.client.Set(ctx, "balance:"+id, "1000", 0)
		return &domain.Account{ID: id, Balance: 1000}, nil
	}
	balance, _ := strconv.Atoi(val)
	return &domain.Account{ID: id, Balance: balance}, nil
}

func (r *RedisAccountRepo) UpdateBalance(ctx context.Context, id string, amount int) error {
	return r.client.Set(ctx, "balance:"+id, strconv.Itoa(amount), 0).Err()
}

func main() {
	user := flag.String("user", "user1", "User ID")
	amount := flag.Int("amount", 100, "Amount")
	key := flag.String("idempotency-key", "", "Idempotency Key")
	flag.Parse()

	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()

	store := infra.NewRedisIdempotencyStore(rdb)
	repo := &RedisAccountRepo{client: rdb}

	uc := usecase.NewChargeUsecase(repo, store)

	res, err := uc.Execute(ctx, usecase.ChargeRequest{
		IdempotencyKey: *key,
		UserID:         *user,
		Amount:         *amount,
	})

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Result: %+v\n", res)
}
