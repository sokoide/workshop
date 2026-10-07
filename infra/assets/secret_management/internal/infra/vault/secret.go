package vault

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/sokoide/workshop/secret_management/internal/domain"
)

// KVPath is the default KV v2 secrets engine mount path.
const KVPath = "secret"

// VaultSecretRepository implements domain.SecretRepository using HashiCorp Vault KV v2.
type VaultSecretRepository struct {
	client *api.Client
	path   string // KV v2 mount path
}

// NewVaultSecretRepository creates a new Vault-backed secret repository.
func NewVaultSecretRepository(client *api.Client, path string) (*VaultSecretRepository, error) {
	if client == nil {
		return nil, fmt.Errorf("vault client cannot be nil")
	}
	if path == "" {
		path = KVPath
	}
	return &VaultSecretRepository{
		client: client,
		path:   path,
	}, nil
}

// GetSecret retrieves a secret from Vault KV v2.
func (r *VaultSecretRepository) GetSecret(ctx context.Context, key string) (*domain.Secret, error) {
	secret, err := r.client.KVv2(r.path).Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %q: %w", key, err)
	}

	if secret == nil {
		return nil, fmt.Errorf("secret %q not found", key)
	}

	// Extract value from secret data
	value, ok := secret.Data["value"].(string)
	if !ok {
		return nil, fmt.Errorf("secret %q does not contain a string value", key)
	}

	// Extract metadata
	metadata := make(map[string]string)
	if secret.VersionMetadata != nil {
		metadata["created_time"] = secret.VersionMetadata.CreatedTime.Format(time.RFC3339)
		metadata["version"] = fmt.Sprintf("%d", secret.VersionMetadata.Version)
	}

	var createdAt time.Time
	if secret.VersionMetadata != nil && !secret.VersionMetadata.CreatedTime.IsZero() {
		createdAt = secret.VersionMetadata.CreatedTime
	}

	return &domain.Secret{
		Key:       key,
		Value:     value,
		Version:   secret.VersionMetadata.Version,
		CreatedAt: createdAt,
		Metadata:  metadata,
	}, nil
}

// PutSecret stores a secret in Vault KV v2.
func (r *VaultSecretRepository) PutSecret(ctx context.Context, key, value string) error {
	_, err := r.client.KVv2(r.path).Put(ctx, key, map[string]interface{}{
		"value": value,
	})
	if err != nil {
		return fmt.Errorf("failed to put secret %q: %w", key, err)
	}
	return nil
}

// ListSecrets lists all secrets in Vault KV v2.
func (r *VaultSecretRepository) ListSecrets(ctx context.Context) ([]domain.SecretMetadata, error) {
	// Vault's LIST only returns one level of keys, so walk the tree.
	var result []domain.SecretMetadata
	if err := r.listRecursive(ctx, "", &result); err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	return result, nil
}

// listRecursive appends the metadata of every secret under prefix.
// In a LIST response, Vault marks subdirectories with a trailing "/".
func (r *VaultSecretRepository) listRecursive(ctx context.Context, prefix string, result *[]domain.SecretMetadata) error {
	list, err := r.client.Logical().ListWithContext(ctx, r.path+"/metadata/"+prefix)
	if err != nil {
		return err
	}
	if list == nil || list.Data == nil {
		return nil
	}

	keys, ok := list.Data["keys"].([]interface{})
	if !ok {
		return nil
	}

	for _, k := range keys {
		key, ok := k.(string)
		if !ok {
			continue
		}
		if strings.HasSuffix(key, "/") {
			// Directory: descend into it (key keeps its trailing slash)
			if err := r.listRecursive(ctx, prefix+key, result); err != nil {
				return err
			}
			continue
		}
		// Get metadata for each secret
		metadata, err := r.getMetadata(ctx, prefix+key)
		if err != nil {
			// If we can't get metadata, use default values
			*result = append(*result, domain.SecretMetadata{
				Key:     prefix + key,
				Version: 0,
			})
			continue
		}
		*result = append(*result, *metadata)
	}
	return nil
}

// getMetadata retrieves metadata for a specific secret.
func (r *VaultSecretRepository) getMetadata(ctx context.Context, key string) (*domain.SecretMetadata, error) {
	secret, err := r.client.KVv2(r.path).Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if secret == nil || secret.VersionMetadata == nil {
		return nil, fmt.Errorf("no metadata available for secret %q", key)
	}

	vm := secret.VersionMetadata

	return &domain.SecretMetadata{
		Key:       key,
		Version:   vm.Version,
		CreatedAt: vm.CreatedTime,
		UpdatedAt: vm.CreatedTime, // KV v2 uses CreatedTime for both
	}, nil
}
