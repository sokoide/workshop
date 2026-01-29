package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
	redisrepo "github.com/sokoide/workshop/leaderboard/infra/redis"
	"github.com/sokoide/workshop/leaderboard/usecase"
)

func main() {
	// Parse flags
	var action, user string
	var score float64
	var n int64

	flag.StringVar(&action, "action", "", "Action to perform: add|top|rank|ban")
	flag.StringVar(&user, "user", "", "User ID")
	flag.Float64Var(&score, "score", 0, "Score to add")
	flag.Int64Var(&n, "n", 10, "Number of top rankers to show")

	flag.Parse()

	// Require action
	if action == "" {
		flag.Usage()
		os.Exit(1)
	}

	// 1. Setup Redis Client
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// 2. Dependency Injection (DI)
	repo := redisrepo.NewRedisLeaderboardRepository(client, "game_leaderboard", "banned_users")
	uc := usecase.NewLeaderboardUsecase(repo)

	ctx := context.Background()

	switch action {
	case "add":
		if user == "" {
			fmt.Fprintln(os.Stderr, "Error: add requires -user flag")
			os.Exit(1)
		}
		if score == 0 {
			fmt.Fprintln(os.Stderr, "Error: add requires -score flag (non-zero)")
			os.Exit(1)
		}
		if err := uc.AddScore(ctx, user, score); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Added score %.2f for user %s\n", score, user)

	case "top":
		rankers, err := uc.GetTopRankers(ctx, n)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("--- Top %d Rankers ---\n", n)
		for _, r := range rankers {
			fmt.Printf("%d. %s: %.2f\n", r.Rank, r.UserID, r.Score)
		}

	case "rank":
		if user == "" {
			fmt.Fprintln(os.Stderr, "Error: rank requires -user flag")
			os.Exit(1)
		}
		rank, err := uc.GetRank(ctx, user)
		if err != nil {
			log.Fatal(err)
		}
		if rank == 0 {
			fmt.Printf("User %s not found in leaderboard\n", user)
		} else {
			fmt.Printf("User %s is ranked #%d\n", user, rank)
		}

	case "ban":
		if user == "" {
			fmt.Fprintln(os.Stderr, "Error: ban requires -user flag")
			os.Exit(1)
		}
		if err := uc.BanUser(ctx, user); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("User %s has been banned\n", user)

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown action %q\n", action)
		flag.Usage()
		os.Exit(1)
	}
}
