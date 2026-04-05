package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sokoide/workshop/infra/assets/rate_limiting/usecase"
)

func main() {
	algo := flag.String("algorithm", "fixed-window", "Algorithm: fixed-window, sliding-window, token-bucket")
	user := flag.String("user", "user1", "User ID")
	limit := flag.Int64("limit", 5, "Request limit")
	window := flag.Duration("window", 10*time.Second, "Time window")
	capacity := flag.Int64("capacity", 10, "Token bucket capacity")
	rate := flag.Float64("rate", 1.0, "Token refill rate per second")
	flag.Parse()

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx := context.Background()

	var allowed bool
	var err error

	switch *algo {
	case "fixed-window":
		limiter := usecase.NewFixedWindowLimiter(rdb, *limit, *window)
		allowed, err = limiter.Allow(ctx, *user)
	case "sliding-window":
		limiter := usecase.NewSlidingWindowLimiter(rdb, *limit, *window)
		allowed, err = limiter.Allow(ctx, *user)
	case "token-bucket":
		limiter := usecase.NewTokenBucketLimiter(rdb, *capacity, *rate)
		allowed, err = limiter.Allow(ctx, *user)
	default:
		log.Fatalf("Unknown algorithm: %s", *algo)
	}

	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	if allowed {
		fmt.Printf("[%s] Allowed: User %s\n", *algo, *user)
	} else {
		fmt.Printf("[%s] Denied: User %s (Rate limit exceeded)\n", *algo, *user)
	}
}
