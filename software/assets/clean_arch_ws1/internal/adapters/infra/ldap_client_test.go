package infra

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLDAPClient_Search(t *testing.T) {
	ctx := context.Background()
	created := time.Date(2019, 4, 1, 0, 0, 0, 0, time.UTC)
	client := NewLDAPClient(map[string]LDAPEntry{
		"alice": {UID: "alice", DisplayName: "Alice", CreationDate: created},
	})

	t.Run("resolves a UID to its directory entry", func(t *testing.T) {
		entry, err := client.Search(ctx, "alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if entry != (LDAPEntry{UID: "alice", DisplayName: "Alice", CreationDate: created}) {
			t.Errorf("unexpected entry: %+v", entry)
		}
	})

	t.Run("returns a driver-style error for a missing UID", func(t *testing.T) {
		_, err := client.Search(ctx, "missing")
		if !errors.Is(err, errNoSuchObject) {
			t.Errorf("expected errNoSuchObject, got %v", err)
		}
	})
}
