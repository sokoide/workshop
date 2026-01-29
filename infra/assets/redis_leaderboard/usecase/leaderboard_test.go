package usecase

import (
	"context"

	"github.com/sokoide/workshop/leaderboard/domain"
	"testing"
)

type mockRepository struct {
	scores    []domain.UserScore
	bannedMap map[string]bool
}

func (m *mockRepository) AddScore(ctx context.Context, userID string, score float64) error {
	return nil
}
func (m *mockRepository) GetTopRankers(ctx context.Context, n int64) ([]domain.UserScore, error) {
	return m.scores, nil
}
func (m *mockRepository) GetRank(ctx context.Context, userID string) (int64, error) { return 0, nil }
func (m *mockRepository) BanUser(ctx context.Context, userID string) error          { return nil }
func (m *mockRepository) IsBanned(ctx context.Context, userID string) (bool, error) {
	return m.bannedMap[userID], nil
}

func TestGetTopRankersFiltering(t *testing.T) {
	repo := &mockRepository{
		scores: []domain.UserScore{
			{UserID: "user3", Score: 300, Rank: 1},
			{UserID: "user2", Score: 200, Rank: 2},
			{UserID: "user1", Score: 100, Rank: 3},
		},
		bannedMap: map[string]bool{
			"user2": true,
		},
	}
	uc := NewLeaderboardUsecase(repo)

	rankers, err := uc.GetTopRankers(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(rankers) != 2 {
		t.Errorf("expected 2 rankers, got %d", len(rankers))
	}

	if rankers[0].UserID != "user3" || rankers[1].UserID != "user1" {
		t.Errorf("incorrect filtering: %+v", rankers)
	}

	if rankers[0].Rank != 1 || rankers[1].Rank != 2 {
		t.Errorf("incorrect rank re-assignment: %+v", rankers)
	}
}
