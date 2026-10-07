package domain

import "time"

// User is the user entity.
type User struct {
	ID       string
	Name     string
	JoinedAt time.Time // Joining date
}

// GetTenureYears returns the number of years of service as of now.
func (u *User) GetTenureYears() int {
	now := time.Now()
	years := now.Year() - u.JoinedAt.Year()
	if now.YearDay() < u.JoinedAt.YearDay() {
		years--
	}
	return years
}
