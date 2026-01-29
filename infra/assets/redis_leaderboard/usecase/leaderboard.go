package usecase

import (
	"context"

	"github.com/sokoide/workshop/leaderboard/domain"
)

type LeaderboardUsecase struct {
	repo domain.LeaderboardRepository
}

func NewLeaderboardUsecase(repo domain.LeaderboardRepository) *LeaderboardUsecase {
	return &LeaderboardUsecase{
		repo: repo,
	}
}

func (u *LeaderboardUsecase) AddScore(ctx context.Context, userID string, score float64) error {
	return u.repo.AddScore(ctx, userID, score)
}

func (u *LeaderboardUsecase) GetTopRankers(ctx context.Context, n int64) ([]domain.UserScore, error) {
	// The Usecase handles the business logic: filtering banned users.
	// Since Redis ZREVRANGE might return users who are banned, we filter them here.
	// Note: For large N, this might need more sophisticated handling (like over-fetching from Redis),
	// but for this workshop, simple filtering is sufficient.

	zs, err := u.repo.GetTopRankers(ctx, n)
	if err != nil {
		return nil, err
	}

	result := make([]domain.UserScore, 0, len(zs))
	rank := int64(1)
	for _, z := range zs {
		banned, err := u.repo.IsBanned(ctx, z.UserID)
		if err != nil {
			return nil, err
		}
		if banned {
			continue
		}
		z.Rank = rank
		result = append(result, z)
		rank++
	}
	return result, nil
}

func (u *LeaderboardUsecase) GetRank(ctx context.Context, userID string) (int64, error) {
	return u.repo.GetRank(ctx, userID)
}

func (u *LeaderboardUsecase) BanUser(ctx context.Context, userID string) error {
	return u.repo.BanUser(ctx, userID)
}
