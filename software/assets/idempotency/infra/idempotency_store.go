package infra

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisIdempotencyStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisIdempotencyStore(client *redis.Client) *RedisIdempotencyStore {
	return &RedisIdempotencyStore{client: client, ttl: 24 * time.Hour}
}

func (s *RedisIdempotencyStore) GetResult(ctx context.Context, key string) ([]byte, error) {
	val, err := s.client.Get(ctx, "idemp:"+key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	return val, err
}

func (s *RedisIdempotencyStore) SaveResult(ctx context.Context, key string, result []byte) error {
	return s.client.Set(ctx, "idemp:"+key, result, s.ttl).Err()
}

func (s *RedisIdempotencyStore) Lock(ctx context.Context, key string) (bool, error) {
	return s.client.SetNX(ctx, "lock:"+key, "1", 30*time.Second).Result()
}

func (s *RedisIdempotencyStore) Unlock(ctx context.Context, key string) error {
	return s.client.Del(ctx, "lock:"+key).Err()
}
