package vault

import (
	"errors"
	"fmt"
	"net/http"
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
//
// A Vault is considered ready when /v1/sys/health returns
// 200 (initialized, unsealed, active) or 429 (unsealed standby).
// 501/503 (uninitialized/sealed) are not ready and are retried.
func WaitForVault(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		config := vaultapi.DefaultConfig()
		config.Address = addr

		client, err := vaultapi.NewClient(config)
		if err == nil {
			ready, err := vaultHealthy(client)
			if err == nil && ready {
				return nil
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("vault did not become ready within %v", timeout)
}

// vaultHealthy reports whether Vault can serve requests, based on
// the /v1/sys/health status code.
func vaultHealthy(client *vaultapi.Client) (bool, error) {
	resp, err := client.RawRequest(client.NewRequest("GET", "/v1/sys/health"))
	if resp != nil {
		defer resp.Body.Close() // RawRequest returns the response even on error
	}
	if err != nil {
		// RawRequest reports 501 (uninitialized) and 503 (sealed) as errors;
		// retry until ready. A 429 reply on /v1/sys/health (unsealed standby)
		// is NOT an error in the vault API library.
		return false, err
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusTooManyRequests: // 200: active, 429: unsealed standby
		return true, nil
	default:
		return false, nil
	}
}
