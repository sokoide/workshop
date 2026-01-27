package domain

import "context"

// SecretRepository defines the interface for secret storage operations.
// This is a pure Go interface with no external dependencies,
// making it easy to mock for testing and swap implementations.
type SecretRepository interface {
	// GetSecret retrieves a secret by its key.
	GetSecret(ctx context.Context, key string) (*Secret, error)

	// PutSecret stores a secret with the given key and value.
	PutSecret(ctx context.Context, key, value string) error

	// ListSecrets returns metadata for all secrets.
	ListSecrets(ctx context.Context) ([]SecretMetadata, error)
}
