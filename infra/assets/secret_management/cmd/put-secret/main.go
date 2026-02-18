package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/sokoide/workshop/secret_management/internal/infra/vault"
	"github.com/sokoide/workshop/secret_management/internal/usecase"
)

func main() {
	// Check command line arguments
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <secret-key> <secret-value>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s api/payment-gateway sk-pay-xyz789\n", os.Args[0])
		os.Exit(1)
	}

	secretKey := os.Args[1]
	secretValue := os.Args[2]

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

	// Store secret
	ctx := context.Background()
	err = sm.StoreSecret(ctx, secretKey, secretValue)
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}

	fmt.Println("=== Secret Stored Successfully ===")
	fmt.Printf("Key:   %s\n", secretKey)
	fmt.Printf("Value: %s\n", secretValue)
	fmt.Println("\nYou can retrieve it with:")
	fmt.Printf("  go run cmd/get-secret/main.go %s\n", secretKey)
}
