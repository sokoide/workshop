package main

import (
	"fmt"
	"sync"
)

type record struct {
	value string
}

type memoryStore struct {
	mu   sync.RWMutex
	data map[string]record
}

func newMemoryStore() *memoryStore {
	return &memoryStore{data: make(map[string]record)}
}

func (s *memoryStore) Put(key string, value record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
	return nil
}

func (s *memoryStore) Get(key string) (record, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, ok := s.data[key]
	return value, ok
}

type memoryCache struct {
	mu   sync.RWMutex
	data map[string]record
}

func newMemoryCache() *memoryCache {
	return &memoryCache{data: make(map[string]record)}
}

func (c *memoryCache) Get(key string) (record, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	value, ok := c.data[key]
	return value, ok
}

func (c *memoryCache) Set(key string, value record) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = value
}

type writeThroughCache struct {
	store *memoryStore
	cache *memoryCache
}

func newWriteThroughCache(store *memoryStore, cache *memoryCache) *writeThroughCache {
	return &writeThroughCache{store: store, cache: cache}
}

func (w *writeThroughCache) Put(key string, value record) error {
	if err := w.store.Put(key, value); err != nil {
		return err
	}
	w.cache.Set(key, value)
	return nil
}

func (w *writeThroughCache) Get(key string) (record, bool) {
	if value, ok := w.cache.Get(key); ok {
		return value, true
	}

	value, ok := w.store.Get(key)
	if !ok {
		return record{}, false
	}
	w.cache.Set(key, value)
	return value, true
}

func main() {
	store := newMemoryStore()
	cache := newMemoryCache()
	writeThrough := newWriteThroughCache(store, cache)

	if err := writeThrough.Put("order:100", record{value: "paid"}); err != nil {
		fmt.Printf("write error: %v\n", err)
		return
	}

	if value, ok := writeThrough.Get("order:100"); ok {
		fmt.Printf("read: %s\n", value.value)
	}
}
