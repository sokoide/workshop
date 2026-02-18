package main

import (
	"context"
	"fmt"
	"log"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/sokoide/workshop/secret_management/internal/infra/vault"
	"github.com/sokoide/workshop/secret_management/internal/usecase"
)

func main() {
	// Load Vault configuration from environment with defaults
	cfg := vault.NewConfigFromEnv()

	log.Printf("[DEBUG] Connecting to Vault at %s", cfg.Address)

	// Wait for Vault to be ready
	err := vault.WaitForVault(cfg.Address, 5*time.Second)
	if err != nil {
		log.Fatalf("[ERROR] Vault is not available: %v", err)
	}

	// Create Vault client
	client, err := vaultapi.NewClient(&vaultapi.Config{
		Address: cfg.Address,
	})
	if err != nil {
		log.Fatalf("[ERROR] Failed to create Vault client: %v", err)
	}
	client.SetToken(cfg.Token)

	// Create repository
	repo, err := vault.NewVaultSecretRepository(client, "secret")
	if err != nil {
		log.Fatalf("[ERROR] Failed to create repository: %v", err)
	}

	// Create usecase
	sm := usecase.NewSecretManager(repo)

	// List all secrets
	ctx := context.Background()
	secrets, err := sm.ListAllSecrets(ctx)
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}

	// Display secrets
	fmt.Println("=== Secrets in Vault ===")
	if len(secrets) == 0 {
		fmt.Println("No secrets found.")
		fmt.Println("\nStore a secret with:")
		fmt.Println("  go run cmd/put-secret/main.go <key> <value>")
	} else {
		for i, secret := range secrets {
			fmt.Printf("%d. %s (v%d)\n", i+1, secret.Key, secret.Version)
			if !secret.CreatedAt.IsZero() {
				fmt.Printf("   Created: %s\n", secret.CreatedAt.Format(time.RFC3339))
			}
		}
	}
}
