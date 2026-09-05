package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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

type storedCharge struct {
	Request  ChargeRequest  `json:"request"`
	Response ChargeResponse `json:"response"`
}

func (uc *ChargeUsecase) cachedResult(ctx context.Context, req ChargeRequest) (*ChargeResponse, error) {
	cached, err := uc.idempotencyStore.GetResult(ctx, req.IdempotencyKey)
	if err != nil {
		return nil, fmt.Errorf("read idempotency result: %w", err)
	}
	if cached == nil {
		return nil, nil
	}
	var stored storedCharge
	if err := json.Unmarshal(cached, &stored); err != nil {
		return nil, fmt.Errorf("decode idempotency result: %w", err)
	}
	if stored.Request != req || stored.Response.Status != "success" {
		return nil, errors.New("idempotency key conflicts with this request or stored result is invalid")
	}
	stored.Response.Source = "Cache (Idempotent)"
	return &stored.Response, nil
}

func (uc *ChargeUsecase) Execute(ctx context.Context, req ChargeRequest) (response *ChargeResponse, err error) {
	if strings.TrimSpace(req.UserID) == "" || req.Amount <= 0 {
		return nil, errors.New("user ID and positive amount are required")
	}
	if req.IdempotencyKey != "" {
		if cached, err := uc.cachedResult(ctx, req); err != nil || cached != nil {
			return cached, err
		}
		token, err := uc.idempotencyStore.Lock(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("lock idempotency key: %w", err)
		}
		if token == "" {
			return nil, errors.New("request in progress or locked")
		}
		defer func() {
			// Cleanup must still run after the request context is canceled.
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			defer cancel()
			if unlockErr := uc.idempotencyStore.Unlock(cleanup, req.IdempotencyKey, token); unlockErr != nil {
				err = errors.Join(err, fmt.Errorf("unlock idempotency key: %w", unlockErr))
			}
		}()
		// Another request may have completed between our first read and lock acquisition.
		if cached, err := uc.cachedResult(ctx, req); err != nil || cached != nil {
			return cached, err
		}
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
		serialized, err := json.Marshal(storedCharge{Request: req, Response: *res})
		if err != nil {
			return nil, fmt.Errorf("encode completed charge: %w", err)
		}
		if err := uc.idempotencyStore.SaveResult(ctx, req.IdempotencyKey, serialized); err != nil {
			return nil, fmt.Errorf("balance updated but result save failed; reconcile before retrying: %w", err)
		}
	}

	return res, nil
}
