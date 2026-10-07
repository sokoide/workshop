package infra

import (
	"context"
	"errors"
	"sync"
	"time"
)

// errCacheMiss is returned by CacheStore.Get when the key is absent or
// expired. It never leaves the Infra layer: the decorator treats every cache
// error as a miss and falls back to the real repository.
var errCacheMiss = errors.New("cache miss")

// CacheStore is the key-value store used by CachedUserRepository. It is an
// infra-internal abstraction: in production you can implement it with Redis
// (e.g. github.com/redis/go-redis/v9) without touching any other layer.
type CacheStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// MemoryCacheStore is an in-memory CacheStore with TTL support. It plays the
// role of the cache database in this workshop.
type MemoryCacheStore struct {
	mu      sync.Mutex
	entries map[string]memoryCacheEntry
	now     func() time.Time // injectable clock for tests
}

type memoryCacheEntry struct {
	value     []byte
	expiresAt time.Time
}

func NewMemoryCacheStore() *MemoryCacheStore {
	return &MemoryCacheStore{
		entries: map[string]memoryCacheEntry{},
		now:     time.Now,
	}
}

func (s *MemoryCacheStore) Get(ctx context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[key]
	if !ok || !s.now().Before(entry.expiresAt) {
		// Absent or expired — indistinguishable from the caller's viewpoint.
		// Drop the stale entry so the map does not grow without bound
		// (a real cache like Redis evicts expired keys itself).
		delete(s.entries, key)
		return nil, errCacheMiss
	}
	return entry.value, nil
}

func (s *MemoryCacheStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries[key] = memoryCacheEntry{value: value, expiresAt: s.now().Add(ttl)}
	return nil
}
