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

var fixedWindowScript = redis.NewScript(`
local count = redis.call('INCR', KEYS[1])
if count == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1]) end
return count
`)

func (f *FixedWindowLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	if f.limit <= 0 || f.window < time.Millisecond {
		return false, fmt.Errorf("positive limit and window of at least 1ms are required")
	}
	windowMS := f.window.Milliseconds()
	key := fmt.Sprintf("rl:fixed:%s:%d", userID, time.Now().UnixMilli()/windowMS)
	count, err := fixedWindowScript.Run(ctx, f.client, []string{key}, windowMS).Int64()
	return err == nil && count <= f.limit, err
}

type SlidingWindowLimiter struct {
	client *redis.Client
	limit  int64
	window time.Duration
}

func NewSlidingWindowLimiter(c *redis.Client, l int64, w time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{client: c, limit: l, window: w}
}

var slidingWindowScript = redis.NewScript(`
local previous = tonumber(redis.call('GET', KEYS[2]) or '0')
local current = redis.call('INCR', KEYS[1])
if current == 1 then redis.call('PEXPIRE', KEYS[1], ARGV[1] * 2) end
local weight = 1 - tonumber(ARGV[2]) / tonumber(ARGV[1])
if previous * weight + current <= tonumber(ARGV[3]) then return 1 end
return 0
`)

func (s *SlidingWindowLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	if s.limit <= 0 || s.window < time.Millisecond || s.window > time.Duration(math.MaxInt64/2) {
		return false, fmt.Errorf("positive limit and window between 1ms and half the maximum duration are required")
	}
	now := time.Now().UnixMilli()
	windowMS := s.window.Milliseconds()
	current := now / windowMS
	keys := []string{fmt.Sprintf("rl:sliding:%s:%d", userID, current), fmt.Sprintf("rl:sliding:%s:%d", userID, current-1)}
	allowed, err := slidingWindowScript.Run(ctx, s.client, keys, windowMS, now%windowMS, s.limit).Int64()
	return allowed == 1 && err == nil, err
}

type TokenBucketLimiter struct {
	client   *redis.Client
	capacity int64
	rate     float64
}

func NewTokenBucketLimiter(c *redis.Client, cap int64, r float64) *TokenBucketLimiter {
	return &TokenBucketLimiter{client: c, capacity: cap, rate: r}
}

var tokenBucketScript = redis.NewScript(`
local capacity, rate = tonumber(ARGV[1]), tonumber(ARGV[2])
local clock = redis.call('TIME')
local now = tonumber(clock[1]) + tonumber(clock[2]) / 1000000
local tokens = tonumber(redis.call('HGET', KEYS[1], 'tokens') or capacity)
local last = tonumber(redis.call('HGET', KEYS[1], 'updated') or now)
tokens = math.min(capacity, tokens + math.max(0, now - last) * rate)
local allowed = 0
if tokens >= 1 then tokens = tokens - 1; allowed = 1 end
redis.call('HSET', KEYS[1], 'tokens', tokens, 'updated', now)
redis.call('PEXPIRE', KEYS[1], math.ceil(capacity / rate * 1000))
return allowed
`)

func (t *TokenBucketLimiter) Allow(ctx context.Context, userID string) (bool, error) {
	if t.capacity <= 0 || t.rate <= 0 || math.IsNaN(t.rate) || math.IsInf(t.rate, 0) || float64(t.capacity)/t.rate*1000 >= float64(math.MaxInt64) {
		return false, fmt.Errorf("positive capacity and finite positive refill rate with representable expiry are required")
	}
	allowed, err := tokenBucketScript.Run(ctx, t.client, []string{"rl:tb:" + userID}, t.capacity, t.rate).Int64()
	return allowed == 1 && err == nil, err
}
