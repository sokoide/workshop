package domain

// VeteranService decides who counts as a veteran.
type VeteranService struct{}

// IsVeteran determines if a user is a veteran (5 or more years of service).
func (s VeteranService) IsVeteran(user *User) bool {
	return user.GetTenureYears() >= 5
}
