package usecase

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/sokoide/workshop/software/assets/idempotency/domain"
)

type ChargeRequest struct {
	IdempotencyKey string
	UserID         string
	Amount         int
}

type ChargeResponse struct {
	Status  string `json:"status"`
	Balance int    `json:"balance"`
	Source  string `json:"source"`
}

type ChargeUsecase struct {
	repo             domain.AccountRepository
	idempotencyStore domain.IdempotencyStore
}

func NewChargeUsecase(repo domain.AccountRepository, store domain.IdempotencyStore) *ChargeUsecase {
	return &ChargeUsecase{repo: repo, idempotencyStore: store}
}

func (uc *ChargeUsecase) Execute(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
	if req.IdempotencyKey != "" {
		if cached, err := uc.idempotencyStore.GetResult(ctx, req.IdempotencyKey); err == nil && cached != nil {
			var res ChargeResponse
			json.Unmarshal(cached, &res)
			res.Source = "Cache (Idempotent)"
			return &res, nil
		}

		locked, err := uc.idempotencyStore.Lock(ctx, req.IdempotencyKey)
		if err != nil || !locked {
			return nil, errors.New("request in progress or locked")
		}
		defer uc.idempotencyStore.Unlock(ctx, req.IdempotencyKey)
	}

	acc, err := uc.repo.Get(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	if acc.Balance < req.Amount {
		return nil, errors.New("insufficient balance")
	}

	newBalance := acc.Balance - req.Amount
	err = uc.repo.UpdateBalance(ctx, req.UserID, newBalance)
	if err != nil {
		return nil, err
	}

	res := &ChargeResponse{
		Status:  "success",
		Balance: newBalance,
		Source:  "DB (Freshly Processed)",
	}

	if req.IdempotencyKey != "" {
		serialized, _ := json.Marshal(res)
		uc.idempotencyStore.SaveResult(ctx, req.IdempotencyKey, serialized)
	}

	return res, nil
}
