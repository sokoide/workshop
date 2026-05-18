package entity

import "testing"

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
	thread := &Thread{Owner: "", OwnerOnly: false}
	if err := thread.EnableOwnerOnlyMode("alice"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if thread.Owner != "alice" {
		t.Errorf("Owner = %q, want %q", thread.Owner, "alice")
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
