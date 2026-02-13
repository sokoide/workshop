package main

import (
	"context"
	"log"
	"os"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	"github.com/sokoide/workshop/secret_management/internal/infra/vault"
	"github.com/sokoide/workshop/secret_management/internal/usecase"
)

const (
	defaultSecretKey   = "api/external-key"
	defaultAPIEndpoint = "https://api.example.com/v1/data"
)

func main() {
	// Get parameters from command line or use defaults
	secretKey := getEnvOrDefault("SECRET_KEY", defaultSecretKey)
	apiEndpoint := getEnvOrDefault("API_ENDPOINT", defaultAPIEndpoint)

	// Override with command line arguments if provided
	if len(os.Args) > 1 {
		secretKey = os.Args[1]
	}
	if len(os.Args) > 2 {
		apiEndpoint = os.Args[2]
	}

	log.Printf("[INFO] Starting API client demo")
	log.Printf("[INFO] Secret Key: %s", secretKey)
	log.Printf("[INFO] API Endpoint: %s", apiEndpoint)

	// Load Vault configuration from environment with defaults
	cfg := vault.NewConfigFromEnv()

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

	// Create usecases
	sm := usecase.NewSecretManager(repo)
	apiClient := usecase.NewAPIClient(sm)

	// Make API call with secret from Vault
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = apiClient.CallAPI(ctx, secretKey, apiEndpoint)
	if err != nil {
		log.Fatalf("[ERROR] API call failed: %v", err)
	}

	log.Println("[INFO] Demo completed successfully")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
