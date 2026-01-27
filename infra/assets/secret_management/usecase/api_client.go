package usecase

import (
	"context"
	"fmt"
	"log"
	"time"
)

// APIClient demonstrates how to use secrets for external API calls.
// This usecase shows the pattern of retrieving credentials and using them
// for API operations without hardcoding secrets.
type APIClient interface {
	// CallAPI retrieves an API key from the secret manager and makes an API call.
	CallAPI(ctx context.Context, secretKey, apiEndpoint string) error
}

type apiClient struct {
	secretManager SecretManager
	httpClient    HTTPClient
}

// HTTPClient defines the interface for making HTTP requests.
// This allows us to mock the HTTP client in tests.
type HTTPClient interface {
	Get(url string, apiKey string) (int, string, error)
}

// DefaultHTTPClient is a real HTTP client implementation.
type DefaultHTTPClient struct{}

// Get performs an HTTP GET request with the API key in the Authorization header.
func (c *DefaultHTTPClient) Get(url, apiKey string) (int, string, error) {
	// In a real implementation, this would make an actual HTTP call.
	// For the workshop, we simulate a successful response.

	// Simulated API call
	time.Sleep(100 * time.Millisecond)

	return 200, "Simulated response from API", nil
}

// NewAPIClient creates a new APIClient.
func NewAPIClient(sm SecretManager) APIClient {
	return &apiClient{
		secretManager: sm,
		httpClient:    &DefaultHTTPClient{},
	}
}

// CallAPI retrieves the API key from Vault and makes an API call.
func (ac *apiClient) CallAPI(ctx context.Context, secretKey, apiEndpoint string) error {
	log.Printf("[INFO] Retrieving API key from Vault...")

	// Step 1: Retrieve API key from secret manager
	secret, err := ac.secretManager.RetrieveSecret(ctx, secretKey)
	if err != nil {
		return fmt.Errorf("failed to retrieve API key: %w", err)
	}

	apiKey := secret.Value
	log.Printf("[INFO] API key retrieved successfully (version %d)", secret.Version)

	// Step 2: Make API call with the key
	log.Printf("[INFO] Calling external API: %s", apiEndpoint)

	statusCode, responseBody, err := ac.httpClient.Get(apiEndpoint, apiKey)
	if err != nil {
		return fmt.Errorf("API call failed: %w", err)
	}

	log.Printf("[INFO] API call successful - Status: %d", statusCode)
	log.Printf("[INFO] Response: %s", responseBody)

	return nil
}
