package usecase

import (
	"context"
	"fmt"
	"log"

	"github.com/sokoide/workshop/secret_management/internal/domain"
)

// SecretManager provides high-level secret management operations.
// This usecase layer orchestrates business logic using the domain repository interface.
type SecretManager interface {
	// RetrieveSecret fetches a secret by key with error handling.
	RetrieveSecret(ctx context.Context, key string) (*domain.Secret, error)

	// StoreSecret saves a secret with validation.
	StoreSecret(ctx context.Context, key, value string) error

	// ListAllSecrets returns all secret metadata.
	ListAllSecrets(ctx context.Context) ([]domain.SecretMetadata, error)
}

// secretManager implements SecretManager interface.
type secretManager struct {
	repo domain.SecretRepository
}

// NewSecretManager creates a new SecretManager.
func NewSecretManager(repo domain.SecretRepository) SecretManager {
	if repo == nil {
		panic("repository cannot be nil")
	}
	return &secretManager{
		repo: repo,
	}
}

// RetrieveSecret fetches a secret with business logic validation.
func (sm *secretManager) RetrieveSecret(ctx context.Context, key string) (*domain.Secret, error) {
	if key == "" {
		return nil, fmt.Errorf("secret key cannot be empty")
	}

	log.Printf("[INFO] Retrieving secret: %s", key)

	secret, err := sm.repo.GetSecret(ctx, key)
	if err != nil {
		log.Printf("[ERROR] Failed to retrieve secret %q: %v", key, err)
		return nil, fmt.Errorf("failed to retrieve secret %q: %w", key, err)
	}

	log.Printf("[INFO] Successfully retrieved secret: %s (version %d)", key, secret.Version)
	return secret, nil
}

// StoreSecret saves a secret with validation.
func (sm *secretManager) StoreSecret(ctx context.Context, key, value string) error {
	if key == "" {
		return fmt.Errorf("secret key cannot be empty")
	}
	if value == "" {
		return fmt.Errorf("secret value cannot be empty")
	}

	log.Printf("[INFO] Storing secret: %s", key)

	err := sm.repo.PutSecret(ctx, key, value)
	if err != nil {
		log.Printf("[ERROR] Failed to store secret %q: %v", key, err)
		return fmt.Errorf("failed to store secret %q: %w", key, err)
	}

	log.Printf("[INFO] Successfully stored secret: %s", key)
	return nil
}

// ListAllSecrets returns all secret metadata.
func (sm *secretManager) ListAllSecrets(ctx context.Context) ([]domain.SecretMetadata, error) {
	log.Printf("[INFO] Listing all secrets")

	secrets, err := sm.repo.ListSecrets(ctx)
	if err != nil {
		log.Printf("[ERROR] Failed to list secrets: %v", err)
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	log.Printf("[INFO] Found %d secret(s)", len(secrets))
	return secrets, nil
}
