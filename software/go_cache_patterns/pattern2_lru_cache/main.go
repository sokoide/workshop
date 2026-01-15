package main

import (
	"container/list"
	"fmt"
	"sync"
)

type lruEntry struct {
	key   string
	value []byte
}

type LRUCache struct {
	mu       sync.Mutex
	capacity int
	items    map[string]*list.Element
	order    *list.List
}

func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		order:    list.New(),
	}
}

func (c *LRUCache) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	element, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(element)
	return element.Value.(lruEntry).value, true
}

func (c *LRUCache) Put(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if element, ok := c.items[key]; ok {
		element.Value = lruEntry{key: key, value: value}
		c.order.MoveToFront(element)
		return
	}

	element := c.order.PushFront(lruEntry{key: key, value: value})
	c.items[key] = element

	if c.order.Len() > c.capacity {
		c.removeOldest()
	}
}

func (c *LRUCache) removeOldest() {
	tail := c.order.Back()
	if tail == nil {
		return
	}
	entry := tail.Value.(lruEntry)
	delete(c.items, entry.key)
	c.order.Remove(tail)
}

func main() {
	cache := NewLRUCache(2)
	cache.Put("a", []byte("alpha"))
	cache.Put("b", []byte("bravo"))

	if value, ok := cache.Get("a"); ok {
		fmt.Printf("hit a: %s\n", value)
	}

	cache.Put("c", []byte("charlie"))

	if _, ok := cache.Get("b"); !ok {
		fmt.Println("b evicted")
	}
}
