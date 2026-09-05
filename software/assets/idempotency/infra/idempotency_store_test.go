package infra

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestExpiredOwnerCannotUnlockNewOwner(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	store := NewRedisIdempotencyStore(client)
	ctx := context.Background()
	first, err := store.Lock(ctx, "key")
	if err != nil || first == "" {
		t.Fatalf("first lock: %q %v", first, err)
	}
	if busy, err := store.Lock(ctx, "key"); err != nil || busy != "" {
		t.Fatalf("busy lock: %q %v", busy, err)
	}
	server.FastForward(31 * time.Second)
	second, err := store.Lock(ctx, "key")
	if err != nil || second == "" || first == second {
		t.Fatalf("second lock: %q %v", second, err)
	}
	if err := store.Unlock(ctx, "key", first); err != nil {
		t.Fatal(err)
	}
	if got, _ := server.Get("lock:key"); got != second {
		t.Fatal("expired owner removed a new lock")
	}
	if err := store.Unlock(ctx, "key", second); err != nil {
		t.Fatal(err)
	}
	if server.Exists("lock:key") {
		t.Fatal("owner failed to release lock")
	}
}
