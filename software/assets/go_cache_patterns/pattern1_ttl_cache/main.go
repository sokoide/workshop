package main

import (
	"fmt"
	"sync"
	"time"
)

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

type TTLCache struct {
	mu    sync.RWMutex
	items map[string]cacheEntry
}

func NewTTLCache() *TTLCache {
	return &TTLCache{items: make(map[string]cacheEntry)}
}

func (c *TTLCache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheEntry{value: value, expiresAt: time.Now().Add(ttl)}
}

func (c *TTLCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}
	return entry.value, true
}

func main() {
	cache := NewTTLCache()
	cache.Set("session:123", []byte("alice"), 2*time.Second)

	if value, ok := cache.Get("session:123"); ok {
		fmt.Printf("hit: %s\n", value)
	}

	time.Sleep(3 * time.Second)
	if _, ok := cache.Get("session:123"); !ok {
		fmt.Println("miss: expired")
	}
}
