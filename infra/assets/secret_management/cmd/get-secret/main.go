package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/sokoide/workshop/secret_management/infra/vault"
	"github.com/sokoide/workshop/secret_management/usecase"
)

func main() {
	// Check command line arguments
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <secret-key>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: %s api/external-key\n", os.Args[0])
		os.Exit(1)
	}

	secretKey := os.Args[1]

	// Load Vault configuration from environment with defaults
	cfg := vault.NewConfigFromEnv()

	log.Printf("[DEBUG] Connecting to Vault at %s", cfg.Address)

	// Wait for Vault to be ready
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

	// Retrieve secret
	secret, err := sm.RetrieveSecret(ctx, secretKey)
	if err != nil {
		log.Fatalf("[ERROR] %v", err)
	}

	// Display secret
	fmt.Println("=== Secret Retrieved ===")
	fmt.Printf("Key:     %s\n", secret.Key)
	fmt.Printf("Value:   %s\n", secret.Value)
	fmt.Printf("Version: %d\n", secret.Version)
	if !secret.CreatedAt.IsZero() {
		fmt.Printf("Created: %s\n", secret.CreatedAt.Format(time.RFC3339))
	}
	for k, v := range secret.Metadata {
		fmt.Printf("Meta[%s]: %s\n", k, v)
	}
}
