package usecase

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestInvalidWindowDoesNotPanic(t *testing.T) {
	for _, window := range []time.Duration{0, -time.Second, time.Nanosecond} {
		if _, err := NewFixedWindowLimiter(nil, 10, window).Allow(context.Background(), "u"); err == nil {
			t.Fatal("invalid fixed window accepted")
		}
		if _, err := NewSlidingWindowLimiter(nil, 10, window).Allow(context.Background(), "u"); err == nil {
			t.Fatal("invalid sliding window accepted")
		}
	}
}
func TestConcurrentTokenBucket(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	limiter := NewTokenBucketLimiter(client, 5, 0.001)
	var accepted atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := limiter.Allow(context.Background(), "u")
			if err != nil {
				t.Error(err)
			}
			if ok {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if accepted.Load() != 5 {
		t.Fatalf("accepted %d requests from a bucket of 5", accepted.Load())
	}
}
func TestRedisFailuresAreNotAllowed(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	server.SetError("ERR unavailable")
	limiter := NewTokenBucketLimiter(client, 1, 1)
	if allowed, err := limiter.Allow(context.Background(), "u"); allowed || err == nil {
		t.Fatalf("allowed=%v err=%v", allowed, err)
	}
}

func TestWindowCountersExpireAndEnforceLimits(t *testing.T) {
	for _, algorithm := range []string{"fixed", "sliding"} {
		t.Run(algorithm, func(t *testing.T) {
			server := miniredis.RunT(t)
			client := redis.NewClient(&redis.Options{Addr: server.Addr()})
			defer client.Close()
			var allow func(context.Context, string) (bool, error)
			if algorithm == "fixed" {
				allow = NewFixedWindowLimiter(client, 2, time.Hour).Allow
			} else {
				allow = NewSlidingWindowLimiter(client, 2, time.Hour).Allow
			}
			for i := 0; i < 3; i++ {
				ok, err := allow(context.Background(), "u")
				if err != nil || ok != (i < 2) {
					t.Fatalf("request %d: %v %v", i, ok, err)
				}
			}
			for _, key := range server.Keys() {
				if server.TTL(key) <= 0 {
					t.Fatalf("counter %s has no expiry", key)
				}
			}
			server.SetError("ERR unavailable")
			if ok, err := allow(context.Background(), "u"); ok || err == nil {
				t.Fatalf("Redis failure: %v %v", ok, err)
			}
		})
	}
}

func TestTokenRefill(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	now := time.Unix(1000, 0)
	server.SetTime(now)
	limiter := NewTokenBucketLimiter(client, 1, 1)
	if ok, err := limiter.Allow(context.Background(), "u"); !ok || err != nil {
		t.Fatalf("first request: %v %v", ok, err)
	}
	if ok, _ := limiter.Allow(context.Background(), "u"); ok {
		t.Fatal("empty bucket allowed")
	}
	server.SetTime(now.Add(time.Second))
	if ok, err := limiter.Allow(context.Background(), "u"); !ok || err != nil {
		t.Fatalf("refill: %v %v", ok, err)
	}
}
