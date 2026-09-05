package main

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestAccountInitializationAndErrors(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	repo := &RedisAccountRepo{client: client}
	ctx := context.Background()
	account, err := repo.Get(ctx, "u")
	if err != nil || account.Balance != 1000 {
		t.Fatalf("initialize: %+v %v", account, err)
	}
	server.Set("balance:u", "900")
	account, err = repo.Get(ctx, "u")
	if err != nil || account.Balance != 900 {
		t.Fatalf("overwrote balance: %+v %v", account, err)
	}
	server.Set("balance:u", "invalid")
	if _, err := repo.Get(ctx, "u"); err == nil {
		t.Fatal("accepted invalid stored balance")
	}
	server.SetError("ERR unavailable")
	if _, err := repo.Get(ctx, "u"); err == nil {
		t.Fatal("ignored Redis failure")
	}
}
