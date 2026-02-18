package vault

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVaultSecretRepository_Integration runs integration tests against a real Vault instance.
// These tests require Vault to be running on localhost:8200 with the dev token.
//
// To run these tests:
//
//	make vault-up
//	vault secrets enable -path=secret kv-v2
//	go test ./infra/vault/... -run TestVaultSecretRepository_Integration
func TestVaultSecretRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Check if Vault is available
	ctx := context.Background()
	cfg := NewConfigFromEnv()

	err := WaitForVault(cfg.Address, 5*time.Second)
	if err != nil {
		t.Skipf("vault not available: %v", err)
	}

	client, err := NewClient(cfg)
	require.NoError(t, err, "failed to create vault client")

	repo, err := NewVaultSecretRepository(client, "secret")
	require.NoError(t, err, "failed to create repository")

	// Test: PutSecret
	t.Run("PutSecret", func(t *testing.T) {
		testKey := "test/integration-put"
		testValue := "test-value-" + time.Now().Format(time.RFC3339Nano)

		err := repo.PutSecret(ctx, testKey, testValue)
		require.NoError(t, err, "failed to put secret")

		// Verify the secret was stored
		secret, err := repo.GetSecret(ctx, testKey)
		require.NoError(t, err, "failed to get secret after put")
		assert.Equal(t, testKey, secret.Key)
		assert.Equal(t, testValue, secret.Value)
	})

	// Test: GetSecret
	t.Run("GetSecret", func(t *testing.T) {
		testKey := "test/integration-get"
		testValue := "test-value-get"

		// First, put a secret
		err := repo.PutSecret(ctx, testKey, testValue)
		require.NoError(t, err)

		// Then, get it
		secret, err := repo.GetSecret(ctx, testKey)
		require.NoError(t, err, "failed to get secret")
		assert.Equal(t, testKey, secret.Key)
		assert.Equal(t, testValue, secret.Value)
		assert.Greater(t, secret.Version, 0)
	})

	// Test: GetSecret_NotFound
	t.Run("GetSecret_NotFound", func(t *testing.T) {
		_, err := repo.GetSecret(ctx, "test/nonexistent-key")
		assert.Error(t, err, "expected error for nonexistent key")
		assert.Contains(t, err.Error(), "not found")
	})

	// Test: ListSecrets
	t.Run("ListSecrets", func(t *testing.T) {
		testPrefix := "test/integration-list-"
		testValues := []string{"value1", "value2", "value3"}

		// Put multiple secrets
		for i, value := range testValues {
			key := testPrefix + string(rune('a'+i))
			err := repo.PutSecret(ctx, key, value)
			require.NoError(t, err)
		}

		// List all secrets
		secrets, err := repo.ListSecrets(ctx)
		require.NoError(t, err, "failed to list secrets")

		// Filter for our test secrets
		found := 0
		for _, s := range secrets {
			if len(s.Key) >= len(testPrefix) && s.Key[:len(testPrefix)] == testPrefix {
				found++
			}
		}
		assert.GreaterOrEqual(t, found, len(testValues), "expected at least our test secrets")
	})

	// Test: Secret Versioning
	t.Run("SecretVersioning", func(t *testing.T) {
		testKey := "test/integration-version"
		value1 := "version-1"
		value2 := "version-2"

		// Put first version
		err := repo.PutSecret(ctx, testKey, value1)
		require.NoError(t, err)

		secret1, err := repo.GetSecret(ctx, testKey)
		require.NoError(t, err)
		assert.Equal(t, value1, secret1.Value)
		version1 := secret1.Version

		// Put second version
		err = repo.PutSecret(ctx, testKey, value2)
		require.NoError(t, err)

		secret2, err := repo.GetSecret(ctx, testKey)
		require.NoError(t, err)
		assert.Equal(t, value2, secret2.Value)
		assert.Equal(t, version1+1, secret2.Version, "version should increment")
	})
}

// TestNewConfigFromEnv tests configuration loading from environment variables.
func TestNewConfigFromEnv(t *testing.T) {
	// Save original values
	origAddr := os.Getenv("VAULT_ADDR")
	origToken := os.Getenv("VAULT_TOKEN")
	defer func() {
		if origAddr != "" {
			os.Setenv("VAULT_ADDR", origAddr)
		} else {
			os.Unsetenv("VAULT_ADDR")
		}
		if origToken != "" {
			os.Setenv("VAULT_TOKEN", origToken)
		} else {
			os.Unsetenv("VAULT_TOKEN")
		}
	}()

	t.Run("WithEnvVars", func(t *testing.T) {
		os.Setenv("VAULT_ADDR", "http://vault.example.com:8200")
		os.Setenv("VAULT_TOKEN", "test-token-123")

		cfg := NewConfigFromEnv()
		assert.Equal(t, "http://vault.example.com:8200", cfg.Address)
		assert.Equal(t, "test-token-123", cfg.Token)
	})

	t.Run("WithDefaults", func(t *testing.T) {
		os.Unsetenv("VAULT_ADDR")
		os.Unsetenv("VAULT_TOKEN")

		cfg := NewConfigFromEnv()
		assert.Equal(t, "http://localhost:8200", cfg.Address)
		assert.Equal(t, "dev-workshop-token", cfg.Token)
	})
}
