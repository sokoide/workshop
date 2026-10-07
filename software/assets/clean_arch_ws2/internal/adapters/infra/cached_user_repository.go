package infra

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws2/internal/domain"
)

// userCacheTTL is how long a cached user entry stays fresh.
const userCacheTTL = 10 * time.Minute

// CachedUserRepository decorates a real domain.UserRepository with a cache
// (Decorator pattern). It satisfies the same port, so neither the UseCase
// layer nor the wrapped repository knows that the cache exists.
type CachedUserRepository struct {
	origin domain.UserRepository // Real repository (DB, LDAP, ...)
	cache  CacheStore            // Cache database (in-memory here, Redis in production)
}

func NewCachedUserRepository(origin domain.UserRepository, cache CacheStore) *CachedUserRepository {
	return &CachedUserRepository{origin: origin, cache: cache}
}

// Compile-time check that the decorator satisfies the same port as the origin.
var _ domain.UserRepository = (*CachedUserRepository)(nil)

func (r *CachedUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	// 1. Return if in cache
	if user, err := r.getFromCache(ctx, id); err == nil {
		return user, nil
	}

	// 2. Otherwise, ask the real repository
	user, err := r.origin.FindByID(ctx, id)
	if err == nil {
		if cacheErr := r.saveToCache(ctx, user); cacheErr != nil {
			slog.Warn("failed to save user to cache", "error", cacheErr) // Log cache failures and proceed
		}
	}
	return user, err
}

// Cache helpers — implementation details of this Decorator.
// They are NOT part of the UserRepository interface, so neither the
// UseCase layer nor the wrapped repository knows about the cache store.

func (r *CachedUserRepository) getFromCache(ctx context.Context, id string) (*domain.User, error) {
	data, err := r.cache.Get(ctx, "user:"+id)
	if err != nil {
		return nil, err // Cache miss or cache failure → fall back to the real repository
	}
	var user domain.User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("unmarshal cached user %s: %w", id, err)
	}
	return &user, nil
}

func (r *CachedUserRepository) saveToCache(ctx context.Context, user *domain.User) error {
	data, err := json.Marshal(user)
	if err != nil {
		return err
	}
	// Set a TTL so stale cache entries expire automatically
	return r.cache.Set(ctx, "user:"+user.ID, data, userCacheTTL)
}
