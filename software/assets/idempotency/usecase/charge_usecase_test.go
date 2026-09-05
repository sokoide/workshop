package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/sokoide/workshop/software/assets/idempotency/domain"
)

type accountRepo struct{ balance, writes int }

func (r *accountRepo) Get(context.Context, string) (*domain.Account, error) {
	return &domain.Account{Balance: r.balance}, nil
}
func (r *accountRepo) UpdateBalance(_ context.Context, _ string, n int) error {
	r.balance = n
	r.writes++
	return nil
}

type resultStore struct {
	result           []byte
	readErr, saveErr error
	reads            int
	afterLock        []byte
}

func (s *resultStore) GetResult(context.Context, string) ([]byte, error) {
	s.reads++
	return s.result, s.readErr
}
func (s *resultStore) SaveResult(_ context.Context, _ string, b []byte) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.result = b
	return nil
}
func (s *resultStore) Lock(context.Context, string) (string, error) {
	if s.afterLock != nil {
		s.result = s.afterLock
	}
	return "token", nil
}
func (s *resultStore) Unlock(context.Context, string, string) error { return nil }

func TestChargeRejectsInvalidInput(t *testing.T) {
	for _, req := range []ChargeRequest{{UserID: "u", Amount: -1}, {UserID: "u", Amount: 0}, {Amount: 1}} {
		repo := &accountRepo{balance: 1000}
		uc := NewChargeUsecase(repo, &resultStore{})
		if _, err := uc.Execute(context.Background(), req); err == nil || repo.writes != 0 {
			t.Fatalf("invalid request processed: %+v", req)
		}
	}
}
func TestChargeFailsClosed(t *testing.T) {
	for _, store := range []*resultStore{{readErr: errors.New("offline")}, {result: []byte("bad json")}} {
		repo := &accountRepo{balance: 1000}
		uc := NewChargeUsecase(repo, store)
		if _, err := uc.Execute(context.Background(), ChargeRequest{UserID: "u", Amount: 100, IdempotencyKey: "key"}); err == nil || repo.writes != 0 {
			t.Fatal("read failure or corrupt result must not charge")
		}
	}
}
func TestResultSaveFailureIsReported(t *testing.T) {
	problem := errors.New("save failed")
	uc := NewChargeUsecase(&accountRepo{balance: 1000}, &resultStore{saveErr: problem})
	if _, err := uc.Execute(context.Background(), ChargeRequest{UserID: "u", Amount: 100, IdempotencyKey: "key"}); !errors.Is(err, problem) {
		t.Fatalf("save error lost: %v", err)
	}
}
func TestRechecksResultAfterLock(t *testing.T) {
	cached, _ := json.Marshal(storedCharge{Request: ChargeRequest{UserID: "u", Amount: 100, IdempotencyKey: "key"}, Response: ChargeResponse{Status: "success", Balance: 900}})
	repo := &accountRepo{balance: 900}
	store := &resultStore{afterLock: cached}
	got, err := NewChargeUsecase(repo, store).Execute(context.Background(), ChargeRequest{UserID: "u", Amount: 100, IdempotencyKey: "key"})
	if err != nil || got.Balance != 900 || repo.writes != 0 {
		t.Fatalf("charged after another request completed: %+v %v writes=%d", got, err, repo.writes)
	}
}

func TestRetryAndConflictingKey(t *testing.T) {
	repo := &accountRepo{balance: 1000}
	uc := NewChargeUsecase(repo, &resultStore{})
	req := ChargeRequest{UserID: "u", Amount: 100, IdempotencyKey: "key"}
	if _, err := uc.Execute(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	got, err := uc.Execute(context.Background(), req)
	if err != nil || got.Balance != 900 || repo.writes != 1 {
		t.Fatalf("retry changed balance: %+v %v", got, err)
	}
	req.Amount = 200
	if _, err := uc.Execute(context.Background(), req); err == nil || repo.writes != 1 {
		t.Fatal("conflicting request accepted")
	}
}
