package main

import "testing"

func TestLRUOwnsValuesAndEvictsLeastRecentlyUsed(t *testing.T) {
	cache := NewLRUCache(2)
	value := []byte("alpha")
	cache.Put("a", value)
	value[0] = 'x'
	cache.Put("b", []byte("bravo"))
	got, ok := cache.Get("a")
	if !ok || string(got) != "alpha" {
		t.Fatalf("stored slice was mutated: %q", got)
	}
	got[0] = 'x'
	if again, _ := cache.Get("a"); string(again) != "alpha" {
		t.Fatalf("returned slice aliases cache: %q", again)
	}
	cache.Put("c", []byte("charlie"))
	if _, ok := cache.Get("b"); ok {
		t.Fatal("least-recently-used entry was not evicted")
	}
}
