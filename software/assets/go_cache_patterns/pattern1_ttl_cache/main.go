package main

import (
	"bytes"
	"fmt"
	"sync"
	"time"
)

type cacheEntry struct {
	value     []byte
	expiresAt time.Time
}

type TTLCache struct {
	mu    sync.Mutex
	items map[string]cacheEntry
}

func NewTTLCache() *TTLCache {
	return &TTLCache{items: make(map[string]cacheEntry)}
}

func (c *TTLCache) Set(key string, value []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheEntry{value: bytes.Clone(value), expiresAt: time.Now().Add(ttl)}
}

func (c *TTLCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if !time.Now().Before(entry.expiresAt) {
		delete(c.items, key)
		return nil, false
	}
	return bytes.Clone(entry.value), true
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
