package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestCanceledWaiterDoesNotWaitForFlight(t *testing.T) {
	c := NewCacheAside(newInMemoryCache(), &slowStore{}, time.Minute)
	release := make(chan struct{})
	entered := make(chan struct{})
	flight := c.group.DoChan("key", func() (interface{}, error) { close(entered); <-release; return "shared", nil })
	<-entered
	defer func() { close(release); <-flight }()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := c.Get(ctx, "key"); done <- err }()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled waiter is blocked by another request")
	}
}

func TestConcurrentMissesFetchOnce(t *testing.T) {
	store := &slowStore{}
	c := NewCacheAside(newInMemoryCache(), store, time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := c.Get(context.Background(), "key")
			if err != nil || value != "value-for-key" {
				t.Errorf("got %q, %v", value, err)
			}
		}()
	}
	wg.Wait()
	if store.Hits() != 1 {
		t.Fatalf("store hits: %d", store.Hits())
	}
}
