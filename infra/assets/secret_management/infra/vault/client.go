package vault

import (
	"errors"
	"fmt"
	"os"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
)

// Config holds Vault connection configuration.
type Config struct {
	Address string
	Token   string
}

// NewConfigFromEnv creates a Config from environment variables with defaults.
func NewConfigFromEnv() *Config {
	cfg := &Config{
		Address: os.Getenv("VAULT_ADDR"),
		Token:   os.Getenv("VAULT_TOKEN"),
	}
	if cfg.Address == "" {
		cfg.Address = "http://localhost:8200"
	}
	if cfg.Token == "" {
		cfg.Token = "dev-workshop-token"
	}
	return cfg
}

// NewClient creates a new Vault API client from the given configuration.
func NewClient(cfg *Config) (*vaultapi.Client, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}
	if cfg.Address == "" {
		return nil, errors.New("vault address is required")
	}
	if cfg.Token == "" {
		return nil, errors.New("vault token is required")
	}

	config := vaultapi.DefaultConfig()
	config.Address = cfg.Address

	client, err := vaultapi.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	client.SetToken(cfg.Token)

	return client, nil
}

// WaitForVault waits for Vault to be ready with a timeout.
func WaitForVault(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &vaultapi.Client{}

	for time.Now().Before(deadline) {
		config := vaultapi.DefaultConfig()
		config.Address = addr

		var err error
		client, err = vaultapi.NewClient(config)
		if err == nil {
			// Try to ping Vault
			_, err = client.RawRequest(client.NewRequest("GET", "/v1/sys/health"))
			if err == nil {
				return nil
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("vault did not become ready within %v", timeout)
}
