package usecase

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
)

type FixedWindowLimiter struct {
	client *redis.Client
	limit  int64
	window time.Duration
}

func NewFixedWindowLimiter(c *redis.Client, l int64, w time.Duration) *FixedWindowLimiter {
	return &FixedWindowLimiter{client: c, limit: l, window: w}
}

func (f *FixedWindowLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	windowSec := int64(f.window.Seconds())
	key := fmt.Sprintf("rl:fixed:%s:%d", userID, time.Now().Unix()/windowSec)

	count, err := f.client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if count == 1 {
		f.client.Expire(ctx, key, f.window)
	}
	return count <= f.limit, nil
}

type SlidingWindowLimiter struct {
	client *redis.Client
	limit  int64
	window time.Duration
}

func NewSlidingWindowLimiter(c *redis.Client, l int64, w time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{client: c, limit: l, window: w}
}

func (s *SlidingWindowLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	now := time.Now().Unix()
	windowSec := int64(s.window.Seconds())
	currWindow := now / windowSec
	prevWindow := currWindow - 1

	keyCurr := fmt.Sprintf("rl:sliding:%s:%d", userID, currWindow)
	keyPrev := fmt.Sprintf("rl:sliding:%s:%d", userID, prevWindow)

	currCount, err := s.client.Incr(ctx, keyCurr).Result()
	if err != nil {
		return false, err
	}
	if currCount == 1 {
		s.client.Expire(ctx, keyCurr, s.window*2)
	}

	prevCount, _ := s.client.Get(ctx, keyPrev).Int64()
	elapsed := float64(now % windowSec)
	weight := 1.0 - (elapsed / float64(windowSec))
	weightedCount := float64(prevCount)*weight + float64(currCount)

	return weightedCount <= float64(s.limit), nil
}

type TokenBucketLimiter struct {
	client   *redis.Client
	capacity int64
	rate     float64
}

func NewTokenBucketLimiter(c *redis.Client, cap int64, r float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{client: c, capacity: cap, rate: r}
}

func (t *TokenBucketLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	keyTokens := fmt.Sprintf("rl:tb:tokens:%s", userID)
	keyTs := fmt.Sprintf("rl:tb:ts:%s", userID)

	now := time.Now().UnixNano()
	val, _ := t.client.Get(ctx, keyTokens).Float64()
	lastTs, _ := t.client.Get(ctx, keyTs).Int64()

	if lastTs == 0 {
		val = float64(t.capacity)
	} else {
		elapsed := float64(now-lastTs) / float64(time.Second)
		val = math.Min(float64(t.capacity), val+(elapsed*t.rate))
	}

	if val >= 1.0 {
		pipe := t.client.Pipeline()
		pipe.Set(ctx, keyTokens, val-1.0, 0)
		pipe.Set(ctx, keyTs, now, 0)
		_, err := pipe.Exec(ctx)
		return err == nil, err
	}
	return false, nil
}
