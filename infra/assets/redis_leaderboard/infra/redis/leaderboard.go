package redis

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/sokoide/workshop/leaderboard/domain"
)

type RedisLeaderboardRepository struct {
	client *redis.Client
	key    string // Sorted Set key
	banKey string // Set key for banned users
}

func NewRedisLeaderboardRepository(client *redis.Client, key, banKey string) *RedisLeaderboardRepository {
	return &RedisLeaderboardRepository{
		client: client,
		key:    key,
		banKey: banKey,
	}
}

func (r *RedisLeaderboardRepository) AddScore(ctx context.Context, userID string, score float64) error {
	return r.client.ZAdd(ctx, r.key, redis.Z{
		Score:  score,
		Member: userID,
	}).Err()
}

func (r *RedisLeaderboardRepository) GetTopRankers(ctx context.Context, n int64) ([]domain.UserScore, error) {
	zs, err := r.client.ZRevRangeWithScores(ctx, r.key, 0, n-1).Result()
	if err != nil {
		return nil, err
	}

	result := make([]domain.UserScore, 0, len(zs))
	for i, z := range zs {
		userID, ok := z.Member.(string)
		if !ok {
			return nil, fmt.Errorf("invalid member type at index %d: expected string, got %T", i, z.Member)
		}
		result = append(result, domain.UserScore{
			UserID: userID,
			Score:  z.Score,
			Rank:   int64(i + 1),
		})
	}
	return result, nil
}

func (r *RedisLeaderboardRepository) GetRank(ctx context.Context, userID string) (int64, error) {
	rank, err := r.client.ZRevRank(ctx, r.key, userID).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil // Not found
		}
		return 0, err
	}
	return rank + 1, nil // Convert to 1-based
}

func (r *RedisLeaderboardRepository) BanUser(ctx context.Context, userID string) error {
	return r.client.SAdd(ctx, r.banKey, userID).Err()
}

func (r *RedisLeaderboardRepository) IsBanned(ctx context.Context, userID string) (bool, error) {
	return r.client.SIsMember(ctx, r.banKey, userID).Result()
}
