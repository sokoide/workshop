package domain

import "time"

// Secret represents a secret stored in the vault.
type Secret struct {
	Key       string
	Value     string
	Version   int
	CreatedAt time.Time
	Metadata  map[string]string
}

// SecretMetadata represents metadata about a secret without its value.
type SecretMetadata struct {
	Key       string
	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
}
