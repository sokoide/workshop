package redis

import (
	"context"
	"testing"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
)

func TestAddScore(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		score   float64
		wantErr bool
	}{
		{
			name:    "normal score",
			userID:  "user1",
			score:   100.0,
			wantErr: false,
		},
		{
			name:    "zero score",
			userID:  "user2",
			score:   0.0,
			wantErr: false,
		},
		{
			name:    "negative score",
			userID:  "user3",
			score:   -50.0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			repo := NewRedisLeaderboardRepository(db, "leaderboard", "banned")

			ctx := context.Background()
			mock.ExpectZAdd("leaderboard", redis.Z{
				Score:  tt.score,
				Member: tt.userID,
			}).SetVal(1)

			err := repo.AddScore(ctx, tt.userID, tt.score)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddScore() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestGetTopRankers(t *testing.T) {
	tests := []struct {
		name    string
		n       int64
		mockVal []redis.Z
		wantLen int
		wantErr bool
	}{
		{
			name: "three rankers",
			n:    3,
			mockVal: []redis.Z{
				{Score: 300, Member: "user3"},
				{Score: 200, Member: "user2"},
				{Score: 100, Member: "user1"},
			},
			wantLen: 3,
			wantErr: false,
		},
		{
			name:    "empty leaderboard",
			n:       10,
			mockVal: []redis.Z{},
			wantLen: 0,
			wantErr: false,
		},
		{
			name: "single ranker",
			n:    1,
			mockVal: []redis.Z{
				{Score: 100, Member: "user1"},
			},
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			repo := NewRedisLeaderboardRepository(db, "leaderboard", "banned")

			ctx := context.Background()
			mock.ExpectZRevRangeWithScores("leaderboard", 0, tt.n-1).SetVal(tt.mockVal)

			rankers, err := repo.GetTopRankers(ctx, tt.n)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTopRankers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(rankers) != tt.wantLen {
				t.Errorf("GetTopRankers() got %d rankers, want %d", len(rankers), tt.wantLen)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestGetRank(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		mockRank    int64
		mockErr     error
		wantRank    int64
		wantErr     bool
	}{
		{
			name:     "user found at rank 2",
			userID:   "user2",
			mockRank: 1,
			mockErr:  nil,
			wantRank: 2,
			wantErr:  false,
		},
		{
			name:     "user at rank 1",
			userID:   "top_user",
			mockRank: 0,
			mockErr:  nil,
			wantRank: 1,
			wantErr:  false,
		},
		{
			name:     "user not found",
			userID:   "nonexistent",
			mockRank: 0,
			mockErr:  redis.Nil,
			wantRank: 0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			repo := NewRedisLeaderboardRepository(db, "leaderboard", "banned")

			ctx := context.Background()
			if tt.mockErr != nil {
				mock.ExpectZRevRank("leaderboard", tt.userID).SetErr(tt.mockErr)
			} else {
				mock.ExpectZRevRank("leaderboard", tt.userID).SetVal(tt.mockRank)
			}

			rank, err := repo.GetRank(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRank() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if rank != tt.wantRank {
				t.Errorf("GetRank() got rank %d, want %d", rank, tt.wantRank)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestBanUser(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "ban cheater",
			userID:  "cheater1",
			wantErr: false,
		},
		{
			name:    "ban empty user",
			userID:  "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			repo := NewRedisLeaderboardRepository(db, "leaderboard", "banned")

			ctx := context.Background()
			mock.ExpectSAdd("banned", tt.userID).SetVal(1)

			err := repo.BanUser(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("BanUser() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestIsBanned(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		mockBanned bool
		wantBanned bool
		wantErr    bool
	}{
		{
			name:       "banned user",
			userID:     "cheater1",
			mockBanned: true,
			wantBanned: true,
			wantErr:    false,
		},
		{
			name:       "clean user",
			userID:     "user1",
			mockBanned: false,
			wantBanned: false,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock := redismock.NewClientMock()
			repo := NewRedisLeaderboardRepository(db, "leaderboard", "banned")

			ctx := context.Background()
			mock.ExpectSIsMember("banned", tt.userID).SetVal(tt.mockBanned)

			banned, err := repo.IsBanned(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsBanned() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if banned != tt.wantBanned {
				t.Errorf("IsBanned() got %v, want %v", banned, tt.wantBanned)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
		})
	}
}
