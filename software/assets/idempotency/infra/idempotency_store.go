package infra

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// Lock returns an ownership token, or an empty string if another request owns the lock.
func (s *RedisIdempotencyStore) Lock(ctx context.Context, key string) (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	token := hex.EncodeToString(random[:])
	ok, err := s.client.SetNX(ctx, "lock:"+key, token, 30*time.Second).Result()
	if err != nil || !ok {
		return "", err
	}
	return token, nil
}

var unlockScript = redis.NewScript(`
if redis.call('GET', KEYS[1]) == ARGV[1] then
 return redis.call('DEL', KEYS[1])
end
return 0
`)

func (s *RedisIdempotencyStore) Unlock(ctx context.Context, key, token string) error {
	return unlockScript.Run(ctx, s.client, []string{"lock:" + key}, token).Err()
}
