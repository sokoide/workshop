package entity

import "testing"

func TestNewThread(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		thread, err := NewThread(1, "title", "alice")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if thread.Owner != "alice" {
			t.Errorf("Owner = %q, want %q", thread.Owner, "alice")
		}
		if thread.Title != "title" {
			t.Errorf("Title = %q, want %q", thread.Title, "title")
		}
	})

	t.Run("empty title", func(t *testing.T) {
		_, err := NewThread(1, "", "alice")
		if err == nil {
			t.Error("expected error for empty title")
		}
	})

	t.Run("empty owner", func(t *testing.T) {
		_, err := NewThread(1, "title", "")
		if err == nil {
			t.Error("expected error for empty owner")
		}
	})
}

func TestCanPost_OwnerOnlyDisabled(t *testing.T) {
	thread := &Thread{Owner: "alice", OwnerOnly: false}
	if !thread.CanPost("bob") {
		t.Error("anyone can post when OwnerOnly is false")
	}
	if !thread.CanPost("alice") {
		t.Error("owner can always post")
	}
}

func TestCanPost_OwnerOnlyEnabled(t *testing.T) {
	thread := &Thread{Owner: "alice", OwnerOnly: true}
	if thread.CanPost("bob") {
		t.Error("non-owner should be rejected when OwnerOnly is true")
	}
	if !thread.CanPost("alice") {
		t.Error("owner should be allowed when OwnerOnly is true")
	}
}

func TestEnableOwnerOnlyMode_Success(t *testing.T) {
	thread, _ := NewThread(1, "title", "alice")
	thread.OwnerOnly = false // Ensure it's false initially
	if err := thread.EnableOwnerOnlyMode("bob"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if thread.Owner != "bob" {
		t.Errorf("Owner = %q, want %q", thread.Owner, "bob")
	}
	if !thread.OwnerOnly {
		t.Error("OwnerOnly should be true")
	}
}

func TestEnableOwnerOnlyMode_EmptyOwner(t *testing.T) {
	thread := &Thread{}
	err := thread.EnableOwnerOnlyMode("")
	if err == nil {
		t.Fatal("expected error for empty owner")
	}
	if thread.OwnerOnly {
		t.Error("OwnerOnly should remain false on error")
	}
}
