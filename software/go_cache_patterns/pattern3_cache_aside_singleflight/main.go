package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type cacheItem struct {
	value     string
	expiresAt time.Time
}

type inMemoryCache struct {
	mu    sync.RWMutex
	items map[string]cacheItem
}

func newInMemoryCache() *inMemoryCache {
	return &inMemoryCache{items: make(map[string]cacheItem)}
}

func (c *inMemoryCache) Get(key string) (string, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return "", false
	}
	if time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return "", false
	}
	return item.value, true
}

func (c *inMemoryCache) Set(key, value string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheItem{value: value, expiresAt: time.Now().Add(ttl)}
}

type slowStore struct {
	mu   sync.Mutex
	hits int
}

func (s *slowStore) Fetch(ctx context.Context, key string) (string, error) {
	s.mu.Lock()
	s.hits++
	s.mu.Unlock()

	select {
	case <-time.After(300 * time.Millisecond):
		return "value-for-" + key, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (s *slowStore) Hits() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits
}

type CacheAside struct {
	cache  *inMemoryCache
	store  *slowStore
	group  singleflight.Group
	cacheT time.Duration
}

func NewCacheAside(cache *inMemoryCache, store *slowStore, ttl time.Duration) *CacheAside {
	return &CacheAside{cache: cache, store: store, cacheT: ttl}
}

func (c *CacheAside) Get(ctx context.Context, key string) (string, error) {
	if value, ok := c.cache.Get(key); ok {
		return value, nil
	}

	// singleflight collapses concurrent cache misses into one store fetch.
	value, err, _ := c.group.Do(key, func() (interface{}, error) {
		if value, ok := c.cache.Get(key); ok {
			return value, nil
		}
		fresh, err := c.store.Fetch(ctx, key)
		if err != nil {
			return "", err
		}
		c.cache.Set(key, fresh, c.cacheT)
		return fresh, nil
	})
	if err != nil {
		return "", err
	}
	return value.(string), nil
}

func main() {
	cache := newInMemoryCache()
	store := &slowStore{}
	cacheAside := NewCacheAside(cache, store, 2*time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			value, err := cacheAside.Get(context.Background(), "user:42")
			if err != nil {
				fmt.Printf("worker %d error: %v\n", id, err)
				return
			}
			fmt.Printf("worker %d got %s\n", id, value)
		}(i)
	}

	wg.Wait()
	fmt.Printf("store hits: %d\n", store.Hits())
}
