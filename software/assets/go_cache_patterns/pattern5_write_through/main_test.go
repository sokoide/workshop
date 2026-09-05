package main

import (
	"fmt"
	"sync"
	"testing"
)

func TestConcurrentWritesKeepCacheAndStoreConsistent(t *testing.T) {
	for round := 0; round < 100; round++ {
		store, cache := newMemoryStore(), newMemoryCache()
		w := newWriteThroughCache(store, cache)
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				if err := w.Put("key", record{value: fmt.Sprint(i)}); err != nil {
					t.Error(err)
				}
				w.Get("key")
			}(i)
		}
		wg.Wait()
		cached, _ := w.Get("key")
		stored, _ := store.Get("key")
		if cached != stored {
			t.Fatalf("cache=%v store=%v", cached, stored)
		}
	}
}
