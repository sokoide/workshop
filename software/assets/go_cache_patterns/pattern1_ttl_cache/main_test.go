package main

import (
	"bytes"
	"sync"
	"testing"
	"time"
)

func TestTTLCacheOwnsValues(t *testing.T) {
	c := NewTTLCache()
	input := []byte("alice")
	c.Set("user", input, time.Hour)
	input[0] = 'x'
	got, ok := c.Get("user")
	if !ok || !bytes.Equal(got, []byte("alice")) {
		t.Fatalf("stored value changed: %q", got)
	}
	got[0] = 'x'
	got, _ = c.Get("user")
	if string(got) != "alice" {
		t.Fatalf("returned value aliases cache: %q", got)
	}
}

func TestTTLCacheExpiryAndReplacement(t *testing.T) {
	c := NewTTLCache()
	c.Set("user", []byte("old"), -time.Second)
	if _, ok := c.Get("user"); ok {
		t.Fatal("expired value returned")
	}
	c.Set("user", []byte("new"), time.Hour)
	if got, ok := c.Get("user"); !ok || string(got) != "new" {
		t.Fatalf("replacement missing: %q", got)
	}
}

func TestTTLCacheConcurrentAccess(t *testing.T) {
	c := NewTTLCache()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				c.Set("key", []byte("value"), time.Second)
				c.Get("key")
			}
		}()
	}
	wg.Wait()
}
