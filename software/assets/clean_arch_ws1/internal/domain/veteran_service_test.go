package domain

import (
	"testing"
	"time"
)

func TestVeteranService_IsVeteran(t *testing.T) {
	svc := VeteranService{}
	now := time.Now()

	tests := []struct {
		name     string
		joinedAt time.Time
		want     bool
	}{
		{"ten years of service is a veteran", now.AddDate(-10, 0, 0), true},
		{"five years of service is a veteran", now.AddDate(-5, 0, 0), true},
		{"four years of service is not a veteran", now.AddDate(-4, 0, 0), false},
		{"one year of service is not a veteran", now.AddDate(-1, 0, 0), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{ID: "u1", Name: "TestUser", JoinedAt: tt.joinedAt}
			if got := svc.IsVeteran(user); got != tt.want {
				t.Errorf("IsVeteran(joinedAt=%s) = %v, want %v", tt.joinedAt, got, tt.want)
			}
		})
	}
}
